package llm

import (
	"context"
	"fmt"
	"strings"
	"github.com/agenthands/carbon/internal/config"
)

func NewClient(ctx context.Context, cfg config.LLMConfig) (LLMClient, EmbedderClient, error) {
	provider := strings.ToLower(cfg.Provider)
	
	switch provider {
	case "openai":
		c := NewOpenAIClient(cfg.APIKey, cfg.Model, cfg.EmbeddingModel, cfg.BaseURL)
		return c, c, nil
	
	case "gemini":
		c, err := NewGeminiClient(ctx, cfg.APIKey, cfg.Model, cfg.EmbeddingModel)
		if err != nil {
			return nil, nil, err
		}
		return c, c, nil
	
	case "claude":
		c := NewClaudeClient(cfg.APIKey, cfg.Model, cfg.BaseURL)
		return c, nil, nil // Return nil for EmbedderClient so application knows it's not supported
	
	case "ollama":
		// Switch to OpenAI-compatible client for Ollama to enable usage tracking
		baseURL := cfg.BaseURL
		if !strings.HasSuffix(baseURL, "/v1") {
			baseURL = fmt.Sprintf("%s/v1", strings.TrimRight(baseURL, "/"))
		}
		
		fmt.Printf("Initializing Ollama via OpenAI-compatible API at %s (enables usage tracking)\n", baseURL)
		
		// Create OpenAI client pointing to Ollama
		// Note: API Key is ignored by Ollama but required by client config (can be dummy)
		apiKey := cfg.APIKey
		if apiKey == "" {
			apiKey = "ollama" // Dummy key
		}
		
		c := NewOpenAIClient(apiKey, cfg.Model, cfg.EmbeddingModel, baseURL)
		return c, c, nil
		
	default:
		return nil, nil, fmt.Errorf("unsupported llm provider: %s", provider)
	}
}
