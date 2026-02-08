package summary

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
