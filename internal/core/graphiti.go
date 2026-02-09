package core

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/agenthands/carbon/internal/config"
	"github.com/agenthands/carbon/internal/core/community"
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
	CommunityDetector community.CommunityDetector
	Reranker     llm.RerankerClient
	Config       *config.Config
	UUIDGenerator func() string
}

func NewGraphiti(driver driver.GraphDriver, llmClient llm.LLMClient, embedderClient llm.EmbedderClient, reranker llm.RerankerClient, cfg *config.Config) *Graphiti {
	if reranker == nil {
		reranker = llm.NewSimpleLLMReranker(llmClient)
	}
	return &Graphiti{
		Driver:       driver,
		LLM:          llmClient,
		Embedder:     embedderClient,
		Reranker:     reranker,
		Extractor:    extraction.NewExtractor(llmClient, cfg.Extraction),
		Deduplicator: dedupe.NewDeduplicator(llmClient, cfg.Deduplication),
		Summarizer:   summary.NewSummarizer(llmClient, cfg.Summary),
		CommunityDetector: community.NewSimpleDetector(),
		Config:       cfg,
		UUIDGenerator: func() string { return uuid.New().String() },
	}
}

func (g *Graphiti) BuildIndices(ctx context.Context) error {
	return g.Driver.BuildIndices(ctx)
}

