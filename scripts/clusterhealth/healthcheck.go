package clusterhealth

import (
	"context"
	"fmt"
	"net"
	"time"

	apicassemdb "github.com/yeqown/cassem/internal/cassemdb/api"
)

func Check(endpoints []string, timeout time.Duration) error {
	if timeout <= 0 {
		return fmt.Errorf("cassemdb write healthcheck failed: timeout must be positive")
	}

	for _, endpoint := range endpoints {
		if err := checkEndpointTCP(endpoint); err != nil {
			return err
		}
	}

	deadline := time.Now().Add(timeout)
	var lastErr error
	for time.Now().Before(deadline) {
		cc, err := apicassemdb.DialWithMode(endpoints, apicassemdb.Mode_X)
		if err == nil {
			client := apicassemdb.NewKVClient(cc)
			ctx, cancel := context.WithTimeout(context.Background(), time.Second)
			_, lastErr = client.SetKV(ctx, &apicassemdb.SetKVReq{
				Key:       fmt.Sprintf("scripts/health/%d", time.Now().UnixNano()),
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
		if !time.Now().Before(deadline) {
			break
		}
		time.Sleep(500 * time.Millisecond)
	}
	return fmt.Errorf("cassemdb write healthcheck failed: %w", lastErr)
}

func checkEndpointTCP(endpoint string) error {
	conn, err := net.DialTimeout("tcp", endpoint, 200*time.Millisecond)
	if err != nil {
		return fmt.Errorf("wait tcp %s: %w", endpoint, err)
	}
	return conn.Close()
}
