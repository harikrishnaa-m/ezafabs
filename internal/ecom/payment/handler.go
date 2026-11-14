package payment

import (
	"database/sql"
	"encoding/json"
	"log"
	"os"
	"strings"

	ecomMw "defab-erp/internal/ecom/middleware"

	"github.com/gofiber/fiber/v2"
)

type Handler struct {
	store *Store
	db    *sql.DB
}

func NewHandler(s *Store, db *sql.DB) *Handler {
	return &Handler{store: s, db: db}
}

// POST /ecom/payments/initiate
// Called after checkout with payment_method=ONLINE. Creates a CoFee order.
func (h *Handler) Initiate(c *fiber.Ctx) error {
	cust := c.Locals("ecom_customer").(*ecomMw.EcomCustomer)

	var in struct {
		OrderID string `json:"order_id"`
	}
	if err := c.BodyParser(&in); err != nil || in.OrderID == "" {
		return c.Status(400).JSON(fiber.Map{"error": "order_id is required"})
	}

	// Load order
	var orderNumber, custName, custPhone, custEmail string
	var grandTotal float64
	var payMethod, payStatus string
	var custPhone_ sql.NullString

	err := h.db.QueryRow(`
		SELECT o.order_number, o.grand_total, o.payment_method, o.payment_status,
		       ec.name, ec.email, ec.phone
		FROM ecom_orders o
		JOIN ecom_customers ec ON ec.id = o.customer_id
		WHERE o.id = $1 AND o.customer_id = $2
	`, in.OrderID, cust.ID).Scan(&orderNumber, &grandTotal, &payMethod, &payStatus, &custName, &custEmail, &custPhone_)
	if err != nil {
		return c.Status(404).JSON(fiber.Map{"error": "order not found"})
	}
	if payMethod != "ONLINE" {
		return c.Status(400).JSON(fiber.Map{"error": "order is not an online payment order"})
	}
	if payStatus == "PAID" {
		return c.Status(400).JSON(fiber.Map{"error": "order already paid"})
	}
	if custPhone_.Valid {
		custPhone = custPhone_.String
	}
	if custPhone == "" {
		return c.Status(400).JSON(fiber.Map{"error": "customer phone number is required for payment"})
	}
	cofeeMobile, err := normalizeCoFeeMobile(custPhone)
	if err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "customer phone number must be a valid Indian mobile number"})
	}

	branchID, err := h.store.GetCentralCoFeeBranchID()
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": err.Error()})
	}
	returnURL := os.Getenv("COFEE_REDIRECT_URL")
	if returnURL == "" {
		returnURL = "https://yoursite.com/payment/result"
	}

	cofeeReq := CoFeeCreateOrderRequest{
		BranchID:        branchID,
		GroupID:         os.Getenv("COFEE_GROUP_ID"),
		Amount:          grandTotal,
		Currency:        "INR",
		MerchantOrderID: orderNumber,
		OrderPurpose:    "E-commerce order " + orderNumber,
		NotifyCustomer:  false,
		CustomerDetails: CoFeeCustomer{
			Name:                custName,
			Email:               custEmail,
			Mobile:              custPhone,
			CustomerReferenceID: cust.ID,
		},
		RedirectURL:           returnURL + "?order_id=" + in.OrderID,
		SendReceiptToCustomer: true,
		HideReviewPage:        false,
	}

	cofeeReq.CustomerDetails.Mobile = cofeeMobile
	cofeeResp, err := CreateCoFeeOrder(cofeeReq)
	if err != nil {
		return c.Status(502).JSON(fiber.Map{"error": "payment gateway error: " + err.Error()})
	}

	if err := h.store.SaveCoFeeOrder(in.OrderID, cofeeResp.OrderID, cofeeResp.PaymentLink); err != nil {
		return c.Status(500).JSON(fiber.Map{"error": "failed to save payment session"})
	}

	return c.JSON(fiber.Map{
		"order_id":       in.OrderID,
		"cofee_order_id": cofeeResp.OrderID,
		"payment_link":   cofeeResp.PaymentLink,
		"amount":         float64(cofeeResp.Amount),
		"expires_at":     cofeeResp.ExpiryTime,
	})
}

