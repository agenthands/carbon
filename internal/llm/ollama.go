package llm

import (
	"context"
	"fmt"

	"github.com/XiaoConstantine/dspy-go/pkg/core"
	"github.com/XiaoConstantine/dspy-go/pkg/llms"
)

type OllamaClient struct {
	llm *llms.OllamaLLM
}

func NewOllamaClient(modelName string, baseURL string) (*OllamaClient, error) {
	// Initialize Ollama LLM via DSPy
	// Assuming dspy-go's NewOllamaLLM usage based on earlier successful read
	// We'll use the "modern" OpenAI-compatible mode by default as per dspy-go code
    // Model ID needs to be cast to core.ModelID
	
	opts := []llms.OllamaOption{
		llms.WithBaseURL(baseURL),
		llms.WithOpenAIAPI(), // Use OpenAI compatible API
	}

	ollamaLLM, err := llms.NewOllamaLLM(core.ModelID(modelName), opts...)
	if err != nil {
		return nil, fmt.Errorf("failed to create ollama llm: %w", err)
	}

	return &OllamaClient{llm: ollamaLLM}, nil
}

func (c *OllamaClient) Generate(ctx context.Context, prompt string) (string, error) {
	response, err := c.llm.Generate(ctx, prompt)
	if err != nil {
		return "", err
	}
	return response.Content, nil
}

func (c *OllamaClient) Embed(ctx context.Context, text string) ([]float32, error) {
	// Using DSPy's CreateEmbedding
	result, err := c.llm.CreateEmbedding(ctx, text)
	if err != nil {
		return nil, err
	}
	// DSPy returns []float32 directly
	return result.Vector, nil
}
