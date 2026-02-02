package main

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/google/generative-ai-go/genai"
	"google.golang.org/api/option"
)

// GeminiClient wraps the Gemini AI client
type GeminiClient struct {
	client *genai.Client
	model  string
	ctx    context.Context
}

// GeminiResponse represents a response from Gemini
type GeminiResponse struct {
	Text       string
	Success    bool
	Error      string
	TokenCount int
}

// NewGeminiClient creates a new Gemini client
func NewGeminiClient(ctx context.Context) (*GeminiClient, error) {
	apiKey := os.Getenv("GEMINI_API_KEY")
	if apiKey == "" {
		return nil, fmt.Errorf("GEMINI_API_KEY environment variable not set")
	}

	client, err := genai.NewClient(ctx, option.WithAPIKey(apiKey))
	if err != nil {
		return nil, fmt.Errorf("failed to create Gemini client: %v", err)
	}

	modelName := os.Getenv("GEMINI_MODEL")
	if modelName == "" {
		modelName = "gemini-1.5-flash" // Default model
	}

	return &GeminiClient{
		client: client,
		model:  modelName,
		ctx:    ctx,
	}, nil
}

// Close closes the Gemini client
func (g *GeminiClient) Close() error {
	return g.client.Close()
}

// GenerateText generates text from a prompt
func (g *GeminiClient) GenerateText(prompt string) (*GeminiResponse, error) {
	model := g.client.GenerativeModel(g.model)

	// Configure model parameters
	model.SetTemperature(0.2) // Lower temperature for more consistent responses
	model.SetTopP(0.8)
	model.SetTopK(40)
	model.SetMaxOutputTokens(2048)

	resp, err := model.GenerateContent(g.ctx, genai.Text(prompt))
	if err != nil {
		return &GeminiResponse{
			Success: false,
			Error:   fmt.Sprintf("Failed to generate content: %v", err),
		}, err
	}

	if len(resp.Candidates) == 0 {
		return &GeminiResponse{
			Success: false,
			Error:   "No response candidates returned",
		}, fmt.Errorf("no response candidates")
	}

	var text string
	for _, part := range resp.Candidates[0].Content.Parts {
		text += fmt.Sprintf("%v", part)
	}

	return &GeminiResponse{
		Text:       text,
		Success:    true,
		TokenCount: int(resp.UsageMetadata.TotalTokenCount),
	}, nil
}

// AnalyzeReceiptTextWithPrompt analyzes receipt text using a custom prompt from environment
func (g *GeminiClient) AnalyzeReceiptTextWithPrompt(ocrText string) (*GeminiResponse, error) {
	promptTemplate := os.Getenv("GEMINI_PROMPT")
	if promptTemplate == "" {
		// Default prompt if not set
		promptTemplate = `Analyze the following receipt text and extract structured information in JSON format.

Extract the following information:
- date: transaction date (YYYY-MM-DD format)
- merchant_raw: merchant name as it appears
- merchant_clean: cleaned/normalized merchant name
- category: spending category (e.g., groceries, restaurant, gas, shopping, entertainment, etc.)
- amount: total amount
- currency: currency code (e.g., USD, EUR, IDR)
- confidence: your confidence level (0.0 to 1.0)

Return ONLY a valid JSON object with these fields. If you cannot extract a field, use null.
Example: {"date":"2024-01-15","merchant_raw":"WALMART #1234","merchant_clean":"Walmart","category":"groceries","amount":45.67,"currency":"USD","confidence":0.95}`
	}

	prompt := fmt.Sprintf("%s\n\nReceipt Text:\n%s", promptTemplate, ocrText)
	return g.GenerateText(prompt)
}

// AnalyzeReceiptText analyzes receipt text and extracts structured data
func (g *GeminiClient) AnalyzeReceiptText(ocrText string) (*GeminiResponse, error) {
	prompt := fmt.Sprintf(`Analyze the following receipt text and extract structured information in JSON format.

Receipt Text:
%s

Extract the following information:
- date: transaction date (YYYY-MM-DD format)
- merchant_raw: merchant name as it appears
- merchant_clean: cleaned/normalized merchant name
- category: spending category (e.g., groceries, restaurant, gas, shopping, entertainment, etc.)
- amount: total amount
- currency: currency code (e.g., USD, EUR, IDR)
- confidence: your confidence level (0.0 to 1.0)

Return ONLY a valid JSON object with these fields. If you cannot extract a field, use null.
Example: {"date":"2024-01-15","merchant_raw":"WALMART #1234","merchant_clean":"Walmart","category":"groceries","amount":45.67,"currency":"USD","confidence":0.95}`, ocrText)

	return g.GenerateText(prompt)
}

// AnalyzeReceiptWithContext provides more detailed receipt analysis
func (g *GeminiClient) AnalyzeReceiptWithContext(ocrText string, additionalContext string) (*GeminiResponse, error) {
	prompt := fmt.Sprintf(`Analyze this receipt with additional context.

Receipt Text:
%s

Additional Context:
%s

Provide a detailed JSON response with:
- date, merchant, category, amount, currency
- items: array of purchased items (if available)
- payment_method: detected payment method
- confidence: overall confidence score

Return ONLY valid JSON.`, ocrText, additionalContext)

	return g.GenerateText(prompt)
}

// TestConnection tests the Gemini API connection
func (g *GeminiClient) TestConnection() error {
	resp, err := g.GenerateText("Hello, respond with 'OK' if you can understand this.")
	if err != nil {
		return err
	}

	if !resp.Success {
		return fmt.Errorf("test failed: %s", resp.Error)
	}

	log.Printf("Gemini connection test successful. Response: %s", resp.Text)
	return nil
}

// ListModels lists all available Gemini models
func (g *GeminiClient) ListModels() ([]string, error) {
	iter := g.client.ListModels(g.ctx)
	var models []string

	for {
		model, err := iter.Next()
		if err != nil {
			break
		}
		models = append(models, model.Name)
	}

	if len(models) == 0 {
		return nil, fmt.Errorf("no models found")
	}

	return models, nil
}
