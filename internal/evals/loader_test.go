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

func TestLoadContextEvalCasesSupportsLiveFields(t *testing.T) {
	dir := t.TempDir()
	data := `{
		"case_id":"ctx-live-001",
		"query":"我是谁？",
		"conversation":[{"role":"user","content":"我是谁？"}],
		"memory_seeds":[{"id":"identity.name","memory":"用户说他的名字是 xyz","scope":"user","kind":"preference"}],
		"active_plan":["回答身份问题"],
		"recent_tools":["read_file"],
		"expected_answer_contains":["xyz"],
		"required_context_ids":["identity.name"],
		"agent_id":"main",
		"project_id":"/repo",
		"token_budget":128,
		"k":1
	}`
	if err := os.WriteFile(filepath.Join(dir, "ctx-live.json"), []byte(data), 0644); err != nil {
		t.Fatal(err)
	}
	cases, err := LoadContextEvalCases(dir)
	if err != nil {
		t.Fatalf("LoadContextEvalCases() = %v", err)
	}
	got := cases[0]
	if got.Query != "我是谁？" || got.Conversation[0].Content != "我是谁？" {
		t.Fatalf("live prompt fields not loaded: %+v", got)
	}
	if len(got.MemorySeeds) != 1 || got.MemorySeeds[0].ID != "identity.name" {
		t.Fatalf("memory seeds not loaded: %+v", got.MemorySeeds)
	}
	if got.ExpectedAnswerContains[0] != "xyz" || got.TokenBudget != 128 {
		t.Fatalf("live expectations not loaded: %+v", got)
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

func TestRepositoryContextEvalCasesHaveEnoughCoverage(t *testing.T) {
	cases, err := LoadContextEvalCases(filepath.Join("..", "..", "testdata", "evals", "context"))
	if err != nil {
		t.Fatalf("LoadContextEvalCases() = %v", err)
	}
	if len(cases) < 8 {
		t.Fatalf("context fixture count = %d, want at least 8", len(cases))
	}
}

func TestRepositoryTeammateEvalCasesHaveEnoughCoverage(t *testing.T) {
	cases, err := LoadTeammateEvalCases(filepath.Join("..", "..", "testdata", "evals", "teammate"))
	if err != nil {
		t.Fatalf("LoadTeammateEvalCases() = %v", err)
	}
	if len(cases) < 9 {
		t.Fatalf("teammate fixture count = %d, want at least 9", len(cases))
	}
}
