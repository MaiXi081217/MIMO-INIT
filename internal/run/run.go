package run

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"resourcemgr/internal/decompress"
	"resourcemgr/internal/env"
	"resourcemgr/internal/fileops"
	"resourcemgr/internal/grub"
	"resourcemgr/internal/motd"
	"resourcemgr/internal/spdk"
	"resourcemgr/internal/system"
	"resourcemgr/internal/systemd"
	"resourcemgr/internal/transaction"
)

const (
	tmpDir          = "/tmp/mimo-output"
	spdkSock        = "/var/tmp/spdk.sock"
)

func RunPkgDep() {
	mimoRoot := env.EnsureMimoRoot()
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

func RunTransaction(cfg *fileops.Config) {
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

func RunUpdate() {
	if os.Geteuid() != 0 {
		log.Fatalf("[ERROR] must run as root")
	}

	env.EnsureMimoRoot()

	if err := decompress.ExtractResources(tmpDir); err != nil {
		log.Fatalf("[ERROR] extracting resources failed: %v", err)
	}
	if ok := decompress.VerifyHash(); !ok {
		log.Fatalf("[ERROR] SHA256 verification failed")
	}

	RunPkgDep()

	configPath := filepath.Join(tmpDir, "config.json")
	cfg := env.LoadFileOpsConfig(configPath)

	RunTransaction(cfg)

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

// ----------------- Target Update Main -----------------
func RuntgtUpdate() {
	if os.Geteuid() != 0 {
		log.Fatalf("[ERROR] must run as root")
	}

	env.EnsureMimoRoot()

	fmt.Println("[INFO] Extracting resource package...")
	if err := decompress.ExtractResources(tmpDir); err != nil {
		log.Fatalf("[ERROR] extracting resources failed: %v", err)
	}
	if ok := decompress.VerifyHash(); !ok {
		log.Fatalf("[ERROR] SHA256 verification failed")
	}

	configPath := filepath.Join(tmpDir, "config.json")
	cfg := env.LoadVersionConfig(configPath)

	newVerFile := cfg.Version[0].Src
	oldVerFile := cfg.Version[1].Dst
	oldVer := env.ReadMimoVersion(oldVerFile)
	newVer := env.ReadMimoVersion(newVerFile)

	fmt.Printf("[INFO] Installed MIMO version: %s\n", oldVer)
	fmt.Printf("[INFO] New MIMO version      : %s\n", newVer)

	if !env.ConfirmPrompt("[PROMPT] Do you want to update? [y/N]: ") {
		fmt.Println("[INFO] Update aborted by user.")
		_ = os.RemoveAll(tmpDir)
		return
	}

	// Step: Stop SPDK and save config
	if _, err := os.Stat(spdkSock); err == nil {
		fmt.Println("[INFO] Detected running SPDK instance at", spdkSock)
		fmt.Println("[INFO] SPDK is currently running. I/O must be stopped for safe update.")
		if env.ConfirmPrompt("[PROMPT] Do you want to stop SPDK processes now? [y/N]: ") {
			spdk.SaveSpdkConfigAndGetCommand()
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
	fileOpsCfg := env.LoadFileOpsConfig(fileOpsConfigPath)

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
	spdk.RestartSpdkWithSavedConfig()
}
