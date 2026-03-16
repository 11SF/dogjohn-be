package access

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"mime/multipart"
	"net/http"
	"strconv"

	"gitdev.devops.krungthai.com/starwolf/backend/common/httpclient"
)

// SlipOK error codes
var (
	ErrSlipOKMissingData      = errors.New("slipok: missing qr/image/url data (1000)")
	ErrSlipOKBranchNotFound   = errors.New("slipok: branch ID not found (1001)")
	ErrSlipOKUnauthorized     = errors.New("slipok: invalid authorization header (1002)")
	ErrSlipOKPackageExpired   = errors.New("slipok: package expired (1003)")
	ErrSlipOKQuotaExceeded    = errors.New("slipok: quota exceeded (1004)")
	ErrSlipOKNotImageFile     = errors.New("slipok: file is not an image (1005)")
	ErrSlipOKInvalidImage     = errors.New("slipok: invalid image (1006)")
	ErrSlipOKNoQRCode         = errors.New("slipok: image has no qr code (1007)")
	ErrSlipOKNotPaymentQR     = errors.New("slipok: qr is not a payment qr (1008)")
	ErrSlipOKBankUnavailable  = errors.New("slipok: bank data temporarily unavailable (1009)")
	ErrSlipOKPleaseWait       = errors.New("slipok: please wait before verifying this slip (1010)")
	ErrSlipOKQRExpired        = errors.New("slipok: qr code expired or transaction not found (1011)")
	ErrSlipOKDuplicate        = errors.New("slipok: duplicate slip (1012)")
	ErrSlipOKAmountMismatch   = errors.New("slipok: amount mismatch (1013)")
	ErrSlipOKReceiverMismatch = errors.New("slipok: receiver account mismatch (1014)")
)

var slipOKErrorMap = map[int]error{
	1000: ErrSlipOKMissingData,
	1001: ErrSlipOKBranchNotFound,
	1002: ErrSlipOKUnauthorized,
	1003: ErrSlipOKPackageExpired,
	1004: ErrSlipOKQuotaExceeded,
	1005: ErrSlipOKNotImageFile,
	1006: ErrSlipOKInvalidImage,
	1007: ErrSlipOKNoQRCode,
	1008: ErrSlipOKNotPaymentQR,
	1009: ErrSlipOKBankUnavailable,
	1010: ErrSlipOKPleaseWait,
	1011: ErrSlipOKQRExpired,
	1012: ErrSlipOKDuplicate,
	1013: ErrSlipOKAmountMismatch,
	1014: ErrSlipOKReceiverMismatch,
}

// ─── Interface ────────────────────────────────────────────────────────────────

type SlipOKClient interface {
	VerifySlip(ctx context.Context, req VerifySlipRequest) (httpclient.Response[VerifySlipResponse], error)
}

type slipOKClient struct {
	baseURL  string
	branchID string
	apiKey   string
	client   *http.Client
}

func NewSlipOKClient(baseURL, branchID, apiKey string, client *http.Client) SlipOKClient {
	return &slipOKClient{
		baseURL:  baseURL,
		branchID: branchID,
		apiKey:   apiKey,
		client:   client,
	}
}

// ─── Request ──────────────────────────────────────────────────────────────────

type VerifySlipRequest struct {
	// SlipImage is the raw bytes of the slip image (JPG, JPEG, PNG, JFIF, WEBP).
	SlipImage []byte
	// Amount is optional — when set, SlipOK validates it against the slip amount.
	Amount *float64
	// Log = true enables duplicate-slip detection and logs the amount in the SlipOK dashboard.
	Log bool
}

// ─── Response ─────────────────────────────────────────────────────────────────

type VerifySlipResponse struct {
	Success bool        `json:"success"`
	Code    int         `json:"code,omitempty"`
	Data    *SlipOKData `json:"data"`
}

type SlipOKData struct {
	Success           bool       `json:"success"`
	Message           string     `json:"message"`
	Language          string     `json:"language"`
	ReceivingBank     string     `json:"receivingBank"`
	SendingBank       string     `json:"sendingBank"`
	TransRef          string     `json:"transRef"`
	TransDate         string     `json:"transDate"`
	TransTime         string     `json:"transTime"`
	TransTimestamp    string     `json:"transTimestamp"`
	Sender            SlipParty  `json:"sender"`
	Receiver          SlipParty  `json:"receiver"`
	Amount            float64    `json:"amount"`
	PaidLocalAmount   float64    `json:"paidLocalAmount"`
	PaidLocalCurrency string     `json:"paidLocalCurrency"`
	CountryCode       string     `json:"countryCode"`
	TransFeeAmount    string     `json:"transFeeAmount"`
	Ref1              string     `json:"ref1"`
	Ref2              string     `json:"ref2"`
	Ref3              string     `json:"ref3"`
	ToMerchantId      string     `json:"toMerchantId"`
}

type SlipParty struct {
	DisplayName string      `json:"displayName"`
	Name        string      `json:"name"`
	Proxy       SlipProxy   `json:"proxy"`
	Account     SlipAccount `json:"account"`
}

type SlipProxy struct {
	Type  string `json:"type"`
	Value string `json:"value"`
}

type SlipAccount struct {
	Type  string `json:"type"`
	Value string `json:"value"`
}

// ─── Implementation ───────────────────────────────────────────────────────────

func (c *slipOKClient) VerifySlip(ctx context.Context, req VerifySlipRequest) (httpclient.Response[VerifySlipResponse], error) {
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)

	part, err := writer.CreateFormFile("files", "slip.jpg")
	if err != nil {
		return httpclient.Response[VerifySlipResponse]{}, fmt.Errorf("slipok: create form file: %w", err)
	}
	if _, err := part.Write(req.SlipImage); err != nil {
		return httpclient.Response[VerifySlipResponse]{}, fmt.Errorf("slipok: write slip image: %w", err)
	}

	logVal := "false"
	if req.Log {
		logVal = "true"
	}
	if err := writer.WriteField("log", logVal); err != nil {
		return httpclient.Response[VerifySlipResponse]{}, fmt.Errorf("slipok: write log field: %w", err)
	}

	if req.Amount != nil {
		amountStr := strconv.FormatFloat(*req.Amount, 'f', -1, 64)
		if err := writer.WriteField("amount", amountStr); err != nil {
			return httpclient.Response[VerifySlipResponse]{}, fmt.Errorf("slipok: write amount field: %w", err)
		}
	}

	writer.Close()

	url := fmt.Sprintf("%s/api/line/apikey/%s", c.baseURL, c.branchID)
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, url, body)
	if err != nil {
		return httpclient.Response[VerifySlipResponse]{}, fmt.Errorf("slipok: create request: %w", err)
	}
	httpReq.Header.Set("Content-Type", writer.FormDataContentType())
	httpReq.Header.Set("x-authorization", c.apiKey)

	return httpclient.DoRequest[VerifySlipResponse](c.client, httpReq)
}

// SlipOKError maps a VerifySlipResponse error code to a sentinel error.
// Returns nil if the response was successful.
func SlipOKError(resp VerifySlipResponse) error {
	if resp.Success {
		return nil
	}
	if sentinel, ok := slipOKErrorMap[resp.Code]; ok {
		return sentinel
	}
	return fmt.Errorf("slipok: unknown error code %d", resp.Code)
}
