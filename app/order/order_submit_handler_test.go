package order

import (
	"bytes"
	"encoding/json"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"testing"

	"gitdev.devops.krungthai.com/starwolf/backend/common/app"
	"gitdev.devops.krungthai.com/starwolf/backend/common/httpclient"
	"github.com/11SF/dogjohn-be/app/order/access"
	mocks "github.com/11SF/dogjohn-be/app/order/access/mocks"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func newTestHandler(t *testing.T) (*handler, *mocks.OrderRepositoryMock, *mocks.SlipOKClientMock, *mocks.HomeAssistantClientMock) {
	repoMock := mocks.NewOrderRepositoryMock(t)
	slipOKMock := mocks.NewSlipOKClientMock(t)
	haMock := mocks.NewHomeAssistantClientMock(t)
	h := NewHandler(HandlerConfig{
		OrderRepo: repoMock,
		SlipOK:    slipOKMock,
		HAClient:  haMock,
	})
	return h, repoMock, slipOKMock, haMock
}

func buildMultipartRequest(t *testing.T, priceID, customerName string, slipImage []byte) *http.Request {
	t.Helper()
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	_ = writer.WriteField("priceId", priceID)
	_ = writer.WriteField("customerName", customerName)
	if slipImage != nil {
		part, _ := writer.CreateFormFile("slipImage", "slip.jpg")
		_, _ = part.Write(slipImage)
	}
	writer.Close()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/order/submit", body)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	return req
}

func TestSubmitOrder_ShouldReturn200(t *testing.T) {
	gin.SetMode(gin.TestMode)
	h, repoMock, slipOKMock, haMock := newTestHandler(t)

	repoMock.EXPECT().SubmitOrder(mock.Anything, mock.Anything).
		Return(&access.SubmitOrderResponse{OrderID: "order-123"}, nil)
	slipOKMock.EXPECT().VerifySlip(mock.Anything, mock.Anything).
		Return(httpclient.Response[access.VerifySlipResponse]{
			Response: access.VerifySlipResponse{
				Success: true,
				Data:    &access.SlipOKData{TransRef: "TXN-001"},
			},
		}, nil)
	repoMock.EXPECT().SavePaymentTxnLog(mock.Anything, "order-123", mock.Anything).Return(nil)
	repoMock.EXPECT().IsPaymentTxnRefDuplicate(mock.Anything, "TXN-001").Return(false, nil)
	repoMock.EXPECT().UpdateOrderProcessing(mock.Anything, "order-123", "TXN-001").Return(nil)
	haMock.EXPECT().TriggerFeeder(mock.Anything).Return(nil)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = buildMultipartRequest(t, "price-1", "John", []byte("fake-slip"))

	h.SubmitOrder(c)

	assert.Equal(t, http.StatusOK, w.Code)
	var resp app.Response[SubmitOrderResponse]
	assert.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.NotNil(t, resp.Data)
	assert.Equal(t, "order-123", resp.Data.OrderID)
}

func TestSubmitOrder_MissingPriceId_ShouldReturn400(t *testing.T) {
	gin.SetMode(gin.TestMode)
	h, _, _, _ := newTestHandler(t)

	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	_ = writer.WriteField("customerName", "John")
	writer.Close()

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodPost, "/api/v1/order/submit", body)
	c.Request.Header.Set("Content-Type", writer.FormDataContentType())

	h.SubmitOrder(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestSubmitOrder_MissingSlipImage_ShouldReturn400(t *testing.T) {
	gin.SetMode(gin.TestMode)
	h, _, _, _ := newTestHandler(t)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = buildMultipartRequest(t, "price-1", "John", nil)

	h.SubmitOrder(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestSubmitOrder_DBError_ShouldReturn500(t *testing.T) {
	gin.SetMode(gin.TestMode)
	h, repoMock, _, _ := newTestHandler(t)

	repoMock.EXPECT().SubmitOrder(mock.Anything, mock.Anything).Return(nil, assert.AnError)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = buildMultipartRequest(t, "price-1", "John", []byte("fake-slip"))

	h.SubmitOrder(c)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

func TestSubmitOrder_SlipInvalid_ShouldReturn422(t *testing.T) {
	gin.SetMode(gin.TestMode)
	h, repoMock, slipOKMock, _ := newTestHandler(t)

	repoMock.EXPECT().SubmitOrder(mock.Anything, mock.Anything).
		Return(&access.SubmitOrderResponse{OrderID: "order-123"}, nil)
	slipOKMock.EXPECT().VerifySlip(mock.Anything, mock.Anything).
		Return(httpclient.Response[access.VerifySlipResponse]{
			Response: access.VerifySlipResponse{Success: false, Code: 1013},
		}, nil)
	repoMock.EXPECT().SavePaymentTxnLog(mock.Anything, "order-123", mock.Anything).Return(nil)
	repoMock.EXPECT().UpdateOrderFailed(mock.Anything, "order-123", mock.Anything).Return(nil)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = buildMultipartRequest(t, "price-1", "John", []byte("fake-slip"))

	h.SubmitOrder(c)

	assert.Equal(t, http.StatusUnprocessableEntity, w.Code)
}

func TestSubmitOrder_DuplicateSlip_ShouldReturn422(t *testing.T) {
	gin.SetMode(gin.TestMode)
	h, repoMock, slipOKMock, _ := newTestHandler(t)

	repoMock.EXPECT().SubmitOrder(mock.Anything, mock.Anything).
		Return(&access.SubmitOrderResponse{OrderID: "order-123"}, nil)
	slipOKMock.EXPECT().VerifySlip(mock.Anything, mock.Anything).
		Return(httpclient.Response[access.VerifySlipResponse]{
			Response: access.VerifySlipResponse{
				Success: true,
				Data:    &access.SlipOKData{TransRef: "TXN-DUP"},
			},
		}, nil)
	repoMock.EXPECT().SavePaymentTxnLog(mock.Anything, "order-123", mock.Anything).Return(nil)
	repoMock.EXPECT().IsPaymentTxnRefDuplicate(mock.Anything, "TXN-DUP").Return(true, nil)
	repoMock.EXPECT().UpdateOrderFailed(mock.Anything, "order-123", "duplicate slip").Return(nil)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = buildMultipartRequest(t, "price-1", "John", []byte("fake-slip"))

	h.SubmitOrder(c)

	assert.Equal(t, http.StatusUnprocessableEntity, w.Code)
}
