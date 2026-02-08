package core

import (
	"context"
	"fmt"
	"testing"
	
	"github.com/agenthands/carbon/internal/config"
	"github.com/neo4j/neo4j-go-driver/v5/neo4j"
	"github.com/stretchr/testify/assert"
)

func TestAddEpisode(t *testing.T) {
	// Mock LLM Responses in order
	// 1. ExtractNodes: Returns "Alice", "Bob"
	// 2. ResolveDuplicates: Returns empty (no duplicates) if prompted
	//    Wait, Deduplicator is called only if existing nodes exist.
	//    AddEpisode logic:
	//    - saves episode
	//    - extract nodes
	//    - get existing nodes
	//    - dedupe (if duplicates exist)
	//    - save entities
	//    - save MENTIONS
	//    - extract edges
	//    - save edges
	//    - summarize nodes (if facts)
	
	entitiesJSON := `{
		"extracted_entities": [
			{"name": "Alice", "entity_type_id": 1},
			{"name": "Bob", "entity_type_id": 1}
		]
	}`
	
	// Predicted UUID sequence:
	// 1. Episode: "uuid-1"
	// 2. Alice: "uuid-2"
	// 3. Bob: "uuid-3"
	// 4. Edge (Alice): "uuid-4" (mentions)
	// 5. Edge (Bob): "uuid-5" (mentions)
	// 6. Edge (Alice-Bob): "uuid-6" (extracted edge)
	
	// So ExtractEdges Mock JSON should refer to uuid-2 (Alice) and uuid-3 (Bob)
	edgesJSON := `{
		"extracted_edges": [
			{"source_node_uuid": "uuid-2", "target_node_uuid": "uuid-3", "relation_type": "FRIEND", "fact": "Alice is friends with Bob"}
		]
	}`
	
	summaryJSON := `{
		"summary": "Alice is friends with Bob."
	}`

	mockLLM := &MockLLM{
		ResponseQueue: []string{
			entitiesJSON, // Extraction
			// Dedupe skipped
			edgesJSON,    // Edges
			summaryJSON,  // Summary for Alice
			summaryJSON,  // Summary for Bob
		},
	}
	
	mockDriver := &MockDriver{
		MockResult: neo4j.EagerResult{Records: []*neo4j.Record{}}, // Searching for existing nodes returns empty
	}
	
	cfg := &config.Config{
		Extraction: config.ExtractionPrompts{Nodes: "%s", Edges: "%s"},
		Deduplication: config.DeduplicationPrompts{Nodes: "%s"},
		Summary: config.SummaryPrompts{Nodes: "%s"},
	}
	
	g := NewGraphiti(mockDriver, mockLLM, &MockEmbedder{}, cfg)
	
	uuidCounter := 0
	g.UUIDGenerator = func() string {
		uuidCounter++
		return fmt.Sprintf("uuid-%d", uuidCounter)
	}
	
	err := g.AddEpisode(context.Background(), "group-1", "Ep1", "Alice met Bob.")
	
	assert.NoError(t, err)
	// ... existing test content ...
}

func TestAddEpisode_Deduplication(t *testing.T) {
	// Scenario: New "Alice" is duplicate of Existing "Alice"
	// Calls:
	// 1. Extract: "Alice"
	// 2. GetGroupNodes: "Alice" exists
	// 3. Dedupe: Returns Alice->Alice
	// 4. SaveEntity (updated UUID)
	// 5. Edges: skipped or extracted
	
	entitiesJSON := `{ "extracted_entities": [ {"name": "Alice", "entity_type_id": 1} ] }`
	dedupeJSON := `{ "duplicates": [ {"original_uuid": "existing-uuid-1", "duplicate_uuid": "mock-uuid-2", "confidence": 0.9} ] }`
	// Note: We need to coordinate UUIDs.
	// Sequence:
	// 1. Episode: "mock-uuid-1"
	// 2. Extracted "Alice". New UUID: "mock-uuid-2" (from generator)
	
	mockLLM := &MockLLM{
		ResponseQueue: []string{
			entitiesJSON,
			dedupeJSON,
			// No edges or summary needed if we stop or verify mock logic
			// Actually AddEpisode continues.
			// It will create Mentions edge ("mock-uuid-3").
			// It will call ExtractEdges.
			`{"extracted_edges": []}`, // Empty edges
		},
	}
	
	// empty line or remove
	// Mocking record is hard without internal neo4j utils or interface.
	// Our MockDriver returns *neo4j.EagerResult which has Records []*Record.
	// The `Record` struct has private fields? No, `Values`?
	// `neo4j.Record` exports `Values`, `Keys`.
	// Check `getGroupNodes` implementation: uses `rec.Get("uuid")`.
	// We need to construct a Record that works with `Get`.
	// `neo4j.Record` is concrete struct. We can construct it?
	// Records: []*neo4j.Record{ {Keys: ["uuid", "name", "summary"], Values: ["existing-uuid-1", "Alice", "Old summary"]} }

	mockDriver := &MockDriver{
		MockResult: neo4j.EagerResult{
			Records: []*neo4j.Record{
				{
					Keys: []string{"uuid", "name", "summary"},
					Values: []interface{}{"existing-uuid-1", "Alice", "Old summary"},
				},
			},
		},
	}
	
	cfg := &config.Config{
		Extraction: config.ExtractionPrompts{Nodes: "foo", Edges: "bar"},
		Deduplication: config.DeduplicationPrompts{Nodes: "baz"},
		Summary: config.SummaryPrompts{Nodes: "qux"},
	}
	
	g := NewGraphiti(mockDriver, mockLLM, &MockEmbedder{}, cfg)
	
	uuidCounter := 0
	g.UUIDGenerator = func() string {
		uuidCounter++
		return fmt.Sprintf("mock-uuid-%d", uuidCounter)
	}
	
	// Add Episode
	err := g.AddEpisode(context.Background(), "group-1", "Ep2", "Alice is back.")
	assert.NoError(t, err)
	
	// Verify Dedupe Logic:
	// New Node UUID "mock-uuid-2" should have been replaced by "existing-uuid-1" in `saveEntity`.
	// We can check `MockDriver.QueryExecuted` or params.
	// But `ExecuteQuery` is called multiple times. MockDriver overwrite params.
	// Last call is `SaveEpisodicEdgeQuery` or `SaveEntityEdgeQuery`?
    // If no entity edges, last call is `SaveEpisodicEdgeQuery` (Mentions).
    // Params: source_uuid=episode, target_uuid=node.UUID.
    // target_uuid should be "existing-uuid-1".
    
	assert.Equal(t, "existing-uuid-1", mockDriver.QueryParams["target_uuid"])
}

func TestAddEpisode_ExtractionError(t *testing.T) {
	mockLLM := &MockLLM{
		Response: "invalid json", 
		// MockLLM without queue returns Response always. 
		// Or with queue, simpler to just fail first call.
	}
	
	mockDriver := &MockDriver{}
	
	cfg := &config.Config{
		Extraction: config.ExtractionPrompts{Nodes: "foo"},
	}
	
	g := NewGraphiti(mockDriver, mockLLM, &MockEmbedder{}, cfg)
	
	err := g.AddEpisode(context.Background(), "group-1", "Ep1", "content")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "extraction failed")
}
