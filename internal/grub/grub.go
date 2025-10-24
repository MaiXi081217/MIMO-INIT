/*
Changes summary:
- Converted user-visible outputs to concise English.
- Use transaction.Action objects (matches internal/transaction) instead of old txn.Add signature.
- Normalize paths with filepath.Clean, preserve original content for Undo.
- Keep behavior: modify /etc/default/grub and add initramfs script, run update-grub/update-initramfs.
- Why: clearer, consistent transaction API, safer rollback, less noisy output, easier maintenance.
- Difference: previous file used a different txn.Add call shape and printed Chinese messages; now it's English, transactional, and returns proper errors.
*/
package grub

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"

	"resourcemgr/internal/transaction"
)

func RegisterGrubAndInitActions(txn *transaction.Transaction) error {
	if txn == nil {
		return fmt.Errorf("nil transaction")
	}

	// -------------------------------
	// 1. Modify GRUB_CMDLINE_LINUX_DEFAULT
	// -------------------------------
	grubFile := filepath.Clean("/etc/default/grub")
	var origGrub []byte
	if b, err := os.ReadFile(grubFile); err == nil {
		origGrub = b
	}

	modifyGrub := &transaction.Action{
		Name: "modify grub cmdline",
		Do: func() error {
			data := string(origGrub)
			re := regexp.MustCompile(`(?m)^GRUB_CMDLINE_LINUX_DEFAULT=.*$`)
			newLine := `GRUB_CMDLINE_LINUX_DEFAULT="quiet loglevel=0 systemd.show_status=0"`

			if re.MatchString(data) {
				data = re.ReplaceAllString(data, newLine)
			} else {
				if !strings.HasSuffix(data, "\n") {
					data += "\n"
				}
				data += newLine + "\n"
			}

			if err := os.WriteFile(grubFile, []byte(data), 0644); err != nil {
				return fmt.Errorf("write grub file: %w", err)
			}

			// update-grub: surface concise error on failure
			if out, err := exec.Command("update-grub").CombinedOutput(); err != nil {
				return fmt.Errorf("update-grub failed: %v: %s", err, strings.TrimSpace(string(out)))
			}

			fmt.Println("INFO: grub updated")
			return nil
		},
		Undo: func() error {
			// restore original grub file if existed
			if len(origGrub) > 0 {
				if err := os.WriteFile(grubFile, origGrub, 0644); err != nil {
					return fmt.Errorf("restore grub file: %w", err)
				}
				// best-effort update-grub, ignore error
				_ = exec.Command("update-grub").Run()
			}
			return nil
		},
	}
	txn.Add(modifyGrub)

	// -------------------------------
	// 2. initramfs script to show brief message (init-top)
	// -------------------------------
	initPath := filepath.Clean("/etc/initramfs-tools/scripts/init-top/mimo-msg")
	var origInit []byte
	if b, err := os.ReadFile(initPath); err == nil {
		origInit = b
	}

	initContent := `#!/bin/sh
echo ">>> Initializing MIMO Live Server (initramfs) <<<" > /dev/console
`

	addInit := &transaction.Action{
		Name: "add initramfs mimo-msg",
		Do: func() error {
			if err := os.MkdirAll(filepath.Dir(initPath), 0755); err != nil {
				return fmt.Errorf("mkdir init path: %w", err)
			}
			if err := os.WriteFile(initPath, []byte(initContent), 0755); err != nil {
				return fmt.Errorf("write init script: %w", err)
			}
			if out, err := exec.Command("update-initramfs", "-u").CombinedOutput(); err != nil {
				return fmt.Errorf("update-initramfs failed: %v: %s", err, strings.TrimSpace(string(out)))
			}
			fmt.Println("INFO: initramfs script installed")
			return nil
		},
		Undo: func() error {
			// restore original or remove
			if len(origInit) > 0 {
				if err := os.WriteFile(initPath, origInit, 0755); err != nil {
					return fmt.Errorf("restore init script: %w", err)
				}
			} else {
				_ = os.Remove(initPath)
			}
			_ = exec.Command("update-initramfs", "-u").Run()
			return nil
		},
	}
	txn.Add(addInit)

	return nil
}