func (g *Graphiti) SaveEntityNode(ctx context.Context, name, groupID, summary string) (*model.EntityNode, error) {
	node := &model.EntityNode{
		UUID:      g.UUIDGenerator(),
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

	var attrsJSON string
	if len(node.Attributes) > 0 {
		if b, err := json.Marshal(node.Attributes); err == nil {
			attrsJSON = string(b)
		} else {
			attrsJSON = "{}" // Default on error
		}
	} else {
		attrsJSON = "{}"
	}

	params := map[string]interface{}{
		"uuid":           node.UUID,
		"name":           node.Name,
		"group_id":       node.GroupID,
		"created_at":     node.CreatedAt,
		"summary":        node.Summary,
		"name_embedding": node.NameEmbedding,
		"attributes":     attrsJSON,
		"labels":         node.Labels,
	}

	_, err := g.Driver.ExecuteQuery(ctx, driver.SaveEntityNodeQuery, params)
	if err != nil {
		return nil, err
	}

	return node, nil
}

func (g *Graphiti) AddEpisode(ctx context.Context, groupID, name, content, saga, schema string) error {
	return g.addEpisodeInternal(ctx, groupID, name, content, saga, schema, nil)
}

func (g *Graphiti) addEpisodeInternal(ctx context.Context, groupID, name, content, saga, schema string, preResolvedNodes []model.EntityNode) error {
	episodeUUID := g.UUIDGenerator()
	now := time.Now().UTC()

	// 1. Create Episode Node
	if err := g.saveEpisodeNode(ctx, episodeUUID, name, groupID, content, now); err != nil {
		return fmt.Errorf("failed to save episode: %w", err)
	}

	var nodes []model.EntityNode

	if preResolvedNodes != nil {
		nodes = preResolvedNodes
	} else {
		// 2. Extract Entities
		// Get context from previous episodes
		prevEpisodes, _ := g.retrievePreviousEpisodes(ctx, groupID, episodeUUID, 5)

		if schema == "" {
			schema = "Person, Place, Organization"
		}
		extractedEntities, err := g.Extractor.ExtractNodes(ctx, content, schema, prevEpisodes)
		if err != nil {
			return fmt.Errorf("extraction failed: %w", err)
		}

		// Convert Extracted to EntityNode
		newNodes := g.convertToEntityNodes(extractedEntities, groupID, now)

		// 3. Deduplicate against existing
		existingNodes, err := g.getGroupNodes(ctx, groupID)
		if err == nil && len(existingNodes) > 0 && len(newNodes) > 0 {
			newNodes = g.resolveDuplicates(ctx, newNodes, existingNodes)
		}
		nodes = newNodes
	}

	// 4. Save Entities and MENTIONS edges
	// Note: If preResolvedNodes were passed, they are already saved/resolved by BulkAddEpisodes.
	// But we still need to create MENTIONS edges.
	// saveNewEntitiesAndMentions executes MERGE for nodes, so it's safe to run again.
	g.saveNewEntitiesAndMentions(ctx, nodes, episodeUUID, groupID, now)

	// 5. Extract Edges (Entity-Entity) & Summarize
	if len(nodes) > 1 {
		if err := g.processEntityEdgesAndSummaries(ctx, nodes, episodeUUID, groupID, now); err != nil {
			// Log error but continue
		}
	}

	// 6. Start Saga Processing if saga name is provided
	if saga != "" {
		if err := g.handleSaga(ctx, saga, groupID, episodeUUID, now); err != nil {
			return fmt.Errorf("failed to handle saga: %w", err)
		}
	}

	return nil
}

// ---------------- Helper Methods ----------------

func (g *Graphiti) retrievePreviousEpisodes(ctx context.Context, groupID string, excludeUUID string, limit int) ([]string, error) {
	if limit <= 0 {
		limit = 5 // Default context window
	}

	res, err := g.Driver.ExecuteQuery(ctx, driver.GetRecentEpisodesQuery, map[string]interface{}{
		"group_id": groupID,
		"limit":    limit + 1, // Fetch +1 to account for potential exclusion
	})
	if err != nil {
		return nil, err
	}

	var episodes []string
	// Iterate through records
	for _, rec := range res.Records {
		uuid, _ := rec.Get("uuid")
		if uuidStr, ok := uuid.(string); ok && uuidStr == excludeUUID {
			continue
		}
		
		if content, ok := rec.Get("content"); ok && content != nil {
			episodes = append(episodes, content.(string))
		}
		if len(episodes) >= limit {
			break
		}
	}
	return episodes, nil
}

func (g *Graphiti) saveEpisodeNode(ctx context.Context, uuid, name, groupID, content string, now time.Time) error {
	params := map[string]interface{}{
		"uuid":               uuid,
		"name":               name, 
		"group_id":           groupID, 
		"created_at":         now.Format(time.RFC3339),
		"valid_at":           now.Format(time.RFC3339),
		"content":            content,
		"source":             "user", 
		"source_description": "user message",
		"entity_edges":       []string{},
	}
	_, err := g.Driver.ExecuteQuery(ctx, driver.SaveEpisodicNodeQuery, params)
	return err
}

func (g *Graphiti) convertToEntityNodes(extracted []model.ExtractedEntity, groupID string, now time.Time) []model.EntityNode {
	var nodes []model.EntityNode
	for _, e := range extracted {
		nodes = append(nodes, model.EntityNode{
			UUID:       g.UUIDGenerator(),
			Name:       e.Name,
			GroupID:    groupID,
			CreatedAt:  now,
			Attributes: e.Attributes,
			Labels:     []string{"Entity"},
		})
	}
	return nodes
}

func (g *Graphiti) resolveDuplicates(ctx context.Context, newNodes, existingNodes []model.EntityNode) []model.EntityNode {
	duplicates, err := g.Deduplicator.ResolveDuplicates(ctx, newNodes, existingNodes)
	if err != nil {
		return newNodes // Fallback: treat as new
	}

	dupMap := make(map[string]string) 
	for _, d := range duplicates {
		dupMap[d.DuplicateUUID] = d.OriginalUUID
	}
	
	for i := range newNodes {
		if existingUUID, found := dupMap[newNodes[i].UUID]; found {
			newNodes[i].UUID = existingUUID
			// Fetch existing summary
			for _, en := range existingNodes {
				if en.UUID == existingUUID {
					newNodes[i].Summary = en.Summary
					break
				}
			}
		}
	}
	return newNodes
}

func (g *Graphiti) saveNewEntitiesAndMentions(ctx context.Context, nodes []model.EntityNode, episodeUUID, groupID string, now time.Time) {
	for _, node := range nodes {
		if err := g.saveEntity(ctx, node); err != nil {
			continue
		}
		
		edgeUUID := g.UUIDGenerator()
		edgeParams := map[string]interface{}{
			"uuid":        edgeUUID,
			"source_uuid": episodeUUID,
			"target_uuid": node.UUID,
			"group_id":    groupID,
			"created_at":  now.Format(time.RFC3339),
		}
		
		g.Driver.ExecuteQuery(ctx, driver.SaveEpisodicEdgeQuery, edgeParams)
	}
}

func (g *Graphiti) processEntityEdgesAndSummaries(ctx context.Context, nodes []model.EntityNode, episodeUUID, groupID string, now time.Time) error {
	edges, err := g.Extractor.ExtractEdges(ctx, nodes, nil)
	if err != nil {
		return err
	}
	
	nodeFacts := make(map[string][]string)
	
	for _, e := range edges {
		// 1. Get existing edges from source node (needed for contradiction check across targets)
		relatedEdges, err := g.getEdgesFromSource(ctx, e.SourceNodeUUID)
		if err != nil {
			continue
		}

		// 2. Check for Exact Match (Deduplication)
		isDuplicate := false
		for _, re := range relatedEdges {
			// Strict dedupe: source (implicit), target, relation, fact MUST match
			if re.TargetUUID == e.TargetNodeUUID && re.Fact == e.Fact && re.Name == e.RelationType {
				isDuplicate = true
				break
			}
		}

		if isDuplicate {
			// Edge exists, track fact for summary but skip saving edge
			nodeFacts[e.SourceNodeUUID] = append(nodeFacts[e.SourceNodeUUID], e.Fact)
			nodeFacts[e.TargetNodeUUID] = append(nodeFacts[e.TargetNodeUUID], e.Fact)
			continue
		}

		// 3. Check for Contradictions
		if len(relatedEdges) > 0 {
			contradictedUUIDs, err := g.Deduplicator.ResolveEdgeContradictions(ctx, e.Fact, relatedEdges)
			if err != nil {
				fmt.Printf("Error checking contradictions: %v\n", err)
			} else if len(contradictedUUIDs) > 0 {
				// Invalidate contradicted edges
				for _, cuuid := range contradictedUUIDs {
					// Use new edge validity as invalid_at for old edge
					// Actually, use current time or episode time
					g.invalidateEdge(ctx, cuuid, now)
				}
			}
		}

		edgeParams := map[string]interface{}{
			"uuid":           g.UUIDGenerator(),
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
			if emb, err := g.Embedder.Embed(ctx, e.Fact); err == nil {
				edgeParams["fact_embedding"] = emb
			}
		}

		g.Driver.ExecuteQuery(ctx, driver.SaveEntityEdgeQuery, edgeParams)
		
		nodeFacts[e.SourceNodeUUID] = append(nodeFacts[e.SourceNodeUUID], e.Fact)
		nodeFacts[e.TargetNodeUUID] = append(nodeFacts[e.TargetNodeUUID], e.Fact)
	}

	// Summarize Nodes
	for _, node := range nodes {
		if facts, hasFacts := nodeFacts[node.UUID]; hasFacts {
			if newSummary, err := g.Summarizer.SummarizeNode(ctx, node, facts); err == nil {
				node.Summary = newSummary
				g.saveEntity(ctx, node)
			}
		}
	}
	return nil
}

func (g *Graphiti) linkNextEpisode(ctx context.Context, prevUUID, nextUUID, groupID string, now time.Time) error {
	params := map[string]interface{}{
		"uuid":        g.UUIDGenerator(),
		"source_uuid": prevUUID,
		"target_uuid": nextUUID,
		"group_id":    groupID,
		"created_at":  now.Format(time.RFC3339),
	}
	_, err := g.Driver.ExecuteQuery(ctx, driver.SaveNextEpisodeEdgeQuery, params)
	return err
}

func (g *Graphiti) linkSagaHasEpisode(ctx context.Context, sagaUUID, episodeUUID, groupID string, now time.Time) error {
	params := map[string]interface{}{
		"uuid":        g.UUIDGenerator(),
		"source_uuid": sagaUUID,
		"target_uuid": episodeUUID,
		"group_id":    groupID,
		"created_at":  now.Format(time.RFC3339),
	}
	_, err := g.Driver.ExecuteQuery(ctx, driver.SaveHasEpisodeEdgeQuery, params)
	return err
}

func (g *Graphiti) DetectAndSummarizeCommunities(ctx context.Context, groupID string) error {
	// 1. Fetch Group Nodes
	nodes, err := g.getGroupNodes(ctx, groupID)
	if err != nil { return err }
	
	// 2. Fetch Group Edges
	edges, err := g.getGroupEdges(ctx, groupID)
	if err != nil { return err }
	
	// 3. Detect Communities
	communities, err := g.CommunityDetector.Detect(nodes, edges)
	if err != nil { return err }
	
	now := time.Now().UTC()
	
	fmt.Printf("Detected %d communities for group %s\n", len(communities), groupID)

	// 4. Summarize and Save
	for i, commNodes := range communities {
		if len(commNodes) == 0 { continue }
		
		summaryText, err := g.Summarizer.SummarizeCommunity(ctx, commNodes)
		if err != nil {
			fmt.Printf("Error summarizing community: %v\n", err)
			continue
		}
		
		name := fmt.Sprintf("Community %d", i+1)
		
		if summaryText != "" {
			if n, err := g.Summarizer.GenerateCommunityName(ctx, summaryText); err == nil && n != "" {
				name = n
			}
		}
		
		commUUID := g.UUIDGenerator()
		
		// Save Community Node
		commParams := map[string]interface{}{
			"uuid":           commUUID,
			"name":           name,
			"group_id":       groupID,
			"created_at":     now.Format(time.RFC3339),
			"summary":        summaryText,
			"name_embedding": nil,
		}
		
		if g.Embedder != nil {
			vec, err := g.Embedder.Embed(ctx, name)
			if err == nil {
				commParams["name_embedding"] = vec
			}
		}

		if _, err := g.Driver.ExecuteQuery(ctx, driver.SaveCommunityNodeQuery, commParams); err != nil {
			fmt.Printf("Error saving community node: %v\n", err)
			continue
		}
		
		// Save Membership Edges
		for _, n := range commNodes {
			edgeParams := map[string]interface{}{
				"uuid":        g.UUIDGenerator(),
				"source_uuid": commUUID,
				"target_uuid": n.UUID,
				"group_id":    groupID,
				"created_at":  now.Format(time.RFC3339),
			}
			if _, err := g.Driver.ExecuteQuery(ctx, driver.SaveCommunityEdgeQuery, edgeParams); err != nil {
				fmt.Printf("Error saving community edge: %v\n", err)
			}
		}
	}
	return nil
}

func (g *Graphiti) getGroupNodes(ctx context.Context, groupID string) ([]model.EntityNode, error) {
	res, err := g.Driver.ExecuteQuery(ctx, driver.GetGroupNodesQuery, map[string]interface{}{
		"group_id": groupID,
	})
	if err != nil {
		return nil, err
	}
	
	var nodes []model.EntityNode
	for _, rec := range res.Records {
		uuidVal, _ := rec.Get("uuid")
		name, _ := rec.Get("name")
		summaryVal, _ := rec.Get("summary")
		
		sumStr := ""
		if summaryVal != nil {
			sumStr = summaryVal.(string)
		}

		nodes = append(nodes, model.EntityNode{
			UUID:    uuidVal.(string),
			Name:    name.(string),
			Summary: sumStr,
			GroupID: groupID,
		})
	}
	// Debug:
	// fmt.Printf("DEBUG: DETECT Nodes: %d\n", len(nodes))
	// nodeMap := make(map[string]string)
	// for _, n := range nodes { 
	// 	fmt.Printf(" - Node: %s (Name: %s)\n", n.UUID, n.Name) 
	// 	nodeMap[n.UUID] = n.Name
	// }
	
	return nodes, nil
}

func (g *Graphiti) getGroupEdges(ctx context.Context, groupID string) ([]model.EntityEdge, error) {
	res, err := g.Driver.ExecuteQuery(ctx, driver.GetGroupEdgesQuery, map[string]interface{}{
		"group_id": groupID,
	})
	if err != nil {
		return nil, err
	}
	
	var edges []model.EntityEdge
	for _, rec := range res.Records {
		uuidVal, _ := rec.Get("uuid")
		source, _ := rec.Get("source_uuid")
		target, _ := rec.Get("target_uuid")
		fact, _ := rec.Get("fact")
		
		edges = append(edges, model.EntityEdge{
			UUID:       uuidVal.(string),
			SourceUUID: source.(string),
			TargetUUID: target.(string),
			Fact:       fact.(string),
			GroupID:    groupID,
		})
	}
	
	// Debug:
	// fmt.Printf("DEBUG: DETECT Edges: %d\n", len(edges))
	// for _, e := range edges { fmt.Printf(" - Edge: %s --[%s]-> %s\n", e.SourceUUID, e.Fact, e.TargetUUID) }
	
	return edges, nil
}

func (g *Graphiti) checkEdgeExists(ctx context.Context, source, target, name, fact string) (bool, error) {
	res, err := g.Driver.ExecuteQuery(ctx, driver.GetActiveEdgesQuery, map[string]interface{}{
		"source_uuid": source,
		"target_uuid": target,
		"name":        name,
	})
	if err != nil {
		return false, err
	}
	
	for _, rec := range res.Records {
		fVal, ok := rec.Get("fact")
		if ok && fVal != nil && fVal.(string) == fact {
			return true, nil
		}
	}
	return false, nil
}

func (g *Graphiti) getEdgesFromSource(ctx context.Context, source string) ([]model.EntityEdge, error) {
	res, err := g.Driver.ExecuteQuery(ctx, driver.GetActiveEdgesFromSourceQuery, map[string]interface{}{
		"source_uuid": source,
	})
	if err != nil {
		return nil, err
	}
	
	var edges []model.EntityEdge
	for _, rec := range res.Records {
		uuid, _ := rec.Get("uuid")
		fact, _ := rec.Get("fact")
		name, _ := rec.Get("name")
		target, _ := rec.Get("target_uuid")
		
		edges = append(edges, model.EntityEdge{
			UUID:         uuid.(string),
			SourceUUID:   source,
			TargetUUID:   target.(string),
			Name:         name.(string),
			Fact:         fact.(string),
		})
	}
	return edges, nil
}

func (g *Graphiti) invalidateEdge(ctx context.Context, uuid string, invalidAt time.Time) error {
	_, err := g.Driver.ExecuteQuery(ctx, driver.InvalidateEdgeQuery, map[string]interface{}{
		"uuid":       uuid,
		"invalid_at": invalidAt.Format(time.RFC3339),
	})
	return err
}

func (g *Graphiti) Search(ctx context.Context, groupID, query string) ([]model.EntityEdge, error) {
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
	// By default, text search on Edge Facts
	cypher := `
		MATCH (n:Entity)-[e:RELATES_TO]->(m:Entity)
		WHERE e.group_id = $group_id AND e.fact CONTAINS $query
		RETURN e.uuid AS uuid, 
		       n.uuid AS source_uuid, 
		       m.uuid AS target_uuid, 
		       e.name AS name,
		       e.fact AS fact, 
		       e.created_at AS created_at,
		       e.episodes AS episodes
		LIMIT 20
	`
	
	params := map[string]interface{}{
		"group_id": groupID,
		"query":    query,
	}

	if len(queryVector) > 0 {
		params["embedding"] = queryVector
		// Vector Search on Edge Fact Embeddings
		cypher = `
            MATCH (n:Entity)-[e:RELATES_TO]->(m:Entity)
            WHERE e.group_id = $group_id AND e.fact_embedding IS NOT NULL
            WITH e, n, m,
                 reduce(dot = 0.0, i in range(0, size(e.fact_embedding)-1) | dot + e.fact_embedding[i] * $embedding[i]) / 
                 (sqrt(reduce(s1 = 0.0, x in e.fact_embedding | s1 + x^2)) * sqrt(reduce(s2 = 0.0, y in $embedding | s2 + y^2))) AS score
            ORDER BY score DESC
            RETURN e.uuid AS uuid, 
                   n.uuid AS source_uuid, 
                   m.uuid AS target_uuid, 
                   e.name AS name,
                   e.fact AS fact, 
                   e.created_at AS created_at,
                   e.episodes AS episodes,
                   score
            LIMIT 20
        `
	}
	
	result, err := g.Driver.ExecuteQuery(ctx, cypher, params)
	if err != nil {
		return nil, fmt.Errorf("search failed: %w", err)
	}
	
	var edges []model.EntityEdge
	for _, record := range result.Records {
		uuidStr, _ := record.Get("uuid")
		sourceStr, _ := record.Get("source_uuid")
		targetStr, _ := record.Get("target_uuid")
		nameStr, _ := record.Get("name")
		factStr, _ := record.Get("fact")
		createdAtStr, _ := record.Get("created_at") // Parsing needed?
		episodesVal, _ := record.Get("episodes")

		edge := model.EntityEdge{
			UUID:       uuidStr.(string),
			SourceUUID: sourceStr.(string),
			TargetUUID: targetStr.(string), // Assuming these are UUID strings from graph
			GroupID:    groupID,
			Name:       nameStr.(string),
			Fact:       factStr.(string),
		}
		
		// Parse CreatedAt
		if tStr, ok := createdAtStr.(string); ok {
			if t, err := time.Parse(time.RFC3339, tStr); err == nil {
				edge.CreatedAt = t
			}
		}

		// Parse Episodes (could be list or string depending on stored format, assuming list of strings in Cypher or serialized)
		if epList, ok := episodesVal.([]interface{}); ok {
			for _, ep := range epList {
				if s, ok := ep.(string); ok {
					edge.Episodes = append(edge.Episodes, s)
				}
			}
		}

		edges = append(edges, edge)
	}

	// Reranking
	if g.Reranker != nil && len(edges) > 1 {
		facts := make([]string, len(edges))
		for i, e := range edges {
			facts[i] = e.Fact
		}

		indices, err := g.Reranker.Rank(ctx, query, facts)
		if err == nil && len(indices) > 0 {
			var reordered []model.EntityEdge
			seen := make(map[int]bool)
			// Reorder based on rank
			for _, idx := range indices {
				if idx >= 0 && idx < len(edges) && !seen[idx] {
					reordered = append(reordered, edges[idx])
					seen[idx] = true
				}
			}
			// Append remaining (if any were missed by reranker)
			for i := range edges {
				if !seen[i] {
					reordered = append(reordered, edges[i])
				}
			}
			edges = reordered
		}
	}

	return edges, nil
}

	// BulkAddEpisodes adds multiple episodes in a true batch process
func (g *Graphiti) BulkAddEpisodes(ctx context.Context, groupID string, episodes []model.EpisodeData) error {
	now := time.Now().UTC()

	// 1. Prepare Episodes and Context
	// Get shared context for batch
	prevEpisodes, _ := g.retrievePreviousEpisodes(ctx, groupID, "", 5)
	
	type extractionResult struct {
		index    int
		entities []model.ExtractedEntity
		err      error
	}

	// Determine concurrency limit
	limit := 2
	if g.Config != nil && g.Config.Concurrency.BulkIngest > 0 {
		limit = g.Config.Concurrency.BulkIngest
	}

	resultsChan := make(chan extractionResult, len(episodes))
	var wg sync.WaitGroup
	sem := make(chan struct{}, limit) // concurrency for LLM calls

	// 2. Concurrent Extraction
	for i, ep := range episodes {
		wg.Add(1)
		sem <- struct{}{}
		go func(idx int, e model.EpisodeData) {
			defer wg.Done()
			defer func() { <-sem }()
			
			// Extract Entities
			entities, err := g.Extractor.ExtractNodes(ctx, e.Content, e.Schema, prevEpisodes) // Use shared context
			resultsChan <- extractionResult{index: idx, entities: entities, err: err}
		}(i, ep)
	}
	wg.Wait()
	close(resultsChan)

	// Collect results mapped by index
	episodeExtracted := make(map[int][]model.ExtractedEntity)
	var errs []string

	for res := range resultsChan {
		if res.err != nil {
			errs = append(errs, fmt.Sprintf("ep[%d]: %v", res.index, res.err))
			continue
		}
		episodeExtracted[res.index] = res.entities
	}

	if len(errs) > 0 {
		return fmt.Errorf("bulk extraction errors: %v", errs)
	}

	// 3. Global Deduplication (Batch + DB)
	// Flatten all extracted entities to nodes
	var allTempNodes []model.EntityNode
	for _, entities := range episodeExtracted {
		nodes := g.convertToEntityNodes(entities, groupID, now)
		allTempNodes = append(allTempNodes, nodes...)
	}

	// First, fetch existing entities from DB
	existingNodes, err := g.getGroupNodes(ctx, groupID)
	if err != nil {
		return fmt.Errorf("failed to fetch existing nodes: %w", err)
	}
	
	// Dedupe within batch (ByName)
	uniqueBatchNodes := make(map[string]model.EntityNode)
	for _, n := range allTempNodes {
		if _, exists := uniqueBatchNodes[n.Name]; !exists {
			uniqueBatchNodes[n.Name] = n
		}
	}
	
	var batchNodes []model.EntityNode
	for _, n := range uniqueBatchNodes {
		batchNodes = append(batchNodes, n)
	}

	// Resolve against DB
	finalNodes := g.resolveDuplicates(ctx, batchNodes, existingNodes)

	// 4. Save Nodes
	// Build a map of Name -> FinalNode for quick lookup later
	finalNodeMap := make(map[string]model.EntityNode)
	for _, n := range finalNodes {
		if err := g.saveEntity(ctx, n); err != nil {
			return fmt.Errorf("failed to save node %s: %w", n.Name, err)
		}
		finalNodeMap[n.Name] = n
	}
	
	// Also map existing nodes in case resolution picked one of them and it wasn't in finalNodes (resolveDuplicates returns mixed?)
	// resolveDuplicates returns the list of nodes we passed in, but with UUIDs updated to match existing if found.
	// So `finalNodes` contains the resolved state of `batchNodes`.
	// Correct.
	
	// 5. Run AddEpisode Concurrently (using pre-resolved nodes)
	
	sem2 := make(chan struct{}, limit)
	errChan2 := make(chan error, len(episodes))
	
	for i, ep := range episodes {
		// Reconstruct the node list for this episode using the resolved map
		extracted := episodeExtracted[i]
		var episodeResolvedNodes []model.EntityNode
		for _, ex := range extracted {
			if resolved, ok := finalNodeMap[ex.Name]; ok {
				episodeResolvedNodes = append(episodeResolvedNodes, resolved)
			}
		}

		wg.Add(1)
		sem2 <- struct{}{}
		go func(e model.EpisodeData, nodes []model.EntityNode) {
			defer wg.Done()
			defer func() { <-sem2 }()
			
			// Call internal method with pre-resolved nodes to skip double extraction
			if err := g.addEpisodeInternal(ctx, groupID, "message", e.Content, e.Saga, e.Schema, nodes); err != nil {
				errChan2 <- fmt.Errorf("failed to add episode: %w", err)
			}
		}(ep, episodeResolvedNodes)
	}
	wg.Wait()
	close(errChan2)
	
	if len(errChan2) > 0 {
		var errMsgs []string
		for err := range errChan2 {
			errMsgs = append(errMsgs, err.Error())
		}
		return fmt.Errorf("bulk add (phase 2) errors: %v", errMsgs)
	}

	return nil
}

// BulkSearch executes multiple search queries concurrently
func (g *Graphiti) BulkSearch(ctx context.Context, groupID string, queries []model.BulkSearchQuery) (map[string][]model.EntityEdge, error) {
	limit := 5
	if g.Config != nil && g.Config.Concurrency.BulkSearch > 0 {
		limit = g.Config.Concurrency.BulkSearch
	}
	sem := make(chan struct{}, limit)
	var wg sync.WaitGroup
	results := make(map[string][]model.EntityEdge)
	var mu sync.Mutex
	errChan := make(chan error, len(queries))

	for _, q := range queries {
		wg.Add(1)
		sem <- struct{}{}
		go func(query model.BulkSearchQuery) {
			defer wg.Done()
			defer func() { <-sem }()
			
			res, err := g.Search(ctx, groupID, query.Query)
			if err != nil {
				errChan <- err
				return
			}
			
			mu.Lock()
			results[query.QueryID] = res
			mu.Unlock()
		}(q)
	}
	
	wg.Wait()
	close(errChan)
	
	if len(errChan) > 0 {
		return nil, fmt.Errorf("bulk search encountered %d errors", len(errChan))
	}
	
	return results, nil
}



	// Helper: Save Entity Node
	func (g *Graphiti) saveEntity(ctx context.Context, node model.EntityNode) error {
	var attrsJSON string
	if len(node.Attributes) > 0 {
		if b, err := json.Marshal(node.Attributes); err == nil {
			attrsJSON = string(b)
		} else {
			attrsJSON = "{}" // Default on error
		}
	} else {
		attrsJSON = "{}"
	}

	params := map[string]interface{}{
		"uuid":           node.UUID,
		"name":           node.Name,
		"group_id":       node.GroupID,
		"created_at":     node.CreatedAt.Format(time.RFC3339),
		"summary":        node.Summary, 
		"name_embedding": nil, 
		"attributes":     attrsJSON,
		"labels":         []string{},
	}
	
	if g.Embedder != nil {
		emb, err := g.Embedder.Embed(ctx, node.Name)
		if err == nil {
			params["name_embedding"] = emb
		}
	}

	_, err := g.Driver.ExecuteQuery(ctx, driver.SaveEntityNodeQuery, params)
	return err
}

func (g *Graphiti) SearchEdges(ctx context.Context, groupID, query string) ([]model.EntityEdge, error) {
	cypher := `
		MATCH (s:Entity)-[e:RELATES_TO]->(t:Entity)
		WHERE e.group_id = $group_id AND e.fact CONTAINS $query
		RETURN e.uuid as uuid, e.source_uuid as source, e.target_uuid as target, e.name as name, e.fact as fact
		LIMIT 10
	`
	
	res, err := g.Driver.ExecuteQuery(ctx, cypher, map[string]interface{}{
		"group_id": groupID,
		"query":    query,
	})
	if err != nil {
		return nil, err
	}
	
	var edges []model.EntityEdge
	for _, rec := range res.Records {
		uuid, _ := rec.Get("uuid")
		source, _ := rec.Get("source")
		target, _ := rec.Get("target")
		name, _ := rec.Get("name")
		fact, _ := rec.Get("fact")
		
		edges = append(edges, model.EntityEdge{
			UUID:           uuid.(string),
			SourceUUID:     source.(string),
			TargetUUID:     target.(string),
			Name:           name.(string),
			Fact:           fact.(string),
			GroupID:        groupID,
		})
	}
	return edges, nil
}

// ---------------- Saga Handle Methods ----------------

func (g *Graphiti) handleSaga(ctx context.Context, sagaName, groupID, episodeUUID string, now time.Time) error {
	sagaNode, err := g.getOrCreateSaga(ctx, sagaName, groupID, now)
	if err != nil {
		return err
	}

	prevUUID, err := g.findPreviousEpisodeInSaga(ctx, sagaNode.UUID, episodeUUID)
	if err != nil {
		return err
	}

	if prevUUID != "" {
		if err := g.linkNextEpisode(ctx, prevUUID, episodeUUID, groupID, now); err != nil {
			return err
		}
	}

	return g.linkSagaHasEpisode(ctx, sagaNode.UUID, episodeUUID, groupID, now)
}

func (g *Graphiti) getOrCreateSaga(ctx context.Context, name, groupID string, now time.Time) (*model.SagaNode, error) {
	res, err := g.Driver.ExecuteQuery(ctx, driver.GetSagaByNameQuery, map[string]interface{}{
		"name":     name,
		"group_id": groupID,
	})
	if err != nil {
		return nil, err
	}

	if len(res.Records) > 0 {
		rec := res.Records[0]
		uuidVal, _ := rec.Get("uuid")
		return &model.SagaNode{
			UUID:      uuidVal.(string),
			Name:      name,
			GroupID:   groupID,
		}, nil
	}

	newNode := &model.SagaNode{
		UUID:      g.UUIDGenerator(),
		Name:      name,
		GroupID:   groupID,
		CreatedAt: now,
	}

	params := map[string]interface{}{
		"uuid":       newNode.UUID,
		"name":       newNode.Name,
		"group_id":   newNode.GroupID,
		"created_at": newNode.CreatedAt.Format(time.RFC3339),
	}
	
	if _, err := g.Driver.ExecuteQuery(ctx, driver.SaveSagaNodeQuery, params); err != nil {
		return nil, err
	}
	
	return newNode, nil
}

func (g *Graphiti) findPreviousEpisodeInSaga(ctx context.Context, sagaUUID, currentEpisodeUUID string) (string, error) {
	res, err := g.Driver.ExecuteQuery(ctx, driver.GetPreviousEpisodeInSagaQuery, map[string]interface{}{
		"saga_uuid":            sagaUUID,
		"current_episode_uuid": currentEpisodeUUID,
	})
	if err != nil {
		return "", err
	}

	if len(res.Records) > 0 {
		uuidVal, _ := res.Records[0].Get("uuid")
		return uuidVal.(string), nil
	}
	return "", nil
}


