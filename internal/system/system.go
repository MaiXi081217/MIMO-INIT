package system

import (
	"fmt"
	"os"
	"os/exec"
)

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
