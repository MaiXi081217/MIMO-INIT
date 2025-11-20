package run

import (
	"fmt"
	"mimo/internal/decompress"
	"mimo/internal/env"
	"mimo/internal/fileops"
	"mimo/internal/grub"
	"mimo/internal/motd"
	"mimo/internal/spdk"
	"mimo/internal/system"
	"mimo/internal/systemd"
	"mimo/internal/transaction"
	"os"
	"os/exec"
	"path/filepath"
)

const (
	tmpDir        = "/tmp/mimo-output"
	configFile    = "config.json"
	pkgdepScript  = "pkgdep.sh"
	scriptsSubDir = "scripts"
)

// 注意：spdkSock 已移除，使用 spdk.SPDKSock() 代替

func RunPkgDep() {
	mimoRoot := env.EnsureMimoRoot()

	// look for pkgdep script in a few locations (prefer unpacked resources)
	candidates := []string{
		filepath.Join(tmpDir, "file", "SPDK_for_MIMO", scriptsSubDir, pkgdepScript), // unpacked package (first run)
		filepath.Join(mimoRoot, scriptsSubDir, pkgdepScript),                        // installed location (later runs)
	}

	var pkgdepScript string
	for _, p := range candidates {
		p = filepath.Clean(p)
		if _, err := os.Stat(p); err == nil {
			pkgdepScript = p
			break
		}
	}

	if pkgdepScript == "" {
		fmt.Println("WARN: dependency script not found, skipping")
		return
	}
	fmt.Println("INFO: running 'sudo apt update'...")
	updateCmd := exec.Command("sudo", "apt", "update")
	updateCmd.Stdout = os.Stdout
	updateCmd.Stderr = os.Stderr
	if err := updateCmd.Run(); err != nil {
		fmt.Printf("WARN: 'apt update' failed: %v\n", err)
	} else {
		fmt.Println("INFO: 'apt update' completed")
	}

	fmt.Println("INFO: installing package dependencies; this may take some time...")
	cmd := exec.Command("bash", pkgdepScript)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		fmt.Printf("WARN: dependency installation failed: %v\n", err)
	} else {
		fmt.Println("INFO: dependencies installed")
	}
}

func RunTransaction(cfg *fileops.Config) error {
	txn := transaction.New()
	defer txn.Cleanup()

	if err := motd.RegisterMOTDActions(txn); err != nil {
		return fmt.Errorf("setup MOTD actions failed: %w", err)
	}

	if err := fileops.RegisterCopyActions(txn, cfg); err != nil {
		return fmt.Errorf("setup file copy actions failed: %w", err)
	}

	if err := grub.RegisterGrubAndInitActions(txn); err != nil {
		return fmt.Errorf("setup GRUB actions failed: %w", err)
	}

	if err := txn.Run(); err != nil {
		return fmt.Errorf("executing update actions failed: %w", err)
	}

	return nil
}

// extractAndVerify 提取并验证资源，返回错误
func extractAndVerify(tmpDir string) error {
	if err := decompress.ExtractResources(tmpDir); err != nil {
		return fmt.Errorf("extracting resources failed: %w", err)
	}
	if ok := decompress.VerifyHash(); !ok {
		return fmt.Errorf("resource verification failed")
	}
	return nil
}

func RunUpdate() error {
	env.MustBeRoot()

	env.EnsureMimoRoot()
	cleanTmp := filepath.Clean(tmpDir)
	defer func() {
		_ = os.RemoveAll(cleanTmp)
	}()

	if err := extractAndVerify(cleanTmp); err != nil {
		return err
	}

	RunPkgDep()

	configPath := filepath.Join(cleanTmp, configFile)
	cfg := env.LoadFileOpsConfig(configPath)

	if err := RunTransaction(cfg); err != nil {
		return fmt.Errorf("executing transaction failed: %w", err)
	}

	if err := systemd.EnableServices(cfg); err != nil {
		return fmt.Errorf("enabling system services failed: %w", err)
	}

	if err := system.DisableCloudInit(); err != nil {
		return fmt.Errorf("disabling cloud-init failed: %w", err)
	}

	return nil
}

func RuntgtUpdate() error {
	env.MustBeRoot()

	env.EnsureMimoRoot()
	cleanTmp := filepath.Clean(tmpDir)
	defer func() {
		_ = os.RemoveAll(cleanTmp)
	}()

	fmt.Println("INFO: extracting package...")
	if err := extractAndVerify(cleanTmp); err != nil {
		return err
	}

	configPath := filepath.Join(cleanTmp, configFile)
	cfg := env.LoadVersionConfig(configPath)

	newVerFile := cfg.Version[0].Src
	oldVerFile := cfg.Version[1].Dst
	oldVer := env.ReadMimoVersion(oldVerFile)
	newVer := env.ReadMimoVersion(newVerFile)

	fmt.Printf("INFO: installed version: %s\n", oldVer)
	fmt.Printf("INFO: new version      : %s\n", newVer)

	if !env.ConfirmPrompt("Proceed with update? [y/N]: ") {
		fmt.Println("INFO: update cancelled")
		return nil
	}

	// Stop MIMO and save config
	cleanSock := filepath.Clean(spdk.SPDKSock())
	if _, err := os.Stat(cleanSock); err == nil {
		fmt.Println("INFO: detected running MIMO instance")
		if env.ConfirmPrompt("Stop MIMO now? [y/N]: ") {
			if err := spdk.SaveSpdkConfigAndGetCommand(); err != nil {
				return fmt.Errorf("failed to save SPDK config: %w", err)
			}
			fmt.Println("INFO: MIMO stopped")
		} else {
			fmt.Println("INFO: please stop I/O before updating")
			return nil
		}
	} else if !os.IsNotExist(err) {
		return fmt.Errorf("failed to check socket: %w", err)
	}

	// Copy files according to mappings
	fmt.Println("INFO: applying file mappings...")

	fileOpsConfigPath := filepath.Join(cleanTmp, configFile)
	fileOpsCfg := env.LoadFileOpsConfig(fileOpsConfigPath)

	for _, mapping := range fileOpsCfg.FileMappings {
		srcPath := mapping.Src
		dstPath := mapping.Dst

		fi, err := os.Stat(srcPath)
		if err != nil {
			return fmt.Errorf("source missing %s: %w", srcPath, err)
		}

		// remove old target
		if err := os.RemoveAll(dstPath); err != nil {
			return fmt.Errorf("failed to remove old target %s: %w", dstPath, err)
		}

		if fi.IsDir() {
			if err := fileops.CopyDir(srcPath, dstPath); err != nil {
				return fmt.Errorf("failed to copy directory %s -> %s: %w", srcPath, dstPath, err)
			}
		} else {
			if err := fileops.CopyFile(srcPath, dstPath); err != nil {
				return fmt.Errorf("failed to copy file %s -> %s: %w", srcPath, dstPath, err)
			}
		}
	}

	// Restart MIMO with saved config
	if err := spdk.RestartSpdkWithSavedConfig(); err != nil {
		return fmt.Errorf("failed to restart SPDK: %w", err)
	}

	return nil
}
