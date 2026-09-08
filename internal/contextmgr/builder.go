package contextmgr

import (
	"os"
	"runtime"
	"time"

	"mewcode/internal/conversation"
	"mewcode/internal/prompt"
)

type Builder struct{}

func NewBuilder() *Builder { return &Builder{} }

func (b *Builder) Build(req PrepareRequest) RuntimeContext {
	now := time.Now()
	var messages []conversation.Message
	if req.Conversation != nil {
		messages = req.Conversation.GetMessages()
	}

	activeSkills := make(map[string]string, len(req.ActiveSkills))
	for k, v := range req.ActiveSkills {
		activeSkills[k] = v
	}

	env := prompt.DetectEnvironment(req.WorkDir)
	if env.OS == "" {
		env.OS = runtime.GOOS
	}
	if env.Arch == "" {
		env.Arch = runtime.GOARCH
	}
	if env.Shell == "" {
		env.Shell = os.Getenv("SHELL")
	}

	snap := RuntimeContext{
		ID: req.SessionID,
		User: UserContext{
			SectionName:   SectionUser,
			Instructions:  req.Instructions,
			MemoryContent: req.MemoryContent,
		},
		Session: SessionContext{
			SectionName:  SectionSession,
			SessionID:    req.SessionID,
			Messages:     messages,
			MessageCount: len(messages),
		},
		Business: BusinessContext{
			SectionName:       SectionBusiness,
			ActiveSkills:      activeSkills,
			ToolSchemas:       append([]map[string]any(nil), req.ToolSchemas...),
			DeferredToolNames: append([]string(nil), req.DeferredToolNames...),
			Checker:           req.Checker,
		},
		Execution: ExecutionContext{
			SectionName:     SectionExecution,
			AgentID:         req.AgentID,
			AgentType:       req.AgentType,
			Protocol:        req.Protocol,
			WorkDir:         req.WorkDir,
			Iteration:       req.Iteration,
			MaxIterations:   req.MaxIterations,
			ContextWindow:   req.ContextWindow,
			MaxOutputTokens: req.MaxOutputTokens,
		},
		Environment: EnvironmentContext{
			SectionName: SectionEnvironment,
			OS:          env.OS,
			Arch:        env.Arch,
			Shell:       env.Shell,
			Date:        env.Date,
			IsGitRepo:   env.IsGitRepo,
			GitBranch:   env.GitBranch,
			Model:       req.Model,
		},
		Budget: BudgetContext{
			SectionName:      SectionBudget,
			UsageAnchor:      req.UsageAnchor,
			CompactTracking:  req.CompactTracking,
			ReplacementState: req.ReplacementState,
		},
	}

	if req.Instructions != "" {
		snap.Sources = append(snap.Sources, ContextSource{Section: SectionUser, Kind: "instructions", FreshAt: now})
	}
	if req.MemoryContent != "" {
		snap.Sources = append(snap.Sources, ContextSource{Section: SectionUser, Kind: "memory", FreshAt: now})
	}
	if len(activeSkills) > 0 {
		snap.Sources = append(snap.Sources, ContextSource{Section: SectionBusiness, Kind: "skill", FreshAt: now})
	}
	if len(req.ToolSchemas) > 0 || len(req.DeferredToolNames) > 0 {
		snap.Sources = append(snap.Sources, ContextSource{Section: SectionBusiness, Kind: "tool-registry", FreshAt: now})
	}
	if len(messages) > 0 {
		snap.Sources = append(snap.Sources, ContextSource{Section: SectionSession, Kind: "conversation", FreshAt: now})
	}
	snap.Sources = append(snap.Sources, ContextSource{Section: SectionEnvironment, Kind: "runtime", FreshAt: now})

	return snap
}
