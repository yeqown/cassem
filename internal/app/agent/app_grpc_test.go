package agent

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	pb "github.com/yeqown/cassem/internal/protogen"
	clientpb "github.com/yeqown/cassem/sdk/go/protogen"

	"github.com/stretchr/testify/require"
	"github.com/yeqown/cassem/internal/concept"
	"google.golang.org/grpc/metadata"
)

func TestWatchPersistsWatchingMetadata(t *testing.T) {
	ctx, cancel := context.WithCancel(tokenContext())
	defer cancel()

	agg := &watchPersistAggregate{}
	d := app{
		uniqueId:     "agent-a",
		authorizer:   allowTokenAuthorizer(),
		aggregate:    agg,
		instancePool: NewInstancePool(),
	}
	watching := &concept.Instance_Watching{App: "demo", Env: "prod", WatchKeys: []string{"db_url"}}

	done := make(chan error, 1)
	go func() {
		done <- d.Watch(&clientpb.WatchReq{
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
		authorizer:   allowTokenAuthorizer(),
		aggregate:    &watchPersistAggregate{},
		cache:        NewCache(10),
		instancePool: NewInstancePool(),
	}
	insId := concept.InstanceID(&concept.Instance{ClientId: "client-01", ClientIp: "10.0.0.1"})
	d.instancePool.Register(insId, "demo", "prod", []string{"db_url"})

	done := make(chan error, 1)
	go func() {
		done <- d.Watch(&clientpb.WatchReq{
			ClientId: "client-01",
			ClientIp: "10.0.0.1",
			Watching: []*concept.Instance_Watching{{App: "demo", Env: "prod", WatchKeys: []string{"db_url"}}},
		}, watchServer{ctx: tokenContext(), sendErr: sendErr})
	}()

	d.instancePool.Notify(insId, &concept.Element{Metadata: &concept.ElementMetadata{App: "demo", Env: "prod", Key: "db_url"}})
	require.ErrorIs(t, <-done, sendErr)
}

func TestDispatchFiltersTargetInstances(t *testing.T) {
	d := app{instancePool: NewInstancePool()}
	ch1 := d.instancePool.Register("client-1@127.0.0.1", "app", "dev", []string{"key"})
	ch2 := d.instancePool.Register("client-2@127.0.0.1", "app", "dev", []string{"key"})

	elem := &concept.Element{
		Metadata: &concept.ElementMetadata{App: "app", Env: "dev", Key: "key"},
		Raw:      []byte("value"),
		Version:  1,
	}
	_, err := d.Dispatch(context.Background(), &pb.DispatchReq{
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

func (a *watchPersistAggregate) GetApp(context.Context, string) (*pb.AppMetadata, error) {
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

func (a *watchPersistAggregate) Watch(context.Context, chan<- *pb.AgentInstanceChange) error {
	return nil
}

func (a *watchPersistAggregate) Register(context.Context, *pb.AgentInstance, int32) error {
	return nil
}

func (a *watchPersistAggregate) Renew(context.Context, *pb.AgentInstance, int32) error {
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

func (s watchServer) Send(*clientpb.WatchResp) error { return s.sendErr }

func (s watchServer) SetHeader(metadata.MD) error { return nil }

func (s watchServer) SendHeader(metadata.MD) error { return nil }

func (s watchServer) SetTrailer(metadata.MD) {}

func (s watchServer) Context() context.Context { return s.ctx }

func (s watchServer) SendMsg(any) error { return nil }

func (s watchServer) RecvMsg(any) error { return nil }
