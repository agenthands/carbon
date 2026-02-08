package core

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/agenthands/carbon/internal/core/dedupe"
	"github.com/agenthands/carbon/internal/core/extraction"
	"github.com/agenthands/carbon/internal/core/model"
	"github.com/agenthands/carbon/internal/core/summary"
	"github.com/agenthands/carbon/internal/driver"
	"github.com/agenthands/carbon/internal/llm"
)

type Graphiti struct {
	Driver       driver.GraphDriver
	LLM          llm.LLMClient
	Embedder     llm.EmbedderClient
	Extractor    *extraction.Extractor
	Deduplicator *dedupe.Deduplicator
	Summarizer   *summary.Summarizer
}

func NewGraphiti(driver driver.GraphDriver, llmClient llm.LLMClient, embedderClient llm.EmbedderClient) *Graphiti {
	return &Graphiti{
		Driver:       driver,
		LLM:          llmClient,
		Embedder:     embedderClient,
		Extractor:    extraction.NewExtractor(llmClient),
		Deduplicator: dedupe.NewDeduplicator(llmClient),
		Summarizer:   summary.NewSummarizer(llmClient),
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
	// 1. Create Episode Node
	episodeUUID := uuid.New().String()
	now := time.Now().UTC()
	
	params := map[string]interface{}{
		"uuid":               episodeUUID,
		"name":               name, 
		"group_id":           groupID, 
		"created_at":         now.Format(time.RFC3339),
		"valid_at":           now.Format(time.RFC3339),
		"content":            content,
		"source":             "user", // simple default
		"source_description": "user message",
		"entity_edges":       []string{},
	}
	
	if _, err := g.Driver.ExecuteQuery(ctx, driver.SaveEpisodicNodeQuery, params); err != nil {
		return fmt.Errorf("failed to save episode: %w", err)
	}

	// 2. Extract Entities
	extractedEntities, err := g.Extractor.ExtractNodes(ctx, content, "Person, Place, Organization", nil)
	if err != nil {
		return fmt.Errorf("extraction failed: %w", err)
	}

	// Convert Extracted to EntityNode
	var newNodes []model.EntityNode
	for _, e := range extractedEntities {
		newNodes = append(newNodes, model.EntityNode{
			UUID:      uuid.New().String(),
			Name:      e.Name,
			GroupID:   groupID,
			CreatedAt: now,
		})
	}

	// 3. Deduplicate against existing
	existingNodes, err := g.getGroupNodes(ctx, groupID)
	// Only dedupe if we have existing nodes and new nodes
	if err == nil && len(existingNodes) > 0 && len(newNodes) > 0 {
		duplicates, err := g.Deduplicator.ResolveDuplicates(ctx, newNodes, existingNodes)
		if err == nil {
			// Map duplicate UUIDs
			dupMap := make(map[string]string) // newUUID -> existingUUID
			for _, d := range duplicates {
				dupMap[d.DuplicateUUID] = d.OriginalUUID
			}
			
			// Update UUIDs
			for i := range newNodes {
				if existingUUID, found := dupMap[newNodes[i].UUID]; found {
					newNodes[i].UUID = existingUUID
					// Fetch existing summary for context
					for _, en := range existingNodes {
						if en.UUID == existingUUID {
							newNodes[i].Summary = en.Summary
							break
						}
					}
				}
			}
		}
	}

	// 4. Save Entities and MENTIONS edges
	for _, node := range newNodes {
		if err := g.saveEntity(ctx, node); err != nil {
			continue
		}
		
		// Create MENTIONS edge from Episode to Entity
		edgeUUID := uuid.New().String()
		edgeParams := map[string]interface{}{
			"uuid":        edgeUUID,
			"source_uuid": episodeUUID,
			"target_uuid": node.UUID,
			"group_id":    groupID,
			"created_at":  now.Format(time.RFC3339),
		}
		
		if _, err := g.Driver.ExecuteQuery(ctx, driver.SaveEpisodicEdgeQuery, edgeParams); err != nil {
			// Log error but continue
		}
	}

	// 5. Extract Edges (Entity-Entity)
	if len(newNodes) > 1 {
		edges, err := g.Extractor.ExtractEdges(ctx, newNodes, nil)
		if err == nil {
			// Collect facts for summarization
			nodeFacts := make(map[string][]string) // Node UUID -> Facts
			
			for _, e := range edges {
				edgeParams := map[string]interface{}{
					"uuid":           uuid.New().String(),
					"source_uuid":    e.SourceNodeUUID,
					"target_uuid":    e.TargetNodeUUID,
					"name":           e.RelationType,
					"fact":           e.Fact,
					"group_id":       groupID,
					"created_at":     now.Format(time.RFC3339),
					"expired_at":     "",
					"valid_at":       now.Format(time.RFC3339),
					"invalid_at":     "",
					"episodes":       []string{episodeUUID},
					"fact_embedding": nil,
					"attributes":     "{}",
				}
				
				if g.Embedder != nil {
					emb, err := g.Embedder.Embed(ctx, e.Fact)
					if err == nil {
						edgeParams["fact_embedding"] = emb
					}
				}

				if _, err := g.Driver.ExecuteQuery(ctx, driver.SaveEntityEdgeQuery, edgeParams); err != nil {
					// Log error
				}
				
				// Add fact to nodes
				nodeFacts[e.SourceNodeUUID] = append(nodeFacts[e.SourceNodeUUID], e.Fact)
				nodeFacts[e.TargetNodeUUID] = append(nodeFacts[e.TargetNodeUUID], e.Fact)
			}
			
			// 6. Summarize Nodes
			for _, node := range newNodes {
				facts, hasFacts := nodeFacts[node.UUID]
				if hasFacts {
					newSummary, err := g.Summarizer.SummarizeNode(ctx, node, facts)
					if err == nil {
						node.Summary = newSummary
						// Update node with new summary
						g.saveEntity(ctx, node)
					}
				}
			}
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
	cypher := `MATCH (n:Entity {group_id: $group_id}) RETURN n.uuid AS uuid, n.name AS name, n.summary AS summary`
	res, err := g.Driver.ExecuteQuery(ctx, cypher, map[string]interface{}{"group_id": groupID})
	if err != nil {
		return nil, err
	}
	
	var nodes []model.EntityNode
	for _, rec := range res.Records {
		uuid, _ := rec.Get("uuid")
		name, _ := rec.Get("name")
		summary, _ := rec.Get("summary")
		
		sVal := ""
		if summary != nil {
			sVal = summary.(string)
		}
		
		nodes = append(nodes, model.EntityNode{
			UUID:    uuid.(string),
			Name:    name.(string),
			Summary: sVal,
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
		"summary":        node.Summary, 
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
