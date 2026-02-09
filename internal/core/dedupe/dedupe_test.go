package dedupe

import (
	"context"
	"testing"
	
	"github.com/agenthands/carbon/internal/config"
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
	
	cfg := config.DeduplicationPrompts{
		Nodes: "test prompt %s %s",
	}
	deduplicator := NewDeduplicator(mockLLM, cfg)
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

func TestResolveEdgeContradictions(t *testing.T) {
	mockJSON := `{
		"contradicted_edge_uuids": ["uuid-1"]
	}`
	
	mockLLM := &MockLLMClient{
		Response: mockJSON,
	}
	
	cfg := config.DeduplicationPrompts{
		Edges: "test prompt %s %s",
	}
	deduplicator := NewDeduplicator(mockLLM, cfg)
	ctx := context.Background()
	
	newEdgeFact := "Alice moved to SF"
	existingEdges := []model.EntityEdge{
		{UUID: "uuid-1", Fact: "Alice lives in Seattle"},
		{UUID: "uuid-2", Fact: "Alice is a software engineer"},
	}
	
	uuids, err := deduplicator.ResolveEdgeContradictions(ctx, newEdgeFact, existingEdges)
	
	assert.NoError(t, err)
	assert.Len(t, uuids, 1)
	assert.Equal(t, "uuid-1", uuids[0])
}
