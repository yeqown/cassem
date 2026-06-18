package testutil

import (
	"context"
	"fmt"
	"net"
	"time"

	"google.golang.org/grpc"
)

func DialCassemDB(t TB, endpoints []string, mode apikv.Mode) *grpc.ClientConn {
	t.Helper()

	var (
		cc  *grpc.ClientConn
		err error
	)
	deadline := time.Now().Add(30 * time.Second)
	for time.Now().Before(deadline) {
		cc, err = apikv.DialWithMode(endpoints, mode)
		if err == nil {
			return cc
		}
		time.Sleep(300 * time.Millisecond)
	}

	t.Fatalf("dial cassemdb %v: %v", endpoints, err)
	return nil
}

func CheckCassemDB(endpoints []string, timeout time.Duration) error {
	for _, endpoint := range endpoints {
		if err := checkEndpointTCP(endpoint); err != nil {
			return err
		}
	}

	deadline := time.Now().Add(timeout)
	var lastErr error
	for time.Now().Before(deadline) {
		cc, err := apikv.DialWithMode(endpoints, apikv.Mode_X)
		if err == nil {
			client := apikv.NewKVClient(cc)
			ctx, cancel := context.WithTimeout(context.Background(), time.Second)
			_, lastErr = client.SetKV(ctx, &apikv.SetKVReq{
				Key:       fmt.Sprintf("tests/health/%d", time.Now().UnixNano()),
				Val:       []byte("ok"),
				Overwrite: true,
			})
			cancel()
			_ = cc.Close()
			if lastErr == nil {
				return nil
			}
		} else {
			lastErr = err
		}
		time.Sleep(500 * time.Millisecond)
	}

	return fmt.Errorf("cassemdb did not become ready: %w", lastErr)
}

func WaitCassemDB(t TB, endpoints []string) {
	t.Helper()
	if err := CheckCassemDB(endpoints, 45*time.Second); err != nil {
		t.Fatalf("%v", err)
	}
}

func checkEndpointTCP(endpoint string) error {
	conn, err := net.DialTimeout("tcp", endpoint, 200*time.Millisecond)
	if err != nil {
		return fmt.Errorf("wait tcp %s: %w", endpoint, err)
	}
	return conn.Close()
}
