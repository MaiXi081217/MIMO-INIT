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

	// === Step 4: 读取 config.json 中的 src/dst 路径 ===
	configPath := filepath.Join(destDir, "config.json")
	data, err := os.ReadFile(configPath)
	if err != nil {
		log.Fatalf("[ERROR] reading config.json failed: %v", err)
	}

	var cfg struct {
		Src string `json:"src"`
		Dst string `json:"dst"`
	}
	if err := json.Unmarshal(data, &cfg); err != nil {
		log.Fatalf("[ERROR] parsing config.json failed: %v", err)
	}

	if cfg.Src == "" || cfg.Dst == "" {
		log.Fatalf("[ERROR] invalid config.json: missing src/dst fields")
	}

	// === Step 5: 执行版本对比 ===
	srcVerPath := filepath.Join(cfg.Src, "VERSION.json")
	dstVerPath := filepath.Join(cfg.Dst, "SPDK_for_MIMO", "VERSION.json")

	readVer := func(path string) (map[string]interface{}, error) {
		data, err := os.ReadFile(path)
		if err != nil {
			return nil, err
		}
		var v map[string]interface{}
		if err := json.Unmarshal(data, &v); err != nil {
			return nil, err
		}
		return v, nil
	}

	srcVer, err := readVer(srcVerPath)
	if err != nil {
		log.Fatalf("[ERROR] failed to read new version file: %v", err)
	}

	dstVer, err := readVer(dstVerPath)
	if err != nil {
		fmt.Printf("[WARN] failed to read installed version file (%v), assuming fresh install\n", err)
		dstVer = map[string]interface{}{}
	}

	srcSPDK, _ := srcVer["SPDK_for_MIMO"].(string)
	dstSPDK, _ := dstVer["SPDK_for_MIMO"].(string)
	srcMIMO, _ := srcVer["MIMO"].(string)
	dstMIMO, _ := dstVer["MIMO"].(string)

	if srcSPDK == "" {
		srcSPDK = "(unknown)"
	}
	if dstSPDK == "" {
		dstSPDK = "(none)"
	}
	if srcMIMO == "" {
		srcMIMO = "(unknown)"
	}
	if dstMIMO == "" {
		dstMIMO = "(none)"
	}

	fmt.Println("===== Version Comparison =====")
	fmt.Printf("Installed SPDK: %s\n", dstSPDK)
	fmt.Printf("Update SPDK:    %s\n", srcSPDK)
	fmt.Printf("Installed MIMO: %s\n", dstMIMO)
	fmt.Printf("Update MIMO:    %s\n", srcMIMO)
	fmt.Println("===================================")

	if srcSPDK == dstSPDK && srcMIMO == dstMIMO {
		fmt.Println("[INFO] Version unchanged, skip update.")
		_ = os.RemoveAll(destDir)
		return
	}

	fmt.Println("[INFO] Detected new version, proceed with update.")

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
			--sys-update        Execute system update
			--target-update         Execute target update`)
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
	fmt.Println("Specify one of the options: -help")
}
