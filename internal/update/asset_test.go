package update

import "testing"

func TestAssetName(t *testing.T) {
	got, err := AssetName("1.2.0", "windows", "amd64")
	if err != nil || got != "logy_1.2.0_windows_amd64.zip" {
		t.Fatalf("got %q err %v", got, err)
	}
	got, err = AssetName("1.2.0", "linux", "arm64")
	if err != nil || got != "logy_1.2.0_linux_arm64.tar.gz" {
		t.Fatalf("got %q err %v", got, err)
	}
	got, err = AssetName("1.2.0", "darwin", "amd64")
	if err != nil || got != "logy_1.2.0_darwin_amd64.tar.gz" {
		t.Fatalf("got %q err %v", got, err)
	}
	if _, err := AssetName("1.0.0", "windows", "386"); err == nil {
		t.Fatal("expected error for 386")
	}
}

func TestParseChecksums(t *testing.T) {
	in := "abc123  logy_1.2.0_windows_amd64.zip\ndef456  logy_1.2.0_linux_amd64.tar.gz\n"
	m, err := ParseChecksums(in)
	if err != nil {
		t.Fatal(err)
	}
	if m["logy_1.2.0_windows_amd64.zip"] != "abc123" {
		t.Fatalf("%v", m)
	}
}
