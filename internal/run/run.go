package run

import (
	"fmt"
	"log"
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
	tmpDir   = "/tmp/mimo-output"
	spdkSock = "/var/tmp/spdk.sock"
)

func RunPkgDep() {
	mimoRoot := env.EnsureMimoRoot()

	// look for pkgdep script in a few locations (prefer unpacked resources)
	candidates := []string{
		filepath.Join(tmpDir, "file", "SPDK_for_MIMO", "scripts", "pkgdep.sh"), // unpacked package (first run)
		filepath.Join(mimoRoot, "scripts", "pkgdep.sh"),                        // installed location (later runs)
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
		log.Printf("WARN: 'apt update' failed: %v", err)
	} else {
		fmt.Println("INFO: 'apt update' completed")
	}

	fmt.Println("INFO: installing package dependencies; this may take some time...")
	cmd := exec.Command("bash", pkgdepScript)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		log.Printf("WARN: dependency installation failed")
	} else {
		fmt.Println("INFO: dependencies installed")
	}
}

func RunTransaction(cfg *fileops.Config) {
	txn := transaction.New()
	defer txn.Cleanup()

	if err := motd.RegisterMOTDActions(txn); err != nil {
		log.Fatalf("ERROR: setup MOTD actions failed")
	}

	if err := fileops.RegisterCopyActions(txn, cfg); err != nil {
		log.Fatalf("ERROR: setup file copy actions failed")
	}

	if err := grub.RegisterGrubAndInitActions(txn); err != nil {
		log.Fatalf("ERROR: setup GRUB actions failed")
	}

	if err := txn.Run(); err != nil {
		log.Fatalf("ERROR: executing update actions failed")
	}
}

func RunUpdate() {
	env.MustBeRoot()

	env.EnsureMimoRoot()
	cleanTmp := filepath.Clean(tmpDir)
	defer func() {
		_ = os.RemoveAll(cleanTmp)
	}()

	if err := decompress.ExtractResources(cleanTmp); err != nil {
		log.Fatalf("ERROR: extracting resources failed")
	}
	if ok := decompress.VerifyHash(); !ok {
		log.Fatalf("ERROR: resource verification failed")
	}

	RunPkgDep()

	configPath := filepath.Join(cleanTmp, "config.json")
	cfg := env.LoadFileOpsConfig(configPath)

	RunTransaction(cfg)

	if err := systemd.EnableServices(cfg); err != nil {
		log.Fatalf("ERROR: enabling system services failed")
	}

	if err := system.DisableCloudInit(); err != nil {
		log.Fatalf("ERROR: disabling cloud-init failed")
	}
}

func RuntgtUpdate() {
	env.MustBeRoot()

	env.EnsureMimoRoot()
	cleanTmp := filepath.Clean(tmpDir)
	defer func() {
		_ = os.RemoveAll(cleanTmp)
	}()

	fmt.Println("INFO: extracting package...")
	if err := decompress.ExtractResources(cleanTmp); err != nil {
		log.Fatalf("ERROR: extracting resources failed")
	}
	if ok := decompress.VerifyHash(); !ok {
		log.Fatalf("ERROR: resource verification failed")
	}

	configPath := filepath.Join(cleanTmp, "config.json")
	cfg := env.LoadVersionConfig(configPath)

	newVerFile := cfg.Version[0].Src
	oldVerFile := cfg.Version[1].Dst
	oldVer := env.ReadMimoVersion(oldVerFile)
	newVer := env.ReadMimoVersion(newVerFile)

	fmt.Printf("INFO: installed version: %s\n", oldVer)
	fmt.Printf("INFO: new version      : %s\n", newVer)

	if !env.ConfirmPrompt("Proceed with update? [y/N]: ") {
		fmt.Println("INFO: update cancelled")
		return
	}

	// Stop MIMO and save config
	cleanSock := filepath.Clean(spdkSock)
	if _, err := os.Stat(cleanSock); err == nil {
		fmt.Println("INFO: detected running MIMO instance")
		if env.ConfirmPrompt("Stop MIMO now? [y/N]: ") {
			spdk.SaveSpdkConfigAndGetCommand()
			fmt.Println("INFO: MIMO stopped")
		} else {
			fmt.Println("INFO: please stop I/O before updating")
			return
		}
	} else if !os.IsNotExist(err) {
		log.Fatalf("ERROR: failed to check socket")
	}

	// Copy files according to mappings
	fmt.Println("INFO: applying file mappings...")

	fileOpsConfigPath := filepath.Join(cleanTmp, "config.json")
	fileOpsCfg := env.LoadFileOpsConfig(fileOpsConfigPath)

	for _, mapping := range fileOpsCfg.FileMappings {
		srcPath := mapping.Src
		dstPath := mapping.Dst

		//fmt.Printf("INFO: %s -> %s\n", srcPath, dstPath)

		fi, err := os.Stat(srcPath)
		if err != nil {
			log.Fatalf("ERROR: source missing")
		}

		// remove old target
		if err := os.RemoveAll(dstPath); err != nil {
			log.Fatalf("ERROR: failed to remove old target")
		}

		if fi.IsDir() {
			if err := fileops.CopyDir(srcPath, dstPath); err != nil {
				log.Fatalf("ERROR: failed to copy directory")
			}
		} else {
			if err := fileops.CopyFile(srcPath, dstPath); err != nil {
				log.Fatalf("ERROR: failed to copy file")
			}
		}
	}

	// Restart MIMO with saved config
	spdk.RestartSpdkWithSavedConfig()
}
