package main

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	gosseract "github.com/otiai10/gosseract/v2"
)

func main() {
	app := fiber.New()

	// Create uploads directory if it doesn't exist
	if err := os.MkdirAll("./uploads", os.ModePerm); err != nil {
		log.Fatal(err)
	}

	app.Get("/", func(c *fiber.Ctx) error {
		return c.JSON(fiber.Map{
			"message": "Receipt Processor API",
			"version": "1.0.0",
			"endpoints": fiber.Map{
				"POST /ocr":             "Upload an image to extract text using OCR",
				"POST /receipts/ingest": "Upload and store a receipt file",
			},
		})
	})

	// OCR endpoint
	app.Post("/ocr", func(c *fiber.Ctx) error {
		// Get uploaded file
		file, err := c.FormFile("image")
		if err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error": "No image file provided",
			})
		}

		// Save file temporarily
		tempPath := filepath.Join("./uploads", file.Filename)
		if err := c.SaveFile(file, tempPath); err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"error": "Failed to save file",
			})
		}
		defer os.Remove(tempPath)

		// Initialize OCR client
		client := gosseract.NewClient()
		defer client.Close()

		// Set image path
		if err := client.SetImage(tempPath); err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"error": fmt.Sprintf("Failed to set image: %v", err),
			})
		}

		// Extract text
		text, err := client.Text()
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"error": fmt.Sprintf("OCR failed: %v", err),
			})
		}

		return c.JSON(fiber.Map{
			"success":  true,
			"filename": file.Filename,
			"text":     text,
		})
	})
	// Receipt ingest endpoint
	app.Post("/receipts/ingest", func(c *fiber.Ctx) error {
		// Get uploaded file
		file, err := c.FormFile("file")
		if err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error": "No file provided",
			})
		}

		// Validate file type (optional - allow images and PDFs)
		allowedTypes := map[string]bool{
			"image/jpeg":      true,
			"image/jpg":       true,
			"image/png":       true,
			"image/gif":       true,
			"image/webp":      true,
			"application/pdf": true,
		}

		contentType := file.Header.Get("Content-Type")
		if contentType != "" && !allowedTypes[contentType] {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error": "Invalid file type. Allowed: images (jpg, png, gif, webp) and PDF",
			})
		}

		// Generate unique filename
		receiptID := uuid.New().String()
		ext := filepath.Ext(file.Filename)
		uniqueFilename := fmt.Sprintf("%s_%s%s", receiptID, time.Now().Format("20060102_150405"), ext)
		savePath := filepath.Join("./uploads", uniqueFilename)

		// Save the file
		if err := c.SaveFile(file, savePath); err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"error": "Failed to save file",
			})
		}

		// Get file info
		fileInfo, err := os.Stat(savePath)
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"error": "Failed to get file info",
			})
		}

		return c.Status(fiber.StatusCreated).JSON(fiber.Map{
			"success":       true,
			"receipt_id":    receiptID,
			"original_name": file.Filename,
			"stored_name":   uniqueFilename,
			"file_size":     fileInfo.Size(),
			"content_type":  contentType,
			"upload_time":   time.Now().Format(time.RFC3339),
			"file_path":     savePath,
		})
	})

	log.Println("Server starting on :3000")
	log.Fatal(app.Listen(":3000"))
}
