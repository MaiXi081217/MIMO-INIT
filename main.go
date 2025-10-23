package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"resourcemgr/internal/decompress"
	"resourcemgr/internal/fileops"
	"resourcemgr/internal/grub"
	"resourcemgr/internal/motd"
	"resourcemgr/internal/system"
	"resourcemgr/internal/systemd"
	"resourcemgr/internal/transaction"
	"strings"
)

func ensureMimoRoot() string {
	const defaultPath = "/usr/local/mimo"
	mimoRoot := os.Getenv("MIMO_ROOT")

	if mimoRoot == "" {
		mimoRoot = defaultPath
		if err := os.Setenv("MIMO_ROOT", mimoRoot); err != nil {
			log.Fatalf("[ERROR] failed to set MIMO_ROOT: %v", err)
		}
		fmt.Printf("[INFO] Environment variable MIMO_ROOT not set, defaulting to %s\n", mimoRoot)

		// 写入系统配置文件，使其永久生效
		profilePath := "/etc/profile.d/mimo_root.sh"
		content := fmt.Sprintf("export MIMO_ROOT=%s\n", mimoRoot)
		if err := os.WriteFile(profilePath, []byte(content), 0644); err != nil {
			log.Printf("[WARN] failed to persist MIMO_ROOT to %s: %v\n", profilePath, err)
		} else {
			fmt.Printf("[INFO] Persisted MIMO_ROOT to %s\n", profilePath)
		}
	}
	return mimoRoot
}

func runUpdate() {
	if os.Geteuid() != 0 {
		log.Fatalf("[ERROR] must run as root")
	}

	destDir := "/tmp/mimo-output"

	ensureMimoRoot()

	// 解压资源
	if err := decompress.ExtractResources(destDir); err != nil {
		log.Fatalf("[ERROR] extracting resources failed: %v", err)
	}

	if ok := decompress.VerifyHash(); !ok {
		log.Fatalf("[ERROR] SHA256 verification failed")
	}

	txn := transaction.New()
	defer txn.Cleanup()

	// 注册 MOTD 备份 & 清理动作
	if err := motd.RegisterMOTDActions(txn); err != nil {
		log.Fatalf("[ERROR] registering MOTD actions failed: %v", err)
	}

	configPath := filepath.Join(destDir, "config.json")
	data, err := os.ReadFile(configPath)
	if err != nil {
		log.Fatalf("[ERROR] reading config.json failed: %v", err)
	}

	cfg := &fileops.Config{}
	if err := json.Unmarshal(data, cfg); err != nil {
		log.Fatalf("[ERROR] parsing config.json failed: %v", err)
	}

	// 注册文件复制动作
	if err := fileops.RegisterCopyActions(txn, cfg); err != nil {
		log.Fatalf("[ERROR] registering copy actions failed: %v", err)
	}
	// 修改 GRUB 内核参数
	if err := grub.RegisterGrubAndInitActions(txn); err != nil {
		log.Fatalf("[ERROR] registering GRUB actions failed: %v", err)
	}

	// 执行事务（包含 MOTD 清理和文件复制）
	if err := txn.Run(); err != nil {
		log.Fatalf("[ERROR] executing transaction failed: %v", err)
	}

	// 启用 systemd 服务
	if err := systemd.EnableServices(cfg); err != nil {
		log.Fatalf("[ERROR] enabling systemd services failed: %v", err)
	}

	// 禁用 cloud-init
	if err := system.DisableCloudInit(); err != nil {
		log.Fatalf("[ERROR] disabling cloud-init failed: %v", err)
	}

	// 删除临时目录
	if err := os.RemoveAll(destDir); err != nil {
		fmt.Printf("[WARN] failed to delete temporary directory %s: %v\n", destDir, err)
	}
}

