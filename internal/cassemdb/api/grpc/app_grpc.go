package grpc

import (
	"context"
	"net"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	pb "github.com/yeqown/cassem/internal/cassemdb/api/grpc/gen"
	"github.com/yeqown/cassem/internal/cassemdb/app"
	"github.com/yeqown/cassem/pkg/types"
	"github.com/yeqown/cassem/pkg/watcher"

	"github.com/yeqown/log"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
)

type grpcServer struct {
	quit  chan struct{}
	coord app.ICoordinator
}

func New(coord app.ICoordinator) *grpc.Server {
	srv := &grpcServer{
		quit:  make(chan struct{}, 1),
		coord: coord,
	}

	// DONE(@yeqown): recover and logger interceptor needed
	s := grpc.NewServer(
		grpc.UnaryInterceptor(chainUnaryServer(serverRecovery(), serverLogger())),
	)
	pb.RegisterApiServer(s, srv)
	reflection.Register(s)

	return s
}

func (s grpcServer) GetKV(ctx context.Context, req *pb.GetKVReq) (*pb.GetKVResp, error) {
	v, err := s.coord.GetKV(req.GetKey())
	if err != nil {
		return nil, err
	}

	resp := &pb.GetKVResp{
		Entity: convertStoreValue(v),
	}
	return resp, nil
}

func (s grpcServer) SetKV(ctx context.Context, req *pb.SetKVReq) (*pb.Empty, error) {
	err := s.coord.SetKV(req.GetKey(), req.GetEntity().GetVal())
	return _empty, err
}

var _empty = new(pb.Empty)

func (s grpcServer) UnsetKV(ctx context.Context, req *pb.UnsetKVReq) (*pb.Empty, error) {
	err := s.coord.UnsetKV(req.GetKey(), req.GetIsDir())
	return _empty, err
}

func (s grpcServer) Watch(req *pb.WatchReq, stream pb.Api_WatchServer) (err error) {
	keys := req.GetKeys()
	// changeCh := make(chan watcher.IChange, len(keys))
	// once := sync.Once{}

	ob, cancel := s.coord.Watch(keys...)
	defer cancel()

	var v watcher.IChange

	for {
		select {
		case v = <-ob.Outbound():
			log.
				WithField("change", v).
				Debug("grpcServer.Watch will be send to client")

			c, ok := v.(*types.Change)
			if !ok {
				log.Debug("grpcServer.Watch skip the change, not the type(*types.Change)")
				continue
			}

			if err = stream.Send(convertChange(c)); err != nil {
				log.
					Errorf("grpcServer(grpc).Watch gets failed to send to client: %v", err)
				continue
			}

		case <-stream.Context().Done():
			// FIXED: what is the timing to quit and release resources timely.
			log.Debug("grpcServer(grpc).Watch received stream done signal, now quit")
			return

		case <-s.quit:
			// if server quit, all watcher should quit too.
			return
		}
	}
}

func convertStoreValue(v *types.StoreValue) *pb.Entity {
	if v == nil {
		return nil
	}

	return &pb.Entity{
		Fingerprint: v.Fingerprint,
		Key:         v.Key.String(),
		Val:         v.Val,
		CreatedAt:   v.CreatedAt,
		UpdatedAt:   v.UpdatedAt,
	}
}

func convertChange(c *types.Change) *pb.Change {
	if c == nil {
		return nil
	}

	return &pb.Change{
		Op:      pb.ChangeOp(c.Op),
		Key:     c.Key.String(),
		Last:    convertStoreValue(c.Last),
		Current: convertStoreValue(c.Current),
	}
}

// isClientClosed check whether the error contains any code which indicates client is offline.
// These codes includes: codes.Unavailable
func isClientClosed(err error) bool {
	return status.Convert(err).Code() == codes.Unavailable
}

// ListenAndServe [DO NOT USE] only works for testing.
func ListenAndServe(addr string) error {
	srv := &grpcServer{
		quit: make(chan struct{}, 1),
	}

	lis, err := net.Listen("tcp", addr)
	if err != nil {
		return err
	}

	s := grpc.NewServer()
	pb.RegisterApiServer(s, srv)
	reflection.Register(s)

	return s.Serve(lis)
}
