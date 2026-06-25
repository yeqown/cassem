//go:build integration

package integration_test

import (
	"context"
	"fmt"
	apikv "github.com/yeqown/cassem/api/kv"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/yeqown/cassem/tests/testutil"
)

func TestCassemKVReadWriteClient(t *testing.T) {
	cluster := testutil.UseDBCluster(t)
	cc := testutil.DialCassemKV(t, cluster.DBEndpoints, apikv.Mode_X)
	t.Cleanup(func() { _ = cc.Close() })

	client := apikv.NewKVClient(cc)
	key := fmt.Sprintf("tests/integration/cassemkv/%d", time.Now().UnixNano())
	value := []byte("ok")

	_, err := client.SetKV(context.Background(), &apikv.SetKVReq{
		Key:       key,
		Val:       value,
		Overwrite: true,
	})
	require.NoError(t, err)

	resp, err := client.GetKV(context.Background(), &apikv.GetKVReq{Key: key})
	require.NoError(t, err)
	require.Equal(t, value, resp.GetEntity().GetVal())
}

func TestCassemKVDistributedLock(t *testing.T) {
	cluster := testutil.UseDBCluster(t)
	cc := testutil.DialCassemKV(t, cluster.DBEndpoints, apikv.Mode_X)
	t.Cleanup(func() { _ = cc.Close() })

	kv := apikv.NewKVClient(cc)
	lockKey := fmt.Sprintf("locks/tests/integration/%d", time.Now().UnixNano())
	entered := make(chan struct{})
	release := make(chan struct{})
	results := make(chan bool, 2)
	wg := sync.WaitGroup{}
	wg.Add(2)

	go func() {
		defer wg.Done()
		panicked := false
		func() {
			defer func() { panicked = recover() != nil }()
			apikv.WithLock(kv, lockKey, 10, func() {
				close(entered)
				<-release
			})
		}()
		results <- !panicked
	}()

	go func() {
		defer wg.Done()
		<-entered
		panicked := false
		func() {
			defer func() { panicked = recover() != nil }()
			apikv.WithLock(kv, lockKey, 10, func() {})
		}()
		close(release)
		results <- panicked
	}()

	wg.Wait()
	close(results)
	for ok := range results {
		require.True(t, ok)
	}
}
