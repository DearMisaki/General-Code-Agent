package contextmgr

import (
	"strings"

	"mewcode/internal/permissions"
	"mewcode/internal/planfile"
	"mewcode/internal/prompt"
)

type Renderer struct{}

func NewRenderer() *Renderer { return &Renderer{} }

func (r *Renderer) Render(req PrepareRequest, snap RuntimeContext) []string {
	var reminders []string

	if snap.User.Instructions != "" || snap.User.MemoryContent != "" {
		var sections []string
		if snap.User.Instructions != "" {
			sections = append(sections, "# mewcodeMd\nCodebase and user instructions are shown below. Be sure to adhere to these instructions. IMPORTANT: These instructions OVERRIDE any default behavior and you MUST follow them exactly as written.\n\n"+snap.User.Instructions)
		}
		if snap.User.MemoryContent != "" {
			sections = append(sections, "# autoMemory\n"+snap.User.MemoryContent)
		}
		reminders = append(reminders, "As you answer the user's questions, you can use the following context:\n"+
			strings.Join(sections, "\n\n")+
			"\n\n      IMPORTANT: this context may or may not be relevant to your tasks. You should not respond to this context unless it is highly relevant to your task.")
	}

	if req.Checker != nil && req.Checker.Mode == permissions.ModePlan {
		planPath := planfile.GetOrCreatePlanPath(req.WorkDir)
		req.Checker.PlanFilePath = planPath
		planExists := planfile.PlanExists(req.WorkDir)
		reminders = append(reminders, prompt.BuildPlanModeReminder(planPath, planExists, req.Iteration))
	}

	reminders = append(reminders, req.Notifications...)

	if active := renderActiveSkills(snap.Business.ActiveSkills); active != "" {
		reminders = append(reminders, active)
	}

	if len(req.DeferredToolNames) > 0 {
		reminders = append(reminders, "The following deferred tools are available via ToolSearch. Their schemas are NOT loaded - use ToolSearch with query \"select:<name>[,<name>...]\" to load tool schemas before calling them:\n"+strings.Join(req.DeferredToolNames, "\n"))
	}

	return reminders
}

func renderActiveSkills(active map[string]string) string {
	if len(active) == 0 {
		return ""
	}
	var sb strings.Builder
	sb.WriteString("# Active Skills\n\nThe following Skill SOPs are pinned to the environment context. Follow each SOP when its triggering condition applies.\n\n")
	for name, body := range active {
		sb.WriteString("## Active Skill: ")
		sb.WriteString(name)
		sb.WriteString("\n\n")
		sb.WriteString(body)
		sb.WriteString("\n\n")
	}
	return sb.String()
}
