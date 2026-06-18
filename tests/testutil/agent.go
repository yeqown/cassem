package testutil

import (
	"context"
	"fmt"
	"time"

	"github.com/yeqown/cassem/api/agent"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

// CheckCassemAgent verifies the edge proxy is ready by exercising the generated gRPC client.
func CheckCassemAgent(endpoint string, timeout, interval time.Duration) error {
	deadline := time.Now().Add(timeout)
	var lastErr error
	for {
		remaining := time.Until(deadline)
		if remaining <= 0 {
			break
		}
		if err := checkCassemAgentOnce(endpoint, deadline); err == nil {
			return nil
		} else {
			lastErr = err
		}
		sleepUntilNextProbe(interval, deadline)
	}
	return fmt.Errorf("cassemagent %s did not become ready: %w", endpoint, lastErr)
}

func checkCassemAgentOnce(endpoint string, deadline time.Time) error {
	cc, err := grpc.NewClient(endpoint, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return fmt.Errorf("create agent client: %w", err)
	}
	defer cc.Close()

	client := agent.NewAgentClient(cc)
	clientID := fmt.Sprintf("ready-%d", time.Now().UnixNano())
	registerCtx, registerCancel := context.WithTimeout(context.Background(), time.Until(deadline))
	_, err = client.Register(registerCtx, &agent.RegisterReq{
		ClientId: clientID,
		ClientIp: "127.0.0.1",
	})
	registerCancel()
	if err != nil {
		return fmt.Errorf("register readiness client: %w", err)
	}

	remaining := time.Until(deadline)
	if remaining <= 0 {
		return fmt.Errorf("unregister readiness client: deadline exceeded")
	}
	unregisterCtx, unregisterCancel := context.WithTimeout(context.Background(), remaining)
	_, err = client.Unregister(unregisterCtx, &agent.UnregisterReq{
		ClientId: clientID,
		ClientIp: "127.0.0.1",
	})
	unregisterCancel()
	if err != nil {
		return fmt.Errorf("unregister readiness client: %w", err)
	}
	return nil
}

func sleepUntilNextProbe(interval time.Duration, deadline time.Time) {
	remaining := time.Until(deadline)
	if remaining <= 0 {
		return
	}
	if interval <= 0 || interval > remaining {
		interval = remaining
	}
	time.Sleep(interval)
}
