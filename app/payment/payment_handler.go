package payment

import (
	"net/http"

	"github.com/11SF/go-common/response"
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
		response.NewGinResponseError(c, http.StatusInternalServerError,
			response.NewError(response.GenericError, "Internal Server Error"))
		return
	}

	prices := make([]Price, len(data.Prices))
	for i, p := range data.Prices {
		prices[i] = Price{ID: p.ID, Price: p.Price, Description: p.Description}
	}

	response.NewGinResponse(c, http.StatusOK, PaymentDetailsResponse{
		PromptPayID: data.PromptPayID,
		Prices:      prices,
	})
}
