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

func TestSummarizationIntegration(t *testing.T) {
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
		}
	}
	
	if apiKey := os.Getenv("LLM_API_KEY"); apiKey != "" {
		cfg.LLM.APIKey = apiKey
	}
	if dbPass := os.Getenv("MEMGRAPH_PASSWORD"); dbPass != "" {
		cfg.Memgraph.Password = dbPass
	}

	d, err := driver.NewMemgraphDriver(cfg.Memgraph.URI, cfg.Memgraph.User, cfg.Memgraph.Password)
	require.NoError(t, err)
	defer d.Close(context.Background())

	ctx := context.Background()
	llmClient, embedder, err := llm.NewClient(ctx, cfg.LLM)
	require.NoError(t, err)

	g := core.NewGraphiti(d, llmClient, embedder, nil, cfg)
	
	groupID := fmt.Sprintf("summary-group-%s", uuid.New().String())
	t.Logf("DEBUG: Test GroupID: %s", groupID)

	defer func() {
		_, _ = d.ExecuteQuery(context.Background(), `MATCH (n {group_id: $gid}) DETACH DELETE n`, map[string]interface{}{"gid": groupID})
		t.Logf("Cleaned up test group: %s", groupID)
	}()

	// 2. Add Episode with facts
	// Use standard entities: Bob (Person), Google (Organization).
	content := "Bob is a software engineer working at Google."
	err = g.AddEpisode(ctx, groupID, "Ep1", content, "", "")
	require.NoError(t, err)

	// 3. Verify Summaries
	// Fetch Bob
	res, err := d.ExecuteQuery(ctx, `MATCH (n:Entity {group_id: $gid, name: 'Bob'}) RETURN n.summary as summary`, map[string]interface{}{"gid": groupID})
	require.NoError(t, err)
	if len(res.Records) == 0 {
		t.Fatal("Bob not found after Ep1")
	}
	
	summaryVal, _ := res.Records[0].Get("summary")
	summaryStr, _ := summaryVal.(string)
	
	t.Logf("Bob's Summary Ep1: %s", summaryStr)
	
	// 4. Add more info to trigger update
	content2 := "Bob lives in Paris."
	err = g.AddEpisode(ctx, groupID, "Ep2", content2, "", "")
	require.NoError(t, err)
	
	res2, err := d.ExecuteQuery(ctx, `MATCH (n:Entity {group_id: $gid, name: 'Bob'}) RETURN n.summary as summary`, map[string]interface{}{"gid": groupID})
	require.NoError(t, err)
	summaryVal2, _ := res2.Records[0].Get("summary")
	summaryStr2, _ := summaryVal2.(string)
	
	t.Logf("Bob's Updated Summary: %s", summaryStr2)
	
	// Should ideally be different or longer than before
	assert.NotEqual(t, summaryStr, summaryStr2, "Summary should have updated")
}
