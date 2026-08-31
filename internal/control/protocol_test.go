package control

import (
	"bytes"
	"encoding/json"
	"errors"
	"net"
	"testing"
)

func TestDecodeRequest(t *testing.T) {
	t.Parallel()
	raw := []byte(`{"version":"1","command":"status"}`)
	req, err := DecodeRequest(bytes.NewReader(raw))
	if err != nil {
		t.Fatalf("DecodeRequest() error = %v", err)
	}
	if req.Version != Version || req.Command != "status" {
		t.Fatalf("request = %#v", req)
	}
}

func TestHandleUnknownCommand(t *testing.T) {
	t.Parallel()
	resp := Handle(Request{Version: Version, Command: "explode"}, Handler{})
	if resp.OK {
		t.Fatalf("Handle() ok = true, want false: %#v", resp)
	}
	if resp.Error == "" {
		t.Fatal("expected error message for unknown command")
	}
}

func TestHandleStatus(t *testing.T) {
	t.Parallel()
	resp := Handle(Request{Version: Version, Command: "status"}, Handler{
		Status: func() StatusPayload {
			return StatusPayload{Running: true, Collectors: []string{"git"}}
		},
	})
	if !resp.OK || resp.Status == nil || !resp.Status.Running {
		t.Fatalf("status response = %#v", resp)
	}
	if len(resp.Status.Collectors) != 1 || resp.Status.Collectors[0] != "git" {
		t.Fatalf("collectors = %#v", resp.Status.Collectors)
	}
}

func TestHandleStop(t *testing.T) {
	t.Parallel()
	stopped := false
	resp := Handle(Request{Version: Version, Command: "stop"}, Handler{
		Stop: func() error {
			stopped = true
			return nil
		},
	})
	if !resp.OK {
		t.Fatalf("stop response = %#v", resp)
	}
	if !stopped {
		t.Fatal("Stop handler was not called")
	}
}

func TestCallRoundTrip(t *testing.T) {
	t.Parallel()
	server, client := net.Pipe()
	t.Cleanup(func() {
		server.Close()
		client.Close()
	})
	go func() {
		_ = ServeConn(server, Handler{
			Status: func() StatusPayload { return StatusPayload{Running: true} },
		})
	}()

	resp, err := Call(client, Request{Version: Version, Command: "status"})
	if err != nil {
		t.Fatalf("Call() error = %v", err)
	}
	if !resp.OK || resp.Status == nil || !resp.Status.Running {
		t.Fatalf("Call() = %#v", resp)
	}
}

func TestUnavailableError(t *testing.T) {
	t.Parallel()
	err := Unavailable("pipe missing")
	if !errors.Is(err, ErrUnavailable) {
		t.Fatalf("Unavailable() = %v, want ErrUnavailable", err)
	}
}

func TestEncodeResponseJSON(t *testing.T) {
	t.Parallel()
	var buf bytes.Buffer
	if err := EncodeResponse(&buf, Response{Version: Version, OK: true}); err != nil {
		t.Fatalf("EncodeResponse() error = %v", err)
	}
	var decoded Response
	if err := json.Unmarshal(bytes.TrimSpace(buf.Bytes()), &decoded); err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}
	if !decoded.OK || decoded.Version != Version {
		t.Fatalf("encoded = %#v", decoded)
	}
}
