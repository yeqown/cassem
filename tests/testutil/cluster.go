package testutil

import (
	"context"
	"fmt"
	"testing"
	"time"
)

const (
	cassemdbHost    = "127.0.0.1"
	cassemdbPort1   = 2021
	cassemdbPort2   = 2022
	cassemdbPort3   = 2023
	cassemadmURL    = "http://127.0.0.1:20218"
	cassemadmPort   = 20218
	cassemagent     = "127.0.0.1:20219"
	cassemagentPort = 20219
)

type Cluster struct {
	DBEndpoints   []string
	AdmBaseURL    string
	AgentEndpoint string
}

func TestDBCluster() *Cluster {
	return &Cluster{
		DBEndpoints: []string{
			fmt.Sprintf("%s:%d", cassemdbHost, cassemdbPort1),
			fmt.Sprintf("%s:%d", cassemdbHost, cassemdbPort2),
			fmt.Sprintf("%s:%d", cassemdbHost, cassemdbPort3),
		},
	}
}

func TestAdmCluster() *Cluster {
	cluster := TestDBCluster()
	cluster.AdmBaseURL = cassemadmURL
	return cluster
}

func TestFullCluster() *Cluster {
	cluster := TestAdmCluster()
	cluster.AgentEndpoint = cassemagent
	return cluster
}

func UseDBCluster(t testing.TB) *Cluster {
	t.Helper()
	cluster := TestDBCluster()
	if err := checkDBCluster(cluster); err != nil {
		t.Skipf("external cassemdb cluster is not ready: %v; run make cluster.start", err)
		return nil
	}
	return cluster
}

func UseAdmCluster(t testing.TB) *Cluster {
	t.Helper()
	cluster := TestAdmCluster()
	if err := checkDBCluster(cluster); err != nil {
		t.Skipf("external cassemdb cluster is not ready: %v; run make cluster.start", err)
		return nil
	}
	if err := checkTCP(cassemadmPort); err != nil {
		t.Skipf("external cassemadm is not ready: %v; run make cluster.start", err)
		return nil
	}
	return cluster
}

func UseFullCluster(t testing.TB) *Cluster {
	t.Helper()
	cluster := TestFullCluster()
	if err := checkDBCluster(cluster); err != nil {
		t.Skipf("external cassemdb cluster is not ready: %v; run make cluster.start", err)
		return nil
	}
	if err := checkTCP(cassemadmPort); err != nil {
		t.Skipf("external cassemadm is not ready: %v; run make cluster.start", err)
		return nil
	}
	if err := checkTCP(cassemagentPort); err != nil {
		t.Skipf("external cassemagent is not ready: %v; run make cluster.start", err)
		return nil
	}
	return cluster
}

func StartCluster(t testing.TB) *Cluster {
	t.Helper()
	return UseFullCluster(t)
}

func checkDBCluster(cluster *Cluster) error {
	return CheckCassemDB(cluster.DBEndpoints, 3*time.Second)
}

func checkTCP(port int) error {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	return WaitTCP(ctx, "127.0.0.1", port)
}
