/*
Changes:
- Implement EnableServices: collect .service files from file mappings, run daemon-reload once, then enable+start each service.
- Non-fatal per-service: warn and continue. Returns error only if daemon-reload fails.
- Keeps user-facing messages concise.
*/
package systemd

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"mimo/internal/fileops"
)

func EnableServices(cfg *fileops.Config) error {
	if cfg == nil {
		return nil
	}
	services := make(map[string]struct{})
	for _, m := range cfg.FileMappings {
		dst := filepath.Clean(m.Dst)
		if strings.HasSuffix(strings.ToLower(dst), ".service") {
			if _, err := os.Stat(dst); err == nil {
				services[filepath.Base(dst)] = struct{}{}
			}
		}
	}
	if len(services) == 0 {
		return nil
	}

	// reload once
	if err := exec.Command("systemctl", "daemon-reload").Run(); err != nil {
		return fmt.Errorf("daemon-reload failed: %w", err)
	}

	for s := range services {
		// enable (synchronous)
		if err := exec.Command("systemctl", "enable", s).Run(); err != nil {
			log.Printf("WARN: enable %s failed", s)
			// continue to attempt start --no-block even if enable failed
		}

		// start non-blocking to avoid waiting for service activation
		if err := exec.Command("systemctl", "start", "--no-block", s).Run(); err != nil {
			log.Printf("WARN: start (non-blocking) %s failed", s)
			continue
		}
	}
	return nil
}
