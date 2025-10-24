package decompress

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"crypto/sha256"
	"fmt"
	"io"
	"os"
	"path/filepath"

	_ "embed"
)

//go:embed resources.tar.gz
var embeddedResources []byte

//go:embed resources.sha256
var embeddedHash []byte

// ExtractResources extracts embedded resources into the target directory
func ExtractResources(dest string) error {
	reader := bytes.NewReader(embeddedResources)
	gzr, err := gzip.NewReader(reader)
	if err != nil {
		return fmt.Errorf("failed to create gzip reader: %v", err)
	}
	defer gzr.Close()

	tr := tar.NewReader(gzr)

	for {
		hdr, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return fmt.Errorf("failed to read tar entry: %v", err)
		}

		targetPath := filepath.Join(dest, hdr.Name)

		switch hdr.Typeflag {
		case tar.TypeDir:
			if err := os.MkdirAll(targetPath, 0755); err != nil {
				return fmt.Errorf("failed to create directory: %v", err)
			}
			fmt.Printf("[INFO] Created directory: %s\n", targetPath)

		case tar.TypeReg:
			if err := os.MkdirAll(filepath.Dir(targetPath), 0755); err != nil {
				return fmt.Errorf("failed to create parent directory: %v", err)
			}
			f, err := os.OpenFile(targetPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, os.FileMode(hdr.Mode))
			if err != nil {
				return fmt.Errorf("failed to create file: %v", err)
			}
			if _, err := io.Copy(f, tr); err != nil {
				f.Close()
				return fmt.Errorf("failed to write file: %v", err)
			}
			f.Close()
			fmt.Printf("[INFO] Extracted file: %s\n", targetPath)

		case tar.TypeSymlink:
			linkTarget := filepath.Join(dest, hdr.Linkname)
			if err := os.MkdirAll(filepath.Dir(targetPath), 0755); err != nil {
				return fmt.Errorf("failed to create directory for symlink: %v", err)
			}
			os.Remove(targetPath) // remove existing file if any
			if err := os.Symlink(linkTarget, targetPath); err != nil {
				return fmt.Errorf("failed to create symlink: %v", err)
			}
			fmt.Printf("[INFO] Created symlink: %s -> %s\n", targetPath, linkTarget)

		default:
			fmt.Printf("[WARN] Skipping unsupported type: %s\n", hdr.Name)
		}
	}

	return nil
}

// VerifyHash verifies the SHA256 of the embedded resources
func VerifyHash() bool {
	sum := sha256.Sum256(embeddedResources)
	calculated := fmt.Sprintf("%x", sum[:])
	expected := string(bytes.TrimSpace(embeddedHash))
	if calculated != expected {
		fmt.Printf("[ERROR] SHA256 mismatch! Calculated: %s, Expected: %s\n", calculated, expected)
		return false
	}
	fmt.Println("[INFO] SHA256 verified successfully")
	return true
}
