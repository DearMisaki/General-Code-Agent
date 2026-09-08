package contextmgr

import "mewcode/internal/compact"

type RecoveryTracker struct {
	state *compact.RecoveryState
}

func NewRecoveryTracker() *RecoveryTracker {
	return &RecoveryTracker{state: compact.NewRecoveryState()}
}

func WrapRecoveryState(state *compact.RecoveryState) *RecoveryTracker {
	if state == nil {
		state = compact.NewRecoveryState()
	}
	return &RecoveryTracker{state: state}
}

func (r *RecoveryTracker) CompactState() *compact.RecoveryState {
	if r == nil {
		return nil
	}
	return r.state
}

func (r *RecoveryTracker) RecordFileRead(path, content string) {
	if r == nil || r.state == nil {
		return
	}
	r.state.RecordFileRead(path, content)
}

func (r *RecoveryTracker) RecordSkillInvocation(name, body string) {
	if r == nil || r.state == nil {
		return
	}
	r.state.RecordSkillInvocation(name, body)
}

func (r *RecoveryTracker) BuildAttachment(toolSchemas []map[string]any) string {
	if r == nil {
		return ""
	}
	return compact.BuildRecoveryAttachment(r.state, toolSchemas)
}
