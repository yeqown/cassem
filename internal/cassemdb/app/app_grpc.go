package app

import (
	"context"
	"net"

	pb "github.com/yeqown/cassem/internal/cassemdb/api/gen"
	"github.com/yeqown/cassem/internal/cassemdb/infras/repository"
	"github.com/yeqown/cassem/pkg/watcher"

	"github.com/yeqown/log"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
)

type grpcServer struct {
	quit  chan struct{}
	coord ICoordinator
}

func gRPC(coord ICoordinator) *grpc.Server {
	srv := &grpcServer{
		quit:  make(chan struct{}, 1),
		coord: coord,
	}

	// DONE(@yeqown): recover and logger interceptor needed
	s := grpc.NewServer(
		grpc.UnaryInterceptor(chainUnaryServer(serverRecovery(), serverLogger(), sevrerErrorx())),
	)
	pb.RegisterKVServer(s, srv)
	reflection.Register(s)

	return s
}

func (s grpcServer) GetKV(ctx context.Context, req *pb.GetKVReq) (*pb.GetKVResp, error) {
	v, err := s.coord.getKV(req.GetKey())
	if err != nil {
		return nil, err
	}

	resp := &pb.GetKVResp{
		Entity: convertStoreValue(v),
	}
	return resp, nil
}

func (s grpcServer) GetKVs(ctx context.Context, req *pb.GetKVsReq) (*pb.GetKVsResp, error) {
	entities := make([]*pb.Entity, 0, len(req.GetKeys()))
	for _, k := range req.GetKeys() {
		v, err := s.coord.getKV(k)
		if err != nil {
			continue
		}

		entities = append(entities, convertStoreValue(v))
	}

	resp := &pb.GetKVsResp{
		Entities: entities,
	}
	return resp, nil
}

func (s grpcServer) SetKV(ctx context.Context, req *pb.SetKVReq) (*pb.Empty, error) {
	err := s.coord.setKV(&setKVParam{
		key:       req.GetKey(),
		val:       req.GetVal(),
		isDir:     req.GetIsDir(),
		ttl:       req.GetTtl(),
		overwrite: req.GetOverwrite(),
	})

	return _empty, err
}

var _empty = new(pb.Empty)

func (s grpcServer) UnsetKV(ctx context.Context, req *pb.UnsetKVReq) (*pb.Empty, error) {
	err := s.coord.unsetKV(&unsetKVParam{
		key:   req.GetKey(),
		isDir: req.GetIsDir(),
	})
	return _empty, err
}

func (s grpcServer) Watch(req *pb.WatchReq, stream pb.KV_WatchServer) (err error) {
	keys := req.GetKeys()
	// changeCh := make(chan watcher.IChange, len(keys))
	// once := sync.Once{}

	ob, cancel := s.coord.watch(keys...)
	defer cancel()

	var v watcher.IChange

	for {
		select {
		case v = <-ob.Outbound():
			log.
				WithField("change", v).
				Debug("grpcServer.watch will be send to client")

			c, ok := v.(*repository.Change)
			if !ok {
				log.Debug("grpcServer.watch skip the change, not the type(*types.Change)")
				continue
			}

			if err = stream.Send(convertChange(c)); err != nil {
				log.
					Errorf("grpcServer(grpc).watch gets failed to send to client: %v", err)
				continue
			}

		case <-stream.Context().Done():
			// FIXED: what is the timing to quit and release resources timely.
			log.Debug("grpcServer(grpc).watch received stream done signal, now quit")
			return

		case <-s.quit:
			// if server quit, all watcher should quit too.
			return
		}
	}
}

func (s grpcServer) TTL(ctx context.Context, req *pb.TtlReq) (*pb.TtlResp, error) {
	ttl, err := s.coord.ttl(req.GetKey())
	return &pb.TtlResp{Ttl: ttl}, err
}

func (s grpcServer) Expire(ctx context.Context, req *pb.ExpireReq) (*pb.Empty, error) {
	err := s.coord.expire(req.GetKey())
	return _empty, err
}

func (s grpcServer) Range(ctx context.Context, req *pb.RangeReq) (*pb.RangeResp, error) {
	result, err := s.coord.iter(&rangeParam{
		key:   req.GetKey(),
		seek:  req.GetSeek(),
		limit: int(req.GetLimit()),
	})

	if err != nil {
		return nil, err
	}

	entities := make([]*pb.Entity, 0, len(result.Items))
	for _, item := range result.Items {
		entities = append(entities, convertStoreValue(&item))
	}

	r := &pb.RangeResp{
		Entities:    entities,
		HasMore:     result.HasMore,
		NextSeekKey: result.NextSeekKey,
	}

	return r, nil
}

func convertStoreValue(v *repository.StoreValue) *pb.Entity {
	if v == nil {
		return nil
	}

	return &pb.Entity{
		Fingerprint: v.Fingerprint,
		Key:         v.Key.String(),
		Val:         v.Val,
		CreatedAt:   v.CreatedAt,
		UpdatedAt:   v.UpdatedAt,
		Ttl:         v.TTL,
		Typ:         v.Type(),
	}
}

func convertChange(c *repository.Change) *pb.Change {
	if c == nil {
		return nil
	}

	return &pb.Change{
		Op:      pb.Change_ChangeOp(c.Op),
		Key:     c.Key.String(),
		Last:    convertStoreValue(c.Last),
		Current: convertStoreValue(c.Current),
	}
}

//// isClientClosed check whether the error contains any code which indicates client is offline.
//// These codes includes: codes.Unavailable
//func isClientClosed(err error) bool {
//	return status.Convert(err).Code() == codes.Unavailable
//}

func serve(s *grpc.Server, addr string) error {
	lis, err := net.Listen("tcp", addr)
	if err != nil {
		return err
	}

	return s.Serve(lis)
}
