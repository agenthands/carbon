package extraction

import (
	"context"
	"fmt"

	"github.com/agenthands/carbon/internal/config"
	"github.com/agenthands/carbon/internal/core/common"
	"github.com/agenthands/carbon/internal/core/model"
	"github.com/agenthands/carbon/internal/llm"
)

type Extractor struct {
	LLM     llm.LLMClient
	Prompts config.ExtractionPrompts
}

func NewExtractor(llmClient llm.LLMClient, prompts config.ExtractionPrompts) *Extractor {
	return &Extractor{
		LLM:     llmClient,
		Prompts: prompts,
	}
}

// ExtractNodes extracts entities from the given content using the LLM.
func (e *Extractor) ExtractNodes(ctx context.Context, content string, schema string, previousEpisodes []string) ([]model.ExtractedEntity, error) {
	// Construct the prompt similar to Python's extract_message
	prompt := fmt.Sprintf(e.Prompts.Nodes, schema, content)

	response, err := e.LLM.Generate(ctx, prompt)
	if err != nil {
		return nil, fmt.Errorf("failed to generate entities: %w", err)
	}

	result, err := common.ParseJSON[model.ExtractedEntities](response)
	if err != nil {
		return nil, fmt.Errorf("failed to extract entities: %w", err)
	}

	return result.ExtractedEntities, nil
}

func (e *Extractor) ExtractEdges(ctx context.Context, nodes []model.EntityNode, previousEpisodes []string) ([]model.ExtractedEdge, error) {
	// Simple serialization of nodes for context
	var nodeContext string
	for _, n := range nodes {
		nodeContext += fmt.Sprintf("- UUID: %s, Name: %s\n", n.UUID, n.Name)
	}

	prompt := fmt.Sprintf(e.Prompts.Edges, nodeContext)

	response, err := e.LLM.Generate(ctx, prompt)
	if err != nil {
		return nil, fmt.Errorf("failed to generate edges: %w", err)
	}
	
	result, err := common.ParseJSON[model.ExtractedEdges](response)
	if err != nil {
		return nil, fmt.Errorf("failed to extract edges: %w", err)
	}

	return result.ExtractedEdges, nil
}
