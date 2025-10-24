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
	"resourcemgr/internal/grub"
	"resourcemgr/internal/motd"
	"resourcemgr/internal/system"
	"resourcemgr/internal/systemd"
	"resourcemgr/internal/transaction"
	"strconv"
	"strings"
)

const (
	defaultMimoRoot = "/usr/local/mimo"
	tmpDir          = "/tmp/mimo-output"
	spdkSock        = "/var/tmp/spdk.sock"
)

// ----------------- Package Dependency Installer -----------------
func runPkgDep() {
	mimoRoot := ensureMimoRoot()
	pkgdepScript := filepath.Join(mimoRoot, "scripts", "pkgdep.sh")

	if _, err := os.Stat(pkgdepScript); os.IsNotExist(err) {
		fmt.Printf("[WARN] pkgdep.sh script not found at %s, skipping dependency installation\n", pkgdepScript)
		return
	}

	fmt.Println("[INFO] Starting package dependency installation...")
	fmt.Println("[INFO] This may take a while depending on your network and system configuration.")

	cmd := exec.Command("bash", pkgdepScript)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		log.Printf("[ERROR] pkgdep.sh execution failed: %v\n", err)
	} else {
		fmt.Println("[INFO] Package dependencies installed successfully")
	}
}

// ----------------- Environment -----------------
func ensureMimoRoot() string {
	mimoRoot := os.Getenv("MIMO_ROOT")
	if mimoRoot == "" {
		mimoRoot = defaultMimoRoot
		if err := os.Setenv("MIMO_ROOT", mimoRoot); err != nil {
			log.Fatalf("[ERROR] failed to set MIMO_ROOT: %v", err)
		}
		fmt.Printf("[INFO] Environment variable MIMO_ROOT not set, defaulting to %s\n", mimoRoot)

		// Persist environment variable
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

// ----------------- Transaction Utilities -----------------
func runTransaction(cfg *fileops.Config) {
	txn := transaction.New()
	defer txn.Cleanup()

	if err := motd.RegisterMOTDActions(txn); err != nil {
		log.Fatalf("[ERROR] registering MOTD actions failed: %v", err)
	}

	if err := fileops.RegisterCopyActions(txn, cfg); err != nil {
		log.Fatalf("[ERROR] registering copy actions failed: %v", err)
	}

	if err := grub.RegisterGrubAndInitActions(txn); err != nil {
		log.Fatalf("[ERROR] registering GRUB actions failed: %v", err)
	}

	if err := txn.Run(); err != nil {
		log.Fatalf("[ERROR] executing transaction failed: %v", err)
	}
}

// ----------------- Config Utilities -----------------
func loadFileOpsConfig(path string) *fileops.Config {
	data, err := os.ReadFile(path)
	if err != nil {
		log.Fatalf("[ERROR] reading config.json failed: %v", err)
	}

	cfg := &fileops.Config{}
	if err := json.Unmarshal(data, cfg); err != nil {
		log.Fatalf("[ERROR] parsing config.json failed: %v", err)
	}
	return cfg
}

type VersionMapping struct {
	Src string `json:"src"`
	Dst string `json:"dst"`
}

type VersionConfig struct {
	Version []VersionMapping `json:"version"`
}

func loadVersionConfig(path string) *VersionConfig {
	data, err := os.ReadFile(path)
	if err != nil {
		log.Fatalf("[ERROR] reading config.json failed: %v", err)
	}

	cfg := &VersionConfig{}
	if err := json.Unmarshal(data, cfg); err != nil {
		log.Fatalf("[ERROR] parsing config.json failed: %v", err)
	}
	if len(cfg.Version) < 2 {
		log.Fatalf("[ERROR] version field malformed, need src and dst")
	}
	return cfg
}

func readMimoVersion(path string) string {
	data, err := os.ReadFile(path)
	if err != nil {
		return "v0.0.0"
	}
	var v map[string]interface{}
	if err := json.Unmarshal(data, &v); err != nil {
		return "v0.0.0"
	}
	if mimo, ok := v["MIMO"].(string); ok && mimo != "" {
		return mimo
	}
	return "v0.0.0"
}

// ----------------- Update Functions -----------------
func runUpdate() {
	if os.Geteuid() != 0 {
		log.Fatalf("[ERROR] must run as root")
	}

	ensureMimoRoot()

	if err := decompress.ExtractResources(tmpDir); err != nil {
		log.Fatalf("[ERROR] extracting resources failed: %v", err)
	}
	if ok := decompress.VerifyHash(); !ok {
		log.Fatalf("[ERROR] SHA256 verification failed")
	}

	runPkgDep()

	configPath := filepath.Join(tmpDir, "config.json")
	cfg := loadFileOpsConfig(configPath)

	runTransaction(cfg)

	if err := systemd.EnableServices(cfg); err != nil {
		log.Fatalf("[ERROR] enabling systemd services failed: %v", err)
	}

	if err := system.DisableCloudInit(); err != nil {
		log.Fatalf("[ERROR] disabling cloud-init failed: %v", err)
	}

	if err := os.RemoveAll(tmpDir); err != nil {
		fmt.Printf("[WARN] failed to delete temporary directory %s: %v\n", tmpDir, err)
	}
}

// ----------------- Target Update Helpers -----------------
func confirmPrompt(msg string) bool {
	fmt.Print(msg)
	var ans string
	fmt.Scanln(&ans)
	return strings.ToLower(strings.TrimSpace(ans)) == "y"
}

var spdkConfigPath = "/tmp/spdk_full_config.json"
var spdkOrigCmd string

func saveSpdkConfigAndGetCommand() {
	fmt.Println("[DEBUG] Entering saveSpdkConfigAndGetCommand()")

	out, err := exec.Command("lsof", "-t", spdkSock).Output()
	if err != nil || len(out) == 0 {
		fmt.Println("[INFO] No SPDK process running")
		return
	}
	pidStr := strings.Fields(string(out))[0]
	pid, _ := strconv.Atoi(pidStr)
	fmt.Printf("[DEBUG] Detected SPDK PID: %d\n", pid)

	psOut, err := exec.Command("ps", "-p", strconv.Itoa(pid), "-o", "args=").Output()
	if err != nil {
		log.Printf("[WARN] failed to get original SPDK command: %v", err)
		return
	}
	spdkOrigCmd = strings.TrimSpace(string(psOut))
	fmt.Println("[INFO] Original SPDK command:", spdkOrigCmd)

	// ✅ Step 1: Save configuration BEFORE killing SPDK
	mimoRoot := ensureMimoRoot()
	rpcPath := filepath.Join(mimoRoot, "scripts", "rpc.py")
	fmt.Printf("[DEBUG] MIMO root: %s\n", mimoRoot)
	fmt.Printf("[DEBUG] RPC path: %s\n", rpcPath)

	cmdLine := fmt.Sprintf("%s save_config -i 2 > %s", rpcPath, spdkConfigPath)
	fmt.Printf("[DEBUG] Running command: %s\n", cmdLine)

	saveCmd := exec.Command("bash", "-c", cmdLine)
	saveCmd.Stderr = os.Stderr
	if err := saveCmd.Run(); err != nil {
		log.Printf("[WARN] failed to save SPDK config: %v", err)
	} else {
		fmt.Printf("[INFO] SPDK config saved to %s\n", spdkConfigPath)
	}

	// ✅ Step 2: Kill SPDK AFTER saving config
	fmt.Printf("[INFO] Killing SPDK PID %d\n", pid)
	if err := exec.Command("kill", "-9", strconv.Itoa(pid)).Run(); err != nil {
		log.Printf("[WARN] Failed to kill PID %d: %v", pid, err)
	} else {
		fmt.Printf("[DEBUG] Successfully killed SPDK PID %d\n", pid)
	}

	fmt.Println("[DEBUG] Exiting saveSpdkConfigAndGetCommand()")
}

// Step 2: Restart SPDK with saved config and original args
func restartSpdkWithSavedConfig() {
	if spdkOrigCmd == "" {
		fmt.Println("[WARN] No original SPDK command captured, cannot restart")
		return
	}

	mimoRoot := ensureMimoRoot()
	newCmd := fmt.Sprintf("%s/build/bin/spdk_tgt -c %s", mimoRoot, spdkConfigPath)

	parts := strings.Fields(spdkOrigCmd)
	newParts := []string{}
	for i := 1; i < len(parts); i++ {
		if parts[i] == "-c" && i+1 < len(parts) {
			i++ // skip old config file
			continue
		}
		newParts = append(newParts, parts[i])
	}
	if len(newParts) > 0 {
		newCmd += " " + strings.Join(newParts, " ")
	}

	fmt.Println("[INFO] Restarting SPDK with command:", newCmd)

	bgCmd := exec.Command("bash", "-c", newCmd)
	bgCmd.Stdout = os.Stdout
	bgCmd.Stderr = os.Stderr
	if err := bgCmd.Start(); err != nil {
		log.Printf("[ERROR] failed to restart SPDK: %v", err)
	} else {
		fmt.Printf("[INFO] SPDK restarted with PID %d\n", bgCmd.Process.Pid)
	}
}

// ----------------- Target Update Main -----------------
func runtgtUpdate() {
	if os.Geteuid() != 0 {
		log.Fatalf("[ERROR] must run as root")
	}

	ensureMimoRoot()

	fmt.Println("[INFO] Extracting resource package...")
	if err := decompress.ExtractResources(tmpDir); err != nil {
		log.Fatalf("[ERROR] extracting resources failed: %v", err)
	}
	if ok := decompress.VerifyHash(); !ok {
		log.Fatalf("[ERROR] SHA256 verification failed")
	}

	configPath := filepath.Join(tmpDir, "config.json")
	cfg := loadVersionConfig(configPath)

	newVerFile := cfg.Version[0].Src
	oldVerFile := cfg.Version[1].Dst
	oldVer := readMimoVersion(oldVerFile)
	newVer := readMimoVersion(newVerFile)

	fmt.Printf("[INFO] Installed MIMO version: %s\n", oldVer)
	fmt.Printf("[INFO] New MIMO version      : %s\n", newVer)

	if !confirmPrompt("[PROMPT] Do you want to update? [y/N]: ") {
		fmt.Println("[INFO] Update aborted by user.")
		_ = os.RemoveAll(tmpDir)
		return
	}

	// Step: Stop SPDK and save config
	if _, err := os.Stat(spdkSock); err == nil {
		fmt.Println("[INFO] Detected running SPDK instance at", spdkSock)
		fmt.Println("[INFO] SPDK is currently running. I/O must be stopped for safe update.")
		if confirmPrompt("[PROMPT] Do you want to stop SPDK processes now? [y/N]: ") {
			saveSpdkConfigAndGetCommand()
			fmt.Println("[INFO] SPDK stopped and configuration saved.")
		} else {
			fmt.Println("[ABORTED] Please stop I/O operations before updating target.")
			return
		}
	} else if !os.IsNotExist(err) {
		log.Fatalf("[ERROR] checking %s failed: %v", spdkSock, err)
	}

	// Step: Delete old and copy new based on file_mappings
	fmt.Println("[INFO] Copying files and directories from file_mappings...")

	fileOpsConfigPath := filepath.Join(tmpDir, "config.json")
	fileOpsCfg := loadFileOpsConfig(fileOpsConfigPath)

	for _, mapping := range fileOpsCfg.FileMappings {
		srcPath := mapping.Src
		dstPath := mapping.Dst

		fmt.Printf("[INFO] Processing mapping: %s -> %s\n", srcPath, dstPath)

		fi, err := os.Stat(srcPath)
		if err != nil {
			log.Fatalf("[ERROR] cannot stat source path %s: %v", srcPath, err)
		}

		// 删除目标路径
		if err := os.RemoveAll(dstPath); err != nil {
			log.Fatalf("[ERROR] failed to remove old path %s: %v", dstPath, err)
		}

		if fi.IsDir() {
			fmt.Printf("[INFO] Copying directory %s to %s\n", srcPath, dstPath)
			if err := fileops.CopyDir(srcPath, dstPath); err != nil {
				log.Fatalf("[ERROR] failed to copy directory: %v", err)
			}
		} else {
			fmt.Printf("[INFO] Copying file %s to %s\n", srcPath, dstPath)
			if err := fileops.CopyFile(srcPath, dstPath); err != nil {
				log.Fatalf("[ERROR] failed to copy file: %v", err)
			}
		}
	}

	// Step: Restart SPDK with saved config
	restartSpdkWithSavedConfig()
}

// ----------------- Main -----------------
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
	--help          Show help message
	--sys-update    Execute system update
	--target-update Execute target update`)
		return
	}

	if *update {
		runUpdate()
		return
	}
	if *tgt {
		runtgtUpdate()
		return
	}

	fmt.Println("Specify one of the options: -help")
}
