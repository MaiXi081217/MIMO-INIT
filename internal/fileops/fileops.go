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

func RegisterCopyActions(txn *transaction.Transaction, cfg *Config) error {
	for _, m := range cfg.FileMappings {
		src := m.Src
		dst := m.Dst

		backupPath := ""
		if _, err := os.Stat(dst); err == nil {
			backupPath = fmt.Sprintf("%s.bak.%d", dst, os.Getpid())
			os.MkdirAll(filepath.Dir(backupPath), 0755)
			if err := os.Rename(dst, backupPath); err != nil {
				return fmt.Errorf("备份目标失败: %v", err)
			}
		}

		do := func() error {
			info, err := os.Stat(src)
			if err != nil {
				return fmt.Errorf("source not found: %s", src)
			}

			fmt.Printf("[INFO] 正在复制文件: %s -> %s\n", src, dst)
			os.MkdirAll(filepath.Dir(dst), 0755)

			if info.IsDir() {
				return copyDir(src, dst)
			}
			return copyFile(src, dst)
		}

		undo := func() error {
			if backupPath != "" {
				_ = os.RemoveAll(dst)
				return os.Rename(backupPath, dst)
			}
			return os.RemoveAll(dst)
		}

		txn.Add(fmt.Sprintf("copy %s -> %s", src, dst), do, undo)
	}
	return nil
}

// ========== 辅助函数 ==========

// 复制单个文件
func copyFile(src, dst string) error {
	sf, err := os.Open(src)
	if err != nil {
		return err
	}
	defer sf.Close()

	df, err := os.OpenFile(dst, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0644)
	if err != nil {
		return err
	}
	defer df.Close()

	if _, err := io.Copy(df, sf); err != nil {
		return err
	}

	// 自动加执行权限
	ext := filepath.Ext(dst)
	if ext == ".sh" || ext == ".service" {
		if err := os.Chmod(dst, 0755); err != nil {
			return fmt.Errorf("设置执行权限失败: %v", err)
		}
	}

	return nil
}

// 递归复制整个目录
func copyDir(src, dst string) error {
	entries, err := os.ReadDir(src)
	if err != nil {
		return err
	}

	if err := os.MkdirAll(dst, 0755); err != nil {
		return err
	}

	for _, entry := range entries {
		srcPath := filepath.Join(src, entry.Name())
		dstPath := filepath.Join(dst, entry.Name())

		info, err := entry.Info()
		if err != nil {
			return err
		}

		if info.IsDir() {
			if err := copyDir(srcPath, dstPath); err != nil {
				return err
			}
		} else {
			if err := copyFile(srcPath, dstPath); err != nil {
				return err
			}
		}
	}
	return nil
}
