package payment

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"
	"unicode"
)

func cofeeBaseURL() string {
	if strings.EqualFold(os.Getenv("COFEE_ENV"), "production") {
		return "https://partner-api.cofee.life"
	}
	return "https://partner-api.sandbox.cofee.life"
}

func normalizeCoFeeMobile(phone string) (string, error) {
	digits := strings.Map(func(r rune) rune {
		if unicode.IsDigit(r) {
			return r
		}
		return -1
	}, phone)

	switch {
	case len(digits) == 10:
		if digits[0] < '6' || digits[0] > '9' {
			return "", fmt.Errorf("invalid Indian mobile number")
		}
		return "+91" + digits, nil
	case len(digits) == 11 && strings.HasPrefix(digits, "0"):
		if digits[1] < '6' || digits[1] > '9' {
			return "", fmt.Errorf("invalid Indian mobile number")
		}
		return "+91" + digits[1:], nil
	case len(digits) == 12 && strings.HasPrefix(digits, "91"):
		if digits[2] < '6' || digits[2] > '9' {
			return "", fmt.Errorf("invalid Indian mobile number")
		}
		return "+" + digits, nil
	default:
		return "", fmt.Errorf("invalid Indian mobile number")
	}
}

type CoFeeCustomer struct {
	Name                string `json:"name"`
	Email               string `json:"email"`
	Mobile              string `json:"mobile"`
	CustomerReferenceID string `json:"customer_reference_id,omitempty"`
}

type CoFeeAmount float64

func (a *CoFeeAmount) UnmarshalJSON(data []byte) error {
	value := strings.TrimSpace(string(data))
	if len(value) >= 2 && value[0] == '"' && value[len(value)-1] == '"' {
		value = strings.Trim(value, `"`)
	}
	amount, err := strconv.ParseFloat(value, 64)
	if err != nil {
		return fmt.Errorf("invalid cofee amount %q: %w", value, err)
	}
	*a = CoFeeAmount(amount)
	return nil
}

type CoFeeCreateOrderRequest struct {
	BranchID               string        `json:"branch_id"`
	GroupID                string        `json:"group_id,omitempty"`
	Amount                 float64       `json:"amount"`
	Currency               string        `json:"currency"`
	MerchantOrderID        string        `json:"merchant_order_id"`
	OrderPurpose           string        `json:"order_purpose"`
	NotifyCustomer         bool          `json:"notify_customer"`
	CustomerDetails        CoFeeCustomer `json:"customer_details"`
	RedirectURL            string        `json:"redirect_url,omitempty"`
	SendReceiptToCustomer  bool          `json:"send_receipt_to_customer"`
	HideReviewPage         bool          `json:"hide_review_page"`
	PreferredPaymentMethod string        `json:"preferred_payment_method,omitempty"`
}

type CoFeeCreateOrderResponse struct {
	Status string            `json:"status"`
	Data   CoFeePaymentOrder `json:"data"`
	Error  *CoFeeAPIError    `json:"error"`
}

type CoFeePaymentOrder struct {
	Amount          CoFeeAmount `json:"amount"`
	MerchantOrderID string      `json:"merchant_order_id"`
	PaymentLink     string      `json:"payment_link"`
	OrderStatus     string      `json:"order_status"`
	OrderID         string      `json:"order_id"`
	CreatedAt       time.Time   `json:"created_at"`
	ExpiryTime      time.Time   `json:"expiry_time"`
	PaidAt          *time.Time  `json:"paid_at"`
}

type CoFeeAPIError struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

type CoFeeWebhook struct {
	EventName string `json:"event_name"`
	Data      struct {
		Amount          string `json:"amount"`
		TotalAmount     string `json:"total_amount"`
		MerchantOrderID string `json:"merchant_order_id"`
		OrderID         string `json:"order_id"`
		OrderStatus     string `json:"order_status"`
		PaymentStatus   string `json:"payment_status"`
	} `json:"data"`
}

func cofeeRequest(method, path string, body interface{}) (*http.Response, error) {
	var reader io.Reader
	if body != nil {
		payload, err := json.Marshal(body)
		if err != nil {
			return nil, err
		}
		reader = bytes.NewReader(payload)
	}

	req, err := http.NewRequest(method, cofeeBaseURL()+path, reader)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("x-api-key", os.Getenv("COFEE_API_KEY"))

	resp, err := (&http.Client{Timeout: 15 * time.Second}).Do(req)
	if err != nil {
		return nil, fmt.Errorf("cofee request failed: %w", err)
	}
	return resp, nil
}

func CreateCoFeeOrder(req CoFeeCreateOrderRequest) (*CoFeePaymentOrder, error) {
	resp, err := cofeeRequest(http.MethodPost, "/v2/payment-order", req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("cofee error %d: %s", resp.StatusCode, string(body))
	}

	var result CoFeeCreateOrderResponse
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("failed to parse cofee response: %w", err)
	}
	if result.Status != "SUCCESS" {
		if result.Error != nil {
			return nil, fmt.Errorf("cofee error %s: %s", result.Error.Code, result.Error.Message)
		}
		return nil, fmt.Errorf("cofee returned status %q", result.Status)
	}
	return &result.Data, nil
}

func GetCoFeeOrder(orderID string) (*CoFeePaymentOrder, error) {
	resp, err := cofeeRequest(http.MethodGet, "/v1/payment-order/"+orderID, nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("cofee error %d: %s", resp.StatusCode, string(body))
	}

	var result CoFeeCreateOrderResponse
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("failed to parse cofee response: %w", err)
	}
	if result.Status != "SUCCESS" {
		return nil, fmt.Errorf("cofee returned status %q", result.Status)
	}
	return &result.Data, nil
}

func DisableCoFeeOrder(orderID string) error {
	resp, err := cofeeRequest(http.MethodPatch, "/v1/payment-order/"+orderID+"/disable", nil)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("cofee cancellation error %d: %s", resp.StatusCode, string(body))
	}
	return nil
}
