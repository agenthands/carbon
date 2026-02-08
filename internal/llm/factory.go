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
		c := NewOpenAIClient(cfg.APIKey, cfg.Model, cfg.BaseURL)
		return c, c, nil
	
	case "gemini":
		c, err := NewGeminiClient(ctx, cfg.APIKey, cfg.Model)
		if err != nil {
			return nil, nil, err
		}
		return c, c, nil
	
	case "claude":
		c := NewClaudeClient(cfg.APIKey, cfg.Model, cfg.BaseURL)
		return c, nil, nil // Return nil for EmbedderClient so application knows it's not supported
	
	case "ollama":
		c, err := NewOllamaClient(cfg.Model, cfg.BaseURL)
		if err != nil {
			return nil, nil, err
		}
		return c, c, nil
		
	default:
		return nil, nil, fmt.Errorf("unsupported llm provider: %s", provider)
	}
}
