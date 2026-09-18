package evals

import (
	"context"
	"fmt"
	"strings"
	"time"

	"mewcode/internal/agent"
	"mewcode/internal/conversation"
	"mewcode/internal/llm"
	"mewcode/internal/memory"
	"mewcode/internal/tools"
)

type ContextLiveRunnerOptions struct {
	Store           memory.MemoryStore
	Client          llm.Client
	Provider        string
	Model           string
	Protocol        string
	UserID          string
	AppID           string
	AgentID         string
	ProjectID       string
	SessionID       string
	RunID           string
	ContextWindow   int
	MaxOutputTokens int
	TokenBudget     int
	TopK            int
	Threshold       float64
	Rerank          bool
	Timeout         time.Duration
}

type ContextLiveRunner struct {
	opts ContextLiveRunnerOptions
}

type SeededMemory struct {
	LogicalID  string `json:"logical_id"`
	PhysicalID string `json:"physical_id"`
	Scope      string `json:"scope"`
}

type ContextLiveEvalResult struct {
	CaseID               string           `json:"case_id"`
	Runner               string           `json:"runner"`
	Mode                 string           `json:"mode"`
	Provider             string           `json:"provider,omitempty"`
	Model                string           `json:"model,omitempty"`
	SessionID            string           `json:"session_id,omitempty"`
	DurationMS           int64            `json:"duration_ms"`
	SeededMemories       []SeededMemory   `json:"seeded_memories,omitempty"`
	SelectedContexts     []RankedContext  `json:"selected_contexts,omitempty"`
	Score                ContextEvalScore `json:"score"`
	LLMText              string           `json:"llm_text,omitempty"`
	AnswerContainsPassed bool             `json:"answer_contains_passed"`
	Error                string           `json:"error,omitempty"`
}

func NewContextLiveRunner(opts ContextLiveRunnerOptions) *ContextLiveRunner {
	if opts.UserID == "" {
		opts.UserID = "eval-user"
	}
	if opts.AppID == "" {
		opts.AppID = "mewcode-eval"
	}
	if opts.AgentID == "" {
		opts.AgentID = "main"
	}
	if opts.RunID == "" {
		opts.RunID = "eval-run"
	}
	opts.AppID = scopedEvalAppID(opts.AppID, opts.RunID)
	if opts.TopK == 0 {
		opts.TopK = 12
	}
	return &ContextLiveRunner{opts: opts}
}

