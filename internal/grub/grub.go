package grub

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"

	"resourcemgr/internal/transaction"
)

// RegisterGrubAndInitActions 将 GRUB 修改和 initramfs 脚本加入事务，GRUB 静默启动，initramfs 保留提示
func RegisterGrubAndInitActions(txn *transaction.Transaction) error {
	// -------------------------------
	// 1. GRUB 修改
	// -------------------------------
	grubFile := "/etc/default/grub"
	origGrub := []byte{}
	if b, err := os.ReadFile(grubFile); err == nil {
		origGrub = b
	}

	txn.Add("modify GRUB_CMDLINE_LINUX_DEFAULT",
		func() error {
			data := string(origGrub)
			re := regexp.MustCompile(`(?m)^GRUB_CMDLINE_LINUX_DEFAULT=.*$`)
			newLine := `GRUB_CMDLINE_LINUX_DEFAULT="quiet loglevel=0 systemd.show_status=0"`

			if re.MatchString(data) {
				data = re.ReplaceAllString(data, newLine)
			} else {
				data += "\n" + newLine + "\n"
			}

			if err := os.WriteFile(grubFile, []byte(data), 0644); err != nil {
				return fmt.Errorf("写入 grub 文件失败: %v", err)
			}

			fmt.Println("[INFO] /etc/default/grub 已修改，执行 update-grub")
			cmd := exec.Command("update-grub")
			if out, err := cmd.CombinedOutput(); err != nil {
				return fmt.Errorf("update-grub 执行失败: %v, out: %s", err, out)
			}
			fmt.Println("[INFO] GRUB 更新完成")
			return nil
		},
		func() error {
			if len(origGrub) > 0 {
				if err := os.WriteFile(grubFile, origGrub, 0644); err != nil {
					return fmt.Errorf("恢复 grub 文件失败: %v", err)
				}
				exec.Command("update-grub").Run()
			}
			return nil
		},
	)

	// -------------------------------
	// 2. initramfs 脚本（保留初始化提示信息）
	// -------------------------------
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
			// 回滚 initramfs 脚本
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
