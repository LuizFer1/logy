package update

import (
	"archive/tar"
	"archive/zip"
	"bytes"
	"compress/gzip"
	"os"
	"path/filepath"
	"testing"
)

func TestExtractBinaryZip(t *testing.T) {
	payload := []byte("fake-logy-binary")
	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)
	w, err := zw.Create("subdir/logy.exe")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := w.Write(payload); err != nil {
		t.Fatal(err)
	}
	// Extra file should be ignored.
	w2, err := zw.Create("README.md")
	if err != nil {
		t.Fatal(err)
	}
	_, _ = w2.Write([]byte("docs"))
	if err := zw.Close(); err != nil {
		t.Fatal(err)
	}

	dest := filepath.Join(t.TempDir(), "out.exe")
	if err := ExtractBinary(buf.Bytes(), "logy_1.0.0_windows_amd64.zip", dest); err != nil {
		t.Fatal(err)
	}
	got, err := os.ReadFile(dest)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(got, payload) {
		t.Fatalf("got %q want %q", got, payload)
	}
}

func TestExtractBinaryTarGz(t *testing.T) {
	payload := []byte("unix-logy")
	var buf bytes.Buffer
	gw := gzip.NewWriter(&buf)
	tw := tar.NewWriter(gw)
	hdr := &tar.Header{
		Name: "logy_1.0.0_linux_amd64/logy",
		Mode: 0755,
		Size: int64(len(payload)),
	}
	if err := tw.WriteHeader(hdr); err != nil {
		t.Fatal(err)
	}
	if _, err := tw.Write(payload); err != nil {
		t.Fatal(err)
	}
	extra := []byte("license")
	if err := tw.WriteHeader(&tar.Header{Name: "LICENSE", Mode: 0644, Size: int64(len(extra))}); err != nil {
		t.Fatal(err)
	}
	if _, err := tw.Write(extra); err != nil {
		t.Fatal(err)
	}
	if err := tw.Close(); err != nil {
		t.Fatal(err)
	}
	if err := gw.Close(); err != nil {
		t.Fatal(err)
	}

	dest := filepath.Join(t.TempDir(), "logy")
	if err := ExtractBinary(buf.Bytes(), "logy_1.0.0_linux_amd64.tar.gz", dest); err != nil {
		t.Fatal(err)
	}
	got, err := os.ReadFile(dest)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(got, payload) {
		t.Fatalf("got %q want %q", got, payload)
	}
}

func TestExtractBinaryMissing(t *testing.T) {
	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)
	w, err := zw.Create("README.md")
	if err != nil {
		t.Fatal(err)
	}
	_, _ = w.Write([]byte("no binary"))
	_ = zw.Close()
	dest := filepath.Join(t.TempDir(), "out")
	if err := ExtractBinary(buf.Bytes(), "empty.zip", dest); err == nil {
		t.Fatal("expected error")
	}
}
