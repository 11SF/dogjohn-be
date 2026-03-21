package order

import (
	"net/http"

	"github.com/11SF/go-common/response"
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
		response.NewGinResponseError(c, http.StatusInternalServerError,
			response.NewError(response.GenericError, "Internal Server Error"))
		return
	}

	response.NewGinResponse(c, http.StatusOK, OrderSummaryResponse{
		Today:       data.Today,
		DogCount:    data.DogCount,
		TotalAmount: data.TotalAmount,
	})
}
