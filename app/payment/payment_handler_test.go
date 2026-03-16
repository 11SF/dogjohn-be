package payment

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/11SF/dogjohn-be/app/payment/access"
	mocks "github.com/11SF/dogjohn-be/app/payment/access/mocks"
	"github.com/11SF/go-common/response"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func newTestHandler(t *testing.T) (*handler, *mocks.PaymentRepositoryMock) {
	repoMock := mocks.NewPaymentRepositoryMock(t)
	h := NewHandler(HandlerConfig{PaymentRepo: repoMock})
	return h, repoMock
}

func TestGetPaymentDetails_ShouldReturn200(t *testing.T) {
	gin.SetMode(gin.TestMode)
	h, repoMock := newTestHandler(t)

	repoMock.EXPECT().GetDetails(mock.Anything).Return(&access.PaymentDetailsResponse{
		PromptPayID: "0812345678",
		Prices: []access.Price{
			{ID: "price-1", Price: 10, Description: "นิดหน่อย"},
			{ID: "price-2", Price: 20, Description: "พอดี"},
			{ID: "price-3", Price: 30, Description: "เยอะ"},
		},
	}, nil)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/api/v1/payment/details", nil)

	h.GetPaymentDetails(c)

	assert.Equal(t, http.StatusOK, w.Code)
	var resp response.Type[PaymentDetailsResponse]
	assert.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.NotNil(t, resp.Data)
	assert.Equal(t, "0812345678", resp.Data.PromptPayID)
	assert.Len(t, resp.Data.Prices, 3)
	assert.Equal(t, "นิดหน่อย", resp.Data.Prices[0].Description)
}

func TestGetPaymentDetails_DBError_ShouldReturn500(t *testing.T) {
	gin.SetMode(gin.TestMode)
	h, repoMock := newTestHandler(t)

	repoMock.EXPECT().GetDetails(mock.Anything).Return(nil, assert.AnError)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/api/v1/payment/details", nil)

	h.GetPaymentDetails(c)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
	var resp response.Type[PaymentDetailsResponse]
	assert.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Equal(t, string(response.GenericError), string(resp.Code))
}
