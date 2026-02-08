package summary

import (
	"context"
	"testing"
	
	"github.com/agenthands/carbon/internal/config"
	"github.com/agenthands/carbon/internal/core/model"
	"github.com/stretchr/testify/assert"
)

func TestSummarizeNodes(t *testing.T) {
	// Scenario: Update summary for "Alice" based on new mentions
	mockJSON := `{
		"summary": "Alice is a software engineer living in Paris."
	}`
	
	mockLLM := &MockLLMClient{
		Response: mockJSON,
	}
	
	cfg := config.SummaryPrompts{
		Nodes: "test prompt %s %s",
	}
	summarizer := NewSummarizer(mockLLM, cfg)
	ctx := context.Background()
	
	node := model.EntityNode{
		UUID:    "uuid-1",
		Name:    "Alice",
		Summary: "Alice is a software engineer.",
	}
	
	newMentions := []string{
		"Alice moved to Paris recently.",
	}
	
	updatedSummary, err := summarizer.SummarizeNode(ctx, node, newMentions)
	
	assert.NoError(t, err)
	assert.Equal(t, "Alice is a software engineer living in Paris.", updatedSummary)
}
