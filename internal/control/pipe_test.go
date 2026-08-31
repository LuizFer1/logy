package control

import (
	"runtime"
	"strings"
	"testing"
)

func TestPipeNameIsUserLocal(t *testing.T) {
	t.Parallel()
	name := PipeName()
	if name == "" {
		t.Fatal("PipeName() is empty")
	}
	if runtime.GOOS == "windows" {
		if !strings.HasPrefix(name, `\\.\pipe\logy`) {
			t.Fatalf("PipeName() = %q, want Windows named pipe", name)
		}
		return
	}
	if !strings.HasSuffix(name, ".sock") {
		t.Fatalf("PipeName() = %q, want unix socket", name)
	}
}
