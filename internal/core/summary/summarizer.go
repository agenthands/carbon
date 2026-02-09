package summary

import (
	"context"
	"fmt"
	
	"github.com/agenthands/carbon/internal/config"
	"github.com/agenthands/carbon/internal/core/common"
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

	result, err := common.ParseJSON[model.EntitySummary](response)
	if err != nil {
		return "", fmt.Errorf("failed to parse summary result: %w", err)
	}

	return result.Summary, nil
}

func (s *Summarizer) SummarizeCommunity(ctx context.Context, nodes []model.EntityNode) (string, error) {
	const ChunkSize = 20

	// 1. Base Case: Small enough to fit in context
	if len(nodes) <= ChunkSize {
		summaries := ""
		for _, n := range nodes {
			if n.Summary != "" {
				summaries += fmt.Sprintf("- %s: %s\n", n.Name, n.Summary)
			}
		}
		if summaries == "" {
			return "No significant information.", nil
		}

		prompt := fmt.Sprintf(s.Prompts.Communities, summaries)
		response, err := s.LLM.Generate(ctx, prompt)
		if err != nil {
			return "", fmt.Errorf("failed to generate community summary: %w", err)
		}

		// Try to parse JSON first
		result, err := common.ParseJSON[model.EntitySummary](response)
		if err == nil {
			return result.Summary, nil
		}
		return response, nil
	}

	// 2. Recursive Case: Split and Reduce
	var chunks [][]model.EntityNode
	for i := 0; i < len(nodes); i += ChunkSize {
		end := i + ChunkSize
		if end > len(nodes) {
			end = len(nodes)
		}
		chunks = append(chunks, nodes[i:end])
	}

	var intermediateSummaries []string
	for _, chunk := range chunks {
		summary, err := s.SummarizeCommunity(ctx, chunk)
		if err != nil {
			// Continue with partial results or fail?
			// Let's log and continue to be robust
			continue
		}
		intermediateSummaries = append(intermediateSummaries, summary)
	}

	if len(intermediateSummaries) == 0 {
		return "Failed to generate summary.", nil
	}
	
	// Create pseudo-nodes for the intermediate summaries to recurse
	var metaNodes []model.EntityNode
	for i, summary := range intermediateSummaries {
		metaNodes = append(metaNodes, model.EntityNode{
			Name:    fmt.Sprintf("Part %d", i+1),
			Summary: summary,
		})
	}
	
	// Recurse on the meta nodes
	return s.SummarizeCommunity(ctx, metaNodes)
}

func (s *Summarizer) GenerateCommunityName(ctx context.Context, summary string) (string, error) {
	if s.Prompts.CommunityName == "" {
		return "", nil // Fallback
	}

	prompt := fmt.Sprintf(s.Prompts.CommunityName, summary)
	
	response, err := s.LLM.Generate(ctx, prompt)
	if err != nil {
		return "", fmt.Errorf("failed to generate community name: %w", err)
	}

	result, err := common.ParseJSON[model.CommunityName](response)
	if err == nil {
		return result.Name, nil
	}
	
	// Fallback to response text if short enough? 
	// Or maybe it's just a name in quotes.
	return response, nil
}
