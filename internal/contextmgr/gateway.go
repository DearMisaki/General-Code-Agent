package contextmgr

import (
	"context"
	"fmt"
	"time"

	"mewcode/internal/conversation"
)

type GatewayOptions struct {
	Builder  *Builder
	Renderer *Renderer
	Budgeter *Budgeter
	Audit    *AuditWriter
}

type ContextGateway struct {
	builder  *Builder
	renderer *Renderer
	budgeter *Budgeter
	audit    *AuditWriter
}

func NewGateway(opts GatewayOptions) *ContextGateway {
	g := &ContextGateway{
		builder:  opts.Builder,
		renderer: opts.Renderer,
		budgeter: opts.Budgeter,
		audit:    opts.Audit,
	}
	if g.builder == nil {
		g.builder = NewBuilder()
	}
	if g.renderer == nil {
		g.renderer = NewRenderer()
	}
	if g.budgeter == nil {
		g.budgeter = NewBudgeter()
	}
	return g
}

func (g *ContextGateway) PrepareTurn(ctx context.Context, req PrepareRequest) (PreparedTurn, error) {
	snap := g.builder.Build(req)

	renderConv := conversation.NewManager()
	if req.Conversation != nil {
		renderConv.AppendMessages(req.Conversation.GetMessages())
	}
	rendered := g.renderer.Render(req, snap)
	for _, reminder := range rendered {
		renderConv.AddSystemReminder(reminder)
	}

	budget, err := g.budgeter.PrepareBudget(ctx, BudgetRequest{
		Conversation:     renderConv,
		Client:           req.Client,
		WorkDir:          req.WorkDir,
		SessionID:        req.SessionID,
		ContextWindow:    req.ContextWindow,
		MaxOutputTokens:  req.MaxOutputTokens,
		ReplacementState: req.ReplacementState,
		Recovery:         req.Recovery,
		ToolSchemas:      req.ToolSchemas,
		UsageAnchor:      req.UsageAnchor,
		CompactTracking:  req.CompactTracking,
	})
	if err != nil {
		return PreparedTurn{}, err
	}

	records := []AuditRecord{{
		ID:        auditID("prepare"),
		Time:      time.Now().UTC(),
		Event:     EventContextPrepare,
		AgentID:   req.AgentID,
		SessionID: req.SessionID,
		ContextID: snap.ID,
		Summary:   "prepared context turn",
		Metadata:  map[string]any{"messages": snap.Session.MessageCount},
	}}
	if len(rendered) > 0 {
		records = append(records, AuditRecord{
			ID:        auditID("render"),
			Time:      time.Now().UTC(),
			Event:     EventContextRender,
			AgentID:   req.AgentID,
			SessionID: req.SessionID,
			ContextID: snap.ID,
			Summary:   "rendered context reminders",
			Metadata:  map[string]any{"count": len(rendered)},
		})
	}
	if len(req.DeferredToolNames) > 0 {
		records = append(records, AuditRecord{
			ID:        auditID("tool-exposure"),
			Time:      time.Now().UTC(),
			Event:     EventContextToolExposure,
			AgentID:   req.AgentID,
			SessionID: req.SessionID,
			ContextID: snap.ID,
			Summary:   "deferred tools exposed",
			Metadata:  map[string]any{"count": len(req.DeferredToolNames)},
		})
	}
	if budget.CompactMessage != "" {
		records = append(records, AuditRecord{
			ID:        auditID("compact"),
			Time:      time.Now().UTC(),
			Event:     EventContextCompact,
			AgentID:   req.AgentID,
			SessionID: req.SessionID,
			ContextID: snap.ID,
			Summary:   budget.CompactMessage,
		})
	}
	if len(budget.ToolResultRecords) > 0 {
		records = append(records, AuditRecord{
			ID:        auditID("budget"),
			Time:      time.Now().UTC(),
			Event:     EventContextToolResultBudget,
			AgentID:   req.AgentID,
			SessionID: req.SessionID,
			ContextID: snap.ID,
			Summary:   "tool result replacements applied",
			Metadata:  map[string]any{"count": len(budget.ToolResultRecords)},
		})
	}

	for _, record := range records {
		if g.audit != nil {
			_ = g.audit.Append(record)
		}
	}

	return PreparedTurn{
		Snapshot:         snap,
		APIConversation:  budget.APIConversation,
		ToolSchemas:      req.ToolSchemas,
		AuditRecords:     records,
		UsageAnchorReset: budget.UsageAnchorReset,
		CompactTracking:  budget.CompactTracking,
	}, nil
}

func auditID(suffix string) string {
	return fmt.Sprintf("%d-%s", time.Now().UnixNano(), suffix)
}
