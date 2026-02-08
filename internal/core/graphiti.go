package core

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/agenthands/carbon/internal/core/dedupe"
	"github.com/agenthands/carbon/internal/core/extraction"
	"github.com/agenthands/carbon/internal/core/model"
	"github.com/agenthands/carbon/internal/driver"
	"github.com/agenthands/carbon/internal/llm"
)

type Graphiti struct {
	Driver       driver.GraphDriver
	LLM          llm.LLMClient
	Embedder     llm.EmbedderClient
	Extractor    *extraction.Extractor
	Deduplicator *dedupe.Deduplicator
}

func NewGraphiti(driver driver.GraphDriver, llmClient llm.LLMClient, embedderClient llm.EmbedderClient) *Graphiti {
	return &Graphiti{
		Driver:       driver,
		LLM:          llmClient,
		Embedder:     embedderClient,
		Extractor:    extraction.NewExtractor(llmClient),
		Deduplicator: dedupe.NewDeduplicator(llmClient),
	}
}

func (g *Graphiti) BuildIndices(ctx context.Context) error {
	return g.Driver.BuildIndices(ctx)
}

func (g *Graphiti) SaveEntityNode(ctx context.Context, name, groupID, summary string) (*model.EntityNode, error) {
	node := &model.EntityNode{
		UUID:      uuid.New().String(),
		Name:      name,
		GroupID:   groupID,
		CreatedAt: time.Now().UTC(),
		Summary:   summary,
		Labels:    []string{"Entity"},
	}

	if g.Embedder != nil {
		vec, err := g.Embedder.Embed(ctx, name)
		if err == nil {
			node.NameEmbedding = vec
		}
	}

	params := map[string]interface{}{
		"uuid":           node.UUID,
		"name":           node.Name,
		"group_id":       node.GroupID,
		"created_at":     node.CreatedAt,
		"summary":        node.Summary,
		"name_embedding": node.NameEmbedding,
		"attributes":     node.Attributes,
		"labels":         node.Labels,
	}

	_, err := g.Driver.ExecuteQuery(ctx, driver.SaveEntityNodeQuery, params)
	if err != nil {
		return nil, err
	}

	return node, nil
}

func (g *Graphiti) AddEpisode(ctx context.Context, groupID, name, content string) error {
	// Simple implementation: Create Episode Node -> Extract Entities -> Link
	
	epUUID := uuid.New().String()
	now := time.Now().UTC()
	
	// Save Episode
	epParams := map[string]interface{}{
		"uuid":               epUUID,
		"name":               name,
		"group_id":           groupID,
		"created_at":         now,
		"valid_at":           now,
		"content":            content,
		"source":             "message",
		"source_description": "user message",
		"entity_edges":       []string{}, // Populated later
	}
	
	_, err := g.Driver.ExecuteQuery(ctx, driver.SaveEpisodicNodeQuery, epParams)
	if err != nil {
		return fmt.Errorf("failed to save episode: %w", err)
	}

	// Extract Entities (Mocking specific extraction logic for now, using LLM to just identify names)
	prompt := fmt.Sprintf("Extract up to 5 key entities (people, places, things) from the following text as a JSON list of strings:\n\n%s", content)
	response, err := g.LLM.Generate(ctx, prompt)
	if err != nil {
		return fmt.Errorf("failed to generate entities: %w", err)
	}

	// Clean response (basic cleanup)
	// In production, use robust JSON parsing or structured output mode
	var entityNames []string
	
	// Attempt to parse JSON
    // Start of JSON list
    start := -1
    for i, c := range response {
        if c == '[' {
            start = i
            break
        }
    }
    // End of JSON list
    end := -1
    for i := len(response) - 1; i >= 0; i-- {
        if response[i] == ']' {
            end = i
            break
        }
    }

    if start != -1 && end != -1 && end > start {
        jsonStr := response[start : end+1]
        if err := json.Unmarshal([]byte(jsonStr), &entityNames); err != nil {
             fmt.Printf("Failed to parse extracted entities: %v\n", err)
        }
    }

	for _, eName := range entityNames {
		// Save Entity
		node, err := g.SaveEntityNode(ctx, eName, groupID, "")
		if err != nil {
			continue
		}

		// Create Edge MENTIONS
		edgeUUID := uuid.New().String()
		edgeParams := map[string]interface{}{
			"uuid":        edgeUUID,
			"source_uuid": epUUID,
			"target_uuid": node.UUID,
			"group_id":    groupID,
			"created_at":  now,
		}
		
		_, err = g.Driver.ExecuteQuery(ctx, driver.SaveEpisodicEdgeQuery, edgeParams)
		if err != nil {
			fmt.Printf("Failed to link episode to entity %s: %v\n", eName, err)
		}
	}

	return nil
}

