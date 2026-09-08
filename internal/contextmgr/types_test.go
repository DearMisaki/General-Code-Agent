package contextmgr

import "testing"

func TestHandoffModeConstants(t *testing.T) {
	modes := []HandoffMode{HandoffNone, HandoffRecent, HandoffSummary, HandoffFull, HandoffFork}
	seen := map[HandoffMode]bool{}
	for _, mode := range modes {
		if mode == "" {
			t.Fatalf("handoff mode must not be empty")
		}
		if seen[mode] {
			t.Fatalf("duplicate handoff mode: %q", mode)
		}
		seen[mode] = true
	}
}

func TestRuntimeContextSectionsHaveStableNames(t *testing.T) {
	ctx := RuntimeContext{
		User:        UserContext{SectionName: SectionUser},
		Session:     SessionContext{SectionName: SectionSession},
		Business:    BusinessContext{SectionName: SectionBusiness},
		Execution:   ExecutionContext{SectionName: SectionExecution},
		Environment: EnvironmentContext{SectionName: SectionEnvironment},
		Budget:      BudgetContext{SectionName: SectionBudget},
	}
	if ctx.User.SectionName != "user" {
		t.Fatalf("user section = %q", ctx.User.SectionName)
	}
	if ctx.Session.SectionName != "session" {
		t.Fatalf("session section = %q", ctx.Session.SectionName)
	}
	if ctx.Business.SectionName != "business" {
		t.Fatalf("business section = %q", ctx.Business.SectionName)
	}
	if ctx.Execution.SectionName != "execution" {
		t.Fatalf("execution section = %q", ctx.Execution.SectionName)
	}
	if ctx.Environment.SectionName != "environment" {
		t.Fatalf("environment section = %q", ctx.Environment.SectionName)
	}
	if ctx.Budget.SectionName != "budget" {
		t.Fatalf("budget section = %q", ctx.Budget.SectionName)
	}
}
