package product

import (
	"math"
	"strings"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
)

type Handler struct {
	store *Store
}

func NewHandler(s *Store) *Handler {
	return &Handler{store: s}
}

// GET /ecom/products
func (h *Handler) List(c *fiber.Ctx) error {
	page := c.QueryInt("page", 1)
	limit := c.QueryInt("limit", 20)
	categoryID := c.Query("category_id")
	search := c.Query("q")
	attributeValueIDs, err := parseAttributeValueIDs(c)
	if err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "attribute_value_ids must contain valid UUIDs"})
	}

	products, total, err := h.store.ListProducts(categoryID, search, attributeValueIDs, page, limit)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": "failed to fetch products"})
	}
	if products == nil {
		products = []map[string]interface{}{}
	}

	return c.JSON(fiber.Map{
		"page":        page,
		"limit":       limit,
		"total":       total,
		"total_pages": int(math.Ceil(float64(total) / float64(limit))),
		"data":        products,
	})
}

func parseAttributeValueIDs(c *fiber.Ctx) ([]string, error) {
	var values []string
	c.Context().QueryArgs().VisitAll(func(key, value []byte) {
		if string(key) != "attribute_value_ids" {
			return
		}
		for _, rawValue := range strings.Split(string(value), ",") {
			value := strings.TrimSpace(rawValue)
			if value != "" {
				values = append(values, value)
			}
		}
	})

	for _, value := range values {
		if _, err := uuid.Parse(value); err != nil {
			return nil, err
		}
	}
	return values, nil
}

// GET /ecom/products/:id
func (h *Handler) GetByID(c *fiber.Ctx) error {
	id := c.Params("id")
	product, err := h.store.GetProductDetail(id)
	if err != nil {
		return c.Status(404).JSON(fiber.Map{"error": "product not found"})
	}
	return c.JSON(product)
}

// GET /ecom/categories
func (h *Handler) Categories(c *fiber.Ctx) error {
	cats, err := h.store.ListCategories()
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": "failed to fetch categories"})
	}
	if cats == nil {
		cats = []map[string]interface{}{}
	}
	return c.JSON(fiber.Map{"categories": cats})
}
