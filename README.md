# n8n-receipt-processor

A Go-based receipt processing API with OCR (Optical Character Recognition) capabilities using Tesseract.

## Features

- 🔍 OCR text extraction from images (receipts, documents, etc.)
- 🚀 Fast and lightweight API built with Fiber
- 🐳 Fully Dockerized for easy deployment
- 📦 Ready for n8n workflow integration

## Prerequisites

- Go 1.23+ (for local development)
- Docker and Docker Compose (for containerized deployment)

## Installation & Running

### Using Docker (Recommended)

1. **Build and run with Docker Compose:**
   ```bash
   docker-compose up -d
   ```

2. **Or build and run with Docker directly:**
   ```bash
   docker build -t receipt-processor .
   docker run -p 3000:3000 receipt-processor
   ```

### Local Development

1. **Install Tesseract OCR:**
   - Windows: Download from [GitHub releases](https://github.com/UB-Mannheim/tesseract/wiki)
   - macOS: `brew install tesseract`
   - Linux: `sudo apt-get install tesseract-ocr`

2. **Install dependencies:**
   ```bash
   go mod download
   ```

3. **Run the application:**
   ```bash
   go run main.go
   ```

The server will start on `http://localhost:3000`

## API Endpoints

### GET /
Get API information and available endpoints.

**Response:**
```json
{
  "message": "Receipt Processor API",
  "version": "1.0.0",
  "endpoints": {
    "POST /ocr": "Upload an image to extract text using OCR"
  }
}
```

### POST /ocr
Upload an image to extract text using OCR.

**Request:**
- Method: `POST`
- Content-Type: `multipart/form-data`
- Body: Form data with `image` field containing the image file

**Example using cURL:**
```bash
curl -X POST http://localhost:3000/ocr \
  -F "image=@receipt.jpg"
```

**Example using JavaScript/fetch:**
```javascript
const formData = new FormData();
formData.append('image', fileInput.files[0]);

fetch('http://localhost:3000/ocr', {
  method: 'POST',
  body: formData
})
  .then(response => response.json())
  .then(data => console.log(data));
```

**Response:**
```json
{
  "success": true,
  "filename": "receipt.jpg",
  "text": "Extracted text from the image..."
}
```

## Integration with n8n

1. Use the **HTTP Request** node in n8n
2. Set method to `POST`
3. Set URL to `http://localhost:3000/ocr` (or your deployed URL)
4. In "Body Content Type", select "Multipart-Form Data"
5. Add parameter: `image` with your file/binary data

## Docker Commands

```bash
# Build the image
docker build -t receipt-processor .

# Run the container
docker run -p 3000:3000 receipt-processor

# Run with Docker Compose
docker-compose up -d

# View logs
docker-compose logs -f

# Stop the container
docker-compose down
```

## Project Structure

```
.
├── main.go              # Main application file
├── go.mod               # Go module dependencies
├── go.sum               # Go module checksums
├── Dockerfile           # Docker build instructions
├── docker-compose.yml   # Docker Compose configuration
├── .dockerignore        # Files to ignore in Docker build
├── uploads/             # Temporary upload directory (created automatically)
└── README.md            # This file
```

## Technologies Used

- [Go](https://golang.org/) - Programming language
- [Fiber](https://gofiber.io/) - Web framework
- [Gosseract](https://github.com/otiai10/gosseract) - Go wrapper for Tesseract OCR
- [Tesseract OCR](https://github.com/tesseract-ocr/tesseract) - OCR engine
- [Docker](https://www.docker.com/) - Containerization

## License

MIT