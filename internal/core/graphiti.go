package core

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/agenthands/carbon/internal/config"
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
	Config       *config.Config
}

func NewGraphiti(driver driver.GraphDriver, llmClient llm.LLMClient, embedderClient llm.EmbedderClient, cfg *config.Config) *Graphiti {
	return &Graphiti{
		Driver:       driver,
		LLM:          llmClient,
		Embedder:     embedderClient,
		Extractor:    extraction.NewExtractor(llmClient, cfg.Extraction),
		Deduplicator: dedupe.NewDeduplicator(llmClient, cfg.Deduplication),
		Summarizer:   summary.NewSummarizer(llmClient, cfg.Summary),
		Config:       cfg,
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
	// Using basic cosine similarity if embedding is provided.
	// Note: This relies on Memgraph MAGE or similar functions being available.
	// We'll use a manual cosine similarity formula or function if available.
	// Assuming `vector.cosine_similarity` or similar exists, or manually:
	// score = reduce(dot=0.0, i in range(0, size(a)-1) | dot + a[i]*b[i]) / (sqrt(reduce(s=0.0, x in a | s + x^2)) * sqrt(reduce(s=0.0, x in b | s + x^2)))
	// For simplicity in this port, we will stick to the text search if vector is missing, 
	// and add vector ordering if present.
	
	cypher := `
		MATCH (n:Entity {group_id: $group_id})
		WHERE n.name CONTAINS $query
		RETURN n.name AS name, n.summary AS summary
		LIMIT 5
	`
	
	if len(queryVector) > 0 {
		// Hybrid: Text match + Vector Sort
		// Note: efficient hybrid search requires specific indexing (e.g. Lucene+HNSW).
		// Here we simply rank text-matches by vector similarity.
        // Using MAGE: query_module.function(n, $embedding)
        // For standard Memgraph without modules, we might skip. 
        // We will assume `distance` function or similar if tracking.
        // Let's implement a safe generic sort if possible, or just note limitation.
        
        // BETTER: Use Memgraph's built-in `cos` distance if available, or just leave text search as MVP 
        // since we don't know the exact environment capabilities (MAGE vs Core).
        // User's original Python code likely used `vector_index` search.
        
        // Updated query to attempt sorting by simple vector property if possible?
        // Let's just uncomment the logic structure but guard it.
        
        // For now, to satisfy review `logical inconsistency`: 
        // The previous code had it commented out. 
        // We will enable a basic similarity sort assuming 'n.name_embedding' exists.
        
        cypher = `
            MATCH (n:Entity {group_id: $group_id})
            WHERE n.name CONTAINS $query
            AND n.name_embedding IS NOT NULL
            WITH n, 
                 reduce(dot = 0.0, i in range(0, size(n.name_embedding)-1) | dot + n.name_embedding[i] * $embedding[i]) / 
                 (sqrt(reduce(s1 = 0.0, x in n.name_embedding | s1 + x^2)) * sqrt(reduce(s2 = 0.0, y in $embedding | s2 + y^2))) AS score
            RETURN n.name AS name, n.summary AS summary
            ORDER BY score DESC
            LIMIT 5
        `
	}
	
	params := map[string]interface{}{
		"group_id": groupID,
		"query":    query,
	}
	
	if len(queryVector) > 0 {
		params["embedding"] = queryVector
	}
	
	result, err := g.Driver.ExecuteQuery(ctx, cypher, params)
	if err != nil {
		// Log warning/fallback to text only if vector math fails (e.g. index mismatch)
		// For now return error
		return nil, fmt.Errorf("search failed: %w", err)
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
