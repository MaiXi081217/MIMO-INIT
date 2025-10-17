package compress

import (
    "archive/tar"
    "bytes"
    "compress/gzip"
    "crypto/sha256"
    "fmt"
    "io"
    "os"
    "path/filepath"
)

// CompressDir 压缩指定目录并返回 tar.gz 数据和 SHA256
func CompressDir(srcDir string) ([]byte, string, error) {
    buf := &bytes.Buffer{}
    gw := gzip.NewWriter(buf)
    defer gw.Close()
    tw := tar.NewWriter(gw)
    defer tw.Close()

    err := filepath.Walk(srcDir, func(path string, info os.FileInfo, err error) error {
        if err != nil { return err }
        relPath, err := filepath.Rel(filepath.Dir(srcDir), path)
        if err != nil { return err }

        hdr, err := tar.FileInfoHeader(info, "")
        if err != nil { return err }
        hdr.Name = relPath
        if err := tw.WriteHeader(hdr); err != nil { return err }

        if info.Mode().IsRegular() {
            f, err := os.Open(path)
            if err != nil { return err }
            defer f.Close()
            _, err = io.Copy(tw, f)
            return err
        }
        return nil
    })
    if err != nil { return nil, "", err }

    hash := sha256.Sum256(buf.Bytes())
    return buf.Bytes(), fmt.Sprintf("%x", hash), nil
}
