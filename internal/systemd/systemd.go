package systemd

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"resourcemgr/internal/fileops"
)

// EnableServices 启用 .service 文件，不阻塞
func EnableServices(cfg *fileops.Config) error {
	var services []string

	// 先收集存在的 .service 文件
	for _, m := range cfg.FileMappings {
		dst := m.Dst
		if filepath.Ext(dst) != ".service" {
			continue
		}

		// 检查文件是否存在
		if _, err := os.Stat(dst); os.IsNotExist(err) {
			fmt.Printf("[WARN] systemd 服务文件不存在，跳过: %s\n", dst)
			continue
		}

		services = append(services, dst)
	}

	if len(services) == 0 {
		fmt.Println("[INFO] 没有需要启用的 systemd 服务")
		return nil
	}

	// daemon-reload 一次即可
	fmt.Println("[INFO] 执行 daemon-reload")
	cmd := exec.Command("systemctl", "daemon-reload")
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("daemon-reload 失败: %v", err)
	}

	// 启用并启动服务
	for _, s := range services {
		fmt.Printf("[INFO] 启用 systemd 服务: %s\n", s)

		// 启用服务
		cmdEnable := exec.Command("systemctl", "enable", s)
		if err := cmdEnable.Run(); err != nil {
			fmt.Printf("[WARN] enable %s 失败: %v\n", s, err)
		}

		// 启动服务（非阻塞）
		cmdStart := exec.Command("systemctl", "start", s)
		if err := cmdStart.Start(); err != nil {
			fmt.Printf("[WARN] start %s 失败: %v\n", s, err)
		} else {
			fmt.Printf("[INFO] systemd 服务已启动: %s\n", s)
		}
	}

	return nil
}
