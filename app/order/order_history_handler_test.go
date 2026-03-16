package order

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"gitdev.devops.krungthai.com/starwolf/backend/common/app"
	"github.com/11SF/dogjohn-be/app/order/access"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestGetOrderHistory_ShouldReturn200(t *testing.T) {
	gin.SetMode(gin.TestMode)
	h, repoMock, _, _ := newTestHandler(t)

	repoMock.EXPECT().GetOrderHistory(mock.Anything, access.OrderHistoryRequest{Limit: 20, Offset: 0}).
		Return(&access.OrderHistoryResponse{
			Items: []access.OrderHistoryItem{
				{
					OrderID:      "order-1",
					CustomerName: "John",
					PriceDetails: access.PriceDetails{Price: "10.00", Description: "นิดหน่อย"},
					OrderStatus:  access.OrderStatusCompleted,
					OrderedAt:    1700000000,
				},
			},
			Total: 1,
		}, nil)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/api/v1/order/history?limit=20&offset=0", nil)

	h.GetOrderHistory(c)

	assert.Equal(t, http.StatusOK, w.Code)
	var resp app.Response[OrderHistoryResponse]
	assert.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.NotNil(t, resp.Data)
	assert.Equal(t, 1, resp.Data.Total)
	assert.Len(t, resp.Data.Items, 1)
	assert.Equal(t, "order-1", resp.Data.Items[0].OrderID)
}

func TestGetOrderHistory_WithPagination_ShouldPassParams(t *testing.T) {
	gin.SetMode(gin.TestMode)
	h, repoMock, _, _ := newTestHandler(t)

	repoMock.EXPECT().GetOrderHistory(mock.Anything, access.OrderHistoryRequest{Limit: 10, Offset: 20}).
		Return(&access.OrderHistoryResponse{Items: []access.OrderHistoryItem{}, Total: 100}, nil)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/api/v1/order/history?limit=10&offset=20", nil)

	h.GetOrderHistory(c)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestGetOrderHistory_DBError_ShouldReturn500(t *testing.T) {
	gin.SetMode(gin.TestMode)
	h, repoMock, _, _ := newTestHandler(t)

	repoMock.EXPECT().GetOrderHistory(mock.Anything, mock.Anything).Return(nil, assert.AnError)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/api/v1/order/history", nil)

	h.GetOrderHistory(c)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
}
