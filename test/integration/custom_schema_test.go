//go:build integration

package integration

import (
	"context"
	"encoding/json"
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

func TestCustomSchemaAttributes(t *testing.T) {
	// 1. Setup
	_ = godotenv.Load("../../.env")
	cfgPath := "../../config/config.toml"
	cfg, err := config.Load(cfgPath)
	if err != nil {
		t.Logf("Config not found, using minimal config: %v", err)
		cfg = &config.Config{
			LLM: config.LLMConfig{
				Provider: "ollama",
				Model:    "gpt-oss:latest",
				BaseURL:  "http://localhost:11434",
			},
			Memgraph: config.MemgraphConfig{
				URI: "bolt://localhost:7687",
			},
			// Default prompt with attribute support
			Extraction: config.ExtractionPrompts{
				Nodes: `You are an expert Data Extractor. Your task is to identify entities mentioned in the text and extract their properties.

<SCHEMA>
%s
</SCHEMA>

<TEXT>
%s
</TEXT>

Instructions:
1. Identify all entities that match the types defined in the SCHEMA.
2. For each entity, extract its Name.
3. Extract all relevant attributes defined in the SCHEMA for that entity type. If an attribute is missing, omit it or set to null.
4. Return the result as a single JSON object with the key "extracted_entities".

Example JSON:
{
  "extracted_entities": [
    {
      "name": "John Doe", 
      "entity_type_id": 1,
      "attributes": {
        "age": 30,
        "occupation": "Engineer",
        "location": "Boston" 
      }
    }
  ]
}`,
				Edges: "%s", // Minimal edge prompt for this test
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
	
	groupID := fmt.Sprintf("schema-group-%s", uuid.New().String())
	t.Logf("DEBUG: Test GroupID: %s", groupID)

	// 2. Define Custom Schema - Structured
	schema := `
	- Entity Type: Person (ID: 1)
	  Description: A human being.
	  Attributes:
	    - age (integer)
	    - occupation (string)
	    - location (string)

	- Entity Type: Product (ID: 2)
	  Description: An item for sale.
	  Attributes:
	    - price (float)
	    - color (string)
	`

	// 3. Add Episode with custom schema
	content := "Alice is a 28-year-old Data Scientist living in New York. She bought a red Bicycle for $500."

	// Debug: Call ExtractNodes directly to see LLM output
	extracted, err := g.Extractor.ExtractNodes(ctx, content, schema, nil)
	require.NoError(t, err)
	t.Logf("DEBUG: Extracted Entities directly: %+v", extracted)
	for _, e := range extracted {
		t.Logf(" - Entity: %s, Attributes: %+v", e.Name, e.Attributes)
	}
	
	err = g.AddEpisode(ctx, groupID, "Ep1", content, "", schema)
	require.NoError(t, err)

	// 4. Verify Attributes in DB
	// Fetch Alice
	res, err := d.ExecuteQuery(ctx, `MATCH (n:Entity {group_id: $gid, name: 'Alice'}) RETURN n.attributes as attrs`, map[string]interface{}{"gid": groupID})
	require.NoError(t, err)
	if len(res.Records) == 0 {
		t.Fatalf("Alice not found in DB. Extracted was: %+v", extracted)
	}
	
	rawAttrs, _ := res.Records[0].Get("attrs")
	t.Logf("Alice Attributes Raw in DB: %v", rawAttrs)
	
	// Expect JSON string
	attrsStr, ok := rawAttrs.(string)
	require.True(t, ok, "Attributes should be a string")
	
	var attrsMap map[string]interface{}
	err = json.Unmarshal([]byte(attrsStr), &attrsMap)
	require.NoError(t, err, "Failed to parse attributes JSON")

	assert.Equal(t, int64(28), convertToInt64(attrsMap["age"]), "Alice age should be 28")
	assert.Equal(t, "Data Scientist", attrsMap["occupation"], "Alice occupation should be Data Scientist")
	assert.Equal(t, "New York", attrsMap["location"], "Alice location should be New York")
}

func convertToInt64(v interface{}) int64 {
	switch val := v.(type) {
	case int: return int64(val)
	case int64: return val
	case float64: return int64(val)
	default: return 0
	}
}
