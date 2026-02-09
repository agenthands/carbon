//go:build integration

package integration

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/agenthands/carbon/internal/config"
	"github.com/agenthands/carbon/internal/core"
	"github.com/agenthands/carbon/internal/driver"
	"github.com/agenthands/carbon/internal/llm"
	"github.com/joho/godotenv"
)

func TestCommunityDetection(t *testing.T) {
	// 1. Setup
	_ = godotenv.Load("../../.env")
	cfgPath := "../../config/config.toml"
	cfg, err := config.Load(cfgPath)
	if err != nil {
		// Minimal config fallback
		cfg = &config.Config{
			LLM: config.LLMConfig{
				Provider: "ollama",
				Model:    "gpt-oss:latest",
				BaseURL:  "http://localhost:11434",
			},
			Memgraph: config.MemgraphConfig{
				URI: "bolt://localhost:7687",
			},
			Extraction: config.ExtractionPrompts{Nodes: "%s", Edges: "%s"},
		}
	}
	// Verify Summarize Communities prompt is set, else default
	if cfg.Summary.Communities == "" {
		cfg.Summary.Communities = "Summarize the following community of entities:\n%s"
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

	groupID := fmt.Sprintf("community-group-%s", uuid.New().String())
	t.Logf("DEBUG: Test GroupID: %s", groupID)

	// 2. Create Cluster 1: The "Tech" Crowd
	// Alice, Bob, Charlie all know each other.
	err = g.AddEpisode(ctx, groupID, "Ep1", "Alice is a software engineer. She works with Bob.", "", "")
	require.NoError(t, err)
	err = g.AddEpisode(ctx, groupID, "Ep2", "Bob is a manager. He manages Charlie.", "", "")
	require.NoError(t, err)
	err = g.AddEpisode(ctx, groupID, "Ep3", "Charlie is a designer. He often lunches with Alice.", "", "")
	require.NoError(t, err)

	// 3. Create Cluster 2: The "Artist" Crowd (Isolated from Tech)
	// Dave, Eve
	err = g.AddEpisode(ctx, groupID, "Ep4", "Dave is a painter. He shares a studio with Eve.", "", "")
	require.NoError(t, err)

	// Allow DB to settle (optional)
	time.Sleep(1 * time.Second)

	// 4. Run Detection
	err = g.DetectAndSummarizeCommunities(ctx, groupID)
	require.NoError(t, err)

	// 5. Verify Results

	// Fetch all Community Nodes
	commRes, err := d.ExecuteQuery(ctx, `MATCH (c:Community {group_id: $gid}) RETURN c.uuid as uuid, c.name as name, c.summary as summary`, map[string]interface{}{"gid": groupID})
	require.NoError(t, err)

	communities := commRes.Records
	t.Logf("Found %d communities", len(communities))

	// We expect 2 communities: [Alice, Bob, Charlie] and [Dave, Eve]
	require.True(t, len(communities) >= 2, "Should find at least 2 communities")

	for _, c := range communities {
		uuidVal, _ := c.Get("uuid")
		name, _ := c.Get("name")
		summary, _ := c.Get("summary")

		t.Logf("Community: %v, Name: %v, Summary: %v", uuidVal, name, summary)
		assert.NotEmpty(t, summary, "Community should have a summary")

		// Check members
		memRes, err := d.ExecuteQuery(ctx, `MATCH (c:Community {uuid: $cuuid})-[r:HAS_MEMBER]->(e:Entity) RETURN e.name as name`, map[string]interface{}{"cuuid": uuidVal})
		require.NoError(t, err)

		var members []string
		for _, m := range memRes.Records {
			n, _ := m.Get("name")
			members = append(members, n.(string))
		}
		t.Logf("  Members: %v", members)

		require.NotEmpty(t, members, "Community should have members")
	}
}
