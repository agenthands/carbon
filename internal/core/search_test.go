package core

import (
	"context"
	"testing"

	"github.com/agenthands/carbon/internal/config"
	"github.com/neo4j/neo4j-go-driver/v5/neo4j"
	"github.com/stretchr/testify/assert"
)

func TestSearch(t *testing.T) {
	mockDriver := &MockDriver{
		MockResult: neo4j.EagerResult{
			Records: []*neo4j.Record{}, // Empty for now, verified query string mainly
		},
	}
	
	mockEmbedder := &MockEmbedder{
		Vector: []float32{0.1, 0.2, 0.3},
	}
	
	cfg := &config.Config{
		Extraction: config.ExtractionPrompts{Nodes: "foo", Edges: "bar"},
		Deduplication: config.DeduplicationPrompts{Nodes: "baz"},
		Summary: config.SummaryPrompts{Nodes: "qux"},
	}
	g := NewGraphiti(mockDriver, &MockLLM{}, mockEmbedder, cfg)
	
	ctx := context.Background()
	groupID := "test-group"
	query := "some query"
	
	// Default search currently only does text search
	// We want to verify it does vector search too
	_, err := g.Search(ctx, groupID, query)
	
	assert.NoError(t, err)
	
	// Verify that the executed query contains vector search logic
	// Memgraph vector search typically involves `call vector_search.search` or similar if using MAGE
	// Or standard cypher vector index query if supported natively in future
	// For now, let's assume MAGE syntax or simple COSINE similarity manually for MVP
	
	// We want to see if the query uses the embedding parameter
	assert.Contains(t, mockDriver.QueryParams, "embedding")
	assert.Equal(t, mockEmbedder.Vector, mockDriver.QueryParams["embedding"])
}
