/*
Changes:
- RegisterMOTDActions: add a transaction action that backs up /etc/update-motd.d and clears it.
- DisableMotd: helper to remove files under /etc/update-motd.d.
- Purpose: keep MOTD actions transactional and reversible.
*/
package motd

import (
	"fmt"
	"io"
	"os"
	"path/filepath"

	"resourcemgr/internal/transaction"
)

const (
	motdDir     = "/etc/update-motd.d"
	motdBakRoot = "/var/lib/mimo/motd-backup"
)

// RegisterMOTDActions registers an action that backs up motd scripts and clears the directory.
// Undo restores from backup.
func RegisterMOTDActions(txn *transaction.Transaction) error {
	if txn == nil {
		return fmt.Errorf("nil transaction")
	}

	action := &transaction.Action{
		Name: "motd backup and disable",
		Do: func() error {
			// ensure backup dir
			if err := os.MkdirAll(motdBakRoot, 0755); err != nil {
				return fmt.Errorf("mkdir backup: %w", err)
			}
			entries, err := os.ReadDir(motdDir)
			if err != nil && !os.IsNotExist(err) {
				return fmt.Errorf("read motd dir: %w", err)
			}
			for _, e := range entries {
				src := filepath.Join(motdDir, e.Name())
				dst := filepath.Join(motdBakRoot, e.Name())
				// copy file
				if err := copyFile(src, dst); err != nil {
					return err
				}
				// remove original
				if err := os.Remove(src); err != nil {
					return err
				}
			}
			return nil
		},
		Undo: func() error {
			entries, err := os.ReadDir(motdBakRoot)
			if err != nil && !os.IsNotExist(err) {
				return fmt.Errorf("read backup: %w", err)
			}
			for _, e := range entries {
				src := filepath.Join(motdBakRoot, e.Name())
				dst := filepath.Join(motdDir, e.Name())
				if err := os.MkdirAll(motdDir, 0755); err != nil {
					return err
				}
				if err := copyFile(src, dst); err != nil {
					return err
				}
			}
			// best-effort: leave backup in place
			return nil
		},
	}

	txn.Add(action)
	return nil
}

// DisableMotd removes all files in /etc/update-motd.d
func DisableMotd() error {
	entries, err := os.ReadDir(motdDir)
	if err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("read motd dir: %w", err)
	}
	for _, e := range entries {
		p := filepath.Join(motdDir, e.Name())
		if err := os.RemoveAll(p); err != nil {
			return fmt.Errorf("remove %s: %w", p, err)
		}
	}
	return nil
}

// small helper to copy file content
func copyFile(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()

	if err := os.MkdirAll(filepath.Dir(dst), 0755); err != nil {
		return err
	}

	out, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer out.Close()

	if _, err := io.Copy(out, in); err != nil {
		return err
	}
	return nil
}
