package summary

import (
	"context"
	"fmt"
	"encoding/json"
	
	"github.com/agenthands/carbon/internal/core/model"
	"github.com/agenthands/carbon/internal/llm"
)

type Summarizer struct {
	LLM llm.LLMClient
}

func NewSummarizer(llmClient llm.LLMClient) *Summarizer {
	return &Summarizer{
		LLM: llmClient,
	}
}

func (s *Summarizer) SummarizeNode(ctx context.Context, node model.EntityNode, newMentions []string) (string, error) {
	mentionsList := ""
	for _, m := range newMentions {
		mentionsList += fmt.Sprintf("- %s\n", m)
	}

	prompt := fmt.Sprintf(`
<EXISTING SUMMARY>
%s
</EXISTING SUMMARY>

<NEW MENTIONS>
%s
</NEW MENTIONS>

Instructions:
Update the existing summary to incorporate the new information from the mentions.
Return the result as a JSON object with a single key "summary" (string).

Example JSON:
{
  "summary": "Alice is a software engineer living in Paris."
}
`, node.Summary, mentionsList)

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
