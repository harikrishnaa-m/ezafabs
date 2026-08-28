package product

type CreateProductInput struct {
	Name       string `json:"name"`
	CategoryID string `json:"category_id"`
	Brand      string `json:"brand"`

	Description       string `json:"description"`
	FabricComposition string `json:"fabric_composition"`
	Pattern           string `json:"pattern"`
	Occasion          string `json:"occasion"`
	CareInstructions  string `json:"care_instructions"`

	IsWebVisible *bool  `json:"is_web_visible"`
	IsStitched   *bool  `json:"is_stitched"`
	UOM          string `json:"uom"`
}

type CreateProductWithVariantsInput struct {
	CreateProductInput
	MainImageURL     string                      `json:"main_image_url"`
	GalleryImageURLs []string                    `json:"gallery_image_urls"`
	IsActive         *bool                       `json:"is_active"`
	Variants         []CreateProductVariantInput `json:"variants"`
}

type CreateProductVariantInput struct {
	Price             float64  `json:"price"`
	CostPrice         float64  `json:"cost_price"`
	HSNCode           string   `json:"hsn_code"`
	IsActive          *bool    `json:"is_active"`
	AttributeValueIDs []string `json:"attribute_value_ids"`
	ImagePaths        []string `json:"image_paths"`
}

type UpdateProductInput struct {
	Name       *string `json:"name"`
	CategoryID *string `json:"category_id"`
	Brand      *string `json:"brand"`

	Description       *string `json:"description"`
	FabricComposition *string `json:"fabric_composition"`
	Pattern           *string `json:"pattern"`
	Occasion          *string `json:"occasion"`
	CareInstructions  *string `json:"care_instructions"`

	IsWebVisible *bool   `json:"is_web_visible"`
	IsStitched   *bool   `json:"is_stitched"`
	UOM          *string `json:"uom"`
}
