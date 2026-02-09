//go:build integration

package integration

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"github.com/agenthands/carbon/internal/config"
	"github.com/agenthands/carbon/internal/core"
	"github.com/agenthands/carbon/internal/core/model"
	"github.com/agenthands/carbon/internal/driver"
	"github.com/agenthands/carbon/internal/llm"
	"github.com/joho/godotenv"
)

func TestBulkOperations(t *testing.T) {
	// 1. Setup
	_ = godotenv.Load("../../.env")
	cfgPath := "../../config/config.toml"
	cfg, err := config.Load(cfgPath)
	if err != nil {
		t.Logf("Config not found, using default: %v", err)
		cfg = &config.Config{
			LLM: config.LLMConfig{
				Provider: "ollama",
				Model:    "gpt-oss:latest",
				BaseURL:  "http://localhost:11434",
			},
			Memgraph: config.MemgraphConfig{
				URI: "bolt://localhost:7687",
			},
			Concurrency: config.ConcurrencyConfig{
				BulkIngest: 2,
				BulkSearch: 2,
			},
		}
	}

	if apiKey := os.Getenv("LLM_API_KEY"); apiKey != "" {
		cfg.LLM.APIKey = apiKey
	}
	if dbPass := os.Getenv("MEMGRAPH_PASSWORD"); dbPass != "" {
		cfg.Memgraph.Password = dbPass
	}

	// Override concurrency for test stability
	cfg.Concurrency.BulkIngest = 2
	cfg.Concurrency.BulkSearch = 2

	d, err := driver.NewMemgraphDriver(cfg.Memgraph.URI, cfg.Memgraph.User, cfg.Memgraph.Password)
	require.NoError(t, err)
	defer d.Close(context.Background())

	ctx := context.Background()
	llmClient, embedder, err := llm.NewClient(ctx, cfg.LLM)
	require.NoError(t, err)

	g := core.NewGraphiti(d, llmClient, embedder, nil, cfg)

	groupID := fmt.Sprintf("bulk-group-%s", uuid.New().String())
	t.Logf("DEBUG: Test GroupID: %s", groupID)

	// Signup cleanup
	defer func() {
		// Clean up test data
		_, _ = d.ExecuteQuery(context.Background(), `MATCH (n {group_id: $gid}) DETACH DELETE n`, map[string]interface{}{"gid": groupID})
		t.Logf("Cleaned up test group: %s", groupID)
	}()

	// 2. Prepare Bulk Data
	episodes := []model.EpisodeData{
		{Content: "Alpha is a software engineer working on project Z.", Saga: "Saga1"},
		{Content: "Beta manages the team for project Z.", Saga: "Saga1"},
		{Content: "Gamma is Alpha's mentor.", Saga: "Saga1"},
		{Content: "Delta lives in New York.", Saga: "Saga2"},
		{Content: "Epsilon is a musician who is friends with Delta.", Saga: "Saga2"},
	}

	// 3. Execute Bulk Add
	startTime := time.Now()
	err = g.BulkAddEpisodes(ctx, groupID, episodes)
	require.NoError(t, err)
	duration := time.Since(startTime)
	t.Logf("Bulk Add took %v for %d episodes", duration, len(episodes))

	// 4. Verify Data Ingested
	// Count episodes
	res, err := d.ExecuteQuery(ctx, `MATCH (e:Episodic {group_id: $gid}) RETURN count(e) as count`, map[string]interface{}{"gid": groupID})
	require.NoError(t, err)
	count, _ := res.Records[0].Get("count")
	require.Equal(t, int64(5), count.(int64))

	// Debug: Check entities and embeddings
	// entRes, err := d.ExecuteQuery(ctx, `MATCH (n:Entity {group_id: $gid}) RETURN n.name as name, n.name_embedding as embedding`, map[string]interface{}{"gid": groupID})
	// require.NoError(t, err)
	// t.Logf("Found %d entities", len(entRes.Records))

	// 5. Execute Bulk Search
	// Note: Since local embedding model might not be available or configured (returning nil embeddings),
	// we use queries that match entity names to verify the BulkSearch mechanism via text fallback.
	queries := []model.BulkSearchQuery{
		{QueryID: "q1", Query: "Alpha"},
		{QueryID: "q2", Query: "Epsilon"},
		{QueryID: "q3", Query: "Gamma"},
	}

	startTime = time.Now()
	searchResults, err := g.BulkSearch(ctx, groupID, queries)
	require.NoError(t, err)
	duration = time.Since(startTime)
	t.Logf("Bulk Search took %v for %d queries", duration, len(queries))

	// 6. Verify Search Results
	// Results are now []model.EntityEdge
	require.Len(t, searchResults, 3)

	t.Logf("Result Q1: %v", searchResults["q1"])
	require.NotEmpty(t, searchResults["q1"], "Q1 should return results")
	require.IsType(t, []model.EntityEdge{}, searchResults["q1"])
	require.NotEmpty(t, searchResults["q1"][0].UUID)
	require.NotEmpty(t, searchResults["q1"][0].SourceUUID)
	require.NotEmpty(t, searchResults["q1"][0].TargetUUID)

	t.Logf("Result Q2: %v", searchResults["q2"])
	require.NotEmpty(t, searchResults["q2"], "Q2 should return results")

	t.Logf("Result Q3: %v", searchResults["q3"])
	require.NotEmpty(t, searchResults["q3"], "Q3 should return results")
}
