package dedupe

import (
	"context"
	"encoding/json"
	"fmt"
	
	"github.com/agenthands/carbon/internal/config"
	"github.com/agenthands/carbon/internal/core/model"
	"github.com/agenthands/carbon/internal/llm"
)

type Deduplicator struct {
	LLM     llm.LLMClient
	Prompts config.DeduplicationPrompts
}

func NewDeduplicator(llmClient llm.LLMClient, prompts config.DeduplicationPrompts) *Deduplicator {
	return &Deduplicator{
		LLM:     llmClient,
		Prompts: prompts,
	}
}

func (d *Deduplicator) ResolveDuplicates(ctx context.Context, newNodes []model.EntityNode, existingNodes []model.EntityNode) ([]model.DuplicatePair, error) {
	prompt := fmt.Sprintf(d.Prompts.Nodes, serializeNodes(newNodes), serializeNodes(existingNodes))

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
