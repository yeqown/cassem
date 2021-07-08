// Package raftleader is included in your Raft nodes to expose whether this node is the leader.
package raftleader

import (
	"github.com/hashicorp/raft"
	"google.golang.org/grpc"
	"google.golang.org/grpc/health"
	"google.golang.org/grpc/health/grpc_health_v1"

	"github.com/yeqown/cassem/pkg/runtime"
)

// Setup creates a new health.Server for you and registers it on s.
// It's a convenience wrapper around report.
func Setup(r *raft.Raft, s *grpc.Server, services []string) {
	h := health.NewServer()
	report(r, h, services)
	grpc_health_v1.RegisterHealthServer(s, h)
}

// report starts a goroutine that updates the given health.Server with whether we are the Raft leader.
// It will set the given services as SERVING if we are the leader, and as NOT_SERVING otherwise.
func report(r *raft.Raft, h *health.Server, services []string) {
	ch := make(chan raft.Observation, 1)
	r.RegisterObserver(raft.NewObserver(ch, true, func(o *raft.Observation) bool {
		_, ok := o.Data.(raft.LeaderObservation)
		return ok
	}))

	updateServingStatus(h, services, r.State() == raft.Leader)

	// run forever
	runtime.GoFunc("", func() error {
		for range ch {
			// TODO(quis, https://github.com/hashicorp/raft/issues/426): Use a safer method to decide if we are the leader.
			updateServingStatus(h, services, r.State() == raft.Leader)
		}

		return nil
	})
}

func updateServingStatus(h *health.Server, services []string, isLeader bool) {
	status := grpc_health_v1.HealthCheckResponse_NOT_SERVING
	if isLeader {
		status = grpc_health_v1.HealthCheckResponse_SERVING
	}
	for _, srv := range services {
		h.SetServingStatus(srv, status)
	}
	h.SetServingStatus(_gRPC_HEALTH_SERVICE, status)
}

const (
	_gRPC_HEALTH_SERVICE = "cassemdb.RaftLeader"
)
