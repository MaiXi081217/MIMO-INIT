package grub

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"resourcemgr/internal/transaction"
)

// RegisterGrubActions 将 GRUB 修改和 initramfs 消息加入事务
func RegisterGrubActions(txn *transaction.Transaction) error {
	grubFile := "/etc/default/grub"
	orig := []byte{}
	if b, err := os.ReadFile(grubFile); err == nil {
		orig = b
	}
	// 修改 GRUB
	txn.Add("modify /etc/default/grub",
		func() error {
			// 读取、替换行
			data := string(orig)
			lines := strings.Split(data, "\n")
			found := false
			for i, l := range lines {
				if strings.HasPrefix(l, "GRUB_CMDLINE_LINUX_DEFAULT") {
					lines[i] = `GRUB_CMDLINE_LINUX_DEFAULT="quiet loglevel=0"`
					found = true
				}
			}
			if !found {
				lines = append(lines, `GRUB_CMDLINE_LINUX_DEFAULT="quiet loglevel=0"`)
			}
			if err := os.WriteFile(grubFile, []byte(strings.Join(lines, "\n")), 0644); err != nil {
				return err
			}
			// 更新 grub 配置
			if out, err := exec.Command("update-grub").CombinedOutput(); err != nil {
				return fmt.Errorf("update-grub failed: %v, out: %s", err, out)
			}
			return nil
		},
		func() error {
			// 恢复原始 grub
			if len(orig) == 0 {
				// 无原始文件则不删除
				return nil
			}
			_ = os.WriteFile(grubFile, orig, 0644)
			exec.Command("update-grub").Run()
			return nil
		},
	)

	// initramfs script
	initPath := "/etc/initramfs-tools/scripts/init-top/mimo-msg"
	origInit := []byte{}
	if b, err := os.ReadFile(initPath); err == nil {
		origInit = b
	}
	content := `#!/bin/sh
echo ">>> Please wait, initializing MIMO Live Server (initramfs stage) <<<" > /dev/console
`
	txn.Add("add initramfs mimo-msg",
		func() error {
			if err := os.MkdirAll(filepath.Dir(initPath), 0755); err != nil {
				return err
			}
			if err := os.WriteFile(initPath, []byte(content), 0755); err != nil {
				return err
			}
			if out, err := exec.Command("update-initramfs", "-u").CombinedOutput(); err != nil {
				return fmt.Errorf("update-initramfs failed: %v, out: %s", err, out)
			}
			return nil
		},
		func() error {
			// 恢复或删除
			if len(origInit) > 0 {
				_ = os.WriteFile(initPath, origInit, 0755)
			} else {
				_ = os.Remove(initPath)
			}
			exec.Command("update-initramfs", "-u").Run()
			return nil
		},
	)
	return nil
}
