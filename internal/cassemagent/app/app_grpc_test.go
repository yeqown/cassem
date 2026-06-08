package app

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/yeqown/cassem/api/agent"
	"github.com/yeqown/cassem/api/concept"
	"github.com/yeqown/cassem/internal/cassemagent/domain"
)

func TestDispatchFiltersTargetInstances(t *testing.T) {
	d := app{instancePool: domain.NewInstancePool()}
	ch1 := d.instancePool.Register("client-1@127.0.0.1", "app", "dev", []string{"key"})
	ch2 := d.instancePool.Register("client-2@127.0.0.1", "app", "dev", []string{"key"})

	elem := &concept.Element{
		Metadata: &concept.ElementMetadata{App: "app", Env: "dev", Key: "key"},
		Raw:      []byte("value"),
		Version:  1,
	}
	_, err := d.Dispatch(context.Background(), &agent.DispatchReq{
		Elems:       []*concept.Element{elem},
		InstanceIds: []string{"client-1@127.0.0.1"},
	})
	require.NoError(t, err)

	select {
	case got := <-ch1:
		require.Equal(t, elem, got)
	case <-time.After(time.Second):
		t.Fatal("target instance did not receive dispatch")
	}

	select {
	case got := <-ch2:
		t.Fatalf("non-target instance received dispatch: %v", got)
	case <-time.After(50 * time.Millisecond):
	}
}
