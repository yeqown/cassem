package testutil

import (
	"context"
	"fmt"
	"net"
	"time"
)

func WaitTCP(ctx context.Context, host string, port int) error {
	addr := net.JoinHostPort(host, fmt.Sprintf("%d", port))
	var lastErr error
	for {
		dialer := net.Dialer{Timeout: 200 * time.Millisecond}
		conn, err := dialer.DialContext(ctx, "tcp", addr)
		if err == nil {
			_ = conn.Close()
			return nil
		}
		lastErr = err

		select {
		case <-ctx.Done():
			return fmt.Errorf("wait tcp %s: %w; last error: %v", addr, ctx.Err(), lastErr)
		case <-time.After(200 * time.Millisecond):
		}
	}
}
