package order

import (
	"errors"
	"net/http"

	"github.com/11SF/dogjohn-be/app/order/access"
	"github.com/11SF/go-common/response"
	"github.com/gin-gonic/gin"
)

type PriceDetails struct {
	Price       string `json:"price"`
	Description string `json:"description"`
}

type GetOrderResponse struct {
	OrderID       string       `json:"orderId"`
	CustomerName  string       `json:"customerName"`
	PriceDetails  PriceDetails `json:"priceDetails"`
	OrderStatus   string       `json:"orderStatus"`
	FailureReason *string      `json:"failureReason"`
	OrderedAt     int64        `json:"orderedAt"`
}

func (h *handler) GetOrder(c *gin.Context) {
	ctx := c.Request.Context()
	orderID := c.Param("orderId")

	data, err := h.orderRepo.GetOrder(ctx, orderID)
	if errors.Is(err, access.ErrNotFound) {
		response.NewGinResponseError(c, http.StatusNotFound,
			response.NewError(response.NotFoundCode, "Not Found"))
		return
	}
	if err != nil {
		response.NewGinResponseError(c, http.StatusInternalServerError,
			response.NewError(response.GenericError, "Internal Server Error"))
		return
	}

	response.NewGinResponse(c, http.StatusOK, GetOrderResponse{
		OrderID:      data.OrderID,
		CustomerName: data.CustomerName,
		PriceDetails: PriceDetails{
			Price:       data.PriceDetails.Price,
			Description: data.PriceDetails.Description,
		},
		OrderStatus:   string(data.OrderStatus),
		FailureReason: data.FailureReason,
		OrderedAt:     data.OrderedAt,
	})
}

// orderHistoryItemResponse is the response shape for a single history item,
// shared with the history handler.
type orderHistoryItemResponse struct {
	OrderID      string       `json:"orderId"`
	CustomerName string       `json:"customerName"`
	PriceDetails PriceDetails `json:"priceDetails"`
	OrderStatus  string       `json:"orderStatus"`
	OrderedAt    int64        `json:"orderedAt"`
}

func mapOrderHistoryItem(item access.OrderHistoryItem) orderHistoryItemResponse {
	return orderHistoryItemResponse{
		OrderID:      item.OrderID,
		CustomerName: item.CustomerName,
		PriceDetails: PriceDetails{
			Price:       item.PriceDetails.Price,
			Description: item.PriceDetails.Description,
		},
		OrderStatus: string(item.OrderStatus),
		OrderedAt:   item.OrderedAt,
	}
}
