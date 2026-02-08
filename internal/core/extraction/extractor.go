package extraction

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/agenthands/carbon/internal/config"
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
func (e *Extractor) ExtractNodes(ctx context.Context, content string, entityTypes string, previousEpisodes []string) ([]model.ExtractedEntity, error) {
	// Construct the prompt similar to Python's extract_message
	prompt := fmt.Sprintf(e.Prompts.Nodes, entityTypes, content)

	response, err := e.LLM.Generate(ctx, prompt)
	if err != nil {
		return nil, fmt.Errorf("failed to generate entities: %w", err)
	}

	// Basic JSON cleanup/extraction if the model adds markdown code blocks
	jsonStr := response
	// Find first '{' and last '}'
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

	var result model.ExtractedEntities
	if err := json.Unmarshal([]byte(jsonStr), &result); err != nil {
		return nil, fmt.Errorf("failed to unmarshal extraction result: %w\nResponse was: %s", err, response)
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

	var result model.ExtractedEdges
	if err := json.Unmarshal([]byte(jsonStr), &result); err != nil {
		return nil, fmt.Errorf("failed to unmarshal edge extraction result: %w\nResponse: %s", err, response)
	}

	return result.ExtractedEdges, nil
}
