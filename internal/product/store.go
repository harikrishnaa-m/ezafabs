package product

import (
	"crypto/rand"
	"database/sql"
	"fmt"
	"math/big"
	"strings"
)

type Store struct {
	db *sql.DB
}

func NewStore(db *sql.DB) *Store {
	return &Store{db: db}
}

//
// CREATE
//

// func (s *Store) Create(in CreateProductInput) error {
// 	_, err := s.db.Exec(`
// 	INSERT INTO products
// 	(name, category_id, brand, image_url, is_web_visible, is_stitched, uom)
// 	VALUES ($1,$2,$3,$4,
// 	        COALESCE($5, TRUE),
// 	        COALESCE($6, FALSE),
// 	        COALESCE($7,'Unit'))
// 	`,
// 		in.Name,
// 		in.CategoryID,
// 		in.Brand,
// 		in.ImageURL,
// 		in.IsWebVisible,
// 		in.IsStitched,
// 		in.UOM,
// 	)
// 	return err
// }

//
// LIST ACTIVE + category join + pagination
//

func (s *Store) ListImages(productID string) (*sql.Rows, error) {
	return s.db.Query(`
		SELECT id, image_url
		FROM product_images
		WHERE product_id = $1
		ORDER BY created_at
	`, productID)
}

// func (s *Store) ListImages(productID string) (*sql.Rows, error) {
// 	return s.db.Query(`
// 	SELECT image_url
// 	FROM product_images
// 	WHERE product_id=$1
// 	ORDER BY created_at
// 	`, productID)
// }

func (s *Store) List(limit, offset int) (*sql.Rows, error) {

	return s.db.Query(`
	SELECT
		p.id,
		p.name,
		p.brand,
		p.main_image_url,
		p.is_web_visible,
		p.is_stitched,
		p.uom,
		p.created_at,
		c.id,
		c.name,
		p.is_active,
		p.description,
		p.fabric_composition,
		p.pattern,
		p.occasion,
		p.care_instructions
	FROM products p
	JOIN categories c ON c.id = p.category_id
	ORDER BY p.created_at DESC
	LIMIT $1 OFFSET $2
	`, limit, offset)
}

func (s *Store) CountActive() (int, error) {

	var n int

	err := s.db.QueryRow(`
	SELECT COUNT(*)
	FROM products
	WHERE is_active = TRUE
	`).Scan(&n)

	return n, err
}

//
// GET BY ID
//

func (s *Store) Get(id string) *sql.Row {
	return s.db.QueryRow(`
		SELECT
			p.id,
			p.name,
			p.brand,
			p.main_image_url,
			p.is_web_visible,
			p.is_stitched,
			p.uom,
			p.is_active,
			c.id,
			c.name,
			p.description,
			p.fabric_composition,
			p.pattern,
			p.occasion,
			p.care_instructions
		FROM products p
		JOIN categories c ON p.category_id = c.id
		WHERE p.id=$1
		`, id)
}

//
// UPDATE
//

func (s *Store) Update(id string, in UpdateProductInput) error {

	_, err := s.db.Exec(`
	UPDATE products SET
	  name = COALESCE($1,name),
	  category_id = COALESCE($2,category_id),
	  brand = COALESCE($3,brand),
	  is_web_visible = COALESCE($4,is_web_visible),
	  is_stitched = COALESCE($5,is_stitched),
	  uom = COALESCE($6,uom),
	  updated_at = NOW()
	WHERE id=$7
	`,
		in.Name,
		in.CategoryID,
		in.Brand,
		in.IsWebVisible,
		in.IsStitched,
		in.UOM,
		id,
	)

	return err
}

//
// SOFT DELETE / RESTORE
//

func (s *Store) SetActive(id string, active bool) error {
	_, err := s.db.Exec(
		`UPDATE products SET is_active=$1 WHERE id=$2`,
		active, id,
	)
	return err
}

func (s *Store) IncrementCategoryProductCount(categoryID string) error {
	_, err := s.db.Exec(`UPDATE categories SET products_count = products_count + 1 WHERE id = $1`, categoryID)
	return err
}

