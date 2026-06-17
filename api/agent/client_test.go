package agent

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"

	"github.com/yeqown/cassem/api/concept"
)

func TestWatchStreamFailureKeepsWatchingForRenew(t *testing.T) {
	clientCtx, cancel := context.WithCancel(context.Background())
	defer cancel()

	streamDone := make(chan struct{})
	fake := &failingWatchAgentClient{streamDone: streamDone}
	c := newTestAgentInstanceClient(fake, clientCtx, cancel)

	err := c.Watch(context.Background(), "test", "default", func(*concept.Element) {}, "ele1", "config")
	require.NoError(t, err)

	<-streamDone
	time.Sleep(10 * time.Millisecond)
	c.renewSelf()

	require.Len(t, fake.renewReq.GetWatching(), 1)
	require.Equal(t, "test", fake.renewReq.GetWatching()[0].GetApp())
	require.Equal(t, "default", fake.renewReq.GetWatching()[0].GetEnv())
	require.Equal(t, []string{"ele1", "config"}, fake.renewReq.GetWatching()[0].GetWatchKeys())
}

func TestWatchContextCancellationRemovesWatchingForRenew(t *testing.T) {
	clientCtx, cancelClient := context.WithCancel(context.Background())
	defer cancelClient()

	fake := &cancelWatchAgentClient{failingWatchAgentClient: &failingWatchAgentClient{}}
	c := newTestAgentInstanceClient(fake, clientCtx, cancelClient)
	watchCtx, cancelWatch := context.WithCancel(context.Background())

	err := c.Watch(watchCtx, "test", "default", func(*concept.Element) {}, "ele1", "config")
	require.NoError(t, err)

	cancelWatch()
	require.Eventually(t, func() bool {
		c.renewSelf()
		return len(fake.renewReq.GetWatching()) == 0
	}, time.Second, 10*time.Millisecond)
}

func newTestAgentInstanceClient(fake AgentClient, ctx context.Context, cancel context.CancelFunc) *agentInstanceClient {
	return &agentInstanceClient{
		agentClient: fake,
		opt:         &options{clientId: "client-01", clientIp: "127.0.0.1"},
		watching:    make(map[string]*concept.Instance_Watching),
		quit:        make(chan struct{}, 1),
		ctx:         ctx,
		cancel:      cancel,
	}
}

type failingWatchAgentClient struct {
	streamDone chan struct{}
	renewReq   *RegisterReq
}

type cancelWatchAgentClient struct {
	*failingWatchAgentClient
}

func (c *failingWatchAgentClient) GetElement(context.Context, *GetElementReq, ...grpc.CallOption) (*GetElementResp, error) {
	return nil, nil
}

func (c *failingWatchAgentClient) Unregister(context.Context, *UnregisterReq, ...grpc.CallOption) (*EmptyResp, error) {
	return nil, nil
}

func (c *failingWatchAgentClient) Register(context.Context, *RegisterReq, ...grpc.CallOption) (*EmptyResp, error) {
	return nil, nil
}

func (c *failingWatchAgentClient) Renew(_ context.Context, req *RegisterReq, _ ...grpc.CallOption) (*EmptyResp, error) {
	c.renewReq = req
	return &EmptyResp{}, nil
}

func (c *failingWatchAgentClient) Watch(ctx context.Context, _ *WatchReq, _ ...grpc.CallOption) (Agent_WatchClient, error) {
	return failingWatchStream{ctx: ctx, done: c.streamDone}, nil
}

func (c *cancelWatchAgentClient) Watch(ctx context.Context, _ *WatchReq, _ ...grpc.CallOption) (Agent_WatchClient, error) {
	return cancelWatchStream{ctx: ctx}, nil
}

type failingWatchStream struct {
	ctx  context.Context
	done chan struct{}
}

type cancelWatchStream struct {
	ctx context.Context
}

func (s failingWatchStream) Recv() (*WatchResp, error) {
	return nil, s.RecvMsg(nil)
}

func (s failingWatchStream) Header() (metadata.MD, error) { return nil, nil }

func (s failingWatchStream) Trailer() metadata.MD { return nil }

func (s failingWatchStream) CloseSend() error { return nil }

func (s failingWatchStream) Context() context.Context { return s.ctx }

func (s failingWatchStream) SendMsg(any) error { return nil }

func (s failingWatchStream) RecvMsg(any) error {
	select {
	case <-s.done:
	default:
		close(s.done)
	}
	return errors.New("stream failed")
}

func (s cancelWatchStream) Recv() (*WatchResp, error) {
	return nil, s.RecvMsg(nil)
}

func (s cancelWatchStream) Header() (metadata.MD, error) { return nil, nil }

func (s cancelWatchStream) Trailer() metadata.MD { return nil }

func (s cancelWatchStream) CloseSend() error { return nil }

func (s cancelWatchStream) Context() context.Context { return s.ctx }

func (s cancelWatchStream) SendMsg(any) error { return nil }

func (s cancelWatchStream) RecvMsg(any) error {
	<-s.ctx.Done()
	return s.ctx.Err()
}
