package extraction

import (
	"context"
	"testing"
	
	"github.com/agenthands/carbon/internal/config"
	"github.com/agenthands/carbon/internal/core/model"
	"github.com/stretchr/testify/assert"
)

// TestExtractNodes ensures that ExtractNodes function correctly parses LLM response
// and returns expected ExtractedEntity objects.
func TestExtractNodes(t *testing.T) {
	// Mock LLM Response matching Python's prompt output structure
	mockJSON := `{
		"extracted_entities": [
			{"name": "Alice", "entity_type_id": 1},
			{"name": "Bob", "entity_type_id": 1}
		]
	}`
	
	mockLLM := &MockLLMClient{
		Response: mockJSON,
	}
	
	configPrompts := config.ExtractionPrompts{
		Nodes: "test prompt %s %s",
	}
	extractor := NewExtractor(mockLLM, configPrompts)
	
	ctx := context.Background()
	content := "Alice met Bob yesterday."
	entityTypes := `1: Person, 2: Place`
	
	entities, err := extractor.ExtractNodes(ctx, content, entityTypes, nil)
	
	assert.NoError(t, err)
	assert.Len(t, entities, 2)
	assert.Equal(t, "Alice", entities[0].Name)
	assert.Equal(t, 1, entities[0].EntityTypeID)
	assert.Equal(t, "Bob", entities[1].Name)
	assert.Equal(t, 1, entities[1].EntityTypeID)
}

func TestExtractEdges(t *testing.T) {
	mockJSON := `{
		"extracted_edges": [
			{
				"source_node_uuid": "uuid-1",
				"target_node_uuid": "uuid-2",
				"relation_type": "FRIEND",
				"fact": "Alice is friends with Bob"
			}
		]
	}`
	
	mockLLM := &MockLLMClient{
		Response: mockJSON,
	}
	
	configPrompts := config.ExtractionPrompts{
		Edges: "test prompt %s",
	}
	extractor := NewExtractor(mockLLM, configPrompts)
	ctx := context.Background()
	
	// Assuming ExtractedEntity objects are needed as input, creating mocks
	nodes := []model.EntityNode{
		{UUID: "uuid-1", Name: "Alice"},
		{UUID: "uuid-2", Name: "Bob"},
	}
	
	edges, err := extractor.ExtractEdges(ctx, nodes, nil)
	
	assert.NoError(t, err)
	assert.Len(t, edges, 1)
	assert.Equal(t, "uuid-1", edges[0].SourceNodeUUID)
	assert.Equal(t, "uuid-2", edges[0].TargetNodeUUID)
	assert.Equal(t, "FRIEND", edges[0].RelationType)
}
