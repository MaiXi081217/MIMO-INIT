package spdk

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"resourcemgr/internal/env"
	"strconv"
	"strings"
)

const (
	spdkSock = "/var/tmp/spdk.sock"
)

var spdkConfigPath = "/tmp/spdk_full_config.json"
var spdkOrigCmd string

func RestartSpdkWithSavedConfig() {
	if spdkOrigCmd == "" {
		fmt.Println("[WARN] No original SPDK command captured, cannot restart")
		return
	}

	mimoRoot := env.EnsureMimoRoot()
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

func SaveSpdkConfigAndGetCommand() {
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

	//  Step 1: Save configuration BEFORE killing SPDK
	mimoRoot := env.EnsureMimoRoot()
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

	//  Step 2: Kill SPDK AFTER saving config
	fmt.Printf("[INFO] Killing SPDK PID %d\n", pid)
	if err := exec.Command("kill", "-9", strconv.Itoa(pid)).Run(); err != nil {
		log.Printf("[WARN] Failed to kill PID %d: %v", pid, err)
	} else {
		fmt.Printf("[DEBUG] Successfully killed SPDK PID %d\n", pid)
	}

	fmt.Println("[DEBUG] Exiting saveSpdkConfigAndGetCommand()")
}
