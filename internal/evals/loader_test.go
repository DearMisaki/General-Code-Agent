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

func TestRepositoryTeammateEvalCasesCoverAllCollaborationModes(t *testing.T) {
	cases, err := LoadTeammateEvalCases(filepath.Join("..", "..", "testdata", "evals", "teammate"))
	if err != nil {
		t.Fatalf("LoadTeammateEvalCases() = %v", err)
	}

	covered := make(map[string]bool)
	for _, tc := range cases {
		covered[tc.Mode] = true
	}
	for _, mode := range []string{"delegation", "peer", "task_board"} {
		if !covered[mode] {
			t.Errorf("collaboration mode %q has no golden fixture", mode)
		}
	}
}
