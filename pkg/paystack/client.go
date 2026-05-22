package paystack

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha512"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

const defaultBaseURL = "https://api.paystack.co"

// Client talks to the Paystack REST API.
type Client struct {
	secretKey  string
	baseURL    string
	httpClient *http.Client
}

func NewClient(secretKey string) *Client {
	return &Client{
		secretKey: secretKey,
		baseURL:   defaultBaseURL,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// InitializeParams are sent to POST /transaction/initialize.
type InitializeParams struct {
	Email             string
	Amount            int64 // smallest currency unit (e.g. kobo for NGN)
	Currency          string
	Reference         string
	CallbackURL       string
	Metadata          map[string]string
	Subaccount        string // ACCT_xxx — mechanic subaccount
	TransactionCharge int64  // flat amount (minor units) to main/platform account
	Bearer            string // "account" (default) or "subaccount"
}

// CreateSubaccountParams are sent to POST /subaccount.
type CreateSubaccountParams struct {
	BusinessName       string
	BankCode           string
	AccountNumber      string
	PercentageCharge   float64 // share to main account; use 0 when using transaction_charge per txn
	PrimaryContactName string
	PrimaryContactEmail string
}

type SubaccountData struct {
	SubaccountCode string `json:"subaccount_code"`
	BusinessName   string `json:"business_name"`
	AccountNumber  string `json:"account_number"`
	AccountName    string `json:"account_name"`
	Active         bool   `json:"active"`
	IsVerified     bool   `json:"is_verified"`
}

type SubaccountResponse struct {
	Status  bool           `json:"status"`
	Message string         `json:"message"`
	Data    SubaccountData `json:"data"`
}

type Bank struct {
	Name string `json:"name"`
	Code string `json:"code"`
	Slug string `json:"slug"`
}

type BanksResponse struct {
	Status  bool   `json:"status"`
	Message string `json:"message"`
	Data    []Bank `json:"data"`
}

type InitializeData struct {
	AuthorizationURL string `json:"authorization_url"`
	AccessCode       string `json:"access_code"`
	Reference        string `json:"reference"`
}

type InitializeResponse struct {
	Status  bool           `json:"status"`
	Message string         `json:"message"`
	Data    InitializeData `json:"data"`
}

// VerifyData is returned from GET /transaction/verify/:reference.
type VerifyData struct {
	Reference string `json:"reference"`
	Amount    int64  `json:"amount"`
	Currency  string `json:"currency"`
	Status    string `json:"status"`
	Metadata  struct {
		EntityType string `json:"entity_type"`
		EntityID   string `json:"entity_id"`
	} `json:"metadata"`
}

type VerifyResponse struct {
	Status  bool       `json:"status"`
	Message string     `json:"message"`
	Data    VerifyData `json:"data"`
}

// RefundParams are sent to POST /refund.
type RefundParams struct {
	Transaction  string // Paystack transaction reference
	Amount       int64  // optional partial refund in minor units; 0 = full
	Currency     string
	CustomerNote string
	MerchantNote string
}

type RefundData struct {
	ID     int64  `json:"id"`
	Status string `json:"status"`
	Amount int64  `json:"amount"`
	Transaction struct {
		Reference string `json:"reference"`
	} `json:"transaction"`
}

type RefundResponse struct {
	Status  bool       `json:"status"`
	Message string     `json:"message"`
	Data    RefundData `json:"data"`
}

// WebhookEvent is the top-level Paystack webhook payload.
type WebhookEvent struct {
	Event string          `json:"event"`
	Data  json.RawMessage `json:"data"`
}

// ChargeWebhookData is the data object for charge.success events.
type ChargeWebhookData struct {
	Reference string `json:"reference"`
	Amount    int64  `json:"amount"`
	Currency  string `json:"currency"`
	Status    string `json:"status"`
	Metadata  struct {
		EntityType string `json:"entity_type"`
		EntityID   string `json:"entity_id"`
	} `json:"metadata"`
}

// RefundWebhookData is the data object for refund.* events.
type RefundWebhookData struct {
	ID     int64  `json:"id"`
	Status string `json:"status"`
	Transaction struct {
		Reference string `json:"reference"`
	} `json:"transaction"`
}

func (c *Client) Initialize(p InitializeParams) (*InitializeResponse, error) {
	body := map[string]interface{}{
		"email":  p.Email,
		"amount": p.Amount,
	}
	if p.Currency != "" {
		body["currency"] = p.Currency
	}
	if p.Reference != "" {
		body["reference"] = p.Reference
	}
	if p.CallbackURL != "" {
		body["callback_url"] = p.CallbackURL
	}
	if len(p.Metadata) > 0 {
		body["metadata"] = p.Metadata
	}
	if p.Subaccount != "" {
		body["subaccount"] = p.Subaccount
	}
	if p.TransactionCharge > 0 {
		body["transaction_charge"] = p.TransactionCharge
	}
	if p.Bearer != "" {
		body["bearer"] = p.Bearer
	}

	var out InitializeResponse
	if err := c.do(http.MethodPost, "/transaction/initialize", body, &out); err != nil {
		return nil, err
	}
	if !out.Status {
		return nil, fmt.Errorf("paystack initialize: %s", out.Message)
	}
	return &out, nil
}

func (c *Client) CreateSubaccount(p CreateSubaccountParams) (*SubaccountResponse, error) {
	body := map[string]interface{}{
		"business_name":     p.BusinessName,
		"bank_code":         p.BankCode,
		"account_number":    p.AccountNumber,
		"percentage_charge": p.PercentageCharge,
	}
	if p.PrimaryContactName != "" {
		body["primary_contact_name"] = p.PrimaryContactName
	}
	if p.PrimaryContactEmail != "" {
		body["primary_contact_email"] = p.PrimaryContactEmail
	}
	var out SubaccountResponse
	if err := c.do(http.MethodPost, "/subaccount", body, &out); err != nil {
		return nil, err
	}
	if !out.Status {
		return nil, fmt.Errorf("paystack create subaccount: %s", out.Message)
	}
	return &out, nil
}

func (c *Client) UpdateSubaccount(code string, bankCode, accountNumber string) (*SubaccountResponse, error) {
	body := map[string]interface{}{}
	if bankCode != "" {
		body["bank_code"] = bankCode
	}
	if accountNumber != "" {
		body["account_number"] = accountNumber
	}
	var out SubaccountResponse
	if err := c.do(http.MethodPut, "/subaccount/"+code, body, &out); err != nil {
		return nil, err
	}
	if !out.Status {
		return nil, fmt.Errorf("paystack update subaccount: %s", out.Message)
	}
	return &out, nil
}

func (c *Client) ListBanks(country string) (*BanksResponse, error) {
	path := "/bank"
	if country != "" {
		path += "?country=" + country
	}
	var out BanksResponse
	if err := c.do(http.MethodGet, path, nil, &out); err != nil {
		return nil, err
	}
	if !out.Status {
		return nil, fmt.Errorf("paystack list banks: %s", out.Message)
	}
	return &out, nil
}

func (c *Client) CreateRefund(p RefundParams) (*RefundResponse, error) {
	body := map[string]interface{}{
		"transaction": p.Transaction,
	}
	if p.Amount > 0 {
		body["amount"] = p.Amount
	}
	if p.Currency != "" {
		body["currency"] = p.Currency
	}
	if p.CustomerNote != "" {
		body["customer_note"] = p.CustomerNote
	}
	if p.MerchantNote != "" {
		body["merchant_note"] = p.MerchantNote
	}
	var out RefundResponse
	if err := c.do(http.MethodPost, "/refund", body, &out); err != nil {
		return nil, err
	}
	if !out.Status {
		return nil, fmt.Errorf("paystack refund: %s", out.Message)
	}
	return &out, nil
}

func (c *Client) FetchRefund(id int64) (*RefundResponse, error) {
	var out RefundResponse
	if err := c.do(http.MethodGet, fmt.Sprintf("/refund/%d", id), nil, &out); err != nil {
		return nil, err
	}
	if !out.Status {
		return nil, fmt.Errorf("paystack fetch refund: %s", out.Message)
	}
	return &out, nil
}

func (c *Client) Verify(reference string) (*VerifyResponse, error) {
	var out VerifyResponse
	if err := c.do(http.MethodGet, "/transaction/verify/"+reference, nil, &out); err != nil {
		return nil, err
	}
	if !out.Status {
		return nil, fmt.Errorf("paystack verify: %s", out.Message)
	}
	return &out, nil
}

// VerifyWebhookSignature checks the x-paystack-signature header (HMAC-SHA512 of raw body).
func VerifyWebhookSignature(secret string, body []byte, signature string) bool {
	if secret == "" || signature == "" {
		return false
	}
	mac := hmac.New(sha512.New, []byte(secret))
	mac.Write(body)
	expected := hex.EncodeToString(mac.Sum(nil))
	return hmac.Equal([]byte(expected), []byte(signature))
}

func (c *Client) do(method, path string, body interface{}, out interface{}) error {
	var reqBody io.Reader
	if body != nil {
		b, err := json.Marshal(body)
		if err != nil {
			return err
		}
		reqBody = bytes.NewReader(b)
	}

	req, err := http.NewRequest(method, c.baseURL+path, reqBody)
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+c.secretKey)
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	if resp.StatusCode >= 400 {
		return fmt.Errorf("paystack API %s %d: %s", path, resp.StatusCode, string(raw))
	}
	if out == nil {
		return nil
	}
	return json.Unmarshal(raw, out)
}
