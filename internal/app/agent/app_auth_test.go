package agent

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/yeqown/cassem/internal/concept"
	clientpb "github.com/yeqown/cassem/sdk/go/protogen"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

// A populated cache must never turn a missing or malformed identity into a read.
func TestGetElementRejectsInvalidBearerBeforeCache(t *testing.T) {
	for _, values := range [][]string{nil, {""}, {"Basic secret"}, {"Bearer"}, {"Bearer first second"}, {"Bearer first", "Bearer second"}, {"Bearer first,second"}} {
		t.Run(valuesName(values), func(t *testing.T) {
			d := app{cache: NewCache(10), authorizer: allowTokenAuthorizer()}
			elem := &concept.Element{Metadata: &concept.ElementMetadata{App: "demo", Env: "prod", Key: "config"}, Raw: []byte("private")}
			d.cache.Set("demo", "prod", "config", elem)
			d.cache.Set("demo", "prod", "config", elem)
			_, cached := d.cache.Query("demo", "prod", "config")
			require.True(t, cached)
			ctx := metadata.NewIncomingContext(context.Background(), metadata.MD{"authorization": values})
			got, err := d.GetElement(ctx, &clientpb.GetElementReq{App: "demo", Env: "prod", Keys: []string{"config"}})
			require.ErrorIs(t, err, concept.Err_UNAUTHENTICATED)
			require.Nil(t, got)
		})
	}
}

func valuesName(values []string) string {
	if len(values) == 0 {
		return "missing"
	}
	return values[0]
}

func TestGetElementAuthorizesEveryCacheHit(t *testing.T) {
	revoked := false
	d := app{cache: NewCache(10), authorizer: tokenAuthorizerFunc(func(_ context.Context, token string, targets ...concept.TokenTarget) error {
		require.Equal(t, "test-token", token)
		require.Equal(t, []concept.TokenTarget{{App: "demo", Env: "prod"}}, targets)
		if revoked {
			return concept.Err_UNAUTHENTICATED
		}
		return nil
	})}
	elem := &concept.Element{Metadata: &concept.ElementMetadata{App: "demo", Env: "prod", Key: "config"}, Raw: []byte("private")}
	d.cache.Set("demo", "prod", "config", elem)
	d.cache.Set("demo", "prod", "config", elem)
	req := &clientpb.GetElementReq{App: "demo", Env: "prod", Keys: []string{"config"}}
	got, err := d.GetElement(tokenContext(), req)
	require.NoError(t, err)
	require.Equal(t, []*concept.Element{elem}, got.GetElems())
	revoked = true
	got, err = d.GetElement(tokenContext(), req)
	require.ErrorIs(t, err, concept.Err_UNAUTHENTICATED)
	require.Nil(t, got)
}

func TestGetElementFailsClosedWithoutAuthorizer(t *testing.T) {
	got, err := (&app{}).GetElement(tokenContext(), &clientpb.GetElementReq{App: "demo", Env: "prod", Keys: []string{"config"}})
	require.ErrorIs(t, err, concept.Err_UNAUTHENTICATED)
	require.Nil(t, got)
}

func TestWatchRejectsUnauthorizedBeforeChangingSubscription(t *testing.T) {
	for _, tc := range []struct {
		name       string
		ctx        context.Context
		authorizer concept.TokenAuthorizer
		want       codes.Code
	}{
		{name: "missing identity", ctx: context.Background(), authorizer: allowTokenAuthorizer(), want: codes.Unauthenticated},
		{name: "missing authorizer", ctx: tokenContext(), want: codes.Unauthenticated},
		{name: "denied", ctx: tokenContext(), authorizer: tokenAuthorizerFunc(func(context.Context, string, ...concept.TokenTarget) error { return concept.Err_PERMISSION_DENIED }), want: codes.PermissionDenied},
	} {
		t.Run(tc.name, func(t *testing.T) {
			ctx, cancel := context.WithCancel(tc.ctx)
			cancel()
			pool := NewInstancePool()
			pool.Register("client-01@10.0.0.1", "demo", "prod", []string{"config"})
			d := app{aggregate: &watchPersistAggregate{}, instancePool: pool, authorizer: tc.authorizer}
			err := d.Watch(&clientpb.WatchReq{ClientId: "client-01", ClientIp: "10.0.0.1", Watching: []*clientpb.Instance_Watching{{App: "demo", Env: "prod", WatchKeys: []string{"config"}}}}, watchServer{ctx: ctx})
			require.Equal(t, tc.want, status.Code(err))
			require.Equal(t, []string{"client-01@10.0.0.1"}, pool.ListWatchingInstances("demo", "prod", "config"))
		})
	}
}

