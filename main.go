package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"

	"resourcemgr/internal/decompress"
	"resourcemgr/internal/fileops"
	"resourcemgr/internal/system"
	"resourcemgr/internal/systemd"
	"resourcemgr/internal/transaction"
)

func disableMotd() error {
	dir := "/etc/update-motd.d"

	// 检查目录是否存在
	if _, err := os.Stat(dir); os.IsNotExist(err) {
		fmt.Printf("[WARN] %s 不存在，跳过清理 motd\n", dir)
		return nil
	}

	// 删除目录下所有文件（保留目录本身）
	cmd := exec.Command("bash", "-c", fmt.Sprintf("rm -f %s/*", dir))
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("删除 %s 下文件失败: %v, output: %s", dir, err, string(output))
	}

	fmt.Println("[INFO] 已删除 /etc/update-motd.d 下所有文件")
	return nil
}

func main() {
	help := flag.Bool("help", false, "显示帮助信息")
	flag.Parse()
	if *help {
		fmt.Println("用法: mimo-update --help")
		return
	}

	// 检查 root 权限
	if os.Geteuid() != 0 {
		log.Fatalf("[ERROR] 必须以 root 权限运行")
	}

	fmt.Println("[INFO] 一键升级模式启动...")

	destDir := "/tmp/mimo-output"

	// -------------------------------
	// 步骤 1: 解压资源
	// -------------------------------
	fmt.Println("[STEP 1/8] 正在解压嵌入资源...")
	if err := decompress.ExtractResources(destDir); err != nil {
		log.Fatalf("[ERROR] 解压嵌入资源失败: %v", err)
	}
	fmt.Println("[STEP 1/8] 解压完成")

	// -------------------------------
	// 步骤 2: 校验 SHA256
	// -------------------------------
	fmt.Println("[STEP 2/8] 正在校验 SHA256...")
	if ok := decompress.VerifyHash(); !ok {
		log.Fatalf("[ERROR] SHA256 校验失败！")
	}
	fmt.Println("[STEP 2/8] SHA256 校验通过")

	// -------------------------------
	// 步骤 3: 初始化事务
	// -------------------------------
	fmt.Println("[STEP 3/8] 初始化事务...")
	txn := transaction.New()
	defer txn.Cleanup()
	fmt.Println("[STEP 3/8] 事务初始化完成")

	// -------------------------------
	// 步骤 4: 读取 config.json
	// -------------------------------
	fmt.Println("[STEP 4/8] 正在读取 config.json...")
	configPath := filepath.Join(destDir, "config.json")
	data, err := os.ReadFile(configPath)
	if err != nil {
		log.Fatalf("[ERROR] 读取 config.json 失败: %v", err)
	}
	cfg := &fileops.Config{}
	if err := json.Unmarshal(data, cfg); err != nil {
		log.Fatalf("[ERROR] 解析 config.json 失败: %v", err)
	}
	fmt.Println("[STEP 4/8] config.json 解析完成")
	// 禁用 motd
	fmt.Println("[INFO] 正在禁用 motd...")
	if err := disableMotd(); err != nil {
		log.Fatalf("[ERROR] 禁用 motd 失败: %v", err)
	}

	// -------------------------------
	// 步骤 5: 注册复制动作
	// -------------------------------
	fmt.Println("[STEP 5/8] 注册复制动作...")
	if err := fileops.RegisterCopyActions(txn, cfg); err != nil {
		log.Fatalf("[ERROR] 注册复制动作失败: %v", err)
	}
	fmt.Println("[STEP 5/8] 复制动作注册完成")

	// -------------------------------
	// 步骤 6: 执行事务
	// -------------------------------
	fmt.Println("[STEP 6/8] 正在执行事务...")
	if err := txn.Run(); err != nil {
		log.Fatalf("[ERROR] 事务执行失败: %v", err)
	}
	fmt.Println("[STEP 6/8] 事务执行完成")

	// -------------------------------
	// 步骤 7: 启用 systemd 服务（非阻塞）
	// -------------------------------
	fmt.Println("[STEP 7/8] 启用 systemd 服务...")
	if err := systemd.EnableServices(cfg); err != nil {
		log.Fatalf("[ERROR] 启用 systemd 服务失败: %v", err)
	}
	fmt.Println("[STEP 7/8] systemd 服务已启用")

	// -------------------------------
	// 步骤 8: 修改 GRUB 内核参数
	// -------------------------------
	fmt.Println("[STEP 8/8] 修改 GRUB 内核参数...")
	if err := system.UpdateGRUBKernelCmdline(); err != nil {
		log.Fatalf("[ERROR] 修改 GRUB 失败: %v", err)
	}

	// -------------------------------
	// 步骤 9: 关闭 cloud-init
	// -------------------------------
	fmt.Println("[INFO] 正在禁用 cloud-init...")
	if err := system.DisableCloudInit(); err != nil {
		log.Fatalf("[ERROR] 禁用 cloud-init 失败: %v", err)
	}
	fmt.Println("[INFO] cloud-init 已禁用")

	fmt.Println("[INFO] 所有操作完成")
}
