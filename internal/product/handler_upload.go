package product

import (
	"defab-erp/internal/core/storage"
	"fmt"
	"mime/multipart"

	"github.com/gofiber/fiber/v2"
)

func (h *Handler) UploadImage(c *fiber.Ctx) error {
	folder, err := uploadFolder(c)
	if err != nil {
		return c.Status(400).JSON(fiber.Map{"error": err.Error()})
	}

	file, err := c.FormFile("image")
	if err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "image file is required"})
	}

	url, err := uploadImageToSpaces(file, folder)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": "image upload failed: " + err.Error()})
	}

	return c.Status(201).JSON(fiber.Map{
		"message": "image uploaded",
		"url":     url,
	})
}

func (h *Handler) UploadImages(c *fiber.Ctx) error {
	folder, err := uploadFolder(c)
	if err != nil {
		return c.Status(400).JSON(fiber.Map{"error": err.Error()})
	}

	form, err := c.MultipartForm()
	if err != nil || form == nil || len(form.File["images"]) == 0 {
		return c.Status(400).JSON(fiber.Map{"error": "at least one image file is required"})
	}

	urls := make([]string, 0, len(form.File["images"]))
	for _, file := range form.File["images"] {
		url, uploadErr := uploadImageToSpaces(file, folder)
		if uploadErr != nil {
			return c.Status(500).JSON(fiber.Map{"error": "image upload failed: " + uploadErr.Error()})
		}
		urls = append(urls, url)
	}

	return c.Status(201).JSON(fiber.Map{
		"message": "images uploaded",
		"count":   len(urls),
		"urls":    urls,
	})
}

func uploadFolder(c *fiber.Ctx) (string, error) {
	folder := c.Query("folder", "products")
	if folder != "products" && folder != "variants" {
		return "", fmt.Errorf("folder must be products or variants")
	}
	return folder, nil
}

func uploadImageToSpaces(file *multipart.FileHeader, folder string) (string, error) {
	data, filename, err := storage.ProcessImage(file)
	if err != nil {
		return "", err
	}
	return storage.UploadFile(folder+"/"+filename, data, file.Header.Get("Content-Type"))
}
