package memoryruntime

import (
	"context"
	"path/filepath"

	"mewcode/internal/agent"
	"mewcode/internal/contextmgr"
	"mewcode/internal/conversation"
	"mewcode/internal/llm"
	"mewcode/internal/memory"
	"mewcode/internal/memory/extractor"
	"mewcode/internal/teams"
	"mewcode/internal/tools"
)

type BuildOptions struct {
	Config       memory.MemoryConfig
	Client       llm.Client
	Registry     *tools.Registry
	Conversation *conversation.Manager
	ProjectRoot  string
	Protocol     string
	UserID       string
	AgentID      string
	SessionID    string
	RunID        string
	TeamName     string
	TaskID       string
	AuditPath    string
	AppendSystem func(string)
	DebugLogf    func(format string, args ...any)
}

type Runtime struct {
	Store      memory.MemoryStore
	Pipeline   *memory.MemoryWritePipeline
	Recall     *memory.MemoryRecallService
	Extraction *memory.MemoryExtractionScheduler

	extractor      *extractor.Extractor
	drainTimeoutMs int
}

func Build(opts BuildOptions) (*Runtime, error) {
	lt := opts.Config.LongTerm
	appID := firstNonEmpty(lt.AppID, "mewcode")
	rt := &Runtime{drainTimeoutMs: lt.Extraction.DrainTimeoutMs}
	if !lt.Enabled {
		return rt, nil
	}
	store, err := memory.NewMemoryStoreFromConfig(opts.Config)
	if err != nil {
		return nil, err
	}
	rt.Store = store
	audit := memory.NewMemoryAuditWriter(defaultAuditPath(opts.ProjectRoot, opts.AuditPath))
	rt.Pipeline = memory.NewMemoryWritePipeline(memory.MemoryWritePipelineOptions{
		Store:  store,
		Policy: memory.NewMemoryPolicyFilter(memory.PolicyOptions{MinConfidence: lt.Extraction.MinConfidence}),
		Audit:  audit,
		AppID:  appID,
	})
	if lt.Recall.Enabled {
		rt.Recall = memory.NewMemoryRecallService(memory.MemoryRecallOptions{
			Store:       store,
			Ranker:      memory.NewMemoryRanker(memory.DefaultRankerOptions()),
			AppID:       appID,
			TokenBudget: lt.Recall.TokenBudget,
			TopK:        lt.Recall.TopK,
			Threshold:   lt.Recall.Threshold,
			Rerank:      lt.Recall.Rerank,
			Audit:       audit,
		})
	}
	if lt.Extraction.Enabled && opts.Client != nil && opts.Registry != nil && opts.Conversation != nil {
		rt.extractor = extractor.InitExtractMemories(extractor.Deps{
			MemoryDir:     memory.GetAutoMemPath(opts.ProjectRoot),
			UserMemoryDir: memory.GetUserAutoMemPath(),
			ProjectRoot:   opts.ProjectRoot,
			Client:        opts.Client,
			ToolRegistry:  opts.Registry,
			Protocol:      opts.Protocol,
			Conversation:  opts.Conversation,
			AppendSystem:  opts.AppendSystem,
			DebugLogf:     opts.DebugLogf,
			CandidateMode: true,
			WritePipeline: rt.Pipeline,
			Audit:         audit,
			ExtractionScope: extractor.MemoryExtractionScope{
				UserID:    opts.UserID,
				AppID:     appID,
				ProjectID: opts.ProjectRoot,
				AgentID:   firstNonEmpty(opts.AgentID, "main"),
				SessionID: opts.SessionID,
				RunID:     opts.RunID,
				TeamName:  opts.TeamName,
				TaskID:    opts.TaskID,
			},
		})
		rt.Extraction = memory.NewMemoryExtractionScheduler(memory.MemoryExtractionSchedulerOptions{
			Extract: rt.extractor.Execute,
		})
	}
	return rt, nil
}

func (r *Runtime) AttachAgent(ag *agent.Agent) {
	if r == nil || ag == nil {
		return
	}
	if ag.ContextGateway == nil {
		ag.ContextGateway = contextmgr.NewGateway(contextmgr.GatewayOptions{})
	}
	if ag.ContextLifecycle == nil {
		ag.ContextLifecycle = contextmgr.NewLifecycleManager(nil, contextmgr.WrapRecoveryState(ag.RecoveryState))
	}
	if r.Recall != nil {
		ag.MemoryRecall = r.Recall
	}
	if r.Extraction != nil {
		ag.MemoryExtraction = r.Extraction
	}
}

func (r *Runtime) AttachTeam(team *teams.Team) {
	if r == nil || team == nil || r.Pipeline == nil {
		return
	}
	team.MemoryPipeline = r.Pipeline
}

func (r *Runtime) Drain(ctx context.Context) error {
	if r == nil {
		return nil
	}
	if r.Extraction != nil {
		if err := r.Extraction.Drain(r.drainTimeoutMs); err != nil {
			return err
		}
	}
	if r.extractor != nil {
		return r.extractor.Drain(r.drainTimeoutMs)
	}
	return nil
}

func defaultAuditPath(projectRoot, explicit string) string {
	if explicit != "" {
		return explicit
	}
	if projectRoot == "" {
		projectRoot = "."
	}
	return filepath.Join(projectRoot, ".mewcode", "memory", "audit.jsonl")
}

func firstNonEmpty(values ...string) string {
	for _, v := range values {
		if v != "" {
			return v
		}
	}
	return ""
}
