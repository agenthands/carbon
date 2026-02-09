//go:build integration

package integration

import (
	"context"
	"fmt"
	"os"
	"strings"
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

func TestTemporalContradiction(t *testing.T) {
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
	// Override specific prompts for contradiction scenario if needed
	// The LLM needs to be smart enough to detect "lives in" vs "moved to" -> now "lives in SF"

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

	groupID := fmt.Sprintf("temporal-group-%s", uuid.New().String())
	t.Logf("DEBUG: Test GroupID: %s", groupID)

	// 2. Fact 1: Alice lives in Seattle
	err = g.AddEpisode(ctx, groupID, "Ep1", "Alice is a software engineer who lives in Seattle.", "", "")
	require.NoError(t, err)

	// Allow some time or ensure distinct timestamps (though Graphiti uses time.Now())
	time.Sleep(1 * time.Second)

	// 3. Fact 2: Alice moved to San Francisco
	// This should invalidate the previous "lives in" edge
	err = g.AddEpisode(ctx, groupID, "Ep2", "Alice moved to San Francisco and lives there now.", "", "")
	require.NoError(t, err)

	// 4. Verify

	// Debug: Check entities
	entityRes, _ := d.ExecuteQuery(ctx, `MATCH (n:Entity {group_id: $gid}) RETURN n.name as name`, map[string]interface{}{"gid": groupID})
	var names []string
	for _, r := range entityRes.Records {
		n, _ := r.Get("name")
		names = append(names, n.(string))
	}
	t.Logf("Entities Found: %v", names)

	// Helper to find edges
	searchEdges := func(factQuery string) []map[string]interface{} {
		// Remove :Entity constraint from query as per debug discovery
		cypher := `
			MATCH (s)-[e:RELATES_TO]->(t)
			WHERE e.group_id = $gid
			RETURN e.uuid as uuid, e.fact as fact, e.invalid_at as invalid_at
		`
		res, err := d.ExecuteQuery(ctx, cypher, map[string]interface{}{"gid": groupID})
		require.NoError(t, err)

		var edges []map[string]interface{}
		for _, r := range res.Records {
			uuid, _ := r.Get("uuid")
			fact, _ := r.Get("fact")
			inv, _ := r.Get("invalid_at")

			// simple contains check
			sFact := ""
			if fact != nil {
				sFact = fact.(string)
			}

			if len(factQuery) > 0 && !contains(sFact, factQuery) {
				continue
			}

			edges = append(edges, map[string]interface{}{
				"uuid":       uuid,
				"fact":       fact,
				"invalid_at": inv,
			})
		}
		return edges
	}

	seattleEdges := searchEdges("Seattle")
	sfEdges := searchEdges("San Francisco")

	t.Logf("Seattle Edges: %v", seattleEdges)
	t.Logf("SF Edges: %v", sfEdges)

	require.NotEmpty(t, seattleEdges, "Should have edge for Seattle")
	require.NotEmpty(t, sfEdges, "Should have edge for San Francisco")

	// Check Seattle edge is invalidated
	seagleEdge := seattleEdges[0]
	invalidAt := seagleEdge["invalid_at"]

	// Check if invalidAt is set (Memgraph returns string or nil)
	isInvalid := false
	if invalidAt != nil {
		if s, ok := invalidAt.(string); ok && s != "" {
			isInvalid = true
		}
	}

	// Temporarily assert FALSE to verify test runs, then change to TRUE after impl
	// Actually, stick to TDD: Assert TRUE, expect FAIL (but fail on assertion, not setup)
	assert.True(t, isInvalid, "Seattle edge should be invalidated because she moved")

	// Check SF edge is valid
	sfEdge := sfEdges[0]
	sfInvalidAt := sfEdge["invalid_at"]
	isSFInvalid := false
	if sfInvalidAt != nil {
		if s, ok := sfInvalidAt.(string); ok && s != "" {
			isSFInvalid = true
		}
	}
	assert.False(t, isSFInvalid, "San Francisco edge should be valid")
}

func contains(s, substr string) bool {
	return strings.Contains(s, substr)
}
