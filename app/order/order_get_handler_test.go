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

func TestGetOrder_ShouldReturn200(t *testing.T) {
	gin.SetMode(gin.TestMode)
	h, repoMock, _, _ := newTestHandler(t)

	repoMock.EXPECT().GetOrder(mock.Anything, "order-123").Return(&access.GetOrderResponse{
		OrderID:      "order-123",
		CustomerName: "John",
		PriceDetails: access.PriceDetails{Price: "10.00", Description: "นิดหน่อย"},
		OrderStatus:  access.OrderStatusPending,
		OrderedAt:    1700000000,
	}, nil)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/api/v1/order/order-123", nil)
	c.Params = gin.Params{{Key: "orderId", Value: "order-123"}}

	h.GetOrder(c)

	assert.Equal(t, http.StatusOK, w.Code)
	var resp app.Response[GetOrderResponse]
	assert.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.NotNil(t, resp.Data)
	assert.Equal(t, "order-123", resp.Data.OrderID)
	assert.Equal(t, string(access.OrderStatusPending), resp.Data.OrderStatus)
}

func TestGetOrder_NotFound_ShouldReturn404(t *testing.T) {
	gin.SetMode(gin.TestMode)
	h, repoMock, _, _ := newTestHandler(t)

	repoMock.EXPECT().GetOrder(mock.Anything, "not-found").Return(nil, access.ErrNotFound)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/api/v1/order/not-found", nil)
	c.Params = gin.Params{{Key: "orderId", Value: "not-found"}}

	h.GetOrder(c)

	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestGetOrder_DBError_ShouldReturn500(t *testing.T) {
	gin.SetMode(gin.TestMode)
	h, repoMock, _, _ := newTestHandler(t)

	repoMock.EXPECT().GetOrder(mock.Anything, mock.Anything).Return(nil, assert.AnError)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/api/v1/order/order-123", nil)
	c.Params = gin.Params{{Key: "orderId", Value: "order-123"}}

	h.GetOrder(c)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
}
