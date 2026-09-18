package memory

import "testing"

func TestParseMemoryScope(t *testing.T) {
	cases := map[string]MemoryScope{
		"user":    ScopeUser,
		"project": ScopeProject,
		"agent":   ScopeAgent,
		"team":    ScopeTeam,
		"session": ScopeSession,
	}
	for raw, want := range cases {
		got, ok := ParseMemoryScope(raw)
		if !ok || got != want {
			t.Fatalf("ParseMemoryScope(%q) = (%q, %v), want (%q, true)", raw, got, ok, want)
		}
	}
	if _, ok := ParseMemoryScope("global"); ok {
		t.Fatal("unknown scope should be rejected")
	}
}

func TestMemoryCandidateValidate(t *testing.T) {
	c := MemoryCandidate{
		Scope:      ScopeProject,
		Kind:       KindArchitectureDecision,
		Memory:     "contextmgr.ContextGateway is the context entrypoint.",
		Confidence: 0.91,
	}
	if err := c.Validate(); err != nil {
		t.Fatalf("Validate() = %v", err)
	}
	c.Memory = ""
	if err := c.Validate(); err == nil {
		t.Fatal("empty memory should fail validation")
	}
}
