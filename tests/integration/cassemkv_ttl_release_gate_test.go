//go:build integration

package integration_test

import (
	"context"
	apikv "github.com/yeqown/cassem/api/kv"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/yeqown/cassem/tests/testutil"
)

func TestConfigCenterReleaseGate_DatabaseTTLExpiresTemporaryLeaseKey(t *testing.T) {
	cluster := testutil.RequireDBCluster(t)
	scope := testutil.NewRunScope(t, "database ttl expires temporary lease key")
	cc := testutil.DialCassemKV(t, cluster.DBEndpoints, apikv.Mode_X)
	t.Cleanup(func() { _ = cc.Close() })
	client := apikv.NewKVClient(cc)
	key := scope.TTLKey("leases", "payment-service", "release-lock")
	value := []byte(`{"owner":"release-manager","purpose":"release-gate"}`)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	_, err := client.SetKV(ctx, &apikv.SetKVReq{Key: key, Val: value, Ttl: 2, Overwrite: true})
	cancel()
	require.NoError(t, err)

	ctx, cancel = context.WithTimeout(context.Background(), 5*time.Second)
	got, err := client.GetKV(ctx, &apikv.GetKVReq{Key: key})
	cancel()
	require.NoError(t, err)
	require.Equal(t, value, got.GetEntity().GetVal())

	testutil.RequireEventually(t, 6*time.Second, 200*time.Millisecond, func() (bool, string) {
		ctx, cancel := context.WithTimeout(context.Background(), time.Second)
		defer cancel()
		_, err := client.GetKV(ctx, &apikv.GetKVReq{Key: key})
		if err != nil {
			return true, ""
		}
		return false, "ttl key still exists"
	})
}
