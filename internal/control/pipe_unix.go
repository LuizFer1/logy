//go:build !windows

package control

import (
	"context"
	"net"
	"os"
)

// ListenName listens on a Unix domain socket path.
func ListenName(name string) (net.Listener, error) {
	_ = os.Remove(name)
	return net.Listen("unix", name)
}

// DialName connects to a Unix domain socket.
func DialName(ctx context.Context, name string) (net.Conn, error) {
	var d net.Dialer
	conn, err := d.DialContext(ctx, "unix", name)
	if err != nil {
		return nil, Unavailable(err.Error())
	}
	return conn, nil
}
