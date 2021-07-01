package grpcx

import (
	"context"
	"time"

	"google.golang.org/grpc/resolver"

	"google.golang.org/grpc"
)

// DialCassemDB support multiple backend server and load balance while request
// backend servers in round-robin.
// DialCassemDB("cassemdb://0.0.0.0:2021,1.1.1.1:2021")
func DialCassemDB(target string) (*grpc.ClientConn, error) {

	timeout, cancel := context.WithTimeout(context.TODO(), 10*time.Second)
	defer cancel()
	cc, err := grpc.DialContext(timeout, target,
		grpc.WithInsecure(),
		// grpc.WithBlock(),
	)
	if err != nil {
		return nil, err
	}

	return cc, nil
}

func init() {
	resolver.Register(new(cassemdbResolverBuilder))
}
