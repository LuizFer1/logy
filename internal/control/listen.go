package control

import (
	"context"
	"net"
)

// Listen opens the user-local control endpoint.
func Listen() (net.Listener, error) {
	return ListenName(PipeName())
}

// Dial connects to the user-local control endpoint.
func Dial(ctx context.Context) (net.Conn, error) {
	return DialName(ctx, PipeName())
}

// Serve accepts connections on ln and handles one request per connection.
// It returns nil when ctx is cancelled.
func Serve(ctx context.Context, ln net.Listener, handler Handler) error {
	done := make(chan struct{})
	go func() {
		select {
		case <-ctx.Done():
			_ = ln.Close()
		case <-done:
		}
	}()
	defer close(done)

	for {
		conn, err := ln.Accept()
		if err != nil {
			if ctx.Err() != nil {
				return nil
			}
			return err
		}
		go func() {
			_ = ServeConn(conn, handler)
		}()
	}
}
