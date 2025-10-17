package transaction

import "fmt"

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
	return &Transaction{}
}

// Add 添加动作
func (t *Transaction) Add(name string, do, undo func() error) {
	t.actions = append(t.actions, &Action{
		Name: name,
		Do:   do,
		Undo: undo,
	})
}

// Run 执行事务
func (t *Transaction) Run() error {
	t.executed = []*Action{}
	for _, act := range t.actions {
		if err := act.Do(); err != nil {
			// 出错回滚
			t.Rollback()
			return fmt.Errorf("执行动作 '%s' 失败: %v", act.Name, err)
		}
		t.executed = append(t.executed, act)
	}
	return nil
}

// Rollback 回滚已执行的动作
func (t *Transaction) Rollback() {
	for i := len(t.executed) - 1; i >= 0; i-- {
		act := t.executed[i]
		if err := act.Undo(); err != nil {
			fmt.Printf("回滚动作 '%s' 失败: %v\n", act.Name, err)
		}
	}
	t.executed = nil
}

// Cleanup 清理事务（未执行的动作清理）
func (t *Transaction) Cleanup() {
	t.actions = nil
	t.executed = nil
}
