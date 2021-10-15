// Package raftleader is included in your Raft nodes to expose whether this node is the leader.
package raftleader

import (
	"google.golang.org/grpc"
	"google.golang.org/grpc/health"
	"google.golang.org/grpc/health/grpc_health_v1"

	"github.com/yeqown/cassem/pkg/runtime"
)

// Setup creates a new health.Server for you and registers it on s.
// It's a convenience wrapper around watchAsync. Notice that, services is the
// set of gRPC services in current gRPC server instance.
func Setup(isLeader bool, leadershipChangeCh <-chan bool, s *grpc.Server) {
	h := health.NewServer()

	services := []string{
		"cassem.db.KV",
		"cassem.db.Cluster",
	}

	watchAsync(isLeader, leadershipChangeCh, h, services)
	grpc_health_v1.RegisterHealthServer(s, h)
}

// watchAsync starts a goroutine that updates the given health.Server with whether we are the Raft leader.
// It will set the given services as SERVING if we are the leader, and as NOT_SERVING otherwise.
func watchAsync(isLeader bool, leadershipChangeCh <-chan bool, h *health.Server, services []string) {
	updateServingStatus := func(h *health.Server, services []string, isLeader bool) {
		status := grpc_health_v1.HealthCheckResponse_NOT_SERVING
		if isLeader {
			status = grpc_health_v1.HealthCheckResponse_SERVING
		}
		for _, svc := range services {
			h.SetServingStatus(svc, status)
		}
		// h.SetServingStatus(_gRPC_HEALTH_SERVICE, status)
	}

	updateServingStatus(h, services, isLeader)

	// run forever
	runtime.GoFunc("", func() error {
		for beLeader := range leadershipChangeCh {
			updateServingStatus(h, services, beLeader)
		}

		return nil
	})
}

//const (
//	_gRPC_HEALTH_SERVICE = "cassemdb.RaftLeader"
//)
