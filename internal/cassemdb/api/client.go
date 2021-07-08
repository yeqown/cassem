package api

import (
	"context"
	"time"

	_ "google.golang.org/grpc/health"

	"google.golang.org/grpc"
	"google.golang.org/grpc/resolver"
)

// Dial support multiple backend server and load balance while request
// backend servers in round-robin.
//
// use target = "cassemdb:///0.0.0.0:2021,1.1.1.1:2021" can only communicate to leader,
// target = "cassemdb:/all//0.0.0.0:2021,1.1.1.1:2021" can communicate to other nodes,
// but note that the client can only execute READ operations.
func Dial(target string) (*grpc.ClientConn, error) {
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
