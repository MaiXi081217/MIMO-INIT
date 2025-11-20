package spdk

import (
	"fmt"
	"mimo/internal/env"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
)

const (
	spdkSock       = "/var/tmp/spdk.sock"
	spdkConfigPath = "/tmp/spdk_full_config.json"
	scriptsDir     = "scripts"
	rpcScript      = "rpc.py"
	spdkBinPath    = "build/bin/spdk_tgt"
)

// SPDKSock 返回 SPDK socket 路径
func SPDKSock() string {
	return spdkSock
}

var spdkOrigCmd string

func RestartSpdkWithSavedConfig() error {
	if spdkOrigCmd == "" {
		fmt.Println("INFO: No original MIMO command captured; restart skipped.")
		return nil
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

	newCmd := fmt.Sprintf("%s/%s -c %s", mimoRoot, spdkBinPath, cleanConfigPath)
	if rest != "" {
		newCmd += " " + rest
	}

	fmt.Println("INFO: Restarting MIMO service...")
	bgCmd := exec.Command("bash", "-c", newCmd)
	bgCmd.Stdout = os.Stdout
	bgCmd.Stderr = os.Stderr
	if err := bgCmd.Start(); err != nil {
		return fmt.Errorf("failed to restart MIMO: %w", err)
	}
	if bgCmd.Process != nil {
		fmt.Printf("INFO: MIMO restart initiated (pid=%d)\n", bgCmd.Process.Pid)
	} else {
		fmt.Println("INFO: MIMO restart initiated")
	}
	return nil
}

func SaveSpdkConfigAndGetCommand() error {
	out, err := exec.Command("lsof", "-t", spdkSock).Output()
	if err != nil {
		return fmt.Errorf("failed to check MIMO socket: %w", err)
	}
	pidStr := strings.TrimSpace(string(out))
	if pidStr == "" {
		return fmt.Errorf("no MIMO process found on socket")
	}

	parts := strings.Fields(pidStr)
	pid, err := strconv.Atoi(parts[0])
	if err != nil {
		return fmt.Errorf("failed to parse MIMO pid: %w", err)
	}
	fmt.Printf("INFO: MIMO process detected (pid=%d)\n", pid)

	psOut, err := exec.Command("ps", "-p", strconv.Itoa(pid), "-o", "args=").Output()
	if err != nil {
		return fmt.Errorf("failed to obtain MIMO process info: %w", err)
	}
	spdkOrigCmd = strings.TrimSpace(string(psOut))

	// Step 1: Save configuration BEFORE killing the process
	mimoRoot := env.EnsureMimoRoot()
	rpcPath := filepath.Clean(filepath.Join(mimoRoot, scriptsDir, rpcScript))
	if _, statErr := os.Stat(rpcPath); statErr != nil {
		return fmt.Errorf("required helper not found: %w", statErr)
	}

	cmdLine := fmt.Sprintf("%s save_config -i 2 > %s", rpcPath, filepath.Clean(spdkConfigPath))
	saveCmd := exec.Command("bash", "-c", cmdLine)
	saveCmd.Stdout = os.Stdout
	saveCmd.Stderr = os.Stderr
	if err := saveCmd.Run(); err != nil {
		return fmt.Errorf("failed to save MIMO configuration: %w", err)
	}
	fmt.Printf("INFO: configuration saved\n")

	// Step 2: Kill process AFTER saving config
	fmt.Printf("INFO: stopping MIMO process\n")
	if err := exec.Command("kill", "-9", strconv.Itoa(pid)).Run(); err != nil {
		return fmt.Errorf("failed to stop MIMO process: %w", err)
	}
	fmt.Printf("INFO: MIMO process stopped\n")
	return nil
}
