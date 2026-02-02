package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	_ "github.com/go-sql-driver/mysql"
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	gosseract "github.com/otiai10/gosseract/v2"
)

// Receipt model
type Receipt struct {
	ID          int64
	FileName    string
	DriveFileID sql.NullString
	Status      string
	UploadedAt  time.Time
}

// Transaction model
type Transaction struct {
	ID            int64
	ReceiptID     int64
	Date          sql.NullTime
	MerchantRaw   sql.NullString
	MerchantClean sql.NullString
	Category      sql.NullString
	Amount        sql.NullFloat64
	Currency      sql.NullString
	Confidence    sql.NullFloat64
	CreatedAt     time.Time
}

// GeminiParsedData represents parsed receipt data from Gemini
type GeminiParsedData struct {
	Date          string  `json:"date"`
	MerchantRaw   string  `json:"merchant_raw"`
	MerchantClean string  `json:"merchant_clean"`
	Category      string  `json:"category"`
	Amount        float64 `json:"amount"`
	Currency      string  `json:"currency"`
	Confidence    float64 `json:"confidence"`
}

var db *sql.DB

// Initialize database connection
func initDB() error {
	// Get database connection string from environment variable
	// Format: username:password@tcp(host:port)/database
	dsn := os.Getenv("MYSQL_DSN")
	if dsn == "" {
		dsn = "root:@tcp(127.0.0.1:3306)/receipt_processor?parseTime=true"
		log.Println("MYSQL_DSN not set, using default:", dsn)
	}

	var err error
	db, err = sql.Open("mysql", dsn)
	if err != nil {
		return fmt.Errorf("failed to open database: %v", err)
	}

	// Test connection
	if err := db.Ping(); err != nil {
		return fmt.Errorf("failed to ping database: %v", err)
	}

	// Set connection pool settings
	db.SetMaxOpenConns(25)
	db.SetMaxIdleConns(5)
	db.SetConnMaxLifetime(5 * time.Minute)

	log.Println("Database connection established")
	return nil
}

// Create database tables if they don't exist
func createTables() error {
	receiptsTable := `
	CREATE TABLE IF NOT EXISTS receipts (
		id BIGINT AUTO_INCREMENT PRIMARY KEY,
		file_name VARCHAR(255) NOT NULL,
		drive_file_id VARCHAR(255),
		status ENUM('processed', 'needs_review', 'error') NOT NULL DEFAULT 'needs_review',
		uploaded_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
		INDEX idx_status (status),
		INDEX idx_uploaded_at (uploaded_at)
	) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
	`

	transactionsTable := `
	CREATE TABLE IF NOT EXISTS transactions (
		id BIGINT AUTO_INCREMENT PRIMARY KEY,
		receipt_id BIGINT NOT NULL,
		date DATE,
		merchant_raw VARCHAR(255),
		merchant_clean VARCHAR(255),
		category VARCHAR(100),
		amount DECIMAL(10, 2),
		currency VARCHAR(3),
		confidence DECIMAL(5, 4),
		created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
		FOREIGN KEY (receipt_id) REFERENCES receipts(id) ON DELETE CASCADE,
		INDEX idx_receipt_id (receipt_id),
		INDEX idx_date (date),
		INDEX idx_merchant_clean (merchant_clean),
		INDEX idx_category (category)
	) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
	`

	if _, err := db.Exec(receiptsTable); err != nil {
		return fmt.Errorf("failed to create receipts table: %v", err)
	}

	if _, err := db.Exec(transactionsTable); err != nil {
		return fmt.Errorf("failed to create transactions table: %v", err)
	}

	log.Println("Database tables created/verified")
	return nil
}

