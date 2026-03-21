package order

import (
	"bytes"
	"encoding/json"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/11SF/dogjohn-be/app/order/access"
	mocks "github.com/11SF/dogjohn-be/app/order/access/mocks"
	paymentaccess "github.com/11SF/dogjohn-be/app/payment/access"
	paymentmocks "github.com/11SF/dogjohn-be/app/payment/access/mocks"
	"github.com/11SF/dogjohn-be/config"
	"github.com/11SF/go-common/response"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func newTestHandler(t *testing.T) (*handler, *mocks.OrderRepositoryMock, *paymentmocks.PaymentRepositoryMock, *mocks.SlipOKClientMock, *mocks.HomeAssistantClientMock) {
	repoMock := mocks.NewOrderRepositoryMock(t)
	paymentRepoMock := paymentmocks.NewPaymentRepositoryMock(t)
	slipOKMock := mocks.NewSlipOKClientMock(t)
	haMock := mocks.NewHomeAssistantClientMock(t)
	h := NewHandler(HandlerConfig{
		OrderRepo:   repoMock,
		PaymentRepo: paymentRepoMock,
		SlipOK:      slipOKMock,
		HAClient:    haMock,
	})
	return h, repoMock, paymentRepoMock, slipOKMock, haMock
}

func newTestHandlerWithBypass(t *testing.T) (*handler, *mocks.OrderRepositoryMock, *paymentmocks.PaymentRepositoryMock, *mocks.SlipOKClientMock, *mocks.HomeAssistantClientMock) {
	repoMock := mocks.NewOrderRepositoryMock(t)
	paymentRepoMock := paymentmocks.NewPaymentRepositoryMock(t)
	slipOKMock := mocks.NewSlipOKClientMock(t)
	haMock := mocks.NewHomeAssistantClientMock(t)
	h := NewHandler(HandlerConfig{
		Config:      config.Config{IsByPassVerifySlip: true},
		OrderRepo:   repoMock,
		PaymentRepo: paymentRepoMock,
		SlipOK:      slipOKMock,
		HAClient:    haMock,
	})
	return h, repoMock, paymentRepoMock, slipOKMock, haMock
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

// validSlipData returns a VerifySlipResponse that passes all validation checks
// given PromptPayID = "0812345678" (last 4 = "5678") and price = 50.0.
func validSlipData(transRef string) access.VerifySlipResponse {
	return access.VerifySlipResponse{
		Success: true,
		Data: &access.SlipOKData{
			TransRef:        transRef,
			PaidLocalAmount: 50.0,
			Receiver: access.SlipParty{
				Proxy: access.SlipProxy{Value: "XX-YY-5678"},
			},
		},
	}
}

func TestSubmitOrder_ShouldReturn200(t *testing.T) {
	gin.SetMode(gin.TestMode)
	h, repoMock, paymentRepoMock, slipOKMock, haMock := newTestHandler(t)

	repoMock.EXPECT().SubmitOrder(mock.Anything, mock.Anything).
		Return(&access.SubmitOrderResponse{OrderID: "order-123"}, nil)
	paymentRepoMock.EXPECT().GetPrice(mock.Anything, "price-1").
		Return(&paymentaccess.Price{ID: "price-1", Price: 50.0}, nil)
	paymentRepoMock.EXPECT().GetDetails(mock.Anything).
		Return(&paymentaccess.PaymentDetailsResponse{PromptPayID: "0812345678"}, nil)
	slipOKMock.EXPECT().VerifySlip(mock.Anything, mock.Anything).
		Return(validSlipData("TXN-001"), nil)
	repoMock.EXPECT().SavePaymentTxnLog(mock.Anything, "order-123", mock.Anything).Return(nil)
	repoMock.EXPECT().IsPaymentTxnRefDuplicate(mock.Anything, "TXN-001").Return(false, nil)
	repoMock.EXPECT().UpdateOrderProcessing(mock.Anything, "order-123", mock.Anything).Return(nil)
	haMock.EXPECT().TriggerFeeder(mock.Anything).Return(nil)
	repoMock.EXPECT().UpdateOrderCompleted(mock.Anything, "order-123").Return(nil)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = buildMultipartRequest(t, "price-1", "John", []byte("fake-slip"))

	h.SubmitOrder(c)

	assert.Equal(t, http.StatusOK, w.Code)
	var resp response.Type[SubmitOrderResponse]
	assert.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.NotNil(t, resp.Data)
	assert.Equal(t, "order-123", resp.Data.OrderID)
}

func TestSubmitOrder_AnonymousCustomerName_ShouldReturn200(t *testing.T) {
	gin.SetMode(gin.TestMode)
	h, repoMock, paymentRepoMock, slipOKMock, haMock := newTestHandler(t)

	repoMock.EXPECT().SubmitOrder(mock.Anything, mock.Anything).
		Return(&access.SubmitOrderResponse{OrderID: "order-456"}, nil)
	paymentRepoMock.EXPECT().GetPrice(mock.Anything, "price-1").
		Return(&paymentaccess.Price{ID: "price-1", Price: 50.0}, nil)
	paymentRepoMock.EXPECT().GetDetails(mock.Anything).
		Return(&paymentaccess.PaymentDetailsResponse{PromptPayID: "0812345678"}, nil)
	slipOKMock.EXPECT().VerifySlip(mock.Anything, mock.Anything).
		Return(validSlipData("TXN-002"), nil)
	repoMock.EXPECT().SavePaymentTxnLog(mock.Anything, "order-456", mock.Anything).Return(nil)
	repoMock.EXPECT().IsPaymentTxnRefDuplicate(mock.Anything, "TXN-002").Return(false, nil)
	repoMock.EXPECT().UpdateOrderProcessing(mock.Anything, "order-456", mock.Anything).Return(nil)
	haMock.EXPECT().TriggerFeeder(mock.Anything).Return(nil)
	repoMock.EXPECT().UpdateOrderCompleted(mock.Anything, "order-456").Return(nil)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	// empty customerName → should default to "Anonymous"
	c.Request = buildMultipartRequest(t, "price-1", "", []byte("fake-slip"))

	h.SubmitOrder(c)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestSubmitOrder_MissingPriceId_ShouldReturn400(t *testing.T) {
	gin.SetMode(gin.TestMode)
	h, _, _, _, _ := newTestHandler(t)

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
	h, _, _, _, _ := newTestHandler(t)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = buildMultipartRequest(t, "price-1", "John", nil)

	h.SubmitOrder(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestSubmitOrder_SubmitOrderDBError_ShouldReturn500(t *testing.T) {
	gin.SetMode(gin.TestMode)
	h, repoMock, _, _, _ := newTestHandler(t)

	repoMock.EXPECT().SubmitOrder(mock.Anything, mock.Anything).Return(nil, assert.AnError)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = buildMultipartRequest(t, "price-1", "John", []byte("fake-slip"))

	h.SubmitOrder(c)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

func TestSubmitOrder_GetPriceError_ShouldReturn500(t *testing.T) {
	gin.SetMode(gin.TestMode)
	h, repoMock, paymentRepoMock, _, _ := newTestHandler(t)

	repoMock.EXPECT().SubmitOrder(mock.Anything, mock.Anything).
		Return(&access.SubmitOrderResponse{OrderID: "order-123"}, nil)
	paymentRepoMock.EXPECT().GetPrice(mock.Anything, "price-1").Return(nil, assert.AnError)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = buildMultipartRequest(t, "price-1", "John", []byte("fake-slip"))

	h.SubmitOrder(c)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

func TestSubmitOrder_InvalidPriceId_ShouldReturn400(t *testing.T) {
	gin.SetMode(gin.TestMode)
	h, repoMock, paymentRepoMock, _, _ := newTestHandler(t)

	repoMock.EXPECT().SubmitOrder(mock.Anything, mock.Anything).
		Return(&access.SubmitOrderResponse{OrderID: "order-123"}, nil)
	paymentRepoMock.EXPECT().GetPrice(mock.Anything, "price-bad").Return(nil, nil)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = buildMultipartRequest(t, "price-bad", "John", []byte("fake-slip"))

	h.SubmitOrder(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestSubmitOrder_GetDetailsError_ShouldReturn500(t *testing.T) {
	gin.SetMode(gin.TestMode)
	h, repoMock, paymentRepoMock, _, _ := newTestHandler(t)

	repoMock.EXPECT().SubmitOrder(mock.Anything, mock.Anything).
		Return(&access.SubmitOrderResponse{OrderID: "order-123"}, nil)
	paymentRepoMock.EXPECT().GetPrice(mock.Anything, "price-1").
		Return(&paymentaccess.Price{ID: "price-1", Price: 50.0}, nil)
	paymentRepoMock.EXPECT().GetDetails(mock.Anything).Return(nil, assert.AnError)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = buildMultipartRequest(t, "price-1", "John", []byte("fake-slip"))

	h.SubmitOrder(c)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

func TestSubmitOrder_VerifySlipError_ShouldReturn500(t *testing.T) {
	gin.SetMode(gin.TestMode)
	h, repoMock, paymentRepoMock, slipOKMock, _ := newTestHandler(t)

	repoMock.EXPECT().SubmitOrder(mock.Anything, mock.Anything).
		Return(&access.SubmitOrderResponse{OrderID: "order-123"}, nil)
	paymentRepoMock.EXPECT().GetPrice(mock.Anything, "price-1").
		Return(&paymentaccess.Price{ID: "price-1", Price: 50.0}, nil)
	paymentRepoMock.EXPECT().GetDetails(mock.Anything).
		Return(&paymentaccess.PaymentDetailsResponse{PromptPayID: "0812345678"}, nil)
	slipOKMock.EXPECT().VerifySlip(mock.Anything, mock.Anything).Return(access.VerifySlipResponse{}, assert.AnError)
	repoMock.EXPECT().UpdateOrderFailed(mock.Anything, "order-123", mock.Anything).Return(nil)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = buildMultipartRequest(t, "price-1", "John", []byte("fake-slip"))

	h.SubmitOrder(c)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

func TestSubmitOrder_SlipNilData_ShouldReturn500(t *testing.T) {
	gin.SetMode(gin.TestMode)
	h, repoMock, paymentRepoMock, slipOKMock, _ := newTestHandler(t)

	repoMock.EXPECT().SubmitOrder(mock.Anything, mock.Anything).
		Return(&access.SubmitOrderResponse{OrderID: "order-123"}, nil)
	paymentRepoMock.EXPECT().GetPrice(mock.Anything, "price-1").
		Return(&paymentaccess.Price{ID: "price-1", Price: 50.0}, nil)
	paymentRepoMock.EXPECT().GetDetails(mock.Anything).
		Return(&paymentaccess.PaymentDetailsResponse{PromptPayID: "0812345678"}, nil)
	slipOKMock.EXPECT().VerifySlip(mock.Anything, mock.Anything).
		Return(access.VerifySlipResponse{Success: false, Data: nil}, nil)
	repoMock.EXPECT().SavePaymentTxnLog(mock.Anything, "order-123", mock.Anything).Return(nil)
	repoMock.EXPECT().UpdateOrderFailed(mock.Anything, "order-123", mock.Anything).Return(nil)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = buildMultipartRequest(t, "price-1", "John", []byte("fake-slip"))

	h.SubmitOrder(c)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

func TestSubmitOrder_SavePaymentTxnLogError_ShouldContinue(t *testing.T) {
	gin.SetMode(gin.TestMode)
	h, repoMock, paymentRepoMock, slipOKMock, haMock := newTestHandler(t)

	repoMock.EXPECT().SubmitOrder(mock.Anything, mock.Anything).
		Return(&access.SubmitOrderResponse{OrderID: "order-123"}, nil)
	paymentRepoMock.EXPECT().GetPrice(mock.Anything, "price-1").
		Return(&paymentaccess.Price{ID: "price-1", Price: 50.0}, nil)
	paymentRepoMock.EXPECT().GetDetails(mock.Anything).
		Return(&paymentaccess.PaymentDetailsResponse{PromptPayID: "0812345678"}, nil)
	slipOKMock.EXPECT().VerifySlip(mock.Anything, mock.Anything).
		Return(validSlipData("TXN-003"), nil)
	repoMock.EXPECT().SavePaymentTxnLog(mock.Anything, "order-123", mock.Anything).Return(assert.AnError)
	repoMock.EXPECT().IsPaymentTxnRefDuplicate(mock.Anything, "TXN-003").Return(false, nil)
	repoMock.EXPECT().UpdateOrderProcessing(mock.Anything, "order-123", mock.Anything).Return(nil)
	haMock.EXPECT().TriggerFeeder(mock.Anything).Return(nil)
	repoMock.EXPECT().UpdateOrderCompleted(mock.Anything, "order-123").Return(nil)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = buildMultipartRequest(t, "price-1", "John", []byte("fake-slip"))

	h.SubmitOrder(c)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestSubmitOrder_InsufficientAmount_ShouldReturn422(t *testing.T) {
	gin.SetMode(gin.TestMode)
	h, repoMock, paymentRepoMock, slipOKMock, _ := newTestHandler(t)

	repoMock.EXPECT().SubmitOrder(mock.Anything, mock.Anything).
		Return(&access.SubmitOrderResponse{OrderID: "order-123"}, nil)
	paymentRepoMock.EXPECT().GetPrice(mock.Anything, "price-1").
		Return(&paymentaccess.Price{ID: "price-1", Price: 50.0}, nil)
	paymentRepoMock.EXPECT().GetDetails(mock.Anything).
		Return(&paymentaccess.PaymentDetailsResponse{PromptPayID: "0812345678"}, nil)
	slipOKMock.EXPECT().VerifySlip(mock.Anything, mock.Anything).
		Return(access.VerifySlipResponse{
			Success: true,
			Data: &access.SlipOKData{
				TransRef:        "TXN-LOW",
				PaidLocalAmount: 30.0, // less than 50.0
				Receiver:        access.SlipParty{Proxy: access.SlipProxy{Value: "XX-YY-5678"}},
			},
		}, nil)
	repoMock.EXPECT().SavePaymentTxnLog(mock.Anything, "order-123", mock.Anything).Return(nil)
	repoMock.EXPECT().UpdateOrderFailed(mock.Anything, "order-123", "insufficient amount").Return(nil)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = buildMultipartRequest(t, "price-1", "John", []byte("fake-slip"))

	h.SubmitOrder(c)

	assert.Equal(t, http.StatusUnprocessableEntity, w.Code)
}

func TestSubmitOrder_InvalidReceiver_ShouldReturn422(t *testing.T) {
	gin.SetMode(gin.TestMode)
	h, repoMock, paymentRepoMock, slipOKMock, _ := newTestHandler(t)

	repoMock.EXPECT().SubmitOrder(mock.Anything, mock.Anything).
		Return(&access.SubmitOrderResponse{OrderID: "order-123"}, nil)
	paymentRepoMock.EXPECT().GetPrice(mock.Anything, "price-1").
		Return(&paymentaccess.Price{ID: "price-1", Price: 50.0}, nil)
	paymentRepoMock.EXPECT().GetDetails(mock.Anything).
		Return(&paymentaccess.PaymentDetailsResponse{PromptPayID: "0812345678"}, nil)
	slipOKMock.EXPECT().VerifySlip(mock.Anything, mock.Anything).
		Return(access.VerifySlipResponse{
			Success: true,
			Data: &access.SlipOKData{
				TransRef:        "TXN-RECV",
				PaidLocalAmount: 50.0,
				Receiver: access.SlipParty{Proxy: access.SlipProxy{Value: "XX-YY-WRONG"}}, // last 4 of PromptPayID is "5678"
			},
		}, nil)
	repoMock.EXPECT().SavePaymentTxnLog(mock.Anything, "order-123", mock.Anything).Return(nil)
	repoMock.EXPECT().UpdateOrderFailed(mock.Anything, "order-123", "invalid receiver").Return(nil)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = buildMultipartRequest(t, "price-1", "John", []byte("fake-slip"))

	h.SubmitOrder(c)

	assert.Equal(t, http.StatusUnprocessableEntity, w.Code)
}

func TestSubmitOrder_DuplicateTxnCheckError_ShouldReturn500(t *testing.T) {
	gin.SetMode(gin.TestMode)
	h, repoMock, paymentRepoMock, slipOKMock, _ := newTestHandler(t)

	repoMock.EXPECT().SubmitOrder(mock.Anything, mock.Anything).
		Return(&access.SubmitOrderResponse{OrderID: "order-123"}, nil)
	paymentRepoMock.EXPECT().GetPrice(mock.Anything, "price-1").
		Return(&paymentaccess.Price{ID: "price-1", Price: 50.0}, nil)
	paymentRepoMock.EXPECT().GetDetails(mock.Anything).
		Return(&paymentaccess.PaymentDetailsResponse{PromptPayID: "0812345678"}, nil)
	slipOKMock.EXPECT().VerifySlip(mock.Anything, mock.Anything).
		Return(validSlipData("TXN-CHK"), nil)
	repoMock.EXPECT().SavePaymentTxnLog(mock.Anything, "order-123", mock.Anything).Return(nil)
	repoMock.EXPECT().IsPaymentTxnRefDuplicate(mock.Anything, "TXN-CHK").Return(false, assert.AnError)
	repoMock.EXPECT().UpdateOrderFailed(mock.Anything, "order-123", mock.Anything).Return(nil)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = buildMultipartRequest(t, "price-1", "John", []byte("fake-slip"))

	h.SubmitOrder(c)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

func TestSubmitOrder_DuplicateSlip_ShouldReturn422(t *testing.T) {
	gin.SetMode(gin.TestMode)
	h, repoMock, paymentRepoMock, slipOKMock, _ := newTestHandler(t)

	repoMock.EXPECT().SubmitOrder(mock.Anything, mock.Anything).
		Return(&access.SubmitOrderResponse{OrderID: "order-123"}, nil)
	paymentRepoMock.EXPECT().GetPrice(mock.Anything, "price-1").
		Return(&paymentaccess.Price{ID: "price-1", Price: 50.0}, nil)
	paymentRepoMock.EXPECT().GetDetails(mock.Anything).
		Return(&paymentaccess.PaymentDetailsResponse{PromptPayID: "0812345678"}, nil)
	slipOKMock.EXPECT().VerifySlip(mock.Anything, mock.Anything).
		Return(validSlipData("TXN-DUP"), nil)
	repoMock.EXPECT().SavePaymentTxnLog(mock.Anything, "order-123", mock.Anything).Return(nil)
	repoMock.EXPECT().IsPaymentTxnRefDuplicate(mock.Anything, "TXN-DUP").Return(true, nil)
	repoMock.EXPECT().UpdateOrderFailed(mock.Anything, "order-123", "duplicate slip").Return(nil)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = buildMultipartRequest(t, "price-1", "John", []byte("fake-slip"))

	h.SubmitOrder(c)

	assert.Equal(t, http.StatusUnprocessableEntity, w.Code)
}

func TestSubmitOrder_UpdateOrderProcessingError_ShouldReturn500(t *testing.T) {
	gin.SetMode(gin.TestMode)
	h, repoMock, paymentRepoMock, slipOKMock, _ := newTestHandler(t)

	repoMock.EXPECT().SubmitOrder(mock.Anything, mock.Anything).
		Return(&access.SubmitOrderResponse{OrderID: "order-123"}, nil)
	paymentRepoMock.EXPECT().GetPrice(mock.Anything, "price-1").
		Return(&paymentaccess.Price{ID: "price-1", Price: 50.0}, nil)
	paymentRepoMock.EXPECT().GetDetails(mock.Anything).
		Return(&paymentaccess.PaymentDetailsResponse{PromptPayID: "0812345678"}, nil)
	slipOKMock.EXPECT().VerifySlip(mock.Anything, mock.Anything).
		Return(validSlipData("TXN-PROC"), nil)
	repoMock.EXPECT().SavePaymentTxnLog(mock.Anything, "order-123", mock.Anything).Return(nil)
	repoMock.EXPECT().IsPaymentTxnRefDuplicate(mock.Anything, "TXN-PROC").Return(false, nil)
	repoMock.EXPECT().UpdateOrderProcessing(mock.Anything, "order-123", mock.Anything).Return(assert.AnError)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = buildMultipartRequest(t, "price-1", "John", []byte("fake-slip"))

	h.SubmitOrder(c)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

func TestSubmitOrder_TriggerFeederError_ShouldReturn200(t *testing.T) {
	gin.SetMode(gin.TestMode)
	h, repoMock, paymentRepoMock, slipOKMock, haMock := newTestHandler(t)

	repoMock.EXPECT().SubmitOrder(mock.Anything, mock.Anything).
		Return(&access.SubmitOrderResponse{OrderID: "order-123"}, nil)
	paymentRepoMock.EXPECT().GetPrice(mock.Anything, "price-1").
		Return(&paymentaccess.Price{ID: "price-1", Price: 50.0}, nil)
	paymentRepoMock.EXPECT().GetDetails(mock.Anything).
		Return(&paymentaccess.PaymentDetailsResponse{PromptPayID: "0812345678"}, nil)
	slipOKMock.EXPECT().VerifySlip(mock.Anything, mock.Anything).
		Return(validSlipData("TXN-FEED"), nil)
	repoMock.EXPECT().SavePaymentTxnLog(mock.Anything, "order-123", mock.Anything).Return(nil)
	repoMock.EXPECT().IsPaymentTxnRefDuplicate(mock.Anything, "TXN-FEED").Return(false, nil)
	repoMock.EXPECT().UpdateOrderProcessing(mock.Anything, "order-123", mock.Anything).Return(nil)
	haMock.EXPECT().TriggerFeeder(mock.Anything).Return(assert.AnError) // best effort
	repoMock.EXPECT().UpdateOrderCompleted(mock.Anything, "order-123").Return(nil)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = buildMultipartRequest(t, "price-1", "John", []byte("fake-slip"))

	h.SubmitOrder(c)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestSubmitOrder_UpdateOrderCompletedError_ShouldReturn200(t *testing.T) {
	gin.SetMode(gin.TestMode)
	h, repoMock, paymentRepoMock, slipOKMock, haMock := newTestHandler(t)

	repoMock.EXPECT().SubmitOrder(mock.Anything, mock.Anything).
		Return(&access.SubmitOrderResponse{OrderID: "order-123"}, nil)
	paymentRepoMock.EXPECT().GetPrice(mock.Anything, "price-1").
		Return(&paymentaccess.Price{ID: "price-1", Price: 50.0}, nil)
	paymentRepoMock.EXPECT().GetDetails(mock.Anything).
		Return(&paymentaccess.PaymentDetailsResponse{PromptPayID: "0812345678"}, nil)
	slipOKMock.EXPECT().VerifySlip(mock.Anything, mock.Anything).
		Return(validSlipData("TXN-COMP"), nil)
	repoMock.EXPECT().SavePaymentTxnLog(mock.Anything, "order-123", mock.Anything).Return(nil)
	repoMock.EXPECT().IsPaymentTxnRefDuplicate(mock.Anything, "TXN-COMP").Return(false, nil)
	repoMock.EXPECT().UpdateOrderProcessing(mock.Anything, "order-123", mock.Anything).Return(nil)
	haMock.EXPECT().TriggerFeeder(mock.Anything).Return(nil)
	repoMock.EXPECT().UpdateOrderCompleted(mock.Anything, "order-123").Return(assert.AnError) // best effort

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = buildMultipartRequest(t, "price-1", "John", []byte("fake-slip"))

	h.SubmitOrder(c)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestSubmitOrder_BypassVerifySlip_ShouldReturn200(t *testing.T) {
	gin.SetMode(gin.TestMode)
	h, repoMock, paymentRepoMock, _, haMock := newTestHandlerWithBypass(t)

	repoMock.EXPECT().SubmitOrder(mock.Anything, mock.Anything).
		Return(&access.SubmitOrderResponse{OrderID: "order-123"}, nil)
	paymentRepoMock.EXPECT().GetPrice(mock.Anything, "price-1").
		Return(&paymentaccess.Price{ID: "price-1", Price: 50.0}, nil)
	paymentRepoMock.EXPECT().GetDetails(mock.Anything).
		Return(&paymentaccess.PaymentDetailsResponse{PromptPayID: "0812345678"}, nil)
	repoMock.EXPECT().UpdateOrderProcessing(mock.Anything, "order-123", mock.Anything).Return(nil)
	haMock.EXPECT().TriggerFeeder(mock.Anything).Return(nil)
	repoMock.EXPECT().UpdateOrderCompleted(mock.Anything, "order-123").Return(nil)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = buildMultipartRequest(t, "price-1", "John", []byte("fake-slip"))

	h.SubmitOrder(c)

	assert.Equal(t, http.StatusOK, w.Code)
	var resp response.Type[SubmitOrderResponse]
	assert.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Equal(t, "order-123", resp.Data.OrderID)
}
