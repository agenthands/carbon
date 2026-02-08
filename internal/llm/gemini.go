package llm

import (
	"context"
	"fmt"


	"github.com/google/generative-ai-go/genai"
	"google.golang.org/api/option"
)

type GeminiClient struct {
	client *genai.Client
	model  string
}

func NewGeminiClient(ctx context.Context, apiKey string, model string) (*GeminiClient, error) {
	client, err := genai.NewClient(ctx, option.WithAPIKey(apiKey))
	if err != nil {
		return nil, err
	}
	return &GeminiClient{
		client: client,
		model:  model,
	}, nil
}

func (c *GeminiClient) Generate(ctx context.Context, prompt string) (string, error) {
	model := c.client.GenerativeModel(c.model)
	resp, err := model.GenerateContent(ctx, genai.Text(prompt))
	if err != nil {
		return "", err
	}
	
	if len(resp.Candidates) > 0 {
		part := resp.Candidates[0].Content.Parts[0]
		if txt, ok := part.(genai.Text); ok {
			return string(txt), nil
		}
	}
	
	return "", fmt.Errorf("no response candidates or content")
}

func (c *GeminiClient) Embed(ctx context.Context, text string) ([]float32, error) {
	// For Gemini, embedding model name is usually separate, e.g. "embedding-001" or "text-embedding-004"
	// Hardcoding reasonable default for now or inferring?
	embedModel := c.client.EmbeddingModel("text-embedding-004")
	res, err := embedModel.EmbedContent(ctx, genai.Text(text))
	if err != nil {
		return nil, err
	}
	if res.Embedding != nil {
		return res.Embedding.Values, nil
	}
	return nil, fmt.Errorf("no embedding values")
}