func (r *ContextLiveRunner) RunCase(ctx context.Context, tc ContextEvalCase) ContextLiveEvalResult {
	start := time.Now()
	result := ContextLiveEvalResult{
		CaseID:               tc.CaseID,
		Runner:               "live",
		Mode:                 "context",
		Provider:             r.opts.Provider,
		Model:                r.opts.Model,
		SessionID:            r.sessionID(tc),
		AnswerContainsPassed: true,
	}

	if r.opts.Store == nil {
		return finishLiveResult(r.fail(result, "memory store is required"), start)
	}
	if r.opts.Client == nil {
		return finishLiveResult(r.fail(result, "llm client is required"), start)
	}
	if strings.TrimSpace(tc.Query) == "" && len(tc.Conversation) == 0 {
		return finishLiveResult(r.fail(result, "query or conversation is required for live context eval"), start)
	}

	seeded, err := r.seedMemories(ctx, tc, result.SessionID)
	if err != nil {
		result.SeededMemories = seeded
		return finishLiveResult(r.fail(result, err.Error()), start)
	}
	result.SeededMemories = seeded

	recorder := &recordingRecall{
		inner: memory.NewMemoryRecallService(memory.MemoryRecallOptions{
			Store:       r.opts.Store,
			Ranker:      memory.NewMemoryRanker(memory.DefaultRankerOptions()),
			AppID:       r.opts.AppID,
			TokenBudget: firstPositive(tc.TokenBudget, r.opts.TokenBudget),
			TopK:        r.opts.TopK,
			Threshold:   r.opts.Threshold,
			Rerank:      r.opts.Rerank,
		}),
		userID:      r.opts.UserID,
		appID:       r.opts.AppID,
		activePlan:  strings.Join(tc.ActivePlan, "\n"),
		recentTools: append([]string(nil), tc.RecentTools...),
		tokenBudget: firstPositive(tc.TokenBudget, r.opts.TokenBudget),
	}

	ag := agent.New(r.opts.Client, tools.NewRegistry(), r.opts.Protocol)
	ag.WorkDir = firstNonEmpty(tc.ProjectID, r.opts.ProjectID)
	ag.SessionID = result.SessionID
	ag.ContextWindow = firstPositive(r.opts.ContextWindow, 200000)
	ag.MaxOutputTokens = r.opts.MaxOutputTokens
	ag.MemoryRecall = recorder
	ag.MaxIterations = 1

	runCtx := ctx
	cancel := func() {}
	if r.opts.Timeout > 0 {
		runCtx, cancel = context.WithTimeout(ctx, r.opts.Timeout)
	}
	defer cancel()

	conv := buildEvalConversation(tc)
	for ev := range ag.Run(runCtx, conv) {
		switch e := ev.(type) {
		case agent.StreamText:
			result.LLMText += e.Text
		case agent.ErrorEvent:
			result.Error = e.Message
		}
	}

	result.SelectedContexts = rankedContextsFromBlocks(recorder.blocks)
	result.Score = ScoreContextEval(tc, result.SelectedContexts)
	result.AnswerContainsPassed = answerContains(result.LLMText, tc.ExpectedAnswerContains)
	return finishLiveResult(result, start)
}

func (r *ContextLiveRunner) fail(result ContextLiveEvalResult, message string) ContextLiveEvalResult {
	result.Error = message
	result.SelectedContexts = nil
	result.Score = ScoreContextEval(ContextEvalCase{CaseID: result.CaseID}, nil)
	result.AnswerContainsPassed = false
	return result
}

func finishLiveResult(result ContextLiveEvalResult, start time.Time) ContextLiveEvalResult {
	result.DurationMS = time.Since(start).Milliseconds()
	return result
}

func (r *ContextLiveRunner) seedMemories(ctx context.Context, tc ContextEvalCase, sessionID string) ([]SeededMemory, error) {
	var seeded []SeededMemory
	for _, seed := range tc.MemorySeeds {
		if strings.TrimSpace(seed.ID) == "" {
			return seeded, fmt.Errorf("memory seed id is required")
		}
		candidate, err := r.memoryCandidate(tc, seed, sessionID)
		if err != nil {
			return seeded, err
		}
		stored, err := r.opts.Store.Add(ctx, memory.AddMemoryRequest{Candidates: []memory.MemoryCandidate{candidate}})
		if err != nil {
			return seeded, err
		}
		physicalID := ""
		if len(stored) > 0 {
			physicalID = stored[0].ID
		}
		seeded = append(seeded, SeededMemory{LogicalID: seed.ID, PhysicalID: physicalID, Scope: string(candidate.Scope)})
	}
	return seeded, nil
}

