package dedupe

import (
	"context"
	"encoding/json"
	"fmt"
	
	"github.com/agenthands/carbon/internal/core/model"
	"github.com/agenthands/carbon/internal/llm"
)

type Deduplicator struct {
	LLM llm.LLMClient
}

func NewDeduplicator(llmClient llm.LLMClient) *Deduplicator {
	return &Deduplicator{
		LLM: llmClient,
	}
}

func (d *Deduplicator) ResolveDuplicates(ctx context.Context, newNodes []model.EntityNode, existingNodes []model.EntityNode) ([]model.DuplicatePair, error) {
	prompt := fmt.Sprintf(`
<NEW NODES>
%s
</NEW NODES>

<EXISTING NODES>
%s
</EXISTING NODES>

Instructions:
Identify if any of the NEW NODES are duplicates of the EXISTING NODES.
Return a JSON object with key "duplicates" which is a list of objects.
Each object should have "original_uuid" (existing node UUID), "duplicate_uuid" (new node UUID), and "confidence" (float).

Example JSON:
{
  "duplicates": [
    {"original_uuid": "existing-1", "duplicate_uuid": "new-1", "confidence": 0.9}
  ]
}
`, serializeNodes(newNodes), serializeNodes(existingNodes))

	response, err := d.LLM.Generate(ctx, prompt)
	if err != nil {
		return nil, fmt.Errorf("failed to generate deduplication result: %w", err)
	}

	// Basic JSON cleanup
	jsonStr := response
	start := 0
	end := len(jsonStr)
	
	for i, c := range jsonStr {
		if c == '{' {
			start = i
			break
		}
	}
	for i := len(jsonStr) - 1; i >= 0; i-- {
		if c := jsonStr[i]; c == '}' {
			end = i + 1
			break
		}
	}
	
	if start < end {
		jsonStr = jsonStr[start:end]
	}

	var result model.DeduplicationResult
	if err := json.Unmarshal([]byte(jsonStr), &result); err != nil {
		return nil, fmt.Errorf("failed to unmarshal dedupe result: %w\nResponse: %s", err, response)
	}

	return result.Duplicates, nil
}

func serializeNodes(nodes []model.EntityNode) string {
	var s string
	for _, n := range nodes {
		s += fmt.Sprintf("- UUID: %s, Name: %s\n", n.UUID, n.Name)
	}
	return s
}
