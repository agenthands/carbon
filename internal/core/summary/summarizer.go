package summary

import (
	"context"
	"fmt"
	"encoding/json"
	
	"github.com/agenthands/carbon/internal/config"
	"github.com/agenthands/carbon/internal/core/model"
	"github.com/agenthands/carbon/internal/llm"
)

type Summarizer struct {
	LLM     llm.LLMClient
	Prompts config.SummaryPrompts
}

func NewSummarizer(llmClient llm.LLMClient, prompts config.SummaryPrompts) *Summarizer {
	return &Summarizer{
		LLM:     llmClient,
		Prompts: prompts,
	}
}

func (s *Summarizer) SummarizeNode(ctx context.Context, node model.EntityNode, newMentions []string) (string, error) {
	mentionsList := ""
	for _, m := range newMentions {
		mentionsList += fmt.Sprintf("- %s\n", m)
	}

	prompt := fmt.Sprintf(s.Prompts.Nodes, node.Summary, mentionsList)

	response, err := s.LLM.Generate(ctx, prompt)
	if err != nil {
		return "", fmt.Errorf("failed to generate summary: %w", err)
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

	var result model.EntitySummary
	if err := json.Unmarshal([]byte(jsonStr), &result); err != nil {
		return "", fmt.Errorf("failed to unmarshal summary result: %w\nResponse: %s", err, response)
	}

	return result.Summary, nil
}
