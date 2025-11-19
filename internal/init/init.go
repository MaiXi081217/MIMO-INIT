/*
统一初始化管理系统
将所有初始化文件的内容集中管理在一个配置文件中，而不是分散在多个文件中
*/
package init

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"mimo/internal/fileops"
	"mimo/internal/transaction"
)

// InitFileConfig 定义单个初始化文件的配置
type InitFileConfig struct {
	// 目标路径（系统上的最终位置）
	Path string `json:"path"`
	// 文件内容（直接内嵌）
	Content string `json:"content"`
	// 文件权限（八进制，如 "0755"）
	Mode string `json:"mode"`
	// 文件类型：script, service, config, text
	Type string `json:"type,omitempty"`
	// 是否为目录
	IsDir bool `json:"is_dir,omitempty"`
}

// InitConfig 统一的初始化配置
type InitConfig struct {
	// 所有需要创建的初始化文件
	Files []InitFileConfig `json:"files"`
	// 需要启用的systemd服务列表
	Services []string `json:"services,omitempty"`
	// 版本信息
	Version string `json:"version,omitempty"`
}

// LoadInitConfig 从文件加载初始化配置
func LoadInitConfig(path string) (*InitConfig, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read init config: %w", err)
	}

	var cfg InitConfig
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("parse init config: %w", err)
	}

	return &cfg, nil
}

// RegisterInitActions 注册初始化动作到事务中
// 从统一的配置文件中读取所有文件内容并创建它们
func RegisterInitActions(txn *transaction.Transaction, cfg *InitConfig, tmpDir string) error {
	if txn == nil || cfg == nil {
		return fmt.Errorf("nil txn or cfg")
	}

	// 在临时目录中创建所有文件
	for _, fileCfg := range cfg.Files {
		if fileCfg.IsDir {
			continue // 目录由文件创建时自动处理
		}

		// 在临时目录中创建文件（使用相对路径避免路径冲突）
		// 使用文件路径的hash或相对路径作为临时文件名
		relPath := strings.TrimPrefix(filepath.Clean(fileCfg.Path), "/")
		srcPath := filepath.Join(tmpDir, "init-files", relPath)
		dstPath := filepath.Clean(fileCfg.Path)

		// 捕获变量用于闭包
		src, dst, content, mode := srcPath, dstPath, fileCfg.Content, fileCfg.Mode

		action := &transaction.Action{
			Name: fmt.Sprintf("create init file %s", dst),
			Do: func() error {
				// 创建临时源文件
				if err := os.MkdirAll(filepath.Dir(src), 0755); err != nil {
					return fmt.Errorf("mkdir src dir: %w", err)
				}

				// 解析文件权限
				var fileMode os.FileMode = 0644
				if mode != "" {
					var m uint32
					if _, err := fmt.Sscanf(mode, "%o", &m); err == nil {
						fileMode = os.FileMode(m)
					}
				}

				// 写入内容到临时文件
				if err := os.WriteFile(src, []byte(content), fileMode); err != nil {
					return fmt.Errorf("write src file: %w", err)
				}

				// 复制到目标位置
				if err := fileops.CopyFile(src, dst); err != nil {
					return fmt.Errorf("copy to dst: %w", err)
				}

				return nil
			},
			Undo: func() error {
				// 删除目标文件
				if err := os.RemoveAll(dst); err != nil {
					return fmt.Errorf("remove dst: %w", err)
				}
				return nil
			},
		}
		txn.Add(action)
	}

	return nil
}

// GenerateInitConfigFromFiles 从现有file文件夹生成统一的初始化配置
// 这是一个辅助函数，用于迁移现有系统
func GenerateInitConfigFromFiles(fileDir string, fileMappings []fileops.FileMapping) (*InitConfig, error) {
	cfg := &InitConfig{
		Files:    make([]InitFileConfig, 0),
		Services: make([]string, 0),
	}

	for _, mapping := range fileMappings {
		srcPath := mapping.Src
		dstPath := mapping.Dst

		// 跳过目录（SPDK_for_MIMO）
		if fi, err := os.Stat(srcPath); err == nil && fi.IsDir() {
			continue
		}

		// 读取文件内容
		content, err := os.ReadFile(srcPath)
		if err != nil {
			return nil, fmt.Errorf("read file %s: %w", srcPath, err)
		}

		// 获取文件权限
		fi, err := os.Stat(srcPath)
		if err != nil {
			return nil, fmt.Errorf("stat file %s: %w", srcPath, err)
		}
		mode := fmt.Sprintf("%04o", fi.Mode().Perm())

		// 确定文件类型
		fileType := "text"
		baseName := filepath.Base(dstPath)
		if filepath.Ext(baseName) == ".sh" {
			fileType = "script"
		} else if filepath.Ext(baseName) == ".service" {
			fileType = "service"
			// 提取服务名
			serviceName := baseName
			cfg.Services = append(cfg.Services, serviceName)
		} else if filepath.Ext(baseName) == ".conf" {
			fileType = "config"
		}

		fileCfg := InitFileConfig{
			Path:    dstPath,
			Content: string(content),
			Mode:    mode,
			Type:    fileType,
		}

		cfg.Files = append(cfg.Files, fileCfg)
	}

	return cfg, nil
}
