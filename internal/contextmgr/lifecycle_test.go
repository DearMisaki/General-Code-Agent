package contextmgr

import (
	"context"
	"testing"
)

func TestLifecycleClearResetsRecovery(t *testing.T) {
	lc := NewLifecycleManager(NewBudgeter(), NewRecoveryTracker())
	lc.Recovery().RecordFileRead("a.go", "package a")
	lc.Clear()
	if got := lc.Recovery().BuildAttachment(nil); got != "" {
		t.Fatalf("recovery not cleared:\n%s", got)
	}
}

func TestLifecycleForceCompactWithNilClientIsNoop(t *testing.T) {
	lc := NewLifecycleManager(NewBudgeter(), NewRecoveryTracker())
	msg, err := lc.ForceCompact(context.Background(), BudgetRequest{})
	if err != nil {
		t.Fatalf("force compact: %v", err)
	}
	if msg != "" {
		t.Fatalf("message = %q", msg)
	}
}
