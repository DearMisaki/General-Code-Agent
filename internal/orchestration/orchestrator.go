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
}

func NewFileOrchestrator(teamName, baseDir string) *Orchestrator {
	return &Orchestrator{
		TeamName: teamName,
		Board:    NewTaskBoard(filepath.Join(baseDir, "board.json")),
		Sessions: NewSessionStore(filepath.Join(baseDir, "sessions")),
	}
}

func (o *Orchestrator) StartDelegation(parent, child AgentRef, taskID string) (CollaborationSession, error) {
	if o == nil || o.Sessions == nil {
		return CollaborationSession{}, fmt.Errorf("session store is required")
	}
	return o.Sessions.CreateSession(CollaborationSession{
		TeamName:     o.TeamName,
		Mode:         ModeDelegation,
		Parent:       parent,
		Participants: []AgentRef{child},
		TaskBoardID:  taskID,
	})
}

func (o *Orchestrator) StartPeerSession(parent AgentRef, participants []AgentRef) (CollaborationSession, error) {
	if o == nil || o.Sessions == nil {
		return CollaborationSession{}, fmt.Errorf("session store is required")
	}
	return o.Sessions.CreateSession(CollaborationSession{
		TeamName:     o.TeamName,
		Mode:         ModePeer,
		Parent:       parent,
		Participants: append([]AgentRef(nil), participants...),
	})
}

func (o *Orchestrator) CreateBoardTask(task BoardTask) (BoardTask, error) {
	if o == nil || o.Board == nil {
		return BoardTask{}, fmt.Errorf("task board is required")
	}
	return o.Board.CreateTask(task)
}

func (o *Orchestrator) AssignBoardTask(taskID, owner string) error {
	if o == nil || o.Board == nil || o.Mail == nil {
		return fmt.Errorf("task board and mail router are required")
	}
	if err := o.Board.ClaimTask(taskID, owner); err != nil {
		return err
	}
	task, _, _ := o.Board.GetTask(taskID)
	return o.Mail.SendEnvelope(MailEnvelope{
		Kind:     MailKindAssignment,
		From:     LeadName,
		To:       owner,
		TeamName: o.TeamName,
		TaskID:   taskID,
		Text:     FormatEnvelopeText(MailEnvelope{Kind: MailKindAssignment, TaskID: taskID, Text: task.Description}),
		Metadata: map[string]string{"title": task.Title},
	})
}

func (o *Orchestrator) CompleteBoardTask(taskID, actor, result string) error {
	if o == nil || o.Board == nil || o.Mail == nil {
		return fmt.Errorf("task board and mail router are required")
	}
	if err := o.Board.UpdateTask(taskID, actor, BoardTaskDone, "task completed", result); err != nil {
		return err
	}
	return o.Mail.SendEnvelope(MailEnvelope{
		Kind:     MailKindBoardUpdate,
		From:     actor,
		To:       LeadName,
		TeamName: o.TeamName,
		TaskID:   taskID,
		Text:     FormatEnvelopeText(MailEnvelope{Kind: MailKindBoardUpdate, TaskID: taskID, Text: "completed"}),
	})
}