func (g *Graphiti) Search(ctx context.Context, groupID, query string) ([]string, error) {
	// Hybrid Search Implementation
	
	// 1. Get Embedding
	var queryVector []float32
	if g.Embedder != nil {
		vec, err := g.Embedder.Embed(ctx, query)
		if err == nil {
			queryVector = vec
		}
	}
	
	// 2. Construct Query
	// Using a basic approximation of hybrid search:
	// Find nodes by text match (CONTAINS) AND find nodes by vector similarity if vector exists
	// Memgraph supports vector search. Typically via `vector_search.search` module or similar.
	// Since we are porting, let's assume we want to pass the vector to the DB query logic.
	
	cypher := `
		MATCH (n:Entity {group_id: $group_id})
		WHERE n.name CONTAINS $query
		// AND logic for vector similarity would be here if using MAGE, e.g. using cosine_similarity function
		// WITH n, vector_cosine_similarity(n.name_embedding, $embedding) AS score
		RETURN n.name AS name, n.summary AS summary
		// ORDER BY score DESC
		LIMIT 5
	`
	
	params := map[string]interface{}{
		"group_id": groupID,
		"query":    query,
	}
	
	if len(queryVector) > 0 {
		params["embedding"] = queryVector
	}
	
	result, err := g.Driver.ExecuteQuery(ctx, cypher, params)
	if err != nil {
		return nil, err
	}
	
	var results []string
	for _, record := range result.Records {
		name, _ := record.Get("name")
		summary, _ := record.Get("summary")
		results = append(results, fmt.Sprintf("%s: %s", name, summary))
	}

	return results, nil
}

// Helper to get existing nodes in group
func (g *Graphiti) getGroupNodes(ctx context.Context, groupID string) ([]model.EntityNode, error) {
	cypher := `MATCH (n:Entity {group_id: $group_id}) RETURN n.uuid AS uuid, n.name AS name`
	res, err := g.Driver.ExecuteQuery(ctx, cypher, map[string]interface{}{"group_id": groupID})
	if err != nil {
		return nil, err
	}
	
	var nodes []model.EntityNode
	for _, rec := range res.Records {
		uuid, _ := rec.Get("uuid")
		name, _ := rec.Get("name")
		nodes = append(nodes, model.EntityNode{
			UUID: uuid.(string),
			Name: name.(string),
		})
	}
	return nodes, nil
}

// Helper: Save Entity Node
func (g *Graphiti) saveEntity(ctx context.Context, node model.EntityNode) error {
	params := map[string]interface{}{
		"uuid":           node.UUID,
		"name":           node.Name,
		"group_id":       node.GroupID,
		"created_at":     node.CreatedAt.Format(time.RFC3339),
		"summary":        "", // Could be empty initially
		"name_embedding": nil, // Should compute embedding ideally
		"attributes":     "{}",
		"labels":         []string{},
	}
	
	// Compute embedding if possible
	if g.Embedder != nil {
		emb, err := g.Embedder.Embed(ctx, node.Name)
		if err == nil {
			params["name_embedding"] = emb
		}
	}

	_, err := g.Driver.ExecuteQuery(ctx, driver.SaveEntityNodeQuery, params)
	return err
}