func (s *Store) CreateProduct(
	in CreateProductInput,
	mainImageURL string,
) (string, error) {

	var id string

	err := s.db.QueryRow(`
		       INSERT INTO products
		       (name, category_id, brand, main_image_url, description, fabric_composition, pattern, occasion, care_instructions)
		       VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9)
		       RETURNING id
	       `,
		in.Name,
		in.CategoryID,
		in.Brand,
		mainImageURL,
		in.Description,
		in.FabricComposition,
		in.Pattern,
		in.Occasion,
		in.CareInstructions,
	).Scan(&id)
	if err != nil {
		return "", err
	}
	// Increment products_count for the category
	_ = s.IncrementCategoryProductCount(in.CategoryID)
	return id, nil
}

func (s *Store) CreateProductWithVariants(in CreateProductWithVariantsInput) (string, []map[string]interface{}, error) {
	tx, err := s.db.Begin()
	if err != nil {
		return "", nil, err
	}
	defer tx.Rollback()

	isWebVisible := true
	if in.IsWebVisible != nil {
		isWebVisible = *in.IsWebVisible
	}
	isStitched := false
	if in.IsStitched != nil {
		isStitched = *in.IsStitched
	}
	uom := in.UOM
	if uom == "" {
		uom = "Unit"
	}
	isActive := true
	if in.IsActive != nil {
		isActive = *in.IsActive
	}

	var productID string
	err = tx.QueryRow(`
		INSERT INTO products
		(name, category_id, brand, description, fabric_composition, pattern, occasion,
		 care_instructions, main_image_url, is_web_visible, is_stitched, uom, is_active)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13)
		RETURNING id
	`, in.Name, in.CategoryID, in.Brand, in.Description, in.FabricComposition,
		in.Pattern, in.Occasion, in.CareInstructions, in.MainImageURL, isWebVisible, isStitched,
		uom, isActive).Scan(&productID)
	if err != nil {
		return "", nil, fmt.Errorf("create product: %w", err)
	}

	if _, err = tx.Exec(`
		UPDATE categories SET products_count = products_count + 1 WHERE id = $1
	`, in.CategoryID); err != nil {
		return "", nil, fmt.Errorf("update category count: %w", err)
	}
	for _, imageURL := range in.GalleryImageURLs {
		if _, err = tx.Exec(`
			INSERT INTO product_images (product_id, image_url) VALUES ($1,$2)
		`, productID, imageURL); err != nil {
			return "", nil, fmt.Errorf("save product gallery: %w", err)
		}
	}

	variants := make([]map[string]interface{}, 0, len(in.Variants))
	for _, variant := range in.Variants {
		attributeNames := make([]string, 0, len(variant.AttributeValueIDs))
		for _, attributeValueID := range variant.AttributeValueIDs {
			var value string
			if err = tx.QueryRow(`SELECT value FROM attribute_values WHERE id = $1`, attributeValueID).Scan(&value); err != nil {
				return "", nil, fmt.Errorf("attribute value %s: %w", attributeValueID, err)
			}
			attributeNames = append(attributeNames, strings.TrimSpace(value))
		}
		variantNameParts := append([]string{strings.TrimSpace(in.Name)}, attributeNames...)
		variantName := strings.Join(variantNameParts, " ")

		var variantCode int
		if err = tx.QueryRow(`SELECT nextval('variant_code_seq')`).Scan(&variantCode); err != nil {
			return "", nil, fmt.Errorf("generate variant code: %w", err)
		}

		var sku, barcode string
		var exists bool
		for {
			randomSuffix, randomErr := randomDigits(3)
			if randomErr != nil {
				return "", nil, fmt.Errorf("generate sku suffix: %w", randomErr)
			}
			sku = fmt.Sprintf("%s-%03d-%s", strings.ToUpper(first3(variantName)), variantCode%1000, randomSuffix)
			barcode, randomErr = generateEAN13()
			if randomErr != nil {
				return "", nil, fmt.Errorf("generate barcode: %w", randomErr)
			}
			if err = tx.QueryRow(`
				SELECT EXISTS(SELECT 1 FROM variants WHERE sku = $1 OR barcode = $2)
			`, sku, barcode).Scan(&exists); err != nil {
				return "", nil, fmt.Errorf("check generated identifiers: %w", err)
			}
			if !exists {
				break
			}
		}

		isActive := true
		if variant.IsActive != nil {
			isActive = *variant.IsActive
		}

		var variantID string
		err = tx.QueryRow(`
			INSERT INTO variants
			(product_id, name, sku, price, cost_price, barcode, variant_code, hsn_code, is_active)
			VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9)
			RETURNING id
		`, productID, variantName, sku, variant.Price, variant.CostPrice, barcode,
			variantCode, variant.HSNCode, isActive).Scan(&variantID)
		if err != nil {
			return "", nil, fmt.Errorf("create variant %s: %w", variantName, err)
		}

		if _, err = tx.Exec(`
			INSERT INTO stocks (variant_id, warehouse_id, quantity, stock_type, updated_at)
			SELECT $1, id, 0, 'PRODUCT', NOW()
			FROM warehouses
			ON CONFLICT (variant_id, warehouse_id) DO NOTHING
		`, variantID); err != nil {
			return "", nil, fmt.Errorf("create warehouse stock for variant %s: %w", variantName, err)
		}

		if isWebVisible {
			if _, err = tx.Exec(`
				INSERT INTO online_stocks (variant_id, quantity, updated_at)
				VALUES ($1, 0, NOW())
				ON CONFLICT (variant_id) DO NOTHING
			`, variantID); err != nil {
				return "", nil, fmt.Errorf("create online stock for variant %s: %w", variantName, err)
			}
		}

		for _, attributeValueID := range variant.AttributeValueIDs {
			if _, err = tx.Exec(`
				INSERT INTO variant_attribute_mapping (variant_id, attribute_value_id)
				VALUES ($1,$2)
			`, variantID, attributeValueID); err != nil {
				return "", nil, fmt.Errorf("map attributes for variant %s: %w", variantName, err)
			}
		}

		for _, imagePath := range variant.ImagePaths {
			if _, err = tx.Exec(`
				INSERT INTO variant_images (variant_id, image_url) VALUES ($1,$2)
			`, variantID, imagePath); err != nil {
				return "", nil, fmt.Errorf("save images for variant %s: %w", variantName, err)
			}
		}

		variants = append(variants, map[string]interface{}{
			"id":            variantID,
			"name":          variantName,
			"sku":           sku,
			"barcode":       barcode,
			"variant_code":  variantCode,
			"attribute_ids": variant.AttributeValueIDs,
		})
	}

	if err = tx.Commit(); err != nil {
		return "", nil, fmt.Errorf("commit product and variants: %w", err)
	}
	return productID, variants, nil
}

