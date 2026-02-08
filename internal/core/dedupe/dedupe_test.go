package dedupe

import (
	"context"
	"testing"
	
	"github.com/agenthands/carbon/internal/core/model"
	"github.com/stretchr/testify/assert"
)

func TestDedupeNodes(t *testing.T) {
	// Scenario: LLM decides "Alice" (new) is duplicate of "Alice Smith" (existing)
	mockJSON := `{
		"duplicates": [
			{
				"original_uuid": "existing-uuid-1",
				"duplicate_uuid": "new-uuid-1",
				"confidence": 0.95
			}
		]
	}`
	
	mockLLM := &MockLLMClient{
		Response: mockJSON,
	}
	
	deduplicator := NewDeduplicator(mockLLM)
	ctx := context.Background()
	
	newNodes := []model.EntityNode{
		{UUID: "new-uuid-1", Name: "Alice"},
		{UUID: "new-uuid-2", Name: "Bob"},
	}
	
	existingNodes := []model.EntityNode{
		{UUID: "existing-uuid-1", Name: "Alice Smith"},
		{UUID: "existing-uuid-3", Name: "Charlie"},
	}
	
	// This function asks: "Which of newNodes are duplicates of existingNodes?"
	results, err := deduplicator.ResolveDuplicates(ctx, newNodes, existingNodes)
	
	assert.NoError(t, err)
	assert.Len(t, results, 1)
	assert.Equal(t, "existing-uuid-1", results[0].OriginalUUID)
	assert.Equal(t, "new-uuid-1", results[0].DuplicateUUID)
}
