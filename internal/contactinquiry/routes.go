package contactinquiry

import "github.com/gofiber/fiber/v2"

func RegisterPublicRoutes(r fiber.Router, h *Handler) {
	r.Post("/", h.Create)
}

func RegisterAdminRoutes(r fiber.Router, h *Handler) {
	r.Get("/", h.List)
	r.Get("/:id", h.Get)
}