func first3(value string) string {
	value = strings.ReplaceAll(value, " ", "")
	if len(value) > 3 {
		return value[:3]
	}
	return value
}

func randomDigits(length int) (string, error) {
	var digits strings.Builder
	for index := 0; index < length; index++ {
		digit, err := rand.Int(rand.Reader, big.NewInt(10))
		if err != nil {
			return "", err
		}
		digits.WriteByte(byte('0' + digit.Int64()))
	}
	return digits.String(), nil
}

func generateEAN13() (string, error) {
	digits, err := randomDigits(12)
	if err != nil {
		return "", err
	}

	sum := 0
	for index, digit := range digits {
		value := int(digit - '0')
		if index%2 == 1 {
			sum += value * 3
		} else {
			sum += value
		}
	}
	checkDigit := (10 - sum%10) % 10
	return fmt.Sprintf("%s%d", digits, checkDigit), nil
}

func (s *Store) InsertProductImage(productID, url string) error {

	_, err := s.db.Exec(`
	INSERT INTO product_images (product_id, image_url)
	VALUES ($1,$2)
	`, productID, url)

	return err
}

func (s *Store) GetMainImage(productID string) (string, error) {
	var url string
	err := s.db.QueryRow(`
	SELECT main_image_url
	FROM products
	WHERE id=$1
	`, productID).Scan(&url)
	return url, err
}

func (s *Store) UpdateMainImage(productID, url string) error {
	_, err := s.db.Exec(`
	UPDATE products
	SET main_image_url=$1, updated_at=NOW()
	WHERE id=$2
	`, url, productID)
	return err
}

func (s *Store) GetProductImage(id string) (string, error) {
	var url string
	err := s.db.QueryRow(`
	SELECT image_url
	FROM product_images
	WHERE id=$1
	`, id).Scan(&url)
	return url, err
}

func (s *Store) DeleteProductImage(id string) error {
	_, err := s.db.Exec(`
	DELETE FROM product_images
	WHERE id=$1
	`, id)
	return err
}

func extractKey(url string) string {
	// assumes CDN/base url + key
	// https://cdn/.../products/abc.jpg → products/abc.jpg
	i := strings.Index(url, "/products/")
	if i == -1 {
		return ""
	}
	return url[i+1:]
}
