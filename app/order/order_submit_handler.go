package order

import (
	"io"
	"log/slog"
	"net/http"
	"strings"

	"github.com/11SF/dogjohn-be/app/order/access"
	"github.com/11SF/go-common/logger"
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

	if priceID == "" {
		logger.Error(ctx, "failed to submit order: missing priceId", slog.String("tag", "submit order"))
		response.NewGinResponseError(c, http.StatusBadRequest,
			response.NewError(response.BadRequestCode, "Bad Request"))
		return
	}

	if customerName == "" {
		customerName = "Anonymous"
	}

	fileHeader, err := c.FormFile("slipImage")
	if err != nil {
		logger.Error(ctx, "failed to submit order: missing slipImage", slog.String("err", err.Error()), slog.String("tag", "submit order"))
		response.NewGinResponseError(c, http.StatusBadRequest,
			response.NewError(response.BadRequestCode, "Bad Request"))
		return
	}

	file, err := fileHeader.Open()
	if err != nil {
		logger.Error(ctx, "failed to submit order: failed to open slipImage", slog.String("err", err.Error()), slog.String("tag", "submit order"))
		response.NewGinResponseError(c, http.StatusInternalServerError,
			response.NewError(response.GenericError, "Internal Server Error"))
		return
	}
	defer file.Close()

	slipBytes, err := io.ReadAll(file)
	if err != nil {
		logger.Error(ctx, "failed to submit order: failed to read slipImage", slog.String("err", err.Error()), slog.String("tag", "submit order"))
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
		logger.Error(ctx, "failed to submit order: failed to create order", slog.String("err", err.Error()), slog.String("tag", "submit order"))
		response.NewGinResponseError(c, http.StatusInternalServerError,
			response.NewError(response.GenericError, "Internal Server Error"))
		return
	}

	priceDetail, err := h.paymentRepo.GetPrice(ctx, priceID)
	if err != nil {
		logger.Error(ctx, "failed to submit order: failed to get price details", slog.String("err", err.Error()), slog.String("tag", "submit order"))
		response.NewGinResponseError(c, http.StatusInternalServerError,
			response.NewError(response.GenericError, "Internal Server Error"))
		return
	}

	if priceDetail == nil {
		logger.Error(ctx, "failed to submit order: invalid priceId", slog.String("tag", "submit order"))
		response.NewGinResponseError(c, http.StatusBadRequest,
			response.NewError(response.BadRequestCode, "Invalid priceId"))
		return
	}

	paymentDetail, err := h.paymentRepo.GetDetails(ctx)
	if err != nil {
		logger.Error(ctx, "failed to submit order: failed to get payment details", slog.String("err", err.Error()), slog.String("tag", "submit order"))
		response.NewGinResponseError(c, http.StatusInternalServerError,
			response.NewError(response.GenericError, "Internal Server Error"))
		return
	}

	// 2. Verify payment slip
	slipResp, err := h.slipOK.VerifySlip(ctx, access.VerifySlipRequest{
		Amount:    lo.ToPtr(priceDetail.Price),
		SlipImage: slipBytes,
		Log:       false,
	})
	if err != nil {
		logger.Error(ctx, "failed to submit order: failed to verify slip", slog.String("err", err.Error()), slog.String("tag", "submit order"))
		_ = h.orderRepo.UpdateOrderFailed(ctx, order.OrderID, err.Error())
		response.NewGinResponseError(c, http.StatusInternalServerError,
			response.NewError(response.GenericError, "Internal Server Error"))
		return
	}

	// 3. Save raw SlipOK response
	if logErr := h.orderRepo.SavePaymentTxnLog(ctx, order.OrderID, slipResp); logErr != nil {
		logger.Error(ctx, "failed to save payment txn log", "error", logErr, "orderID", order.OrderID)
	}

	// 4. Check slip validity
	if slipResp.Data == nil {
		_ = h.orderRepo.UpdateOrderFailed(ctx, order.OrderID, "slipok: missing data in response")
		logger.Error(ctx, "failed to submit order: slipok returned nil data", slog.String("tag", "submit order"))
		response.NewGinResponseError(c, http.StatusInternalServerError,
			response.NewError(response.GenericError, "Internal Server Error"))
		return
	}

	logger.Info(ctx, "slip verification result", slog.Bool("success", slipResp.Success), slog.Any("ok slip response", slipResp), slog.String("tag", "submit order"))

	paidLocalAmount := slipResp.Data.PaidLocalAmount
	receiverValue := lo.If(len(strings.Split(slipResp.Data.Receiver.Proxy.Value, "-")) == 3, strings.Split(slipResp.Data.Receiver.Proxy.Value, "-")[2]).Else(slipResp.Data.Receiver.Proxy.Value)
	txnRef := slipResp.Data.TransRef

	if paidLocalAmount < priceDetail.Price {
		_ = h.orderRepo.UpdateOrderFailed(ctx, order.OrderID, "insufficient amount")
		logger.Error(ctx, "failed to submit order: insufficient amount", slog.Float64("paidLocalAmount", paidLocalAmount), slog.Float64("expectedAmount", priceDetail.Price), slog.String("tag", "submit order"))
		response.NewGinResponseError(c, http.StatusUnprocessableEntity,
			response.NewError(response.BadRequestCode, "Unprocessable Entity"))
		return
	}

	if receiverValue != paymentDetail.PromptPayID[len(paymentDetail.PromptPayID)-4:] {
		_ = h.orderRepo.UpdateOrderFailed(ctx, order.OrderID, "invalid receiver")
		logger.Error(ctx, "failed to submit order: invalid receiver", slog.String("receiverValue", receiverValue), slog.String("expectedReceiver", paymentDetail.PromptPayID[len(paymentDetail.PromptPayID)-4:]), slog.String("tag", "submit order"))
		response.NewGinResponseError(c, http.StatusUnprocessableEntity,
			response.NewError(response.BadRequestCode, "Unprocessable Entity"))
		return
	}

	// 5. Check for duplicate slip
	isDup, err := h.orderRepo.IsPaymentTxnRefDuplicate(ctx, txnRef)
	if err != nil {
		_ = h.orderRepo.UpdateOrderFailed(ctx, order.OrderID, "failed to check duplicate txn ref")
		logger.Error(ctx, "failed to submit order: failed to check duplicate txn ref", slog.String("err", err.Error()), slog.String("tag", "submit order"))
		response.NewGinResponseError(c, http.StatusInternalServerError,
			response.NewError(response.GenericError, "Internal Server Error"))
		return
	}
	if isDup {
		_ = h.orderRepo.UpdateOrderFailed(ctx, order.OrderID, "duplicate slip")
		logger.Error(ctx, "failed to submit order: duplicate slip", slog.String("tag", "submit order"))
		response.NewGinResponseError(c, http.StatusUnprocessableEntity,
			response.NewError(response.BadRequestCode, "Unprocessable Entity"))
		return
	}

	// 6. Update order to PROCESSING with txnRef
	if err := h.orderRepo.UpdateOrderProcessing(ctx, order.OrderID, txnRef); err != nil {
		logger.Error(ctx, "failed to submit order: failed to update order", slog.String("err", err.Error()), slog.String("tag", "submit order"))
		response.NewGinResponseError(c, http.StatusInternalServerError,
			response.NewError(response.GenericError, "Internal Server Error"))
		return
	}

	// 7. Trigger dog feeder (best effort)
	if err := h.haClient.TriggerFeeder(ctx); err != nil {
		logger.Error(ctx, "failed to trigger feeder", slog.String("err", err.Error()), slog.String("tag", "submit order"))
	}

	response.NewGinResponse(c, http.StatusOK, SubmitOrderResponse{OrderID: order.OrderID})
}
