package llm

import (
	"context"
)

type LLMClient interface {
	Generate(ctx context.Context, prompt string) (string, error)
}

type EmbedderClient interface {
	Embed(ctx context.Context, text string) ([]float32, error)
}
