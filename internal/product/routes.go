package product

import "github.com/gofiber/fiber/v2"

func RegisterRoutes(r fiber.Router, h *Handler) {
	r.Post("/", h.Create) //tested
	r.Post("/with-variants", h.CreateWithVariants)
	r.Post("/upload-image", h.UploadImage)
	r.Post("/upload-images", h.UploadImages)

	r.Put("/:id/main-image", h.ReplaceMainImage)       //tested
	r.Post("/:id/images", h.AddGalleryImages)          //tested
	r.Delete("/images/:imageId", h.DeleteGalleryImage) //tested

	r.Get("/", h.List)        //tested
	r.Get("/:id", h.Get)      //tested
	r.Patch("/:id", h.Update) //tested

	r.Patch("/:id/deactivate", h.Deactivate) //tested
	r.Patch("/:id/activate", h.Activate)     //tested
}
