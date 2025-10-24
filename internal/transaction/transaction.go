package transaction

import (
	"fmt"
	"strings"
)

// Action 定义单个事务动作
type Action struct {
	Name string
	Do   func() error
	Undo func() error
}

// Transaction 管理多个动作
type Transaction struct {
	actions  []*Action
	executed []*Action
}

// New 创建事务
func New() *Transaction {
	return &Transaction{
		actions:  make([]*Action, 0),
		executed: make([]*Action, 0),
	}
}

// Add 添加动作
func (t *Transaction) Add(a *Action) {
	if a == nil {
		return
	}
	t.actions = append(t.actions, a)
}

// Run 执行事务，遇到错误则回滚已执行的动作并返回错误
func (t *Transaction) Run() error {
	t.executed = t.executed[:0]
	for _, a := range t.actions {
		if a == nil || a.Do == nil {
			continue
		}
		if err := a.Do(); err != nil {
			// 回滚已执行动作
			_ = t.Rollback()
			return fmt.Errorf("action %q failed: %w", a.Name, err)
		}
		t.executed = append(t.executed, a)
	}
	return nil
}

// Rollback 回滚已执行的动作（按相反顺序），返回合并错误或 nil
func (t *Transaction) Rollback() error {
	var errs []string
	for i := len(t.executed) - 1; i >= 0; i-- {
		a := t.executed[i]
		if a == nil || a.Undo == nil {
			continue
		}
		if err := a.Undo(); err != nil {
			errs = append(errs, fmt.Sprintf("%s: %v", a.Name, err))
		}
	}
	// 清空已执行列表以避免重复回滚
	t.executed = t.executed[:0]
	if len(errs) > 0 {
		return fmt.Errorf("rollback errors: %s", strings.Join(errs, "; "))
	}
	return nil
}

// Cleanup 清理事务内部状态（释放引用）
func (t *Transaction) Cleanup() {
	t.actions = nil
	t.executed = nil
}
