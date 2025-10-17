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

	// -------------------------------
	// 嵌入资源
	// -------------------------------
	_ "embed"
)

//go:embed resources.tar.gz
var embeddedResources []byte

// SHA256 值直接嵌入
//
//go:embed resources.sha256
var embeddedHash []byte

// ExtractResources 解压到目标目录
func ExtractResources(dest string) error {
	reader := bytes.NewReader(embeddedResources)
	gzr, err := gzip.NewReader(reader)
	if err != nil {
		return fmt.Errorf("创建 gzip reader 失败: %v", err)
	}
	defer gzr.Close()

	tr := tar.NewReader(gzr)

	for {
		hdr, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return fmt.Errorf("读取 tar 文件失败: %v", err)
		}

		targetPath := filepath.Join(dest, hdr.Name)
		if hdr.Typeflag == tar.TypeDir {
			os.MkdirAll(targetPath, 0755)
			continue
		}

		if err := os.MkdirAll(filepath.Dir(targetPath), 0755); err != nil {
			return fmt.Errorf("创建目录失败: %v", err)
		}

		f, err := os.OpenFile(targetPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0644)
		if err != nil {
			return fmt.Errorf("创建文件失败: %v", err)
		}

		if _, err := io.Copy(f, tr); err != nil {
			f.Close()
			return fmt.Errorf("写入文件失败: %v", err)
		}
		f.Close()
	}
	return nil
}

// VerifyHash 校验嵌入资源 SHA256
func VerifyHash() bool {
	sum := sha256.Sum256(embeddedResources)
	calculated := fmt.Sprintf("%x", sum[:])
	expected := string(bytes.TrimSpace(embeddedHash))
	return calculated == expected
}
