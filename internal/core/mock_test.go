package core

import (
	"context"
	
	"github.com/neo4j/neo4j-go-driver/v5/neo4j"
)

type MockDriver struct {
	QueryExecuted string
	QueryParams   map[string]interface{}
	MockResult    neo4j.EagerResult
	Err           error
}

func (m *MockDriver) ExecuteQuery(ctx context.Context, query string, params map[string]interface{}) (neo4j.EagerResult, error) {
	m.QueryExecuted = query
	m.QueryParams = params
	if m.Err != nil {
		return neo4j.EagerResult{}, m.Err
	}
	return m.MockResult, nil
}

func (m *MockDriver) BuildIndices(ctx context.Context) error {
	return nil
}

func (m *MockDriver) Close(ctx context.Context) error {
	return nil
}

type MockEmbedder struct {
	Vector []float32
	Err    error
}

func (m *MockEmbedder) Embed(ctx context.Context, text string) ([]float32, error) {
	if m.Err != nil {
		return nil, m.Err
	}
	return m.Vector, nil
}

type MockLLM struct {
	Response     string
	ResponseQueue []string
}
func (m *MockLLM) Generate(ctx context.Context, prompt string) (string, error) {
	if len(m.ResponseQueue) > 0 {
		resp := m.ResponseQueue[0]
		m.ResponseQueue = m.ResponseQueue[1:]
		return resp, nil
	}
	return m.Response, nil
}
