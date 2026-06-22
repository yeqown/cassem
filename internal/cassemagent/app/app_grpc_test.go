package app

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/metadata"

	"github.com/yeqown/cassem/api/agent"
	"github.com/yeqown/cassem/api/concept"
	"github.com/yeqown/cassem/internal/cassemagent/domain"
)

func TestWatchPersistsWatchingMetadata(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	agg := &watchPersistAggregate{}
	d := app{
		uniqueId:     "agent-a",
		aggregate:    agg,
		instancePool: domain.NewInstancePool(),
	}
	watching := &concept.Instance_Watching{App: "demo", Env: "prod", WatchKeys: []string{"db.url"}}

	done := make(chan error, 1)
	go func() {
		done <- d.Watch(&agent.WatchReq{
			ClientId: "client-01",
			ClientIp: "10.0.0.1",
			Watching: []*concept.Instance_Watching{watching},
		}, watchServer{ctx: ctx})
	}()

	require.Eventually(t, func() bool {
		return len(agg.renewedInstances()) == 1
	}, time.Second, 10*time.Millisecond)
	renewed := agg.renewedInstances()[0]
	require.Equal(t, "client-01", renewed.GetClientId())
	require.Equal(t, "agent-a", renewed.GetAgentId())
	require.Equal(t, "10.0.0.1", renewed.GetClientIp())
	require.Equal(t, []*concept.Instance_Watching{watching}, renewed.GetWatching())

	cancel()
	require.NoError(t, <-done)
}

func TestWatchReturnsSendErrors(t *testing.T) {
	sendErr := errors.New("send failed")
	d := app{
		uniqueId:     "agent-a",
		aggregate:    &watchPersistAggregate{},
		cache:        domain.NewCache(10),
		instancePool: domain.NewInstancePool(),
	}
	insId := (&concept.Instance{ClientId: "client-01", ClientIp: "10.0.0.1"}).Id()
	d.instancePool.Register(insId, "demo", "prod", []string{"db.url"})

	done := make(chan error, 1)
	go func() {
		done <- d.Watch(&agent.WatchReq{
			ClientId: "client-01",
			ClientIp: "10.0.0.1",
			Watching: []*concept.Instance_Watching{{App: "demo", Env: "prod", WatchKeys: []string{"db.url"}}},
		}, watchServer{ctx: context.Background(), sendErr: sendErr})
	}()

	d.instancePool.Notify(insId, &concept.Element{Metadata: &concept.ElementMetadata{App: "demo", Env: "prod", Key: "db.url"}})
	require.ErrorIs(t, <-done, sendErr)
}

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

type watchPersistAggregate struct {
	mu      sync.Mutex
	renewed []*concept.Instance
}

func (a *watchPersistAggregate) renewedInstances() []*concept.Instance {
	a.mu.Lock()
	defer a.mu.Unlock()
	return append([]*concept.Instance(nil), a.renewed...)
}

func (a *watchPersistAggregate) GetElementWithVersion(context.Context, string, string, string, int) (*concept.Element, error) {
	return nil, nil
}

func (a *watchPersistAggregate) GetElementVersions(context.Context, string, string, string, string, int) (*concept.GetElementsResult, error) {
	return nil, nil
}

func (a *watchPersistAggregate) GetElements(context.Context, string, string, string, int, string) (*concept.GetElementsResult, error) {
	return nil, nil
}

func (a *watchPersistAggregate) GetElementsByKeys(context.Context, string, string, []string) (*concept.GetElementsResult, error) {
	return nil, nil
}

func (a *watchPersistAggregate) GetElementOperations(context.Context, string, string, string, string, int) (*concept.GetElementOperationsResult, error) {
	return nil, nil
}

func (a *watchPersistAggregate) GetApp(context.Context, string) (*concept.AppMetadata, error) {
	return nil, nil
}

func (a *watchPersistAggregate) GetApps(context.Context, string, int, string) (*concept.GetAppsResult, error) {
	return nil, nil
}

func (a *watchPersistAggregate) GetEnvironments(context.Context, string, string, int) (*concept.GetAppEnvsResult, error) {
	return nil, nil
}

func (a *watchPersistAggregate) GetInstances(context.Context, string, int) (*concept.GetInstancesResult, error) {
	return nil, nil
}

func (a *watchPersistAggregate) GetInstancesByElement(context.Context, string, string, string) (*concept.GetInstancesResult, error) {
	return nil, nil
}

func (a *watchPersistAggregate) GetInstance(context.Context, string) (*concept.Instance, error) {
	return nil, nil
}

func (a *watchPersistAggregate) RegisterInstance(context.Context, *concept.Instance) error {
	return nil
}

func (a *watchPersistAggregate) RenewInstance(_ context.Context, ins *concept.Instance) error {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.renewed = append(a.renewed, ins)
	return nil
}

func (a *watchPersistAggregate) UnregisterInstance(context.Context, string) error {
	return nil
}

func (a *watchPersistAggregate) Watch(context.Context, chan<- *concept.AgentInstanceChange) error {
	return nil
}

func (a *watchPersistAggregate) Register(context.Context, *concept.AgentInstance, int32) error {
	return nil
}

func (a *watchPersistAggregate) Renew(context.Context, *concept.AgentInstance, int32) error {
	return nil
}

func (a *watchPersistAggregate) Unregister(context.Context, string) error {
	return nil
}

func (a *watchPersistAggregate) GetAgents(context.Context, string, int) (*concept.GetAgentsResult, error) {
	return nil, nil
}

type watchServer struct {
	ctx     context.Context
	sendErr error
}

func (s watchServer) Send(*agent.WatchResp) error { return s.sendErr }

func (s watchServer) SetHeader(metadata.MD) error { return nil }

func (s watchServer) SendHeader(metadata.MD) error { return nil }

func (s watchServer) SetTrailer(metadata.MD) {}

func (s watchServer) Context() context.Context { return s.ctx }

func (s watchServer) SendMsg(any) error { return nil }

func (s watchServer) RecvMsg(any) error { return nil }
