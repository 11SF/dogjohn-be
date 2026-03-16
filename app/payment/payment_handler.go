package payment

import (
	"net/http"

	"gitdev.devops.krungthai.com/starwolf/backend/common/app"
	"gitdev.devops.krungthai.com/starwolf/backend/common/wrapper"
	"github.com/gin-gonic/gin"
)

type Price struct {
	ID          string  `json:"id"`
	Price       float64 `json:"price"`
	Description string  `json:"description"`
}

type PaymentDetailsResponse struct {
	PromptPayID string  `json:"promptPayId"`
	Prices      []Price `json:"prices"`
}

func (h *handler) GetPaymentDetails(c *gin.Context) {
	ctx := c.Request.Context()

	data, err := h.paymentRepo.GetDetails(ctx)
	if err != nil {
		wrapper.Respond(c, wrapper.ResponseOption[PaymentDetailsResponse]{
			HTTPStatus: http.StatusInternalServerError,
			Code:       app.CodeInternalError,
			Message:    app.MessageInternalError,
			Err:        err,
		})
		return
	}

	prices := make([]Price, len(data.Prices))
	for i, p := range data.Prices {
		prices[i] = Price{ID: p.ID, Price: p.Price, Description: p.Description}
	}

	wrapper.Respond(c, wrapper.ResponseOption[PaymentDetailsResponse]{
		HTTPStatus: http.StatusOK,
		Code:       app.CodeSuccess,
		Message:    app.MessageSuccess,
		Data: &PaymentDetailsResponse{
			PromptPayID: data.PromptPayID,
			Prices:      prices,
		},
	})
}
