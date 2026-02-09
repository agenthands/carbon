//go:build integration

package integration

import (
	"context"
	"fmt"
	"os"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/agenthands/carbon/internal/config"
	"github.com/agenthands/carbon/internal/core"
	"github.com/agenthands/carbon/internal/driver"
	"github.com/agenthands/carbon/internal/llm"
	"github.com/joho/godotenv"
)

func TestFullFlow(t *testing.T) {
	// Load environment if present
	_ = godotenv.Load("../../.env") // Try root .env

	// Load Config
	cfgPath := "../../config/config.toml"
	cfg, err := config.Load(cfgPath)
	if err != nil {
		// Create default config if file missing (or rely on env fallback if we kept it, but user wants config first)
		// For integration test, we assume config file exists or we manually construct config.
		// Let's manually construct minimal defaults if load fails
		cfg = &config.Config{
			LLM: config.LLMConfig{
				Provider: "ollama",
				Model:    "gpt-oss:latest",
				BaseURL:  "http://localhost:11434",
			},
			Memgraph: config.MemgraphConfig{
				URI: "bolt://localhost:7687",
			},
			Extraction: config.ExtractionPrompts{
				Nodes: "Extract entities regarding the content: %s\nTypes: %s",
				Edges: "Extract edges from: %s",
			},
			Deduplication: config.DeduplicationPrompts{
				Nodes: "Deduplicate: %s vs %s",
			},
			Summary: config.SummaryPrompts{
				Nodes: "Summarize: %s with new info: %s",
			},
		}
	}

	// Override Secrets from Env
	if apiKey := os.Getenv("LLM_API_KEY"); apiKey != "" {
		cfg.LLM.APIKey = apiKey
	}
	if dbPass := os.Getenv("MEMGRAPH_PASSWORD"); dbPass != "" {
		cfg.Memgraph.Password = dbPass
	}

	// Connect Driver
	if cfg.Memgraph.URI == "" {
		cfg.Memgraph.URI = "bolt://localhost:7687"
	}
	d, err := driver.NewMemgraphDriver(cfg.Memgraph.URI, cfg.Memgraph.User, cfg.Memgraph.Password)
	require.NoError(t, err)
	defer d.Close(context.Background())

	ctx := context.Background()
	llmClient, embedder, err := llm.NewClient(ctx, cfg.LLM)
	require.NoError(t, err)

	// Initialize Graphiti with Reranker
	reranker := llm.NewSimpleLLMReranker(llmClient)
	g := core.NewGraphiti(d, llmClient, embedder, reranker, cfg)

	// Build Indices
	err = g.BuildIndices(ctx)
	require.NoError(t, err)

	// Unique Group ID for this test run
	groupID := fmt.Sprintf("test-group-%s", uuid.New().String())

	// Step 1: Add Episode 1
	// "Alice is a software engineer living in Seattle."
	episode1 := "Alice is a software engineer living in Seattle."
	err = g.AddEpisode(ctx, groupID, "Ep1", episode1, "TestSaga", "")
	require.NoError(t, err)

	// Step 2: Add Episode 2
	// "Alice met Bob, a data scientist from Portland."
	episode2 := "Alice met Bob, a data scientist from Portland."
	err = g.AddEpisode(ctx, groupID, "Ep2", episode2, "TestSaga", "")
	require.NoError(t, err)

	// Step 3: Search
	// Query: "Who is Alice?"
	results, err := g.Search(ctx, groupID, "Alice")
	require.NoError(t, err)
	assert.NotEmpty(t, results)

	t.Logf("Search Results: %v", results)

	// Verify content if possible (LLM dependent so loose check)
	foundAlice := false
	for _, r := range results {
		if r.Fact != "" { // Just check we got something
			foundAlice = true
		}
	}
	assert.True(t, foundAlice)

	// Step 4: Verify Graph Structure directly (optional)
	cypher := `MATCH (n:Entity {group_id: $gid}) RETURN n.name as name`
	res, err := d.ExecuteQuery(ctx, cypher, map[string]interface{}{"gid": groupID})
	require.NoError(t, err)

	var entityNames []string
	if len(res.Records) > 0 {
		for _, r := range res.Records {
			name, _ := r.Get("name")
			entityNames = append(entityNames, name.(string))
		}
		t.Logf("Entity Count: %d", len(entityNames))
		t.Logf("Entities Found: %v", entityNames)
		assert.True(t, len(entityNames) > 0)
	}

	// Step 5: Verify Saga Structure
	sagaCypher := `MATCH (s:Saga {group_id: $gid, name: $sname}) RETURN count(s) as count`
	sagaRes, err := d.ExecuteQuery(ctx, sagaCypher, map[string]interface{}{"gid": groupID, "sname": "TestSaga"})
	require.NoError(t, err)
	if len(sagaRes.Records) > 0 {
		count, _ := sagaRes.Records[0].Get("count")
		t.Logf("Saga Count: %v", count)
		assert.Equal(t, int64(1), count.(int64))
	}

	// Verify Saga has episodes
	hasEpCypher := `MATCH (s:Saga {name: $sname})-[:HAS_EPISODE]->(e:Episodic) WHERE e.group_id = $gid RETURN count(e) as count`
	hasEpRes, err := d.ExecuteQuery(ctx, hasEpCypher, map[string]interface{}{"gid": groupID, "sname": "TestSaga"})
	require.NoError(t, err)
	if len(hasEpRes.Records) > 0 {
		count, _ := hasEpRes.Records[0].Get("count")
		t.Logf("Saga Episodes: %v", count)
		assert.Equal(t, int64(2), count.(int64))
	}

	// Verify Next Episode link
	nextEpCypher := `MATCH (e1:Episodic)-[:NEXT_EPISODE]->(e2:Episodic) WHERE e1.group_id = $gid RETURN count(*) as count`
	nextEpRes, err := d.ExecuteQuery(ctx, nextEpCypher, map[string]interface{}{"gid": groupID})
	require.NoError(t, err)
	if len(nextEpRes.Records) > 0 {
		count, _ := nextEpRes.Records[0].Get("count")
		t.Logf("Next Episode Links: %v", count)
		assert.Equal(t, int64(1), count.(int64))
	}

	// Cleanup
	cleanupCypher := `MATCH (n {group_id: $gid}) DETACH DELETE n`
	_, _ = d.ExecuteQuery(ctx, cleanupCypher, map[string]interface{}{"gid": groupID})
}
