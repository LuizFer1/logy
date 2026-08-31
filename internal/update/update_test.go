package update

import (
	"archive/tar"
	"archive/zip"
	"bytes"
	"compress/gzip"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

type mapGetter map[string][]byte

func (m mapGetter) Get(url string) ([]byte, error) {
	body, ok := m[url]
	if !ok {
		return nil, fmt.Errorf("unexpected GET %s", url)
	}
	return body, nil
}

func buildTestArchive(t *testing.T, goos string, payload []byte) (name string, data []byte) {
	t.Helper()
	binName := "logy"
	if goos == "windows" {
		binName = "logy.exe"
		var buf bytes.Buffer
		zw := zip.NewWriter(&buf)
		w, err := zw.Create(binName)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := w.Write(payload); err != nil {
			t.Fatal(err)
		}
		if err := zw.Close(); err != nil {
			t.Fatal(err)
		}
		return "logy_1.2.0_windows_amd64.zip", buf.Bytes()
	}
	var buf bytes.Buffer
	gw := gzip.NewWriter(&buf)
	tw := tar.NewWriter(gw)
	hdr := &tar.Header{Name: binName, Mode: 0755, Size: int64(len(payload))}
	if err := tw.WriteHeader(hdr); err != nil {
		t.Fatal(err)
	}
	if _, err := tw.Write(payload); err != nil {
		t.Fatal(err)
	}
	if err := tw.Close(); err != nil {
		t.Fatal(err)
	}
	if err := gw.Close(); err != nil {
		t.Fatal(err)
	}
	return fmt.Sprintf("logy_1.2.0_%s_amd64.tar.gz", goos), buf.Bytes()
}

func TestRunHappyPath(t *testing.T) {
	goos, goarch := runtime.GOOS, "amd64"
	payload := []byte("updated-binary-v1.2.0")
	assetName, archive := buildTestArchive(t, goos, payload)
	sum := sha256.Sum256(archive)
	digest := hex.EncodeToString(sum[:])

	assetURL := "https://example.com/" + assetName
	checksumURL := "https://example.com/checksums.txt"
	apiURL := "https://api.github.com/repos/LuizFer1/logy/releases/latest"

	releaseJSON := fmt.Sprintf(`{
		"tag_name": "v1.2.0",
		"assets": [
			{"name": %q, "browser_download_url": %q},
			{"name": "checksums.txt", "browser_download_url": %q}
		]
	}`, assetName, assetURL, checksumURL)

	g := mapGetter{
		apiURL:      []byte(releaseJSON),
		assetURL:    archive,
		checksumURL: []byte(digest + "  " + assetName + "\n"),
	}

	dir := t.TempDir()
	exeName := "logy"
	if goos == "windows" {
		exeName = "logy.exe"
	}
	exePath := filepath.Join(dir, exeName)
	if err := os.WriteFile(exePath, []byte("old-binary"), 0755); err != nil {
		t.Fatal(err)
	}

	var out bytes.Buffer
	res, err := Run(Options{
		CurrentVersion: "1.1.0",
		GOOS:           goos,
		GOARCH:         goarch,
		ExePath:        exePath,
		Getter:         g,
		Repo:           "LuizFer1/logy",
		Yes:            true,
		Stdout:         &out,
	})
	if err != nil {
		t.Fatal(err)
	}
	if !res.Updated || res.Pending {
		t.Fatalf("result=%+v", res)
	}
	if res.Latest != "v1.2.0" && res.Latest != "1.2.0" {
		// Prefer preserving tag from release; either normalized form is fine if documented.
		t.Fatalf("Latest=%q", res.Latest)
	}
	got, err := os.ReadFile(exePath)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(got, payload) {
		t.Fatalf("binary=%q want %q", got, payload)
	}
	text := out.String()
	if !strings.Contains(text, "Updated successfully") {
		t.Fatalf("stdout=%q", text)
	}
}

func TestRunAlreadyUpToDate(t *testing.T) {
	apiURL := "https://api.github.com/repos/LuizFer1/logy/releases/latest"
	g := mapGetter{
		apiURL: []byte(`{"tag_name":"v1.2.0","assets":[]}`),
	}
	var out bytes.Buffer
	res, err := Run(Options{
		CurrentVersion: "v1.2.0",
		GOOS:           "linux",
		GOARCH:         "amd64",
		ExePath:        filepath.Join(t.TempDir(), "logy"),
		Getter:         g,
		Repo:           "LuizFer1/logy",
		Stdout:         &out,
	})
	if err != nil {
		t.Fatal(err)
	}
	if res.Updated || res.Pending {
		t.Fatalf("result=%+v", res)
	}
	if !strings.Contains(out.String(), "Already up to date") {
		t.Fatalf("stdout=%q", out.String())
	}
}

func TestRunCheckOnlyPending(t *testing.T) {
	apiURL := "https://api.github.com/repos/LuizFer1/logy/releases/latest"
	g := mapGetter{
		apiURL: []byte(`{"tag_name":"v2.0.0","assets":[]}`),
	}
	var out bytes.Buffer
	res, err := Run(Options{
		CurrentVersion: "1.0.0",
		GOOS:           "linux",
		GOARCH:         "amd64",
		ExePath:        filepath.Join(t.TempDir(), "logy"),
		Getter:         g,
		Repo:           "LuizFer1/logy",
		CheckOnly:      true,
		Stdout:         &out,
	})
	if err != nil {
		t.Fatal(err)
	}
	if !res.Pending || res.Updated {
		t.Fatalf("result=%+v", res)
	}
	if !strings.Contains(out.String(), "Current:") || !strings.Contains(out.String(), "Latest:") {
		t.Fatalf("stdout=%q", out.String())
	}
}

func TestRunPromptDeclined(t *testing.T) {
	apiURL := "https://api.github.com/repos/LuizFer1/logy/releases/latest"
	g := mapGetter{
		apiURL: []byte(`{"tag_name":"v2.0.0","assets":[]}`),
	}
	var out bytes.Buffer
	res, err := Run(Options{
		CurrentVersion: "1.0.0",
		GOOS:           "linux",
		GOARCH:         "amd64",
		ExePath:        filepath.Join(t.TempDir(), "logy"),
		Getter:         g,
		Repo:           "LuizFer1/logy",
		Stdout:         &out,
		Prompt:         func(string) (bool, error) { return false, nil },
	})
	if err != nil {
		t.Fatal(err)
	}
	if res.Updated || res.Pending {
		t.Fatalf("result=%+v", res)
	}
}