func (r *ContextLiveRunner) memoryCandidate(tc ContextEvalCase, seed EvalMemorySeed, sessionID string) (memory.MemoryCandidate, error) {
	scope, ok := memory.ParseMemoryScope(firstNonEmpty(seed.Scope, "user"))
	if !ok {
		return memory.MemoryCandidate{}, fmt.Errorf("invalid memory seed scope %q", seed.Scope)
	}
	kind, ok := memory.ParseMemoryKind(firstNonEmpty(seed.Kind, "preference"))
	if !ok {
		return memory.MemoryCandidate{}, fmt.Errorf("invalid memory seed kind %q", seed.Kind)
	}
	meta := cloneMap(seed.Metadata)
	meta["eval_case_id"] = tc.CaseID
	meta["eval_logical_id"] = seed.ID
	meta["eval_run_id"] = r.opts.RunID
	return memory.MemoryCandidate{
		ID:         seed.ID,
		Scope:      scope,
		Kind:       kind,
		Memory:     seed.Memory,
		UserID:     firstNonEmpty(seed.UserID, r.opts.UserID),
		AppID:      firstNonEmpty(seed.AppID, r.opts.AppID),
		ProjectID:  firstNonEmpty(seed.ProjectID, tc.ProjectID, r.opts.ProjectID),
		AgentID:    firstNonEmpty(seed.AgentID, tc.AgentID, r.opts.AgentID),
		SessionID:  firstNonEmpty(seed.SessionID, sessionID),
		RunID:      firstNonEmpty(seed.RunID, r.opts.RunID),
		TeamName:   seed.TeamName,
		TaskID:     seed.TaskID,
		Confidence: 1,
		Metadata:   meta,
	}, nil
}

func (r *ContextLiveRunner) sessionID(tc ContextEvalCase) string {
	if r.opts.SessionID != "" {
		return r.opts.SessionID
	}
	return "eval-" + tc.CaseID
}

type recordingRecall struct {
	inner       *memory.MemoryRecallService
	userID      string
	appID       string
	activePlan  string
	recentTools []string
	tokenBudget int
	blocks      []memory.MemoryBlock
}

func (r *recordingRecall) Recall(ctx context.Context, req memory.RecallRequest) (memory.RecallResult, error) {
	req.UserID = firstNonEmpty(req.UserID, r.userID)
	req.AppID = firstNonEmpty(req.AppID, r.appID)
	req.ActivePlan = firstNonEmpty(req.ActivePlan, r.activePlan)
	if len(req.RecentTools) == 0 {
		req.RecentTools = append([]string(nil), r.recentTools...)
	}
	if req.TokenBudget == 0 {
		req.TokenBudget = r.tokenBudget
	}
	result, err := r.inner.Recall(ctx, req)
	r.blocks = append([]memory.MemoryBlock(nil), result.Blocks...)
	return result, err
}

func buildEvalConversation(tc ContextEvalCase) *conversation.Manager {
	conv := conversation.NewManager()
	for _, msg := range tc.Conversation {
		switch msg.Role {
		case "assistant":
			conv.AddAssistantMessage(msg.Content)
		default:
			conv.AddUserMessage(msg.Content)
		}
	}
	if conv.Len() == 0 && tc.Query != "" {
		conv.AddUserMessage(tc.Query)
	}
	return conv
}

func rankedContextsFromBlocks(blocks []memory.MemoryBlock) []RankedContext {
	selected := make([]RankedContext, 0, len(blocks))
	for index, block := range blocks {
		id := block.ID
		if block.Metadata != nil && block.Metadata["eval_logical_id"] != "" {
			id = block.Metadata["eval_logical_id"]
		}
		selected = append(selected, RankedContext{ID: id, Rank: index + 1, TokenCost: block.TokenCost})
	}
	return selected
}

func answerContains(text string, needles []string) bool {
	for _, needle := range needles {
		if needle != "" && !strings.Contains(text, needle) {
			return false
		}
	}
	return true
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if value != "" {
			return value
		}
	}
	return ""
}

func scopedEvalAppID(appID, runID string) string {
	if appID == "" {
		appID = "mewcode-eval"
	}
	if runID == "" || strings.Contains(appID, runID) {
		return appID
	}
	return strings.Trim(appID+"-"+runID, "-")
}

func firstPositive(values ...int) int {
	for _, value := range values {
		if value > 0 {
			return value
		}
	}
	return 0
}

func cloneMap(in map[string]string) map[string]string {
	out := make(map[string]string, len(in)+3)
	for key, value := range in {
		out[key] = value
	}
	return out
}
