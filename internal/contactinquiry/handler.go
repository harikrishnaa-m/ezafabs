package contactinquiry

import (
	"database/sql"
	"log"
	"net/mail"
	"strings"

	"defab-erp/internal/core/httperr"

	"github.com/gofiber/fiber/v2"
)

type Handler struct {
	store *Store
}

func NewHandler(store *Store) *Handler {
	return &Handler{store: store}
}

func (h *Handler) Create(c *fiber.Ctx) error {
	var in CreateContactInquiryInput
	if err := c.BodyParser(&in); err != nil {
		return httperr.BadRequest(c, "Invalid request body")
	}

	in.FirstName = strings.TrimSpace(in.FirstName)
	in.LastName = strings.TrimSpace(in.LastName)
	in.Email = strings.TrimSpace(in.Email)
	in.Phone = strings.TrimSpace(in.Phone)
	in.Message = strings.TrimSpace(in.Message)

	if in.FirstName == "" || in.LastName == "" || in.Email == "" || in.Message == "" {
		return httperr.BadRequest(c, "first_name, last_name, email, and message are required")
	}
	if _, err := mail.ParseAddress(in.Email); err != nil {
		return httperr.BadRequest(c, "email must be valid")
	}

	id, err := h.store.Create(in)
	if err != nil {
		log.Println("create contact inquiry error:", err)
		return httperr.Internal(c)
	}

	return c.Status(201).JSON(fiber.Map{
		"id":      id,
		"message": "Your message has been submitted successfully.",
	})
}

func (h *Handler) List(c *fiber.Ctx) error {
	page := c.QueryInt("page", 1)
	limit := c.QueryInt("limit", 20)
	limit, offset, err := validateLimit(page, limit)
	if err != nil {
		return httperr.BadRequest(c, err.Error())
	}

	inquiries, err := h.store.List(limit, offset)
	if err != nil {
		log.Println("list contact inquiries error:", err)
		return httperr.Internal(c)
	}
	if inquiries == nil {
		inquiries = []ContactInquiry{}
	}
	total, err := h.store.Count()
	if err != nil {
		log.Println("count contact inquiries error:", err)
		return httperr.Internal(c)
	}

	return c.JSON(fiber.Map{
		"data":  inquiries,
		"page":  page,
		"limit": limit,
		"total": total,
	})
}

func (h *Handler) Get(c *fiber.Ctx) error {
	inquiry, err := h.store.Get(c.Params("id"))
	if err != nil {
		if err == sql.ErrNoRows {
			return httperr.NotFound(c, "Contact inquiry not found")
		}
		log.Println("get contact inquiry error:", err)
		return httperr.Internal(c)
	}
	return c.JSON(inquiry)
}
