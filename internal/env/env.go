package env

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"resourcemgr/internal/fileops"
	"strings"
)

const (
	defaultMimoRoot = "/usr/local/mimo"
)

type VersionMapping struct {
	Src string `json:"src"`
	Dst string `json:"dst"`
}

type VersionConfig struct {
	Version []VersionMapping `json:"version"`
}

func ConfirmPrompt(msg string) bool {
	fmt.Print(msg)
	var ans string
	fmt.Scanln(&ans)
	return strings.ToLower(strings.TrimSpace(ans)) == "y"
}

func EnsureMimoRoot() string {
	mimoRoot := os.Getenv("MIMO_ROOT")
	if mimoRoot == "" {
		mimoRoot = defaultMimoRoot
		if err := os.Setenv("MIMO_ROOT", mimoRoot); err != nil {
			log.Fatalf("[ERROR] failed to set MIMO_ROOT: %v", err)
		}
		fmt.Printf("[INFO] Environment variable MIMO_ROOT not set, defaulting to %s\n", mimoRoot)

		// Persist environment variable
		profilePath := "/etc/profile.d/mimo_root.sh"
		content := fmt.Sprintf("export MIMO_ROOT=%s\n", mimoRoot)
		if err := os.WriteFile(profilePath, []byte(content), 0644); err != nil {
			log.Printf("[WARN] failed to persist MIMO_ROOT to %s: %v\n", profilePath, err)
		} else {
			fmt.Printf("[INFO] Persisted MIMO_ROOT to %s\n", profilePath)
		}
	}
	return mimoRoot
}
func LoadFileOpsConfig(path string) *fileops.Config {
	data, err := os.ReadFile(path)
	if err != nil {
		log.Fatalf("[ERROR] reading config.json failed: %v", err)
	}

	cfg := &fileops.Config{}
	if err := json.Unmarshal(data, cfg); err != nil {
		log.Fatalf("[ERROR] parsing config.json failed: %v", err)
	}
	return cfg
}

func LoadVersionConfig(path string) *VersionConfig {
	data, err := os.ReadFile(path)
	if err != nil {
		log.Fatalf("[ERROR] reading config.json failed: %v", err)
	}

	cfg := &VersionConfig{}
	if err := json.Unmarshal(data, cfg); err != nil {
		log.Fatalf("[ERROR] parsing config.json failed: %v", err)
	}
	if len(cfg.Version) < 2 {
		log.Fatalf("[ERROR] version field malformed, need src and dst")
	}
	return cfg
}

func ReadMimoVersion(path string) string {
	data, err := os.ReadFile(path)
	if err != nil {
		return "v0.0.0"
	}
	var v map[string]interface{}
	if err := json.Unmarshal(data, &v); err != nil {
		return "v0.0.0"
	}
	if mimo, ok := v["MIMO"].(string); ok && mimo != "" {
		return mimo
	}
	return "v0.0.0"
}
