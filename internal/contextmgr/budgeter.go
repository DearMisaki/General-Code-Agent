package contextmgr

import (
	"context"

	"mewcode/internal/compact"
	"mewcode/internal/conversation"
	"mewcode/internal/llm"
	"mewcode/internal/toolresult"
)

type Budgeter struct{}

func NewBudgeter() *Budgeter { return &Budgeter{} }

type BudgetRequest struct {
	Conversation     *conversation.Manager
	Client           llm.Client
	WorkDir          string
	SessionID        string
	ContextWindow    int
	MaxOutputTokens  int
	ReplacementState *toolresult.ContentReplacementState
	Recovery         *compact.RecoveryState
	ToolSchemas      []map[string]any
	UsageAnchor      compact.UsageAnchor
	CompactTracking  compact.AutoCompactTrackingState
}

type BudgetResult struct {
	APIConversation   *conversation.Manager
	ToolResultRecords []toolresult.Record
	CompactMessage    string
	CompactTracking   compact.AutoCompactTrackingState
	UsageAnchorReset  bool
}

func (b *Budgeter) PrepareBudget(ctx context.Context, req BudgetRequest) (BudgetResult, error) {
	tracking := req.CompactTracking
	var compactMessage string
	var usageAnchorReset bool

	if req.Client != nil && req.Conversation != nil {
		msg, err := compact.ManageContext(ctx, req.Conversation, req.Client, req.WorkDir, req.SessionID, req.ContextWindow, req.MaxOutputTokens, &tracking, req.Recovery, req.ToolSchemas, req.UsageAnchor)
		if err != nil {
			return BudgetResult{CompactTracking: tracking}, err
		}
		if msg != "" {
			compactMessage = msg
			usageAnchorReset = true
		}
	}

	state := req.ReplacementState
	if state == nil {
		state = toolresult.New()
	}
	apiConv, records, err := toolresult.Apply(req.Conversation, req.WorkDir, state)
	if err != nil {
		return BudgetResult{CompactTracking: tracking}, err
	}
	if len(records) > 0 {
		_ = toolresult.AppendRecords(req.WorkDir, records)
	}

	return BudgetResult{
		APIConversation:   apiConv,
		ToolResultRecords: records,
		CompactMessage:    compactMessage,
		CompactTracking:   tracking,
		UsageAnchorReset:  usageAnchorReset,
	}, nil
}

func (b *Budgeter) ForceCompact(ctx context.Context, req BudgetRequest) (string, error) {
	if req.Client == nil {
		return "", nil
	}
	return compact.ForceCompact(ctx, req.Conversation, req.Client, req.WorkDir, req.SessionID, req.ContextWindow, req.Recovery, req.ToolSchemas)
}
