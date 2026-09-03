package product

import (
	"math"
	"strconv"
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
	minPrice, maxPrice, err := parsePriceRange(c)
	if err != nil {
		return c.Status(400).JSON(fiber.Map{"error": err.Error()})
	}
	sort, err := parsePriceSort(c.Query("sort"))
	if err != nil {
		return c.Status(400).JSON(fiber.Map{"error": err.Error()})
	}
	attributeValueIDs, err := parseAttributeValueIDs(c)
	if err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "attribute_value_ids must contain valid UUIDs"})
	}

	products, total, err := h.store.ListProducts(categoryID, search, attributeValueIDs, minPrice, maxPrice, sort, page, limit)
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

func parsePriceRange(c *fiber.Ctx) (*float64, *float64, error) {
	var minPrice, maxPrice *float64
	for name, target := range map[string]**float64{
		"min_price": &minPrice,
		"max_price": &maxPrice,
	} {
		rawValue := c.Query(name)
		if rawValue == "" {
			continue
		}
		value, err := strconv.ParseFloat(rawValue, 64)
		if err != nil || value < 0 {
			return nil, nil, fiber.NewError(400, name+" must be a non-negative number")
		}
		*target = &value
	}
	if minPrice != nil && maxPrice != nil && *minPrice > *maxPrice {
		return nil, nil, fiber.NewError(400, "min_price must be less than or equal to max_price")
	}
	return minPrice, maxPrice, nil
}

func parsePriceSort(value string) (string, error) {
	switch value {
	case "":
		return "", nil
	case "price_asc", "low_to_high":
		return "price_asc", nil
	case "price_desc", "high_to_low":
		return "price_desc", nil
	default:
		return "", fiber.NewError(400, "sort must be price_asc or price_desc")
	}
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

// GET /ecom/attributes
func (h *Handler) Attributes(c *fiber.Ctx) error {
	attributes, err := h.store.ListAttributes()
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": "failed to fetch attributes"})
	}
	if attributes == nil {
		attributes = []map[string]interface{}{}
	}
	return c.JSON(fiber.Map{"attributes": attributes})
}
