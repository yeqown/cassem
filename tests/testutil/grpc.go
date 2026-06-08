package testutil

import (
	"context"
	"fmt"
	"time"

	apicassemdb "github.com/yeqown/cassem/internal/cassemdb/api"
	"google.golang.org/grpc"
)

func DialCassemDB(t TB, endpoints []string, mode apicassemdb.Mode) *grpc.ClientConn {
	t.Helper()

	var (
		cc  *grpc.ClientConn
		err error
	)
	deadline := time.Now().Add(30 * time.Second)
	for time.Now().Before(deadline) {
		cc, err = apicassemdb.DialWithMode(endpoints, mode)
		if err == nil {
			return cc
		}
		time.Sleep(300 * time.Millisecond)
	}

	t.Fatalf("dial cassemdb %v: %v", endpoints, err)
	return nil
}

func WaitCassemDB(t TB, endpoints []string) {
	t.Helper()

	deadline := time.Now().Add(45 * time.Second)
	var lastErr error
	for time.Now().Before(deadline) {
		cc, err := apicassemdb.DialWithMode(endpoints, apicassemdb.Mode_X)
		if err == nil {
			client := apicassemdb.NewKVClient(cc)
			ctx, cancel := context.WithTimeout(context.Background(), time.Second)
			_, lastErr = client.SetKV(ctx, &apicassemdb.SetKVReq{
				Key:       fmt.Sprintf("tests/health/%d", time.Now().UnixNano()),
				Val:       []byte("ok"),
				Overwrite: true,
			})
			cancel()
			_ = cc.Close()
			if lastErr == nil {
				return
			}
		} else {
			lastErr = err
		}
		time.Sleep(500 * time.Millisecond)
	}

	t.Fatalf("cassemdb did not become ready: %v", lastErr)
}
