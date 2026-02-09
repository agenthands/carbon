package dedupe

import (
	"context"
	"encoding/json"
	"fmt"
	"regexp"

	"github.com/agenthands/carbon/internal/core/model"
)

func (d *Deduplicator) ResolveEdgeContradictions(ctx context.Context, newFact string, existingEdges []model.EntityEdge) ([]string, error) {
	if len(existingEdges) == 0 {
		return nil, nil // No contradictions possible
	}

	// Construct Existing Facts String
	var existingFactsStr string
	for _, edge := range existingEdges {
		existingFactsStr += fmt.Sprintf("- UUID: %s, Fact: %s\n", edge.UUID, edge.Fact)
	}

	// Use Configured Prompt or Default
	promptTemplate := d.Prompts.Edges
	if promptTemplate == "" {
		promptTemplate = `Does the New Fact contradict any of the Existing Facts?
Be conservative. Only identify contradictions that represent a change in state or a logical impossibility (e.g. "lives in Seattle" vs "moved to SF").
New Fact: %s

Existing Facts:
%s

Return a JSON object with a list of UUIDs of the EXISTING facts that are contradicted by the new fact.
Example: { "contradicted_edge_uuids": ["uuid-1"] }
If none, return empty list.`
	}

	prompt := fmt.Sprintf(promptTemplate, newFact, existingFactsStr)

	response, err := d.LLM.Generate(ctx, prompt)
	if err != nil {
		return nil, fmt.Errorf("failed to generate contradiction check: %w", err)
	}

	// Simple JSON extraction (handle markdown blocks if any)
	jsonStr := extractJSON(response) // Assume extractJSON helper exists or implement inline
	
	var result model.ContradictionResult
	if err := json.Unmarshal([]byte(jsonStr), &result); err != nil {
		// Log warning? For now return nil
		return nil, fmt.Errorf("failed to parse contradiction result: %w", err)
	}

	return result.ContradictedEdgeUUIDs, nil
}

func extractJSON(s string) string {
	re := regexp.MustCompile(`\{[\s\S]*\}`)
	match := re.FindString(s)
	if match != "" {
		return match
	}
	return s
}
