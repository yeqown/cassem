package kv

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/yeqown/log"
	"google.golang.org/grpc"
	"google.golang.org/grpc/connectivity"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/resolver"

	apiinternal "github.com/yeqown/cassem/api/internal"
)

// Mode indicates the way that gRPC client communicate with cassemdb cluster.
type Mode uint8

const (
	// Mode_R means read only
	Mode_R Mode = iota + 1
	// Mode_X means read / write, but only communicate with leader node.
	Mode_X
)

func init() {
	resolver.Register(new(cassemdbResolverBuilder))
}

// DialWithMode support multiple backend server and load balance while request
// backend servers in round-robin.
//
// target = "cassemdb:///0.0.0.0:2021,1.1.1.1:2021" can only communicate to leader,
// target = "cassemdb:/all//0.0.0.0:2021,1.1.1.1:2021" can communicate to other nodes,
// but note that the client can only execute READ operations.
func DialWithMode(endpoints []string, mode Mode) (*grpc.ClientConn, error) {
	timeout, cancel := context.WithTimeout(context.TODO(), 10*time.Second)
	defer cancel()

	return DialWithModeContext(timeout, endpoints, mode)
}

// DialWithModeContext lets callers share one context budget across connection
// establishment and subsequent RPCs, which keeps one-shot workflows from
// spending extra time outside their configured deadline.
func DialWithModeContext(ctx context.Context, endpoints []string, mode Mode) (*grpc.ClientConn, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	if _, ok := ctx.Deadline(); !ok {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, 10*time.Second)
		defer cancel()
	}

	var (
		target  = "cassemdb:/"
		scPlain = _SERVICE_CONFIG_JSON_WITH_HEALTH
	)
	switch mode {
	case Mode_R:
		target += "all//"
		scPlain = _SERVICE_CONFIG_JSON_WITHOUT_HEALTH
	case Mode_X:
		target += "//"
	}
	target += strings.Join(endpoints, ",")

	log.
		WithFields(log.Fields{
			"endpoints": endpoints,
			"mode":      mode,
			"target":    target,
		}).
		Debug("DialWithModeContext calling")

	cc, err := grpc.NewClient(target,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithDefaultServiceConfig(scPlain),
		grpc.WithChainUnaryInterceptor(apiinternal.ClientRecovery(), apiinternal.ClientErrorx(), apiinternal.ClientValidation()),
	)
	if err != nil {
		return nil, fmt.Errorf("DialWithModeContext failed: %w", err)
	}
	if err = waitForConnReady(ctx, cc); err != nil {
		_ = cc.Close()
		return nil, fmt.Errorf("DialWithModeContext failed: %w", err)
	}

	return cc, nil
}

func waitForConnReady(ctx context.Context, cc *grpc.ClientConn) error {
	cc.Connect()
	for {
		state := cc.GetState()
		if state == connectivity.Ready {
			return nil
		}
		if !cc.WaitForStateChange(ctx, state) {
			return ctx.Err()
		}
	}
}
