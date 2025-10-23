package motd

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"resourcemgr/internal/transaction"
)

// RegisterMOTDActions 注册备份并清空 /etc/update-motd.d
func RegisterMOTDActions(txn *transaction.Transaction) error {
	srcDir := "/etc/update-motd.d"
	backupDir := fmt.Sprintf("/root/motd_backup_%d", time.Now().Unix())

	do := func() error {
		if _, err := os.Stat(srcDir); os.IsNotExist(err) {
			// 不存在视为成功
			return nil
		}
		if err := os.MkdirAll(backupDir, 0700); err != nil {
			return err
		}
		entries, err := os.ReadDir(srcDir)
		if err != nil {
			return err
		}
		for _, e := range entries {
			src := filepath.Join(srcDir, e.Name())
			dst := filepath.Join(backupDir, e.Name())
			if err := os.Rename(src, dst); err != nil {
				return err
			}
		}
		return nil
	}
	undo := func() error {
		// 恢复备份
		entries, err := os.ReadDir(backupDir)
		if err != nil {
			return nil
		}
		for _, e := range entries {
			src := filepath.Join(backupDir, e.Name())
			dst := filepath.Join(srcDir, e.Name())
			_ = os.Rename(src, dst)
		}
		_ = os.RemoveAll(backupDir)
		return nil
	}
	txn.Add("backup_and_clear_motd", do, undo)
	return nil
}

// disableMotd removes all files under /etc/update-motd.d using pure Go
func DisableMotd() error {
	dir := "/etc/update-motd.d"

	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			fmt.Printf("[WARN] %s does not exist, skipping motd cleanup\n", dir)
			return nil
		}
		return fmt.Errorf("failed to read %s: %v", dir, err)
	}

	for _, entry := range entries {
		filePath := filepath.Join(dir, entry.Name())
		if err := os.Remove(filePath); err != nil {
			return fmt.Errorf("failed to remove %s: %v", filePath, err)
		}
	}
	return nil
}
