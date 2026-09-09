package orchestration

import (
	"fmt"
	"path/filepath"
)

const LeadName = "lead"

type Orchestrator struct {
	TeamName string
	Board    *TaskBoard
	Mail     *MailRouter
	Sessions *SessionStore
	Monitor  *Monitor
}

func NewFileOrchestrator(teamName, baseDir string) *Orchestrator {
	return &Orchestrator{
		TeamName: teamName,
		Board:    NewTaskBoard(filepath.Join(baseDir, "board.json")),
		Sessions: NewSessionStore(filepath.Join(baseDir, "sessions")),
		Monitor:  NewMonitor(filepath.Join(baseDir, "events.jsonl")),
	}
}

func (o *Orchestrator) StartDelegation(parent, child AgentRef, taskID string) (CollaborationSession, error) {
	if o == nil || o.Sessions == nil {
		return CollaborationSession{}, fmt.Errorf("session store is required")
	}
	session, err := o.Sessions.CreateSession(CollaborationSession{
		TeamName:     o.TeamName,
		Mode:         ModeDelegation,
		Parent:       parent,
		Participants: []AgentRef{child},
		TaskBoardID:  taskID,
	})
	if err == nil {
		o.record(EventSessionCreated, parent.Name, "delegation session created", map[string]string{"session_id": session.ID, "task_id": taskID})
	}
	return session, err
}

func (o *Orchestrator) StartPeerSession(parent AgentRef, participants []AgentRef) (CollaborationSession, error) {
	if o == nil || o.Sessions == nil {
		return CollaborationSession{}, fmt.Errorf("session store is required")
	}
	session, err := o.Sessions.CreateSession(CollaborationSession{
		TeamName:     o.TeamName,
		Mode:         ModePeer,
		Parent:       parent,
		Participants: append([]AgentRef(nil), participants...),
	})
	if err == nil {
		o.record(EventSessionCreated, parent.Name, "peer session created", map[string]string{"session_id": session.ID})
	}
	return session, err
}

func (o *Orchestrator) CreateBoardTask(task BoardTask) (BoardTask, error) {
	if o == nil || o.Board == nil {
		return BoardTask{}, fmt.Errorf("task board is required")
	}
	created, err := o.Board.CreateTask(task)
	if err == nil {
		o.record(EventTaskCreated, LeadName, "task created", map[string]string{"task_id": created.ID})
	}
	return created, err
}

func (o *Orchestrator) AssignBoardTask(taskID, owner string) error {
	if o == nil || o.Board == nil || o.Mail == nil {
		return fmt.Errorf("task board and mail router are required")
	}
	if err := o.Board.ClaimTask(taskID, owner); err != nil {
		return err
	}
	o.record(EventTaskClaimed, owner, "task claimed", map[string]string{"task_id": taskID})
	task, _, _ := o.Board.GetTask(taskID)
	err := o.Mail.SendEnvelope(MailEnvelope{
		Kind:     MailKindAssignment,
		From:     LeadName,
		To:       owner,
		TeamName: o.TeamName,
		TaskID:   taskID,
		Text:     FormatEnvelopeText(MailEnvelope{Kind: MailKindAssignment, TaskID: taskID, Text: task.Description}),
		Metadata: map[string]string{"title": task.Title},
	})
	if err == nil {
		o.record(EventMailSent, LeadName, "assignment sent", map[string]string{"task_id": taskID, "to": owner})
	}
	return err
}

func (o *Orchestrator) CompleteBoardTask(taskID, actor, result string) error {
	if o == nil || o.Board == nil || o.Mail == nil {
		return fmt.Errorf("task board and mail router are required")
	}
	if err := o.Board.UpdateTask(taskID, actor, BoardTaskDone, "task completed", result); err != nil {
		return err
	}
	o.record(EventTaskUpdated, actor, "task completed", map[string]string{"task_id": taskID})
	err := o.Mail.SendEnvelope(MailEnvelope{
		Kind:     MailKindBoardUpdate,
		From:     actor,
		To:       LeadName,
		TeamName: o.TeamName,
		TaskID:   taskID,
		Text:     FormatEnvelopeText(MailEnvelope{Kind: MailKindBoardUpdate, TaskID: taskID, Text: "completed"}),
	})
	if err == nil {
		o.record(EventMailSent, actor, "board update sent", map[string]string{"task_id": taskID, "to": LeadName})
	}
	return err
}

func (o *Orchestrator) record(eventType OrchestrationEventType, actor, summary string, metadata map[string]string) {
	if o == nil || o.Monitor == nil {
		return
	}
	_ = o.Monitor.AppendEvent(OrchestrationEvent{
		Type:     eventType,
		Actor:    actor,
		Summary:  summary,
		Metadata: metadata,
	})
}
