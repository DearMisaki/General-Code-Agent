package extractor

import (
	"encoding/json"
	"fmt"
	"strings"

	"mewcode/internal/memory"
)

func ParseCandidateOutput(raw string) ([]memory.MemoryCandidate, error) {
	clean := extractJSONObject(strings.TrimSpace(stripJSONFence(raw)))
	if clean == "" {
		return nil, fmt.Errorf("no JSON object found in memory candidate output")
	}
	var parsed struct {
		Candidates []memory.MemoryCandidate `json:"candidates"`
	}
	if err := json.Unmarshal([]byte(clean), &parsed); err != nil {
		return nil, err
	}
	for i, candidate := range parsed.Candidates {
		if err := candidate.Validate(); err != nil {
			return nil, fmt.Errorf("candidate %d: %w", i, err)
		}
	}
	return parsed.Candidates, nil
}

func stripJSONFence(raw string) string {
	trimmed := strings.TrimSpace(raw)
	if strings.HasPrefix(trimmed, "```") {
		lines := strings.Split(trimmed, "\n")
		if len(lines) >= 2 {
			lines = lines[1:]
			if strings.HasPrefix(strings.TrimSpace(lines[len(lines)-1]), "```") {
				lines = lines[:len(lines)-1]
			}
			return strings.Join(lines, "\n")
		}
	}
	return raw
}

func extractJSONObject(raw string) string {
	trimmed := strings.TrimSpace(raw)
	if strings.HasPrefix(trimmed, "{") && strings.HasSuffix(trimmed, "}") {
		return trimmed
	}
	start := strings.Index(trimmed, "{")
	end := strings.LastIndex(trimmed, "}")
	if start < 0 || end < start {
		return ""
	}
	return trimmed[start : end+1]
}
