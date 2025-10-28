package spdk

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"mimo/internal/env"
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
		fmt.Println("INFO: No original MIMO command captured; restart skipped.")
		return
	}

	mimoRoot := env.EnsureMimoRoot()
	cleanConfigPath := filepath.Clean(spdkConfigPath)

	// Helper: remove "-c <file>" pair from original args and return remaining args
	stripConfigArg := func(parts []string) string {
		newParts := make([]string, 0, len(parts))
		for i := 1; i < len(parts); i++ {
			if parts[i] == "-c" && i+1 < len(parts) {
				i++ // skip config file arg
				continue
			}
			newParts = append(newParts, parts[i])
		}
		if len(newParts) == 0 {
			return ""
		}
		return strings.Join(newParts, " ")
	}

	parts := strings.Fields(spdkOrigCmd)
	rest := stripConfigArg(parts)

	newCmd := fmt.Sprintf("%s/build/bin/spdk_tgt -c %s", mimoRoot, cleanConfigPath)
	if rest != "" {
		newCmd += " " + rest
	}

	fmt.Println("INFO: Restarting MIMO service...")
	bgCmd := exec.Command("bash", "-c", newCmd)
	bgCmd.Stdout = os.Stdout
	bgCmd.Stderr = os.Stderr
	if err := bgCmd.Start(); err != nil {
		log.Fatalf("ERROR: failed to restart MIMO: %v", err)
	}
	if bgCmd.Process != nil {
		fmt.Printf("INFO: MIMO restart initiated (pid=%d)\n", bgCmd.Process.Pid)
	} else {
		fmt.Println("INFO: MIMO restart initiated")
	}
}

func SaveSpdkConfigAndGetCommand() {
	out, err := exec.Command("lsof", "-t", spdkSock).Output()
	if err != nil {
		log.Fatalf("ERROR: failed to check MIMO socket")
	}
	pidStr := strings.TrimSpace(string(out))
	if pidStr == "" {
		log.Fatalf("ERROR: no MIMO process found on socket")
	}

	parts := strings.Fields(pidStr)
	pid, err := strconv.Atoi(parts[0])
	if err != nil {
		log.Fatalf("ERROR: failed to parse MIMO pid")
	}
	fmt.Printf("INFO: MIMO process detected (pid=%d)\n", pid)

	psOut, err := exec.Command("ps", "-p", strconv.Itoa(pid), "-o", "args=").Output()
	if err != nil {
		log.Fatalf("ERROR: failed to obtain MIMO process info")
	}
	spdkOrigCmd = strings.TrimSpace(string(psOut))

	// Step 1: Save configuration BEFORE killing the process
	mimoRoot := env.EnsureMimoRoot()
	rpcPath := filepath.Clean(filepath.Join(mimoRoot, "scripts", "rpc.py"))
	if _, statErr := os.Stat(rpcPath); statErr != nil {
		log.Fatalf("ERROR: required helper not found")
	}

	cmdLine := fmt.Sprintf("%s save_config -i 2 > %s", rpcPath, filepath.Clean(spdkConfigPath))
	saveCmd := exec.Command("bash", "-c", cmdLine)
	saveCmd.Stdout = os.Stdout
	saveCmd.Stderr = os.Stderr
	if err := saveCmd.Run(); err != nil {
		log.Fatalf("ERROR: failed to save MIMO configuration")
	}
	fmt.Printf("INFO: configuration saved\n")

	// Step 2: Kill process AFTER saving config
	fmt.Printf("INFO: stopping MIMO process\n")
	if err := exec.Command("kill", "-9", strconv.Itoa(pid)).Run(); err != nil {
		log.Fatalf("ERROR: failed to stop MIMO process")
	}
	fmt.Printf("INFO: MIMO process stopped\n")
}