func TestWatchReauthorizesAllNamespacesBeforeDelivery(t *testing.T) {
	for _, denied := range []error{concept.Err_UNAUTHENTICATED, concept.Err_PERMISSION_DENIED} {
		t.Run(denied.Error(), func(t *testing.T) {
			calls := 0
			d := app{aggregate: &watchPersistAggregate{}, cache: NewCache(10), instancePool: NewInstancePool(), authorizer: tokenAuthorizerFunc(func(_ context.Context, token string, targets ...concept.TokenTarget) error {
				require.Equal(t, "test-token", token)
				require.Equal(t, []concept.TokenTarget{{App: "demo", Env: "prod"}, {App: "other", Env: "staging"}}, targets)
				calls++
				if calls > 1 {
					return denied
				}
				return nil
			})}
			insID := "client-01@10.0.0.1"
			d.instancePool.Register(insID, "demo", "prod", []string{"config"})
			d.instancePool.Notify(insID, &concept.Element{Metadata: &concept.ElementMetadata{App: "demo", Env: "prod", Key: "config"}})
			server := &recordingWatchServer{watchServer: watchServer{ctx: tokenContext(), sendErr: errors.New("unexpected delivery")}}
			err := d.Watch(&clientpb.WatchReq{ClientId: "client-01", ClientIp: "10.0.0.1", Watching: []*clientpb.Instance_Watching{{App: "demo", Env: "prod", WatchKeys: []string{"config"}}, {App: "other", Env: "staging", WatchKeys: []string{"limits"}}}}, server)
			require.Equal(t, status.Code(concept.ToStatus(denied)), status.Code(err))
			require.Empty(t, server.sent)
		})
	}
}

func TestWatchFiltersUnrequestedElementsOnCollidingIdentity(t *testing.T) {
	d := app{aggregate: &watchPersistAggregate{}, cache: NewCache(10), instancePool: NewInstancePool(), authorizer: allowTokenAuthorizer()}
	insID := "client-01@10.0.0.1"
	d.instancePool.Register(insID, "demo", "prod", []string{"config"})
	for _, md := range []*concept.ElementMetadata{{App: "other", Env: "prod", Key: "config"}, {App: "demo", Env: "prod", Key: "secret"}, {App: "demo", Env: "prod", Key: "config"}} {
		d.instancePool.Notify(insID, &concept.Element{Metadata: md})
	}
	stopped := errors.New("stop after first allowed delivery")
	server := &recordingWatchServer{watchServer: watchServer{ctx: tokenContext(), sendErr: stopped}}
	err := d.Watch(&clientpb.WatchReq{ClientId: "client-01", ClientIp: "10.0.0.1", Watching: []*clientpb.Instance_Watching{{App: "demo", Env: "prod", WatchKeys: []string{"config"}}}}, server)
	require.ErrorIs(t, err, stopped)
	require.Len(t, server.sent, 1)
	require.Equal(t, "demo", server.sent[0].GetElem().GetMetadata().GetApp())
	require.Equal(t, "config", server.sent[0].GetElem().GetMetadata().GetKey())
}

type tokenAuthorizerFunc func(context.Context, string, ...concept.TokenTarget) error

func (f tokenAuthorizerFunc) AuthorizeToken(ctx context.Context, token string, targets ...concept.TokenTarget) error {
	return f(ctx, token, targets...)
}
func allowTokenAuthorizer() concept.TokenAuthorizer {
	return tokenAuthorizerFunc(func(context.Context, string, ...concept.TokenTarget) error { return nil })
}
func tokenContext() context.Context {
	return metadata.NewIncomingContext(context.Background(), metadata.Pairs("authorization", "Bearer test-token"))
}

type recordingWatchServer struct {
	watchServer
	sent []*clientpb.WatchResp
}

func (s *recordingWatchServer) Send(resp *clientpb.WatchResp) error {
	s.sent = append(s.sent, resp)
	return s.sendErr
}
