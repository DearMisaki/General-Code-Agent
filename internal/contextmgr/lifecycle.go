package contextmgr

import "context"

type LifecycleManager struct {
	budgeter *Budgeter
	recovery *RecoveryTracker
}

func NewLifecycleManager(budgeter *Budgeter, recovery *RecoveryTracker) *LifecycleManager {
	if budgeter == nil {
		budgeter = NewBudgeter()
	}
	if recovery == nil {
		recovery = NewRecoveryTracker()
	}
	return &LifecycleManager{budgeter: budgeter, recovery: recovery}
}

func (m *LifecycleManager) Recovery() *RecoveryTracker {
	if m == nil {
		return nil
	}
	return m.recovery
}

func (m *LifecycleManager) Clear() {
	if m == nil {
		return
	}
	m.recovery = NewRecoveryTracker()
}

func (m *LifecycleManager) ForceCompact(ctx context.Context, req BudgetRequest) (string, error) {
	if m == nil {
		return "", nil
	}
	return m.budgeter.ForceCompact(ctx, req)
}
