package order

import (
	"net/http"
	"strconv"

	"gitdev.devops.krungthai.com/starwolf/backend/common/app"
	"gitdev.devops.krungthai.com/starwolf/backend/common/wrapper"
	"github.com/11SF/dogjohn-be/app/order/access"
	"github.com/gin-gonic/gin"
	"github.com/samber/lo"
)

type OrderHistoryItemResponse = orderHistoryItemResponse

type OrderHistoryResponse struct {
	Items []OrderHistoryItemResponse `json:"items"`
	Total int                        `json:"total"`
}

func (h *handler) GetOrderHistory(c *gin.Context) {
	ctx := c.Request.Context()

	limit := 20
	offset := 0

	if l := c.Query("limit"); l != "" {
		if v, err := strconv.Atoi(l); err == nil {
			limit = v
		}
	}
	if o := c.Query("offset"); o != "" {
		if v, err := strconv.Atoi(o); err == nil {
			offset = v
		}
	}

	data, err := h.orderRepo.GetOrderHistory(ctx, access.OrderHistoryRequest{
		Limit:  limit,
		Offset: offset,
	})
	if err != nil {
		wrapper.Respond(c, wrapper.ResponseOption[OrderHistoryResponse]{
			HTTPStatus: http.StatusInternalServerError,
			Code:       app.CodeInternalError,
			Message:    app.MessageInternalError,
			Err:        err,
		})
		return
	}

	wrapper.Respond(c, wrapper.ResponseOption[OrderHistoryResponse]{
		HTTPStatus: http.StatusOK,
		Code:       app.CodeSuccess,
		Message:    app.MessageSuccess,
		Data: &OrderHistoryResponse{
			Items: lo.Map(data.Items, func(item access.OrderHistoryItem, _ int) OrderHistoryItemResponse {
				return mapOrderHistoryItem(item)
			}),
			Total: data.Total,
		},
	})
}
