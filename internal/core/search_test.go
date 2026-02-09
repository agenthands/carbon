package core

import (
	"context"
	"fmt"
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
		Summary: config.SummaryPrompts{Nodes: "qux"},
	}
	g := NewGraphiti(mockDriver, &MockLLM{}, mockEmbedder, nil, cfg)
	
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

func TestSearch_Error(t *testing.T) {
	mockDriver := &MockDriver{
		Err: fmt.Errorf("db error"),
	}
	
	cfg := &config.Config{}
	g := NewGraphiti(mockDriver, &MockLLM{}, &MockEmbedder{}, nil, cfg)
	
	_, err := g.Search(context.Background(), "g1", "query")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "db error")
}

func TestBuildIndices(t *testing.T) {
	mockDriver := &MockDriver{}
	g := NewGraphiti(mockDriver, &MockLLM{}, &MockEmbedder{}, nil, &config.Config{})
	
	err := g.BuildIndices(context.Background())
	assert.NoError(t, err)
}

func TestSaveEntityNode(t *testing.T) {
	mockDriver := &MockDriver{}
	mockEmbedder := &MockEmbedder{Vector: []float32{1.0, 2.0}}
	
	g := NewGraphiti(mockDriver, &MockLLM{}, mockEmbedder, nil, &config.Config{})
	
	node, err := g.SaveEntityNode(context.Background(), "EntityName", "Group1", "Summary")
	assert.NoError(t, err)
	assert.NotNil(t, node)
	assert.Equal(t, "EntityName", node.Name)
	assert.Equal(t, []float32{1.0, 2.0}, node.NameEmbedding)
}
