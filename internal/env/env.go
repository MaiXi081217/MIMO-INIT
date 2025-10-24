
package env

import (
	"bufio"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"resourcemgr/internal/fileops"
	"strings"
)

const (
	defaultMimoRoot = "/usr/local/mimo"
	profilePath     = "/etc/profile.d/mimo_root.sh"
	defaultVersion  = "v0.0.0"
)

type VersionMapping struct {
	Src string `json:"src"`
	Dst string `json:"dst"`
}

type VersionConfig struct {
	Version []VersionMapping `json:"version"`
}

func MustBeRoot() {
	if os.Geteuid() != 0 {
		log.Fatalf("ERROR: must run as root")
	}
}

// ConfirmPrompt reads a single line from stdin and returns true if the trimmed
// lowercased answer equals "y". On read error it conservatively returns false.
func ConfirmPrompt(msg string) bool {
	fmt.Print(msg)
	r := bufio.NewReader(os.Stdin)
	line, err := r.ReadString('\n')
	if err != nil {
		// Do not expose internal error details to user; return negative response.
		return false
	}
	return strings.ToLower(strings.TrimSpace(line)) == "y"
}

// EnsureMimoRoot ensures MIMO_ROOT is set. If absent, set to default and try to persist.
// On persistence failure a warning is logged but function continues.
func EnsureMimoRoot() string {
	mimoRoot := os.Getenv("MIMO_ROOT")
	if mimoRoot == "" {
		mimoRoot = defaultMimoRoot
		if err := os.Setenv("MIMO_ROOT", mimoRoot); err != nil {
			log.Fatalf("ERROR: failed to set MIMO_ROOT: %v", err)
		}
		fmt.Printf("INFO: MIMO_ROOT set to %s\n", mimoRoot)

		content := fmt.Sprintf("export MIMO_ROOT=%s\n", mimoRoot)
		if err := os.WriteFile(filepath.Clean(profilePath), []byte(content), 0644); err != nil {
			log.Printf("WARN: failed to persist MIMO_ROOT (will continue): %v", err)
		} else {
			fmt.Println("INFO: MIMO_ROOT persisted")
		}
	}
	return mimoRoot
}

// LoadFileOpsConfig loads file operations config from path. Fatal on error (preserve original behavior).
func LoadFileOpsConfig(path string) *fileops.Config {
	cleanPath := filepath.Clean(path)
	data, err := os.ReadFile(cleanPath)
	if err != nil {
		log.Fatalf("ERROR: failed to read %s: %v", cleanPath, err)
	}

	cfg := &fileops.Config{}
	if err := json.Unmarshal(data, cfg); err != nil {
		log.Fatalf("ERROR: failed to parse %s: %v", cleanPath, err)
	}
	return cfg
}

// LoadVersionConfig loads version mapping config from path. Fatal on error or malformed content.
func LoadVersionConfig(path string) *VersionConfig {
	cleanPath := filepath.Clean(path)
	data, err := os.ReadFile(cleanPath)
	if err != nil {
		log.Fatalf("ERROR: failed to read %s: %v", cleanPath, err)
	}

	cfg := &VersionConfig{}
	if err := json.Unmarshal(data, cfg); err != nil {
		log.Fatalf("ERROR: failed to parse %s: %v", cleanPath, err)
	}
	if len(cfg.Version) < 2 {
		log.Fatalf("ERROR: version config malformed: need src and dst entries")
	}
	return cfg
}

// ReadMimoVersion reads a JSON file and returns the MIMO field, or defaultVersion on error.
func ReadMimoVersion(path string) string {
	cleanPath := filepath.Clean(path)
	data, err := os.ReadFile(cleanPath)
	if err != nil {
		return defaultVersion
	}
	var v struct {
		MIMO string `json:"MIMO"`
	}
	if err := json.Unmarshal(data, &v); err != nil {
		return defaultVersion
	}
	if v.MIMO == "" {
		return defaultVersion
	}
	return v.MIMO
}
