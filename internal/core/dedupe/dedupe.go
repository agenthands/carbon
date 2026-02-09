package dedupe

import (
	"context"
	"fmt"
	
	"github.com/agenthands/carbon/internal/config"
	"github.com/agenthands/carbon/internal/core/common"
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

	result, err := common.ParseJSON[model.DeduplicationResult](response)
	if err != nil {
		return nil, fmt.Errorf("failed to parse deduplication result: %w", err)
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
