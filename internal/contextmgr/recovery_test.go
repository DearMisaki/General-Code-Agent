package contextmgr

import (
	"strings"
	"testing"
)

func TestRecoveryTrackerBuildsCompactAttachment(t *testing.T) {
	tracker := NewRecoveryTracker()
	tracker.RecordFileRead("a.go", "package a")
	tracker.RecordSkillInvocation("review", "review SOP")

	attachment := tracker.BuildAttachment([]map[string]any{{
		"name":        "ReadFile",
		"description": "Read a file\nmore",
	}})
	for _, want := range []string{"Recently read files", "a.go", "package a", "Active skills", "review", "Available tools", "ReadFile"} {
		if !strings.Contains(attachment, want) {
			t.Fatalf("missing %q in attachment:\n%s", want, attachment)
		}
	}
}

func TestRecoveryTrackerNilSafe(t *testing.T) {
	var tracker *RecoveryTracker
	tracker.RecordFileRead("a.go", "package a")
	tracker.RecordSkillInvocation("review", "body")
	if got := tracker.BuildAttachment(nil); got != "" {
		t.Fatalf("nil tracker attachment = %q", got)
	}
}