func main() {
	// Initialize database
	if err := initDB(); err != nil {
		log.Fatal("Database initialization failed:", err)
	}
	defer db.Close()

	// Create tables
	if err := createTables(); err != nil {
		log.Fatal("Failed to create tables:", err)
	}

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
				"POST /ocr":                   "Upload an image to extract text using OCR",
				"POST /receipts/ingest":       "Upload and store a receipt file",
				"POST /gemini/test":           "Test Gemini AI connection",
				"GET  /gemini/models":         "List available Gemini AI models",
				"POST /gemini/analyze":        "Analyze text with Gemini AI",
				"POST /receipts/analyze/{id}": "Analyze a receipt using Gemini AI",
			},
		})
	})

	// Gemini test endpoint
	app.Post("/gemini/test", func(c *fiber.Ctx) error {
		geminiClient, err := NewGeminiClient(c.Context())
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"error": fmt.Sprintf("Failed to create Gemini client: %v", err),
			})
		}
		defer geminiClient.Close()

		if err := geminiClient.TestConnection(); err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"error": fmt.Sprintf("Gemini test failed: %v", err),
			})
		}

		return c.JSON(fiber.Map{
			"success": true,
			"message": "Gemini AI is connected and working",
		})
	})

	// Gemini analyze endpoint
	app.Post("/gemini/analyze", func(c *fiber.Ctx) error {
		type AnalyzeRequest struct {
			Text string `json:"text"`
		}

		var req AnalyzeRequest
		if err := c.BodyParser(&req); err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error": "Invalid request body",
			})
		}

		if req.Text == "" {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error": "Text field is required",
			})
		}

		geminiClient, err := NewGeminiClient(c.Context())
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"error": fmt.Sprintf("Failed to create Gemini client: %v", err),
			})
		}
		defer geminiClient.Close()

		response, err := geminiClient.AnalyzeReceiptText(req.Text)
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"error": fmt.Sprintf("Analysis failed: %v", err),
			})
		}

		return c.JSON(fiber.Map{
			"success":     response.Success,
			"analysis":    response.Text,
			"token_count": response.TokenCount,
			"error":       response.Error,
		})
	})

	// List available Gemini models endpoint
	app.Get("/gemini/models", func(c *fiber.Ctx) error {
		geminiClient, err := NewGeminiClient(c.Context())
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"error": fmt.Sprintf("Failed to create Gemini client: %v", err),
			})
		}
		defer geminiClient.Close()

		models, err := geminiClient.ListModels()
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"error": fmt.Sprintf("Failed to list models: %v", err),
			})
		}

		return c.JSON(fiber.Map{
			"success": true,
			"models":  models,
			"count":   len(models),
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

		// Insert receipt into database
		result, err := db.Exec(
			"INSERT INTO receipts (file_name, status, uploaded_at) VALUES (?, ?, ?)",
			uniqueFilename,
			"needs_review",
			time.Now(),
		)
		if err != nil {
			log.Printf("Failed to insert receipt into database: %v", err)
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"error": "Failed to save receipt to database",
			})
		}

		receiptDBID, err := result.LastInsertId()
		if err != nil {
			log.Printf("Failed to get receipt ID: %v", err)
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"error": "Failed to get receipt ID",
			})
		}

		// Perform OCR on the uploaded file (only for images, not PDFs)
		ocrText := ""
		ocrStatus := "success"
		ocrError := ""

		// Check if file is an image (not PDF)
		isImage := contentType != "application/pdf" && contentType != ""

		if !isImage {
			ocrStatus = "skipped"
			ocrError = "PDF files require separate processing"
		} else {
			// Read file bytes for OCR
			fileBytes, err := os.ReadFile(savePath)
			if err != nil {
				log.Printf("OCR: Failed to read file: %v", err)
				ocrStatus = "failed"
				ocrError = fmt.Sprintf("Failed to read file: %v", err)
			} else {
				client := gosseract.NewClient()
				defer client.Close()

				if err := client.SetImageFromBytes(fileBytes); err != nil {
					log.Printf("OCR: Failed to set image: %v", err)
					ocrStatus = "failed"
					ocrError = fmt.Sprintf("Failed to set image: %v", err)
				} else {
					text, err := client.Text()
					if err != nil {
						log.Printf("OCR: Failed to extract text: %v", err)
						ocrStatus = "failed"
						ocrError = fmt.Sprintf("Failed to extract text: %v", err)
					} else {
						ocrText = text
					}
				}
			}
		}

		// Parse OCR text with Gemini if OCR was successful
		var geminiAnalysis string
		var geminiStatus string
		var geminiError string
		var parsedData *GeminiParsedData

		if ocrStatus == "success" && ocrText != "" {
			geminiClient, err := NewGeminiClient(c.Context())
			if err != nil {
				log.Printf("Gemini: Failed to create client: %v", err)
				geminiStatus = "failed"
				geminiError = fmt.Sprintf("Failed to create client: %v", err)
			} else {
				defer geminiClient.Close()

				response, err := geminiClient.AnalyzeReceiptTextWithPrompt(ocrText)
				if err != nil {
					log.Printf("Gemini: Failed to analyze: %v", err)
					geminiStatus = "failed"
					geminiError = fmt.Sprintf("Failed to analyze: %v", err)
				} else if !response.Success {
					log.Printf("Gemini: Analysis unsuccessful: %s", response.Error)
					geminiStatus = "failed"
					geminiError = response.Error
				} else {
					geminiAnalysis = response.Text
					geminiStatus = "success"

					// Try to parse JSON from Gemini response
					// Clean the response - sometimes Gemini wraps JSON in markdown code blocks
					cleanedText := strings.TrimSpace(response.Text)
					cleanedText = strings.TrimPrefix(cleanedText, "```json")
					cleanedText = strings.TrimPrefix(cleanedText, "```")
					cleanedText = strings.TrimSuffix(cleanedText, "```")
					cleanedText = strings.TrimSpace(cleanedText)

					var data GeminiParsedData
					if err := json.Unmarshal([]byte(cleanedText), &data); err != nil {
						log.Printf("Gemini: Failed to parse JSON: %v", err)
						geminiError = fmt.Sprintf("Failed to parse JSON: %v", err)
					} else {
						parsedData = &data

						// Insert into transactions table
						var transactionDate sql.NullTime
						if data.Date != "" {
							if t, err := time.Parse("2006-01-02", data.Date); err == nil {
								transactionDate = sql.NullTime{Time: t, Valid: true}
							}
						}

						_, err := db.Exec(
							`INSERT INTO transactions (receipt_id, date, merchant_raw, merchant_clean, category, amount, currency, confidence, created_at) 
							VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`,
							receiptDBID,
							transactionDate,
							sql.NullString{String: data.MerchantRaw, Valid: data.MerchantRaw != ""},
							sql.NullString{String: data.MerchantClean, Valid: data.MerchantClean != ""},
							sql.NullString{String: data.Category, Valid: data.Category != ""},
							sql.NullFloat64{Float64: data.Amount, Valid: data.Amount > 0},
							sql.NullString{String: data.Currency, Valid: data.Currency != ""},
							sql.NullFloat64{Float64: data.Confidence, Valid: data.Confidence > 0},
							time.Now(),
						)
						if err != nil {
							log.Printf("Failed to insert transaction: %v", err)
						} else {
							// Update receipt status to processed
							db.Exec("UPDATE receipts SET status = ? WHERE id = ?", "processed", receiptDBID)
						}
					}
				}
			}
		} else {
			geminiStatus = "skipped"
			geminiError = "No OCR text available"
		}

		return c.Status(fiber.StatusCreated).JSON(fiber.Map{
			"success":       true,
			"receipt_id":    receiptDBID,
			"uuid":          receiptID,
			"original_name": file.Filename,
			"stored_name":   uniqueFilename,
			"file_size":     fileInfo.Size(),
			"content_type":  contentType,
			"upload_time":   time.Now().Format(time.RFC3339),
			"file_path":     savePath,
			"status":        "needs_review",
			"ocr": fiber.Map{
				"status": ocrStatus,
				"text":   ocrText,
				"error":  ocrError,
			},
			"gemini": fiber.Map{
				"status":   geminiStatus,
				"analysis": geminiAnalysis,
				"error":    geminiError,
				"parsed":   parsedData,
			},
		})
	})

	log.Println("Server starting on :3000")
	log.Fatal(app.Listen(":3000"))
}
