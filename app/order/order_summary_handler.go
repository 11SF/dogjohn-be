package order

import (
	"net/http"

	"gitdev.devops.krungthai.com/starwolf/backend/common/app"
	"gitdev.devops.krungthai.com/starwolf/backend/common/wrapper"
	"github.com/gin-gonic/gin"
)

type OrderSummaryResponse struct {
	Today       int     `json:"today"`
	DogCount    int     `json:"dogCount"`
	TotalAmount float64 `json:"totalAmount"`
}

func (h *handler) GetOrderSummary(c *gin.Context) {
	ctx := c.Request.Context()

	data, err := h.orderRepo.GetOrderSummary(ctx)
	if err != nil {
		wrapper.Respond(c, wrapper.ResponseOption[OrderSummaryResponse]{
			HTTPStatus: http.StatusInternalServerError,
			Code:       app.CodeInternalError,
			Message:    app.MessageInternalError,
			Err:        err,
		})
		return
	}

	wrapper.Respond(c, wrapper.ResponseOption[OrderSummaryResponse]{
		HTTPStatus: http.StatusOK,
		Code:       app.CodeSuccess,
		Message:    app.MessageSuccess,
		Data: &OrderSummaryResponse{
			Today:       data.Today,
			DogCount:    data.DogCount,
			TotalAmount: data.TotalAmount,
		},
	})
}
