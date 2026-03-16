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

func TestGetOrderSummary_ShouldReturn200(t *testing.T) {
	gin.SetMode(gin.TestMode)
	h, repoMock, _, _ := newTestHandler(t)

	repoMock.EXPECT().GetOrderSummary(mock.Anything).Return(&access.OrderSummaryResponse{
		Today:       5,
		DogCount:    3,
		TotalAmount: 150.0,
	}, nil)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/api/v1/order/summary", nil)

	h.GetOrderSummary(c)

	assert.Equal(t, http.StatusOK, w.Code)
	var resp app.Response[OrderSummaryResponse]
	assert.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.NotNil(t, resp.Data)
	assert.Equal(t, 5, resp.Data.Today)
	assert.Equal(t, 3, resp.Data.DogCount)
	assert.Equal(t, 150.0, resp.Data.TotalAmount)
}

func TestGetOrderSummary_DBError_ShouldReturn500(t *testing.T) {
	gin.SetMode(gin.TestMode)
	h, repoMock, _, _ := newTestHandler(t)

	repoMock.EXPECT().GetOrderSummary(mock.Anything).Return(nil, assert.AnError)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/api/v1/order/summary", nil)

	h.GetOrderSummary(c)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
}
