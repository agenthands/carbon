package extraction

import (
	"context"
)

type MockLLMClient struct {
	Response string
	Err      error
}

func (m *MockLLMClient) Generate(ctx context.Context, prompt string) (string, error) {
	if m.Err != nil {
		return "", m.Err
	}
	return m.Response, nil
}

type MockEmbedderClient struct {
	Response []float32
	Err      error
}

func (m *MockEmbedderClient) Embed(ctx context.Context, text string) ([]float32, error) {
	if m.Err != nil {
		return nil, m.Err
	}
	return m.Response, nil
}
