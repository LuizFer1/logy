package control

import (
	"bufio"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"strings"
)

// Version is the control protocol version.
const Version = "1"

// ErrUnavailable is returned when the daemon pipe cannot be reached.
var ErrUnavailable = errors.New("logy daemon is not running")

// Request is a versioned control command.
type Request struct {
	Version string `json:"version"`
	Command string `json:"command"`
}

// Response is a versioned control reply.
type Response struct {
	Version string         `json:"version"`
	OK      bool           `json:"ok"`
	Error   string         `json:"error,omitempty"`
	Status  *StatusPayload `json:"status,omitempty"`
}

// StatusPayload is the daemon status returned over the control pipe.
type StatusPayload struct {
	Running    bool     `json:"running"`
	Collectors []string `json:"collectors,omitempty"`
}

// Handler implements daemon-side control commands.
type Handler struct {
	Status func() StatusPayload
	Stop   func() error
	Reload func() error
}

// Unavailable wraps a transport error as ErrUnavailable.
func Unavailable(detail string) error {
	detail = strings.TrimSpace(detail)
	if detail == "" {
		return ErrUnavailable
	}
	return fmt.Errorf("%w: %s", ErrUnavailable, detail)
}

// DecodeRequest reads one JSON request from r.
func DecodeRequest(r io.Reader) (Request, error) {
	var req Request
	dec := json.NewDecoder(r)
	if err := dec.Decode(&req); err != nil {
		return Request{}, err
	}
	return req, nil
}

// EncodeResponse writes one JSON response followed by a newline.
func EncodeResponse(w io.Writer, resp Response) error {
	if resp.Version == "" {
		resp.Version = Version
	}
	enc := json.NewEncoder(w)
	return enc.Encode(resp)
}

// Handle dispatches a decoded request.
func Handle(req Request, handler Handler) Response {
	resp := Response{Version: Version}
	if req.Version != "" && req.Version != Version {
		resp.Error = "unsupported protocol version"
		return resp
	}
	switch strings.ToLower(strings.TrimSpace(req.Command)) {
	case "status":
		if handler.Status == nil {
			resp.Error = "status is unavailable"
			return resp
		}
		status := handler.Status()
		resp.OK = true
		resp.Status = &status
		return resp
	case "stop":
		if handler.Stop == nil {
			resp.Error = "stop is unavailable"
			return resp
		}
		if err := handler.Stop(); err != nil {
			resp.Error = err.Error()
			return resp
		}
		resp.OK = true
		return resp
	case "reload":
		if handler.Reload == nil {
			resp.Error = "reload is unavailable"
			return resp
		}
		if err := handler.Reload(); err != nil {
			resp.Error = err.Error()
			return resp
		}
		resp.OK = true
		return resp
	default:
		resp.Error = fmt.Sprintf("unknown command: %s", req.Command)
		return resp
	}
}

// ServeConn reads one request from conn and writes a response.
func ServeConn(conn net.Conn, handler Handler) error {
	defer conn.Close()
	reader := bufio.NewReader(conn)
	req, err := DecodeRequest(reader)
	if err != nil {
		return err
	}
	return EncodeResponse(conn, Handle(req, handler))
}

// Call writes req and reads the response from conn.
func Call(conn net.Conn, req Request) (Response, error) {
	if req.Version == "" {
		req.Version = Version
	}
	if err := json.NewEncoder(conn).Encode(req); err != nil {
		return Response{}, err
	}
	var resp Response
	if err := json.NewDecoder(conn).Decode(&resp); err != nil {
		return Response{}, err
	}
	return resp, nil
}
