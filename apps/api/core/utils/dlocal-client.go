package utils

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"
)

type DlocalClient struct {
	httpClient *http.Client
	baseURL    string
	apiKey     string
	apiSecret  string
}

type DlocalCreatePaymentRequest struct {
	Amount          float64 `json:"amount"`
	Currency        string  `json:"currency"`
	OrderID         string  `json:"order_id"`
	Description     string  `json:"description"`
	NotificationURL string  `json:"notification_url"`
	SuccessURL      string  `json:"success_url"`
}

type DlocalCreatePaymentResponse struct {
	ID          string `json:"id"`
	Status      string `json:"status"`
	CheckoutURL string `json:"checkout_url"`
	RedirectURL string `json:"redirect_url"`
}

// URL returns whichever checkout URL field dlocal populated.
func (r *DlocalCreatePaymentResponse) URL() string {
	if r.CheckoutURL != "" {
		return r.CheckoutURL
	}
	return r.RedirectURL
}

func NewDlocalClient() *DlocalClient {
	baseURL := "https://api-sbx.dlocalgo.com/v1"
	if os.Getenv("DLOCAL_ENV") == "production" {
		baseURL = "https://api.dlocalgo.com/v1"
	}
	return &DlocalClient{
		httpClient: &http.Client{Timeout: 30 * time.Second},
		baseURL:    baseURL,
		apiKey:     os.Getenv("DLOCAL_API_KEY"),
		apiSecret:  os.Getenv("DLOCAL_SECRET"),
	}
}

func (c *DlocalClient) CreatePayment(req DlocalCreatePaymentRequest) (*DlocalCreatePaymentResponse, error) {
	body, err := json.Marshal(req)
	if err != nil {
		return nil, err
	}

	httpReq, err := http.NewRequest(http.MethodPost, c.baseURL+"/payments", bytes.NewBuffer(body))
	if err != nil {
		return nil, err
	}

	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", fmt.Sprintf("Bearer %s:%s", c.apiKey, c.apiSecret))

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	rawBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("dlocal API error: status %d — body: %s", resp.StatusCode, rawBody)
	}

	var result DlocalCreatePaymentResponse
	if err := json.Unmarshal(rawBody, &result); err != nil {
		return nil, err
	}

	if result.URL() == "" {
		return nil, fmt.Errorf("dlocal response missing checkout URL — body: %s", rawBody)
	}

	return &result, nil
}
