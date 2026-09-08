package agent

import (
	"strings"
	"testing"

	"mewcode/internal/contextmgr"
)

func TestActivateAndClearSkills(t *testing.T) {
	a := &Agent{}
	a.ActivateSkill("commit", "do git stuff")
	a.ActivateSkill("review", "audit changes")

	if got := a.GetActiveSkills(); len(got) != 2 {
		t.Errorf("expected 2 active skills, got %d", len(got))
	}

	a.ClearActiveSkills()
	if got := a.GetActiveSkills(); len(got) != 0 {
		t.Errorf("ClearActiveSkills did not empty the map; got %d", len(got))
	}
}

func TestActiveSkillsReminderRendering(t *testing.T) {
	active := map[string]string{
		"commit": "git status then conventional commit",
	}
	reminder := buildActiveSkillsReminder(active)
	if !strings.Contains(reminder, "# Active Skills") {
		t.Errorf("missing header section")
	}
	if !strings.Contains(reminder, "## Active Skill: commit") {
		t.Errorf("missing skill subheader")
	}
	if !strings.Contains(reminder, "git status") {
		t.Errorf("body not rendered")
	}
}

func TestActiveSkillsReminderEmpty(t *testing.T) {
	if buildActiveSkillsReminder(nil) != "" {
		t.Errorf("nil map must render empty string (short-circuit)")
	}
	if buildActiveSkillsReminder(map[string]string{}) != "" {
		t.Errorf("empty map must render empty string")
	}
}

func TestActivateSkillRecordsRecovery(t *testing.T) {
	a := &Agent{ContextLifecycle: contextmgr.NewLifecycleManager(nil, contextmgr.NewRecoveryTracker())}
	a.ActivateSkill("review", "review SOP")

	attachment := a.ContextLifecycle.Recovery().BuildAttachment(nil)
	if !strings.Contains(attachment, "review SOP") {
		t.Fatalf("skill recovery not recorded:\n%s", attachment)
	}
}

func TestClearContextStateResetsContextCarriedState(t *testing.T) {
	a := New(nil, nil, "anthropic")
	a.ActivateSkill("review", "review SOP")
	a.ContextLifecycle.Recovery().RecordFileRead("a.go", "package a")

	a.ClearContextState()

	if got := a.GetActiveSkills(); len(got) != 0 {
		t.Fatalf("active skills not cleared: %v", got)
	}
	if a.ReplacementState == nil {
		t.Fatalf("replacement state not reset")
	}
	if a.RecoveryState == nil {
		t.Fatalf("recovery state not reset")
	}
	if got := a.ContextLifecycle.Recovery().BuildAttachment(nil); got != "" {
		t.Fatalf("context lifecycle recovery not cleared:\n%s", got)
	}
}
