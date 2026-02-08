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

	// Memgraph Config
	uri := os.Getenv("MEMGRAPH_URI")
	if uri == "" {
		t.Skip("Skipping integration test: MEMGRAPH_URI not set")
	}
	user := os.Getenv("MEMGRAPH_USER")
	pwd := os.Getenv("MEMGRAPH_PASSWORD")

	// LLM Config
	provider := os.Getenv("LLM_PROVIDER")
	model := os.Getenv("LLM_MODEL")
	baseURL := os.Getenv("OLLAMA_BASE_URL") // Fallback
	if provider == "" {
		provider = "ollama"
	}
	if model == "" {
		model = "gpt-oss:latest"
	}
	if baseURL == "" {
		baseURL = "http://localhost:11434"
	}

	// Connect Driver
	d, err := driver.NewMemgraphDriver(uri, user, pwd)
	require.NoError(t, err)
	defer d.Close(context.Background())

	// Initialize LLM
	llmCfg := config.LLMConfig{
		Provider: provider,
		Model:    model,
		BaseURL:  baseURL,
		APIKey:   os.Getenv("LLM_API_KEY"),
	}
	
	ctx := context.Background()
	llmClient, embedder, err := llm.NewClient(ctx, llmCfg)
	require.NoError(t, err)

	// Load default prompts from file or construct manually
	// Assuming prompts.toml is in config/config.toml relative to root
	cfgPath := "../../config/config.toml"
	cfg, err := config.Load(cfgPath)
	if err != nil {
		// Create default config if file missing
		cfg = &config.Config{
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
	// Ensure LLM config matches environment if loaded from file
	cfg.LLM = llmCfg

	// Initialize Graphiti
	g := core.NewGraphiti(d, llmClient, embedder, cfg)

	// Build Indices
	err = g.BuildIndices(ctx)
	require.NoError(t, err)

	// Unique Group ID for this test run
	groupID := fmt.Sprintf("test-group-%s", uuid.New().String())
	
	// Step 1: Add Episode 1
	// "Alice is a software engineer living in Seattle."
	episode1 := "Alice is a software engineer living in Seattle."
	err = g.AddEpisode(ctx, groupID, "Ep1", episode1)
	require.NoError(t, err)
	
	// Step 2: Add Episode 2
	// "Alice met Bob, a data scientist from Portland."
	episode2 := "Alice met Bob, a data scientist from Portland."
	err = g.AddEpisode(ctx, groupID, "Ep2", episode2)
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
		if len(r) > 0 { // Just check we got something
			foundAlice = true
		}
	}
	assert.True(t, foundAlice)
	
	// Step 4: Verify Graph Structure directly (optional)
	cypher := `MATCH (n:Entity {group_id: $gid}) RETURN count(n) as count`
	res, err := d.ExecuteQuery(ctx, cypher, map[string]interface{}{"gid": groupID})
	require.NoError(t, err)
	if len(res.Records) > 0 {
		count, _ := res.Records[0].Get("count")
		t.Logf("Entity Count: %v", count)
		assert.True(t, count.(int64) > 0)
	}

	// Cleanup
	cleanupCypher := `MATCH (n {group_id: $gid}) DETACH DELETE n`
	_, _ = d.ExecuteQuery(ctx, cleanupCypher, map[string]interface{}{"gid": groupID})
}
