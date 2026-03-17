package order

import (
	"io"
	"log/slog"
	"net/http"

	"github.com/11SF/dogjohn-be/app/order/access"
	"github.com/11SF/go-common/response"
	"github.com/gin-gonic/gin"
	"github.com/samber/lo"
)

type SubmitOrderResponse struct {
	OrderID string `json:"orderId"`
}

func (h *handler) SubmitOrder(c *gin.Context) {
	ctx := c.Request.Context()

	priceID := c.PostForm("priceId")
	customerName := c.PostForm("customerName")

	if priceID == "" || customerName == "" {
		response.NewGinResponseError(c, http.StatusBadRequest,
			response.NewError(response.BadRequestCode, "Bad Request"))
		return
	}

	fileHeader, err := c.FormFile("slipImage")
	if err != nil {
		response.NewGinResponseError(c, http.StatusBadRequest,
			response.NewError(response.BadRequestCode, "Bad Request"))
		return
	}

	file, err := fileHeader.Open()
	if err != nil {
		response.NewGinResponseError(c, http.StatusInternalServerError,
			response.NewError(response.GenericError, "Internal Server Error"))
		return
	}
	defer file.Close()

	slipBytes, err := io.ReadAll(file)
	if err != nil {
		response.NewGinResponseError(c, http.StatusInternalServerError,
			response.NewError(response.GenericError, "Internal Server Error"))
		return
	}

	// 1. Create order with PENDING status
	order, err := h.orderRepo.SubmitOrder(ctx, access.SubmitOrderRequest{
		PriceID:      priceID,
		CustomerName: customerName,
		SlipImage:    slipBytes,
	})
	if err != nil {
		response.NewGinResponseError(c, http.StatusInternalServerError,
			response.NewError(response.GenericError, "Internal Server Error"))
		return
	}

	priceDetail, err := h.paymentRepo.GetPrice(ctx, priceID)
	if err != nil {
		response.NewGinResponseError(c, http.StatusInternalServerError,
			response.NewError(response.GenericError, "Internal Server Error"))
		return
	}

	if priceDetail == nil {
		response.NewGinResponseError(c, http.StatusBadRequest,
			response.NewError(response.BadRequestCode, "Invalid priceId"))
		return
	}

	// 2. Verify payment slip
	slipResp, err := h.slipOK.VerifySlip(ctx, access.VerifySlipRequest{
		Amount:    lo.ToPtr(priceDetail.Price),
		SlipImage: slipBytes,
		Log:       true,
	})
	if err != nil {
		_ = h.orderRepo.UpdateOrderFailed(ctx, order.OrderID, err.Error())
		response.NewGinResponseError(c, http.StatusInternalServerError,
			response.NewError(response.GenericError, "Internal Server Error"))
		return
	}

	// 3. Save raw SlipOK response
	if logErr := h.orderRepo.SavePaymentTxnLog(ctx, order.OrderID, slipResp); logErr != nil {
		slog.ErrorContext(ctx, "failed to save payment txn log", "error", logErr, "orderID", order.OrderID)
	}

	// 4. Check slip validity
	if slipErr := access.SlipOKError(slipResp); slipErr != nil {
		_ = h.orderRepo.UpdateOrderFailed(ctx, order.OrderID, slipErr.Error())
		response.NewGinResponseError(c, http.StatusUnprocessableEntity,
			response.NewError(response.BadRequestCode, "Unprocessable Entity"))
		return
	}

	txnRef := slipResp.Data.TransRef

	// 5. Check for duplicate slip
	isDup, err := h.orderRepo.IsPaymentTxnRefDuplicate(ctx, txnRef)
	if err != nil {
		_ = h.orderRepo.UpdateOrderFailed(ctx, order.OrderID, "failed to check duplicate txn ref")
		response.NewGinResponseError(c, http.StatusInternalServerError,
			response.NewError(response.GenericError, "Internal Server Error"))
		return
	}
	if isDup {
		_ = h.orderRepo.UpdateOrderFailed(ctx, order.OrderID, "duplicate slip")
		response.NewGinResponseError(c, http.StatusUnprocessableEntity,
			response.NewError(response.BadRequestCode, "Unprocessable Entity"))
		return
	}

	// 6. Update order to PROCESSING with txnRef
	if err := h.orderRepo.UpdateOrderProcessing(ctx, order.OrderID, txnRef); err != nil {
		response.NewGinResponseError(c, http.StatusInternalServerError,
			response.NewError(response.GenericError, "Internal Server Error"))
		return
	}

	// 7. Trigger dog feeder (best effort)
	if err := h.haClient.TriggerFeeder(ctx); err != nil {
		slog.ErrorContext(ctx, "failed to trigger feeder", "error", err, "orderID", order.OrderID)
	}

	response.NewGinResponse(c, http.StatusOK, SubmitOrderResponse{OrderID: order.OrderID})
}