func runtgtUpdate() {
	if os.Geteuid() != 0 {
		log.Fatalf("[ERROR] must run as root")
	}

	destDir := "/tmp/mimo-output"

	// === Step 1: 解压资源包 ===
	fmt.Println("[INFO] Extracting resource package...")
	if err := decompress.ExtractResources(destDir); err != nil {
		log.Fatalf("[ERROR] extracting resources failed: %v", err)
	}

	// === Step 2: 校验哈希 ===
	if ok := decompress.VerifyHash(); !ok {
		log.Fatalf("[ERROR] SHA256 verification failed")
	}

	// === Step 3: 检查 SPDK 是否正在运行 ===
	spdkSock := "/var/tmp/spdk.sock"
	if _, err := os.Stat(spdkSock); err == nil {
		fmt.Println("[INFO] Detected running SPDK instance at", spdkSock)
		fmt.Println("SPDK is currently running and may need to be stopped for safe update.")
		fmt.Println("Please ensure all I/O operations have been stopped before proceeding.")
		fmt.Print("Have you stopped all I/O? [y/N]: ")

		var answer string
		fmt.Scanln(&answer)
		if strings.ToLower(strings.TrimSpace(answer)) != "y" {
			fmt.Println("[ABORTED] Please stop I/O operations before running target update.")
			return
		}
	} else if !os.IsNotExist(err) {
		log.Fatalf("[ERROR] checking %s failed: %v", spdkSock, err)
	}

	// === Step 4: 读取旧版本 ===
	mimoRoot := os.Getenv("MIMO_ROOT")
	if mimoRoot == "" {
		log.Fatalf("[ERROR] MIMO_ROOT environment variable not set")
	}

	oldVerPath := filepath.Join(mimoRoot, "VERSION.json")
	oldData, err := os.ReadFile(oldVerPath)
	if err != nil {
		log.Fatalf("[ERROR] failed to read %s: %v", oldVerPath, err)
	}

	var oldVer map[string]interface{}
	if err := json.Unmarshal(oldData, &oldVer); err != nil {
		log.Fatalf("[ERROR] parsing old VERSION.json failed: %v", err)
	}

	oldSPDK, _ := oldVer["SPDK_for_MIMO"].(string)
	if oldSPDK == "" {
		oldSPDK = "(unknown)"
	}

	// === Step 5: 读取新版本 ===
	newVerPath := filepath.Join(destDir, "VERSION.json")
	newData, err := os.ReadFile(newVerPath)
	if err != nil {
		log.Fatalf("[ERROR] failed to read %s: %v", newVerPath, err)
	}

	var newVer map[string]interface{}
	if err := json.Unmarshal(newData, &newVer); err != nil {
		log.Fatalf("[ERROR] parsing new VERSION.json failed: %v", err)
	}

	newSPDK, _ := newVer["SPDK_for_MIMO"].(string)
	if newSPDK == "" {
		newSPDK = "(unknown)"
	}

	// === Step 6: 打印版本对比 ===
	fmt.Println("===== SPDK Version Comparison =====")
	fmt.Printf("Current (Installed): %s\n", oldSPDK)
	fmt.Printf("New (Update File):  %s\n", newSPDK)
	fmt.Println("===================================")

	// === Step 7: （后续可执行升级逻辑）===
	fmt.Println("[INFO] Target update check complete. Ready to continue upgrade steps.")

	// 清理临时目录
	// _ = os.RemoveAll(destDir)
}

func main() {
	help := flag.Bool("help", false, "Show help message")
	update := flag.Bool("sys-update", false, "System update mode")
	tgt := flag.Bool("target-update", false, "Target update mode")

	flag.Parse()
	if *help {
		fmt.Println(`mimo-update: One-step MIMO system resource updater
		Usage:
			mimo-update [options]

		Options:
			--help          Show this help message
			--update        Execute system update
			--tgt           Execute target update`)
		return
	}

	if *update {
		runUpdate()
		return
	}
	if *tgt {
		runtgtUpdate()
		fmt.Println("[INFO] Target update mode is not implemented yet")
	}
	fmt.Println("Specify one of the options: --help | --update")
}
