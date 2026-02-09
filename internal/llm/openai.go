package llm

import (
	"context"
	"fmt"

	"github.com/sashabaranov/go-openai"
)

type OpenAIClient struct {
	client         *openai.Client
	model          string
	embeddingModel string
}

func NewOpenAIClient(apiKey string, model string, embeddingModel string, baseURL string) *OpenAIClient {
	config := openai.DefaultConfig(apiKey)
	if baseURL != "" {
		config.BaseURL = baseURL
	}
	client := openai.NewClientWithConfig(config)
	return &OpenAIClient{
		client:         client,
		model:          model,
		embeddingModel: embeddingModel,
	}
}

func (c *OpenAIClient) Generate(ctx context.Context, prompt string) (string, error) {
	req := openai.ChatCompletionRequest{
		Model: c.model,
		Messages: []openai.ChatCompletionMessage{
			{
				Role:    openai.ChatMessageRoleUser,
				Content: prompt,
			},
		},
	}
	resp, err := c.client.CreateChatCompletion(ctx, req)
	if err != nil {
		return "", err
	}

	// Log Token Usage
	usage := resp.Usage
	if usage.TotalTokens > 0 {
		// Using standard log for simple monitoring as requested
		// In production, this should be structured logging (slog/zap)
		fmt.Printf("LLM Usage: model=%s prompt=%d completion=%d total=%d\n",
			c.model, usage.PromptTokens, usage.CompletionTokens, usage.TotalTokens)
	}

	if len(resp.Choices) > 0 {
		return resp.Choices[0].Message.Content, nil
	}
	return "", fmt.Errorf("no response choices")
}

func (c *OpenAIClient) Embed(ctx context.Context, text string) ([]float32, error) {
	model := c.embeddingModel
	if model == "" {
		model = string(openai.SmallEmbedding3)
	}
	req := openai.EmbeddingRequest{
		Input: []string{text},
		Model: openai.EmbeddingModel(model),
	}
	resp, err := c.client.CreateEmbeddings(ctx, req)
	if err != nil {
		return nil, err
	}

	// Log Token Usage
	usage := resp.Usage
	if usage.TotalTokens > 0 {
		fmt.Printf("LLM Usage (Embedding): model=%s prompt=%d total=%d\n",
			model, usage.PromptTokens, usage.TotalTokens)
	}

	if len(resp.Data) > 0 {
		return resp.Data[0].Embedding, nil
	}
	return nil, fmt.Errorf("no embedding data")
}
