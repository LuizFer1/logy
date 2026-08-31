package control

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"
)

func testPipeName(t *testing.T) string {
	t.Helper()
	if runtime.GOOS == "windows" {
		return fmt.Sprintf(`\\.\pipe\logy-test-%d-%d`, os.Getpid(), time.Now().UnixNano())
	}
	return filepath.Join(t.TempDir(), "logy.sock")
}

func TestDialUnavailable(t *testing.T) {
	t.Parallel()
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	_, err := DialName(ctx, testPipeName(t))
	if err == nil {
		t.Fatal("DialName() error = nil, want ErrUnavailable")
	}
	if !errors.Is(err, ErrUnavailable) {
		t.Fatalf("DialName() error = %v, want ErrUnavailable", err)
	}
}

func TestListenDialStatusRoundTrip(t *testing.T) {
	t.Parallel()
	name := testPipeName(t)
	ln, err := ListenName(name)
	if err != nil {
		t.Fatalf("ListenName() error = %v", err)
	}
	t.Cleanup(func() { _ = ln.Close() })

	handler := Handler{
		Status: func() StatusPayload {
			return StatusPayload{Running: true, Collectors: []string{"git"}}
		},
	}
	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)
	errCh := make(chan error, 1)
	go func() {
		errCh <- Serve(ctx, ln, handler)
	}()

	dialCtx, dialCancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer dialCancel()
	conn, err := DialName(dialCtx, name)
	if err != nil {
		t.Fatalf("DialName() error = %v", err)
	}
	defer conn.Close()

	resp, err := Call(conn, Request{Version: Version, Command: "status"})
	if err != nil {
		t.Fatalf("Call() error = %v", err)
	}
	if !resp.OK || resp.Status == nil || !resp.Status.Running {
		t.Fatalf("status = %#v", resp)
	}
	if len(resp.Status.Collectors) != 1 || resp.Status.Collectors[0] != "git" {
		t.Fatalf("collectors = %#v", resp.Status.Collectors)
	}

	cancel()
	select {
	case err := <-errCh:
		if err != nil {
			t.Fatalf("Serve() error = %v", err)
		}
	case <-time.After(3 * time.Second):
		t.Fatal("Serve() did not return after cancel")
	}
}
