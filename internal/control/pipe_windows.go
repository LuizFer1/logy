//go:build windows

package control

import (
	"context"
	"net"
	"os"
	"sync"
	"time"

	"golang.org/x/sys/windows"
)

const pipeRejectRemoteClients = 0x00000008

type pipeAddr string

func (a pipeAddr) Network() string { return "pipe" }
func (a pipeAddr) String() string  { return string(a) }

type pipeConn struct {
	file *os.File
	addr pipeAddr
}

func newPipeConn(h windows.Handle, addr pipeAddr) net.Conn {
	return &pipeConn{
		file: os.NewFile(uintptr(h), string(addr)),
		addr: addr,
	}
}

func (c *pipeConn) Read(b []byte) (int, error)         { return c.file.Read(b) }
func (c *pipeConn) Write(b []byte) (int, error)        { return c.file.Write(b) }
func (c *pipeConn) Close() error                       { return c.file.Close() }
func (c *pipeConn) LocalAddr() net.Addr                { return c.addr }
func (c *pipeConn) RemoteAddr() net.Addr               { return c.addr }
func (c *pipeConn) SetDeadline(t time.Time) error      { return c.file.SetDeadline(t) }
func (c *pipeConn) SetReadDeadline(t time.Time) error  { return c.file.SetReadDeadline(t) }
func (c *pipeConn) SetWriteDeadline(t time.Time) error { return c.file.SetWriteDeadline(t) }

type pipeListener struct {
	addr   pipeAddr
	mu     sync.Mutex
	closed bool
	handle windows.Handle
}

// ListenName listens on a Windows named pipe path such as \\.\pipe\logy-user.
func ListenName(name string) (net.Listener, error) {
	h, err := createPipe(name, true)
	if err != nil {
		return nil, err
	}
	return &pipeListener{addr: pipeAddr(name), handle: h}, nil
}

func (l *pipeListener) Accept() (net.Conn, error) {
	l.mu.Lock()
	if l.closed {
		l.mu.Unlock()
		return nil, net.ErrClosed
	}
	h := l.handle
	l.handle = 0
	l.mu.Unlock()

	if h == 0 {
		var err error
		h, err = createPipe(string(l.addr), false)
		if err != nil {
			if l.isClosed() {
				return nil, net.ErrClosed
			}
			return nil, err
		}
	}

	err := windows.ConnectNamedPipe(h, nil)
	if err != nil && err != windows.ERROR_PIPE_CONNECTED {
		_ = windows.CloseHandle(h)
		if l.isClosed() {
			return nil, net.ErrClosed
		}
		return nil, err
	}

	l.mu.Lock()
	if l.closed {
		l.mu.Unlock()
		_ = windows.CloseHandle(h)
		return nil, net.ErrClosed
	}
	next, nextErr := createPipe(string(l.addr), false)
	if nextErr == nil {
		l.handle = next
	}
	l.mu.Unlock()

	return newPipeConn(h, l.addr), nil
}

func (l *pipeListener) Close() error {
	l.mu.Lock()
	if l.closed {
		l.mu.Unlock()
		return nil
	}
	l.closed = true
	h := l.handle
	l.handle = 0
	l.mu.Unlock()

	if h != 0 {
		_ = windows.CloseHandle(h)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
	defer cancel()
	if c, err := DialName(ctx, string(l.addr)); err == nil {
		_ = c.Close()
	}
	return nil
}

func (l *pipeListener) Addr() net.Addr {
	return l.addr
}

func (l *pipeListener) isClosed() bool {
	l.mu.Lock()
	defer l.mu.Unlock()
	return l.closed
}

// DialName connects to a Windows named pipe.
func DialName(ctx context.Context, name string) (net.Conn, error) {
	path, err := windows.UTF16PtrFromString(name)
	if err != nil {
		return nil, err
	}

	for {
		if err := ctx.Err(); err != nil {
			return nil, Unavailable(err.Error())
		}

		h, err := windows.CreateFile(
			path,
			windows.GENERIC_READ|windows.GENERIC_WRITE,
			0,
			nil,
			windows.OPEN_EXISTING,
			windows.FILE_ATTRIBUTE_NORMAL,
			0,
		)
		if err == nil {
			return newPipeConn(h, pipeAddr(name)), nil
		}
		if err == windows.ERROR_FILE_NOT_FOUND {
			return nil, Unavailable(err.Error())
		}
		if err != windows.ERROR_PIPE_BUSY {
			return nil, Unavailable(err.Error())
		}

		timer := time.NewTimer(20 * time.Millisecond)
		select {
		case <-ctx.Done():
			timer.Stop()
			return nil, Unavailable(ctx.Err().Error())
		case <-timer.C:
		}
	}
}

func createPipe(name string, first bool) (windows.Handle, error) {
	path, err := windows.UTF16PtrFromString(name)
	if err != nil {
		return 0, err
	}
	openMode := uint32(windows.PIPE_ACCESS_DUPLEX)
	if first {
		openMode |= windows.FILE_FLAG_FIRST_PIPE_INSTANCE
	}
	pipeMode := uint32(windows.PIPE_TYPE_BYTE | windows.PIPE_READMODE_BYTE | pipeRejectRemoteClients)
	h, err := windows.CreateNamedPipe(path, openMode, pipeMode, windows.PIPE_UNLIMITED_INSTANCES, 65536, 65536, 0, nil)
	if err != nil {
		return 0, err
	}
	return h, nil
}
