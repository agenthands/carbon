package common

import (
	"encoding/json"
	"fmt"
)

// ParseJSON cleans and unmarshals a JSON string into a type T.
// It handles common LLM quirks like surrounding markdown or extra text.
func ParseJSON[T any](response string) (T, error) {
	var zero T
	jsonStr := response

	// Find first '{' and last '}'
	start := -1
	end := -1

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

	if start != -1 && end != -1 && start < end {
		jsonStr = jsonStr[start:end]
	} else if start == -1 {
		return zero, fmt.Errorf("no JSON object found in response (missing '{')")
	}

	var result T
	if err := json.Unmarshal([]byte(jsonStr), &result); err != nil {
		return zero, fmt.Errorf("failed to unmarshal JSON: %w\nData: %s", err, jsonStr)
	}

	return result, nil
}
