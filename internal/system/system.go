package system

import (
	"fmt"
	"os"
	"os/exec"
	"regexp"
)

// 修改 GRUB 内核参数
func UpdateGRUBKernelCmdline() error {
	grubFile := "/etc/default/grub"

	data, err := os.ReadFile(grubFile)
	if err != nil {
		return fmt.Errorf("读取 grub 文件失败: %v", err)
	}

	content := string(data)
	re := regexp.MustCompile(`(?m)^GRUB_CMDLINE_LINUX_DEFAULT=.*$`)
	newLine := `GRUB_CMDLINE_LINUX_DEFAULT="quiet loglevel=0"`

	if re.MatchString(content) {
		content = re.ReplaceAllString(content, newLine)
	} else {
		content += "\n" + newLine + "\n"
	}

	if err := os.WriteFile(grubFile, []byte(content), 0644); err != nil {
		return fmt.Errorf("写入 grub 文件失败: %v", err)
	}

	fmt.Println("[INFO] /etc/default/grub 已修改，执行 update-grub")
	cmd := exec.Command("update-grub")
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("update-grub 执行失败: %v", err)
	}

	fmt.Println("[INFO] GRUB 更新完成")
	return nil
}

// 禁用 cloud-init
func DisableCloudInit() error {
	// 停止并禁用服务
	services := []string{
		"cloud-init",
		"cloud-final",
		"cloud-config",
		"cloud-init-local",
	}

	for _, s := range services {
		fmt.Printf("[INFO] 停止 cloud-init 服务: %s\n", s)
		cmd := exec.Command("systemctl", "stop", s)
		cmd.Run() // 忽略错误
		cmd = exec.Command("systemctl", "disable", s)
		cmd.Run()
	}

	// 创建标记文件
	f, err := os.Create("/etc/cloud/cloud-init.disabled")
	if err != nil {
		return fmt.Errorf("创建 cloud-init.disabled 失败: %v", err)
	}
	f.Close()

	fmt.Println("[INFO] cloud-init 已被禁用，系统下次启动时不会执行")
	return nil
}
