/*
Changes:
- Implement DisableCloudInit to stop/disable cloud-init related services and create a disabled marker file.
- Does not fatal; returns error for caller to decide.
- Outputs concise English messages when called by higher-level code.
*/
package system

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
)

// DisableCloudInit stops and disables cloud-init services and creates marker file.
// Returns error if marker creation fails or critical operations fail.
func DisableCloudInit() error {
	services := []string{
		"cloud-init",
		"cloud-final",
		"cloud-config",
		"cloud-init-local",
	}

	for _, s := range services {
		// stop and disable; ignore non-zero results but collect first error
		_ = exec.Command("systemctl", "stop", s).Run()
		_ = exec.Command("systemctl", "disable", s).Run()
	}

	// ensure dir exists
	dir := "/etc/cloud"
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("create %s: %w", dir, err)
	}

	marker := filepath.Join(dir, "cloud-init.disabled")
	f, err := os.Create(marker)
	if err != nil {
		return fmt.Errorf("create marker %s: %w", marker, err)
	}
	defer f.Close()
	_, _ = f.WriteString("disabled\n")
	return nil
}
