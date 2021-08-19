package api

import (
	"context"
	"strings"
	"time"

	"github.com/pkg/errors"
	"github.com/yeqown/log"
	"google.golang.org/grpc"
	_ "google.golang.org/grpc/health"
	"google.golang.org/grpc/resolver"

	"github.com/yeqown/cassem/pkg/grpcx"
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
	var target = "cassemdb:/"
	switch mode {
	case Mode_R:
		target += "all//"
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
		Debug("DialWithMode calling")

	timeout, cancel := context.WithTimeout(context.TODO(), 10*time.Second)
	defer cancel()
	cc, err := grpc.DialContext(timeout, target,
		grpc.WithInsecure(),
		grpc.WithBlock(),
		grpc.WithChainUnaryInterceptor(grpcx.ClientRecovery(), grpcx.ClientErrorx()),
	)
	if err != nil {
		return nil, errors.Wrap(err, "DialWithMode failed")
	}

	return cc, nil
}
