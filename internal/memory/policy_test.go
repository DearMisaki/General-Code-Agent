package memory

import "testing"

func TestMemoryPolicyFilterRejectsCredentials(t *testing.T) {
	filter := NewMemoryPolicyFilter(PolicyOptions{MinConfidence: 0.75})
	decision := filter.Filter(MemoryCandidate{
		Scope:      ScopeUser,
		Kind:       KindPreference,
		Memory:     "The API key is sk-secret.",
		Confidence: 0.99,
	})
	if decision.Accepted {
		t.Fatal("credential-like memory should be rejected")
	}
	if decision.Reason != FilterReasonSensitive {
		t.Fatalf("reason = %q, want %q", decision.Reason, FilterReasonSensitive)
	}
}

func TestMemoryPolicyFilterAcceptsProjectDecision(t *testing.T) {
	filter := NewMemoryPolicyFilter(PolicyOptions{MinConfidence: 0.75})
	decision := filter.Filter(MemoryCandidate{
		Scope:      ScopeProject,
		Kind:       KindArchitectureDecision,
		Memory:     "Use ContextGateway as the only context preparation entrypoint.",
		Confidence: 0.92,
	})
	if !decision.Accepted {
		t.Fatalf("expected accepted, got %q", decision.Reason)
	}
}
