package evals

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadContextEvalCases(t *testing.T) {
	dir := t.TempDir()
	data := `{"case_id":"ctx-001","required_context_ids":["a"],"forbidden_context_ids":["z"]}`
	if err := os.WriteFile(filepath.Join(dir, "ctx.json"), []byte(data), 0644); err != nil {
		t.Fatal(err)
	}
	cases, err := LoadContextEvalCases(dir)
	if err != nil {
		t.Fatalf("LoadContextEvalCases() = %v", err)
	}
	if len(cases) != 1 || cases[0].CaseID != "ctx-001" {
		t.Fatalf("cases = %+v", cases)
	}
}

func TestLoadTeammateEvalCasesSortsByCaseID(t *testing.T) {
	dir := t.TempDir()
	for name, data := range map[string]string{
		"zeta.json":  `{"case_id":"team-002","mode":"delegation"}`,
		"alpha.json": `{"case_id":"team-001","mode":"task_board"}`,
	} {
		if err := os.WriteFile(filepath.Join(dir, name), []byte(data), 0644); err != nil {
			t.Fatal(err)
		}
	}
	cases, err := LoadTeammateEvalCases(dir)
	if err != nil {
		t.Fatalf("LoadTeammateEvalCases() = %v", err)
	}
	if len(cases) != 2 || cases[0].CaseID != "team-001" || cases[1].CaseID != "team-002" {
		t.Fatalf("cases = %+v", cases)
	}
}
