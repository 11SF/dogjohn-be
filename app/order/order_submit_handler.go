package order

import (
	"io"
	"log/slog"
	"net/http"

	"gitdev.devops.krungthai.com/starwolf/backend/common/app"
	"gitdev.devops.krungthai.com/starwolf/backend/common/wrapper"
	"github.com/11SF/dogjohn-be/app/order/access"
	"github.com/gin-gonic/gin"
)

type SubmitOrderResponse struct {
	OrderID string `json:"orderId"`
}

func (h *handler) SubmitOrder(c *gin.Context) {
	ctx := c.Request.Context()

	priceID := c.PostForm("priceId")
	customerName := c.PostForm("customerName")

	if priceID == "" || customerName == "" {
		wrapper.Respond(c, wrapper.ResponseOption[SubmitOrderResponse]{
			HTTPStatus: http.StatusBadRequest,
			Code:       app.CodeBadRequest,
			Message:    app.MessageBadRequest,
		})
		return
	}

	fileHeader, err := c.FormFile("slipImage")
	if err != nil {
		wrapper.Respond(c, wrapper.ResponseOption[SubmitOrderResponse]{
			HTTPStatus: http.StatusBadRequest,
			Code:       app.CodeBadRequest,
			Message:    app.MessageBadRequest,
			Err:        err,
		})
		return
	}

	file, err := fileHeader.Open()
	if err != nil {
		wrapper.Respond(c, wrapper.ResponseOption[SubmitOrderResponse]{
			HTTPStatus: http.StatusInternalServerError,
			Code:       app.CodeInternalError,
			Message:    app.MessageInternalError,
			Err:        err,
		})
		return
	}
	defer file.Close()

	slipBytes, err := io.ReadAll(file)
	if err != nil {
		wrapper.Respond(c, wrapper.ResponseOption[SubmitOrderResponse]{
			HTTPStatus: http.StatusInternalServerError,
			Code:       app.CodeInternalError,
			Message:    app.MessageInternalError,
			Err:        err,
		})
		return
	}

	// 1. Create order with PENDING status
	order, err := h.orderRepo.SubmitOrder(ctx, access.SubmitOrderRequest{
		PriceID:      priceID,
		CustomerName: customerName,
		SlipImage:    slipBytes,
	})
	if err != nil {
		wrapper.Respond(c, wrapper.ResponseOption[SubmitOrderResponse]{
			HTTPStatus: http.StatusInternalServerError,
			Code:       app.CodeInternalError,
			Message:    app.MessageInternalError,
			Err:        err,
		})
		return
	}

	// 2. Verify payment slip
	slipResp, err := h.slipOK.VerifySlip(ctx, access.VerifySlipRequest{
		SlipImage: slipBytes,
		Log:       true,
	})
	if err != nil {
		_ = h.orderRepo.UpdateOrderFailed(ctx, order.OrderID, err.Error())
		wrapper.Respond(c, wrapper.ResponseOption[SubmitOrderResponse]{
			HTTPStatus: http.StatusInternalServerError,
			Code:       app.CodeInternalError,
			Message:    app.MessageInternalError,
			Err:        err,
		})
		return
	}

	// 3. Save raw SlipOK response
	if logErr := h.orderRepo.SavePaymentTxnLog(ctx, order.OrderID, slipResp.Response); logErr != nil {
		slog.ErrorContext(ctx, "failed to save payment txn log", "error", logErr, "orderID", order.OrderID)
	}

	// 4. Check slip validity
	if slipErr := access.SlipOKError(slipResp.Response); slipErr != nil {
		_ = h.orderRepo.UpdateOrderFailed(ctx, order.OrderID, slipErr.Error())
		wrapper.Respond(c, wrapper.ResponseOption[SubmitOrderResponse]{
			HTTPStatus: http.StatusUnprocessableEntity,
			Code:       app.CodeBadRequest,
			Message:    app.MessageBadRequest,
			Err:        slipErr,
		})
		return
	}

	txnRef := slipResp.Response.Data.TransRef

	// 5. Check for duplicate slip
	isDup, err := h.orderRepo.IsPaymentTxnRefDuplicate(ctx, txnRef)
	if err != nil {
		_ = h.orderRepo.UpdateOrderFailed(ctx, order.OrderID, "failed to check duplicate txn ref")
		wrapper.Respond(c, wrapper.ResponseOption[SubmitOrderResponse]{
			HTTPStatus: http.StatusInternalServerError,
			Code:       app.CodeInternalError,
			Message:    app.MessageInternalError,
			Err:        err,
		})
		return
	}
	if isDup {
		_ = h.orderRepo.UpdateOrderFailed(ctx, order.OrderID, "duplicate slip")
		wrapper.Respond(c, wrapper.ResponseOption[SubmitOrderResponse]{
			HTTPStatus: http.StatusUnprocessableEntity,
			Code:       app.CodeBadRequest,
			Message:    app.MessageBadRequest,
		})
		return
	}

	// 6. Update order to PROCESSING with txnRef
	if err := h.orderRepo.UpdateOrderProcessing(ctx, order.OrderID, txnRef); err != nil {
		wrapper.Respond(c, wrapper.ResponseOption[SubmitOrderResponse]{
			HTTPStatus: http.StatusInternalServerError,
			Code:       app.CodeInternalError,
			Message:    app.MessageInternalError,
			Err:        err,
		})
		return
	}

	// 7. Trigger dog feeder (best effort)
	if err := h.haClient.TriggerFeeder(ctx); err != nil {
		slog.ErrorContext(ctx, "failed to trigger feeder", "error", err, "orderID", order.OrderID)
	}

	wrapper.Respond(c, wrapper.ResponseOption[SubmitOrderResponse]{
		HTTPStatus: http.StatusOK,
		Code:       app.CodeSuccess,
		Message:    app.MessageSuccess,
		Data:       &SubmitOrderResponse{OrderID: order.OrderID},
	})
}
