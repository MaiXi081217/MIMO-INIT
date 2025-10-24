/*
Changes:
- Implement CopyFile and CopyDir with proper path cleaning and mode preservation.
- Provide RegisterCopyActions to register copy/undo actions into a Transaction.
- Purpose: make file copy operations reusable and transactional.
*/
package fileops

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"resourcemgr/internal/transaction"
)

type FileMapping struct {
	Src string `json:"src"`
	Dst string `json:"dst"`
}

type Config struct {
	FileMappings []FileMapping `json:"file_mappings"`
}

func LoadConfig(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var cfg Config
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, err
	}
	return &cfg, nil
}

// CopyFile copies a single file from src to dst, creating parent dirs and preserving file mode.
func CopyFile(src, dst string) error {
	src = filepath.Clean(src)
	dst = filepath.Clean(dst)

	sfi, err := os.Stat(src)
	if err != nil {
		return fmt.Errorf("stat src %s: %w", src, err)
	}
	if sfi.IsDir() {
		return fmt.Errorf("source %s is a directory", src)
	}

	if err := os.MkdirAll(filepath.Dir(dst), 0755); err != nil {
		return fmt.Errorf("mkdir dst dir %s: %w", filepath.Dir(dst), err)
	}

	in, err := os.Open(src)
	if err != nil {
		return fmt.Errorf("open src %s: %w", src, err)
	}
	defer in.Close()

	out, err := os.Create(dst)
	if err != nil {
		return fmt.Errorf("create dst %s: %w", dst, err)
	}
	defer func() {
		out.Close()
	}()

	if _, err := io.Copy(out, in); err != nil {
		return fmt.Errorf("copy %s -> %s: %w", src, dst, err)
	}

	if err := out.Chmod(sfi.Mode()); err != nil {
		// not fatal, but propagate
		return fmt.Errorf("chmod %s: %w", dst, err)
	}
	return nil
}

// CopyDir recursively copies src directory to dst.
func CopyDir(src, dst string) error {
	src = filepath.Clean(src)
	dst = filepath.Clean(dst)

	sfi, err := os.Stat(src)
	if err != nil {
		return fmt.Errorf("stat src %s: %w", src, err)
	}
	if !sfi.IsDir() {
		return fmt.Errorf("source %s is not a directory", src)
	}

	err = filepath.Walk(src, func(path string, info os.FileInfo, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		rel, err := filepath.Rel(src, path)
		if err != nil {
			return err
		}
		target := filepath.Join(dst, rel)
		if info.IsDir() {
			if err := os.MkdirAll(target, info.Mode()); err != nil {
				return err
			}
			return nil
		}
		// file
		if err := CopyFile(path, target); err != nil {
			return err
		}
		return nil
	})
	if err != nil {
		return fmt.Errorf("copydir %s -> %s: %w", src, dst, err)
	}
	return nil
}

// RegisterCopyActions registers copy actions described by cfg into txn.
// Each action Do performs copy; Undo removes the destination.
func RegisterCopyActions(txn *transaction.Transaction, cfg *Config) error {
	if txn == nil || cfg == nil {
		return fmt.Errorf("nil txn or cfg")
	}
	for _, m := range cfg.FileMappings {
		src := filepath.Clean(m.Src)
		dst := filepath.Clean(m.Dst)
		name := fmt.Sprintf("copy %s -> %s", src, dst)

		// capture variables for closure
		s, d := src, dst
		action := &transaction.Action{
			Name: name,
			Do: func() error {
				info, err := os.Stat(s)
				if err != nil {
					return fmt.Errorf("stat source %s: %w", s, err)
				}
				// remove existing dst first to ensure clean replace
				_ = os.RemoveAll(d)
				if info.IsDir() {
					return CopyDir(s, d)
				}
				return CopyFile(s, d)
			},
			Undo: func() error {
				// best-effort remove destination
				if err := os.RemoveAll(d); err != nil {
					return fmt.Errorf("undo remove %s: %w", d, err)
				}
				return nil
			},
		}
		txn.Add(action)
	}
	return nil
}

// ========== 辅助函数 ==========
