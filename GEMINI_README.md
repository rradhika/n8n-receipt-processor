# Gemini AI Integration

This project includes Google Gemini AI integration for analyzing receipt text and extracting structured data.

## Setup

1. Get your Gemini API key from [Google AI Studio](https://makersuite.google.com/app/apikey)

2. Create a `.env` file in the project root:
```bash
cp .env.example .env
```

3. Add your Gemini API key to the `.env` file:
```env
GEMINI_API_KEY=your_actual_api_key_here
```

4. Start the services:
```bash
docker-compose up -d
```

## API Endpoints

### Test Gemini Connection
```bash
POST /gemini/test
```

Tests if Gemini AI is properly configured and working.

**Response:**
```json
{
  "success": true,
  "message": "Gemini AI is connected and working"
}
```

### Analyze Receipt Text
```bash
POST /gemini/analyze
Content-Type: application/json

{
  "text": "Your receipt OCR text here..."
}
```

Analyzes receipt text and extracts structured information.

**Response:**
```json
{
  "success": true,
  "analysis": "{\"date\":\"2024-01-15\",\"merchant_raw\":\"WALMART #1234\",\"merchant_clean\":\"Walmart\",\"category\":\"groceries\",\"amount\":45.67,\"currency\":\"USD\",\"confidence\":0.95}",
  "token_count": 150,
  "error": ""
}
```

## Gemini Client Library

### Creating a Client

```go
import "context"

ctx := context.Background()
client, err := NewGeminiClient(ctx)
if err != nil {
    log.Fatal(err)
}
defer client.Close()
```

### Methods

#### `GenerateText(prompt string) (*GeminiResponse, error)`
Generate text from any prompt.

```go
response, err := client.GenerateText("What is Go programming language?")
if err != nil {
    log.Fatal(err)
}
fmt.Println(response.Text)
```

#### `AnalyzeReceiptText(ocrText string) (*GeminiResponse, error)`
Analyze receipt text and extract structured data (date, merchant, amount, etc.).

```go
ocrText := "WALMART\n123 Main St\n01/15/2024\nGroceries $45.67"
response, err := client.AnalyzeReceiptText(ocrText)
if err != nil {
    log.Fatal(err)
}
fmt.Println(response.Text) // Returns JSON with extracted data
```

#### `AnalyzeReceiptWithContext(ocrText, additionalContext string) (*GeminiResponse, error)`
Analyze receipt with additional context for more detailed extraction.

```go
response, err := client.AnalyzeReceiptWithContext(ocrText, "This is from a grocery store")
```

#### `TestConnection() error`
Test if the Gemini API connection is working.

```go
if err := client.TestConnection(); err != nil {
    log.Fatal("Gemini connection failed:", err)
}
```

## Response Structure

```go
type GeminiResponse struct {
    Text       string  // Generated text response
    Success    bool    // Whether the request succeeded
    Error      string  // Error message if failed
    TokenCount int     // Number of tokens used
}
```

## Configuration

Environment variables:
- `GEMINI_API_KEY`: Your Google Gemini API key (required)
- `GEMINI_MODEL`: Model to use (default: `gemini-2.0-flash-exp`)

Available models:
- `gemini-2.0-flash-exp` (recommended - fast and cost-effective)
- `gemini-1.5-pro`
- `gemini-1.5-flash`

## Example: Full Receipt Processing Flow

```go
// 1. Upload receipt (OCR happens automatically)
// POST /receipts/ingest with file

// 2. Get OCR text from response
ocrText := response["ocr"]["text"]

// 3. Analyze with Gemini
client, _ := NewGeminiClient(ctx)
analysis, _ := client.AnalyzeReceiptText(ocrText)

// 4. Parse the JSON response
// analysis.Text contains structured data like:
// {"date":"2024-01-15","merchant":"Walmart","amount":45.67,...}
```

## Error Handling

The Gemini client handles common errors:
- Missing API key
- Network errors
- API rate limits
- Invalid responses

Always check `GeminiResponse.Success` and `GeminiResponse.Error` fields.

## Cost Considerations

Gemini 2.0 Flash pricing (as of 2024):
- Input: $0.075 per 1M tokens
- Output: $0.30 per 1M tokens

A typical receipt analysis uses ~200-500 tokens, costing less than $0.001 per receipt.
