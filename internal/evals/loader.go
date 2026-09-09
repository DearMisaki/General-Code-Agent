package evals

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
)

func LoadContextEvalCases(dir string) ([]ContextEvalCase, error) {
	return loadJSONFiles[ContextEvalCase](dir)
}

func LoadTeammateEvalCases(dir string) ([]TeammateEvalCase, error) {
	return loadJSONFiles[TeammateEvalCase](dir)
}

func loadJSONFiles[T any](dir string) ([]T, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}
	var out []T
	for _, entry := range entries {
		if entry.IsDir() || filepath.Ext(entry.Name()) != ".json" {
			continue
		}
		data, err := os.ReadFile(filepath.Join(dir, entry.Name()))
		if err != nil {
			return nil, err
		}
		var item T
		if err := json.Unmarshal(data, &item); err != nil {
			return nil, fmt.Errorf("%s: %w", entry.Name(), err)
		}
		out = append(out, item)
	}
	sort.Slice(out, func(i, j int) bool { return caseID(out[i]) < caseID(out[j]) })
	return out, nil
}

func caseID[T any](item T) string {
	switch value := any(item).(type) {
	case ContextEvalCase:
		return value.CaseID
	case TeammateEvalCase:
		return value.CaseID
	default:
		return ""
	}
}
