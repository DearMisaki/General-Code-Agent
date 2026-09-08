package contextmgr

import (
	"strings"
	"testing"

	"mewcode/internal/permissions"
)

func TestRendererIncludesCurrentReminderSections(t *testing.T) {
	req := PrepareRequest{
		Iteration:         1,
		WorkDir:           t.TempDir(),
		Notifications:     []string{"team idle"},
		DeferredToolNames: []string{"mcp__docs__search"},
		Checker:           &permissions.Checker{Mode: permissions.ModePlan},
	}
	snap := RuntimeContext{
		User: UserContext{Instructions: "repo rules", MemoryContent: "memory body"},
		Business: BusinessContext{ActiveSkills: map[string]string{
			"review": "review SOP",
		}},
	}

	out := NewRenderer().Render(req, snap)
	joined := strings.Join(out, "\n---\n")
	for _, want := range []string{"repo rules", "memory body", "team idle", "Active Skills", "review SOP", "deferred tools", "mcp__docs__search", "Plan mode"} {
		if !strings.Contains(joined, want) {
			t.Fatalf("rendered reminders missing %q:\n%s", want, joined)
		}
	}
}
