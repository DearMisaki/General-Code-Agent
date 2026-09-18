package memory

import (
	"fmt"
	"strings"
)

type MemoryBudgetLimiter struct {
	budget int
}

func NewMemoryBudgetLimiter(tokenBudget int) *MemoryBudgetLimiter {
	return &MemoryBudgetLimiter{budget: tokenBudget}
}

func (l *MemoryBudgetLimiter) Limit(blocks []MemoryBlock) []MemoryBlock {
	if l == nil || l.budget <= 0 {
		return append([]MemoryBlock(nil), blocks...)
	}
	var kept []MemoryBlock
	remaining := l.budget
	for _, block := range blocks {
		cost := block.TokenCost
		if cost <= 0 {
			cost = estimateMemoryTokens(block.Text)
			block.TokenCost = cost
		}
		if cost > remaining {
			continue
		}
		kept = append(kept, block)
		remaining -= cost
	}
	return kept
}

func RenderMemoryBlocks(blocks []MemoryBlock) string {
	if len(blocks) == 0 {
		return ""
	}
	groups := []struct {
		title string
		match func(MemoryBlock) bool
	}{
		{"User constraints", func(b MemoryBlock) bool {
			return b.Scope == ScopeUser || b.Kind == KindConstraint || b.Kind == KindPreference
		}},
		{"Project decisions", func(b MemoryBlock) bool {
			return b.Scope == ScopeProject || b.Kind == KindArchitectureDecision || b.Kind == KindProjectFact
		}},
		{"Team and task memory", func(b MemoryBlock) bool {
			return b.Scope == ScopeTeam || b.Kind == KindTaskSummary || b.Kind == KindTeamState
		}},
		{"Tool gotchas", func(b MemoryBlock) bool { return b.Kind == KindToolGotcha }},
		{"Other memory", func(b MemoryBlock) bool { return true }},
	}
	used := make(map[string]struct{}, len(blocks))
	var out strings.Builder
	out.WriteString("# longTermMemory\n")
	for _, group := range groups {
		var lines []string
		for _, block := range blocks {
			if _, ok := used[block.ID]; ok {
				continue
			}
			if group.match(block) {
				used[block.ID] = struct{}{}
				label := block.ID
				if label == "" {
					label = string(block.Kind)
				}
				lines = append(lines, fmt.Sprintf("- [%s] %s", label, block.Text))
			}
		}
		if len(lines) == 0 {
			continue
		}
		out.WriteString("\n## ")
		out.WriteString(group.title)
		out.WriteString("\n")
		out.WriteString(strings.Join(lines, "\n"))
		out.WriteString("\n")
	}
	return strings.TrimSpace(out.String())
}
