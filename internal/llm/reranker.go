package llm

import (
	"context"
	"fmt"
	"regexp"
	"strconv"
)

type SimpleLLMReranker struct {
	LLM LLMClient
}

func NewSimpleLLMReranker(client LLMClient) *SimpleLLMReranker {
	return &SimpleLLMReranker{LLM: client}
}

func (r *SimpleLLMReranker) Rank(ctx context.Context, query string, docs []string) ([]int, error) {
	if len(docs) == 0 {
		return nil, nil
	}
	if len(docs) == 1 {
		return []int{0}, nil
	}

	docList := ""
	for i, d := range docs {
		// Truncate very long docs
		content := d
		if len(content) > 200 {
			content = content[:200] + "..."
		}
		docList += fmt.Sprintf("[%d] %s\n", i, content)
	}

	prompt := fmt.Sprintf(`You are a search relevance optimization system.
Query: %s

Documents:
%s

Rank the documents above based on their relevance to the query.
Output ONLY the indices of the documents in order of relevance, separated by commas.
Example: 0, 2, 1
Do not output any other text.`, query, docList)

	resp, err := r.LLM.Generate(ctx, prompt)
	if err != nil {
		// Fallback to original order on error
		indices := make([]int, len(docs))
		for i := range indices { indices[i] = i }
		return indices, nil
	}

	// Parse indices
	return parseIndices(resp), nil
}

func parseIndices(s string) []int {
	re := regexp.MustCompile(`\d+`)
	matches := re.FindAllString(s, -1)
	var indices []int
	for _, m := range matches {
		if i, err := strconv.Atoi(m); err == nil {
			indices = append(indices, i)
		}
	}
	return indices
}
