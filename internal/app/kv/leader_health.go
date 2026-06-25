package kv

import (
	"google.golang.org/grpc"
	"google.golang.org/grpc/health"
	"google.golang.org/grpc/health/grpc_health_v1"

	"github.com/yeqown/cassem/pkg/runtime"
)

// setupLeaderAwareHealthServer registers leader-aware health checks on s.
// This keeps write routing aligned with the current Raft leader without changing service names.
func setupLeaderAwareHealthServer(isLeader bool, leadershipChangeCh <-chan bool, s *grpc.Server) {
	h := health.NewServer()

	services := []string{
		"cassem.api.kv.KV",
		"cassem.api.kv.Cluster",
	}

	watchLeaderHealthAsync(isLeader, leadershipChangeCh, h, services)
	grpc_health_v1.RegisterHealthServer(s, h)
}

// watchLeaderHealthAsync updates health status as leadership changes.
// This preserves the existing health contract so leader-only traffic keeps resolving to the same services.
func watchLeaderHealthAsync(isLeader bool, leadershipChangeCh <-chan bool, h *health.Server, services []string) {
	updateServingStatus := func(h *health.Server, services []string, isLeader bool) {
		status := grpc_health_v1.HealthCheckResponse_NOT_SERVING
		if isLeader {
			status = grpc_health_v1.HealthCheckResponse_SERVING
		}
		for _, svc := range services {
			h.SetServingStatus(svc, status)
		}
		h.SetServingStatus(_gRPCHealthService, status)
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

const (
	_gRPCHealthService = "cassemkv.RaftLeader"
)