// POST /ecom/payments/webhook  (public — no auth, verified by signature)
func (h *Handler) Webhook(c *fiber.Ctx) error {
	rawBody := c.Body()
	log.Printf("[COFEE WEBHOOK] received method=%s path=%s body_bytes=%d", c.Method(), c.Path(), len(rawBody))
	headerName := os.Getenv("COFEE_WEBHOOK_HEADER")
	headerValue := os.Getenv("COFEE_WEBHOOK_SECRET")
	if headerName == "" || headerValue == "" {
		log.Printf("[COFEE WEBHOOK] rejected: webhook verification is not configured")
		return c.Status(500).SendString("cofee webhook verification is not configured")
	}
	if c.Get(headerName) != headerValue {
		log.Printf("[COFEE WEBHOOK] rejected: invalid verification header")
		return c.Status(401).SendString("invalid webhook credentials")
	}
	log.Printf("[COFEE WEBHOOK] verification accepted")

	var payload CoFeeWebhook
	if err := json.Unmarshal(rawBody, &payload); err != nil {
		log.Printf("[COFEE WEBHOOK] rejected: invalid JSON: %v", err)
		return c.Status(400).SendString("invalid JSON")
	}
	log.Printf("[COFEE WEBHOOK] event=%q merchant_order_id=%q cofee_order_id=%q order_status=%q payment_status=%q",
		payload.EventName,
		payload.Data.MerchantOrderID,
		payload.Data.OrderID,
		payload.Data.OrderStatus,
		payload.Data.PaymentStatus,
	)

	if payload.EventName == "payment-order.paid" &&
		strings.EqualFold(payload.Data.PaymentStatus, "success") &&
		strings.EqualFold(payload.Data.OrderStatus, "success") {
		paymentRef := "COFEE-" + payload.Data.OrderID
		if err := h.store.MarkOrderPaid(payload.Data.MerchantOrderID, paymentRef); err != nil {
			log.Printf("[COFEE WEBHOOK] MarkOrderPaid failed: %v", err)
		} else {
			log.Printf("[COFEE WEBHOOK] payment marked PAID merchant_order_id=%q", payload.Data.MerchantOrderID)
		}
	} else {
		log.Printf("[COFEE WEBHOOK] acknowledged without payment update event=%q", payload.EventName)
	}

	return c.SendStatus(200)
}

// GET /ecom/payments/:order_id/verify
// Called by frontend after redirect to check if payment was captured.
func (h *Handler) Verify(c *fiber.Ctx) error {
	cust := c.Locals("ecom_customer").(*ecomMw.EcomCustomer)
	orderID := c.Params("order_id")

	// Fetch order_number and payment_status; order_number is what we sent to Cashfree
	var orderNumber, payStatus string
	err := h.db.QueryRow(`
		SELECT order_number, payment_status FROM ecom_orders
		WHERE id = $1 AND customer_id = $2
	`, orderID, cust.ID).Scan(&orderNumber, &payStatus)
	if err != nil {
		return c.Status(404).JSON(fiber.Map{"error": "order not found"})
	}

	// Already marked paid (e.g. via webhook)
	if payStatus == "PAID" {
		return c.JSON(fiber.Map{"payment_status": "PAID", "order_id": orderID})
	}

	var cofeeOrderID string
	err = h.db.QueryRow(`SELECT COALESCE(cofee_order_id, '') FROM ecom_orders WHERE id = $1`, orderID).Scan(&cofeeOrderID)
	if err != nil || cofeeOrderID == "" {
		return c.Status(409).JSON(fiber.Map{"error": "payment order has not been initialized"})
	}

	cofeeStatus, err := GetCoFeeOrder(cofeeOrderID)
	if err != nil {
		return c.Status(502).JSON(fiber.Map{"error": "could not verify with payment gateway"})
	}

	if cofeeStatus.OrderStatus == "success" {
		if err := h.store.MarkOrderPaid(orderNumber, "COFEE-"+cofeeStatus.OrderID); err != nil {
			return c.Status(500).JSON(fiber.Map{"error": "failed to update order"})
		}
		return c.JSON(fiber.Map{"payment_status": "PAID", "order_id": orderID})
	}

	return c.JSON(fiber.Map{"payment_status": cofeeStatus.OrderStatus, "order_id": orderID})
}

// GET /ecom/payments/:order_id/refund-status
// Manual refunds do not have a gateway status to poll.
func (h *Handler) RefundStatus(c *fiber.Ctx) error {
	cust := c.Locals("ecom_customer").(*ecomMw.EcomCustomer)
	orderID := c.Params("order_id")

	var orderNumber, payStatus string
	err := h.db.QueryRow(`
		SELECT order_number, payment_status FROM ecom_orders
		WHERE id = $1 AND customer_id = $2
	`, orderID, cust.ID).Scan(&orderNumber, &payStatus)
	if err != nil {
		return c.Status(404).JSON(fiber.Map{"error": "order not found"})
	}

	return c.JSON(fiber.Map{"refund_status": payStatus, "order_id": orderID})
}
