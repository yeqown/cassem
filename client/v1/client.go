package v1

import (
	"context"
	"fmt"

	pb "github.com/yeqown/cassem/clientv1/gen"

	"google.golang.org/grpc"
)

type Client struct {
	cc *grpc.ClientConn

	cfg *Config
	//// changeCh is a nonblocking channel, it's buffer size is double time of
	//// Config.Watching's length.
	//changeCh chan Changes

	// AuthConfig *Auth

	changeFn HandlerFunc
}

func New(cfg *Config) (*Client, error) {
	cfg = adaptConfig(cfg)

	timeoutCtx, cancel := context.WithTimeout(context.Background(), cfg.DialTimeout)
	defer cancel()
	cc, err := grpc.DialContext(timeoutCtx, cfg.Endpoint, grpc.WithInsecure())
	if err != nil {
		return nil, err
	}

	client := &Client{
		cc:       cc,
		cfg:      cfg,
		changeFn: cfg.Fn,
		//changeCh: make(chan Changes, len(cfg.Watching)*2),
	}

	if err = client.lazyInitWatcher(); err != nil {
		return nil, err
	}

	return client, nil
}

func defaultChangeHandlerFunc(c Changes) {
	return
}

type HandlerFunc func(c Changes)

// SetChangesHandler
func (c *Client) SetChangesHandler(fn HandlerFunc) {
	if fn == nil {
		return
	}

	c.changeFn = fn
}

func (c *Client) lazyInitWatcher() error {
	if len(c.cfg.Watching) == 0 {
		return nil
	}

	watches := make([]*pb.WatchOption, 0, len(c.cfg.Watching))
	for _, opt := range c.cfg.Watching {
		watches = append(watches, &pb.WatchOption{
			Namespace: opt.Namespace,
			Keys:      opt.Keys,
			Format:    toPBFormat(opt.Format),
		})
	}

	// start watching
	req := pb.WatchReq{
		Watches: watches,
	}
	stream, err := pb.NewWatcherClient(c.cc).Watch(context.TODO(), &req)
	if err != nil {
		return err
	}

	// receive asynchronously
	go c.receive(stream)

	return nil
}

func (c *Client) receive(stream pb.Watcher_WatchClient) {
	var (
		m = new(pb.Changes)
		//ch Changes
	)
	for {
		if err := stream.RecvMsg(m); err != nil {
			fmt.Printf("Client failed to RecvMsg: %v\n", err)
			continue
		}

		// now only need to use handlerFunc, maybe use channel to handle this quickly.
		c.changeFn(Changes{
			Key:       m.GetKey(),
			Namespace: m.GetNamespace(),
			Format:    fromPBFormat(m.GetFormat()),
			CheckSum:  m.GetChecksum(),
			Data:      m.GetData(),
		})
		//
		//// send but not block.
		//select {
		//case c.changeCh <- ch:
		//default:
		//}
	}
}
