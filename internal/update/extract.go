package update

import (
	"archive/tar"
	"archive/zip"
	"bytes"
	"compress/gzip"
	"fmt"
	"io"
	"os"
	"path"
	"path/filepath"
	"strings"
)

// ExtractBinary reads zip or tar.gz and writes the logy binary to destPath (full path).
func ExtractBinary(archive []byte, archiveName, destPath string) error {
	name := strings.ToLower(archiveName)
	var data []byte
	var err error
	switch {
	case strings.HasSuffix(name, ".zip"):
		data, err = extractZipBinary(archive)
	case strings.HasSuffix(name, ".tar.gz") || strings.HasSuffix(name, ".tgz"):
		data, err = extractTarGzBinary(archive)
	default:
		return fmt.Errorf("unsupported archive format: %s", archiveName)
	}
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(destPath), 0755); err != nil {
		return err
	}
	if err := os.WriteFile(destPath, data, 0755); err != nil {
		return err
	}
	return os.Chmod(destPath, 0755)
}

func extractZipBinary(archive []byte) ([]byte, error) {
	zr, err := zip.NewReader(bytes.NewReader(archive), int64(len(archive)))
	if err != nil {
		return nil, err
	}
	for _, f := range zr.File {
		base := path.Base(filepath.ToSlash(f.Name))
		if !isLogyBinaryName(base) || f.FileInfo().IsDir() {
			continue
		}
		rc, err := f.Open()
		if err != nil {
			return nil, err
		}
		data, err := io.ReadAll(rc)
		_ = rc.Close()
		if err != nil {
			return nil, err
		}
		return data, nil
	}
	return nil, fmt.Errorf("logy binary not found in zip archive")
}

func extractTarGzBinary(archive []byte) ([]byte, error) {
	gr, err := gzip.NewReader(bytes.NewReader(archive))
	if err != nil {
		return nil, err
	}
	defer gr.Close()
	tr := tar.NewReader(gr)
	for {
		hdr, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, err
		}
		if hdr.Typeflag != tar.TypeReg && hdr.Typeflag != tar.TypeRegA {
			continue
		}
		base := path.Base(hdr.Name)
		if !isLogyBinaryName(base) {
			continue
		}
		return io.ReadAll(tr)
	}
	return nil, fmt.Errorf("logy binary not found in tar.gz archive")
}

func isLogyBinaryName(base string) bool {
	return base == "logy" || base == "logy.exe"
}

// CleanupStaleOld removes a leftover exePath+".old" from a previous Windows update.
func CleanupStaleOld(exePath string) {
	_ = os.Remove(exePath + ".old")
}
