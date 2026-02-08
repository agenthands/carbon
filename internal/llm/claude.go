package llm

import (
	"context"
	"fmt"

	"github.com/liushuangls/go-anthropic/v2"
)

type ClaudeClient struct {
	client *anthropic.Client
	model  string
}

func NewClaudeClient(apiKey string, model string, baseURL string) *ClaudeClient {
	var opts []anthropic.ClientOption
	if baseURL != "" {
		opts = append(opts, anthropic.WithBaseURL(baseURL))
	}
	
	// apiKey is first argument based on error message usually? Or NewClient takes APIKey as option?
	// Error: `want (string, ...ClientOption)` implies NewClient(apiKey, opts...)
	client := anthropic.NewClient(apiKey, opts...)
	
	return &ClaudeClient{
		client: client,
		model:  model,
	}
}

func (c *ClaudeClient) Generate(ctx context.Context, prompt string) (string, error) {
	resp, err := c.client.CreateMessages(ctx, anthropic.MessagesRequest{
		Model: anthropic.Model(c.model),
		Messages: []anthropic.Message{
			{
				Role: anthropic.RoleUser,
				Content: []anthropic.MessageContent{
					anthropic.NewTextMessageContent(prompt),
				},
			},
		},
		MaxTokens: 1000,
	})
	if err != nil {
		return "", err
	}
	
	if len(resp.Content) > 0 && resp.Content[0].Text != nil {
		return *resp.Content[0].Text, nil
	}
	return "", fmt.Errorf("no response content")
}

func (c *ClaudeClient) Embed(ctx context.Context, text string) ([]float32, error) {
	// Anthropic API does not support embeddings yet officially via same interface.
	// We return nil, nil so the calling code can gracefully handle missing embedding capability if it checks for it,
	// BUT the interface requires []float32, error.
	// If returned error, graphiti logic skips embedding (which is desired).
	return nil, fmt.Errorf("embeddings not supported by Claude client")
}
