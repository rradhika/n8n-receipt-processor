# TODO List

## High Priority

### 1. PDF File Amount Extraction Issue
- [ ] **Problem**: PDF files are returning improper amounts during extraction
- [ ] Investigate why text extraction from PDFs is not capturing amounts correctly
- [ ] Test with multiple PDF types (text-based vs image-based)
- [ ] Compare Gemini parsing results between image OCR and PDF text extraction
- [ ] Consider adding preprocessing/cleaning for PDF extracted text
- [ ] Add validation for amount fields before storing in database

### 2. Add Logging Before Gemini API Call
- [ ] **Problem**: Need to log the JSON/text before sending to Gemini for debugging
- [ ] Create a log file or directory for Gemini requests (`logs/gemini/`)
- [ ] Log the OCR extracted text before Gemini analysis
- [ ] Include timestamp, receipt ID, and file name in logs
- [ ] Log the raw Gemini response as well
- [ ] Add environment variable to enable/disable detailed logging
- [ ] Format logs as JSON for easier parsing

### 3. Code Refactoring - Separate Functions
- [ ] **Problem**: All code is in main.go, need better organization
- [ ] Create separate files for different concerns:
  - [ ] `database.go` - All database connection and query functions
  - [ ] `pdf.go` - PDF detection and text extraction functions
  - [ ] `ocr.go` - OCR-related functions
  - [ ] `gemini.go` - Already exists, ensure it's complete
  - [ ] `handlers.go` or `routes.go` - HTTP handlers
  - [ ] `models.go` - Data structures and models
- [ ] Extract database functions:
  - [ ] `initDB()` → `database.go`
  - [ ] `createTables()` → `database.go`
  - [ ] Receipt insert/update queries → separate functions
- [ ] Extract PDF/OCR functions:
  - [ ] `isPDFTextBased()` → `pdf.go`
  - [ ] `extractTextFromPDF()` → `pdf.go`
  - [ ] `convertPDFToImagesAndOCR()` → `pdf.go`
- [ ] Extract common utilities:
  - [ ] `utils.go` - Helper functions (file handling, validation, etc.)
  - [ ] `logger.go` - Centralized logging utilities
- [ ] Update imports and package references

## Medium Priority

### 4. Improve Error Handling
- [ ] Add better error messages for PDF processing failures
- [ ] Return more detailed error responses to API consumers
- [ ] Add retry logic for Gemini API calls
- [ ] Handle timeout scenarios for long-running OCR operations

### 5. Add Configuration Management
- [ ] Create `config.go` for centralized configuration
- [ ] Move all environment variables to a config struct
- [ ] Add configuration validation on startup
- [ ] Document all required environment variables in README

### 6. Testing
- [ ] Add unit tests for PDF detection
- [ ] Add unit tests for text extraction functions
- [ ] Add integration tests for the ingest endpoint
- [ ] Test with various PDF types (scanned, digital, mixed)
- [ ] Create test fixtures directory with sample receipts

## Low Priority

### 7. Performance Optimization
- [ ] Add caching for Gemini API responses (optional)
- [ ] Optimize image processing for large PDFs
- [ ] Add file size limits and validation
- [ ] Consider async processing for large batches

### 8. Documentation
- [ ] Update README with setup instructions
- [ ] Document API endpoints with examples
- [ ] Add architecture diagram
- [ ] Document the PDF detection logic flow

### 9. Monitoring & Observability
- [ ] Add metrics collection (processing time, success rate)
- [ ] Add health check endpoint with dependency status
- [ ] Log processing statistics (files processed, success/failure rates)

## Notes

- The PDF amount extraction issue might be related to:
  - Text formatting differences between PDFs and images
  - Gemini prompt not handling PDF text format well
  - Need to compare raw OCR output vs Gemini parsed output

- For logging Gemini calls, consider:
  - Sensitive data - don't log personal information
  - Rotation policy for log files
  - Log level configuration (DEBUG, INFO, ERROR)

- For code separation:
  - Keep main.go minimal (just initialization and routing)
  - Use Go packages properly
  - Consider creating a `pkg/` or `internal/` directory structure
