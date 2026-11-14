package ecomreturn

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	ecomMw "defab-erp/internal/ecom/middleware"

	"github.com/gofiber/fiber/v2"
)

type Handler struct {
	store *Store
}

func NewHandler(s *Store) *Handler {
	return &Handler{store: s}
}

// ── Customer endpoints ────────────────────────────────────────────────────────

// POST /ecom/returns
func (h *Handler) RequestReturn(c *fiber.Ctx) error {
	cust := c.Locals("ecom_customer").(*ecomMw.EcomCustomer)

	var in struct {
		OrderID             string  `json:"order_id"`
		Reason              string  `json:"reason"`
		PayoutMethod        *string `json:"payout_method"`
		PayoutUPI           *string `json:"payout_upi"`
		PayoutAccountNumber *string `json:"payout_account_number"`
		PayoutIFSC          *string `json:"payout_ifsc"`
		PayoutAccountName   *string `json:"payout_account_name"`
	}
	if err := c.BodyParser(&in); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "invalid JSON"})
	}
	if in.OrderID == "" || in.Reason == "" {
		return c.Status(400).JSON(fiber.Map{"error": "order_id and reason are required"})
	}

	ret, err := h.store.RequestReturn(
		cust.ID, in.OrderID, in.Reason,
		in.PayoutMethod, in.PayoutUPI, in.PayoutAccountNumber, in.PayoutIFSC, in.PayoutAccountName,
	)
	if err != nil {
		msg := err.Error()
		switch msg {
		case "order not found":
			return c.Status(404).JSON(fiber.Map{"error": msg})
		case "only delivered orders are eligible for return",
			"return window has expired (7 days from delivery)",
			"a return request already exists for this order",
			"delivery date not recorded; cannot process return":
			return c.Status(400).JSON(fiber.Map{"error": msg})
		default:
			if strings.HasPrefix(msg, "payout") {
				return c.Status(400).JSON(fiber.Map{"error": msg})
			}
			return c.Status(500).JSON(fiber.Map{"error": "failed to create return request"})
		}
	}

	return c.Status(201).JSON(ret)
}

// GET /ecom/returns/:id
func (h *Handler) GetReturn(c *fiber.Ctx) error {
	cust := c.Locals("ecom_customer").(*ecomMw.EcomCustomer)
	returnID := c.Params("id")

	ret, err := h.store.GetReturn(cust.ID, returnID)
	if err != nil {
		if err.Error() == "return request not found" {
			return c.Status(404).JSON(fiber.Map{"error": "return request not found"})
		}
		return c.Status(500).JSON(fiber.Map{"error": "failed to fetch return"})
	}
	return c.JSON(ret)
}

// GET /ecom/returns
func (h *Handler) ListReturns(c *fiber.Ctx) error {
	cust := c.Locals("ecom_customer").(*ecomMw.EcomCustomer)

	list, err := h.store.ListReturns(cust.ID)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": "failed to fetch returns"})
	}
	if list == nil {
		list = []ReturnRequest{}
	}
	return c.JSON(list)
}

// ── Admin endpoints ───────────────────────────────────────────────────────────

// GET /admin/ecom-returns
func (h *Handler) AdminListReturns(c *fiber.Ctx) error {
	status := c.Query("status")
	page, _ := strconv.Atoi(c.Query("page", "1"))
	limit, _ := strconv.Atoi(c.Query("limit", "20"))
	if page < 1 {
		page = 1
	}
	if limit < 1 || limit > 100 {
		limit = 20
	}

	list, err := h.store.AdminListReturns(status, page, limit)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": "failed to fetch returns"})
	}
	if list == nil {
		list = []ReturnRequest{}
	}
	return c.JSON(list)
}

// PATCH /admin/ecom-returns/:id/approve
// Approves the return and queues a manual refund.
func (h *Handler) AdminApproveReturn(c *fiber.Ctx) error {
	returnID := c.Params("id")

	ret, info, err := h.store.AdminGetReturnWithOrder(returnID)
	if err != nil {
		if err.Error() == "return request not found" {
			return c.Status(404).JSON(fiber.Map{"error": "return request not found"})
		}
		return c.Status(500).JSON(fiber.Map{"error": err.Error()})
	}

	if ret.Status != "REQUESTED" {
		return c.Status(400).JSON(fiber.Map{"error": fmt.Sprintf("cannot approve a return in %s state", ret.Status)})
	}

	if err := h.store.AdminApproveReturn(returnID); err != nil {
		return c.Status(500).JSON(fiber.Map{"error": err.Error()})
	}

	if err := h.store.MarkRefundNotInitiated(returnID, info.OrderNumber); err != nil {
		return c.Status(500).JSON(fiber.Map{"error": "return approved but refund status update failed"})
	}

	return c.JSON(fiber.Map{"message": "return approved", "refund_status": "REFUND_NOT_INITIATED"})
}

// PATCH /admin/ecom-returns/:id/reject
func (h *Handler) AdminRejectReturn(c *fiber.Ctx) error {
	returnID := c.Params("id")

	if err := h.store.AdminRejectReturn(returnID); err != nil {
		msg := err.Error()
		if msg == "return not found or not in REQUESTED state" {
			return c.Status(404).JSON(fiber.Map{"error": msg})
		}
		return c.Status(500).JSON(fiber.Map{"error": msg})
	}
	return c.JSON(fiber.Map{"message": "return rejected"})
}

// PATCH /admin/ecom-returns/:id/refund
func (h *Handler) AdminCompleteRefund(c *fiber.Ctx) error {
	returnID := c.Params("id")
	var in struct {
		Amount     float64    `json:"refunded_amount"`
		Reference  string     `json:"refund_reference"`
		RefundedAt *time.Time `json:"refunded_at"`
		Note       string     `json:"note"`
	}
	if err := c.BodyParser(&in); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "invalid JSON"})
	}
	if in.Amount <= 0 || in.Reference == "" {
		return c.Status(400).JSON(fiber.Map{"error": "refunded_amount and refund_reference are required"})
	}
	refundedAt := time.Now()
	if in.RefundedAt != nil {
		refundedAt = *in.RefundedAt
	}
	if err := h.store.AdminCompleteRefund(returnID, in.Amount, in.Reference, refundedAt, in.Note); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": err.Error()})
	}
	return c.JSON(fiber.Map{"message": "refund marked completed"})
}
