package app

import (
	"context"
	"net"

	"github.com/yeqown/log"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"

	apicassemdb "github.com/yeqown/cassem/internal/cassemdb/api"
	"github.com/yeqown/cassem/pkg/grpcx"
	"github.com/yeqown/cassem/pkg/watcher"
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
		grpc.UnaryInterceptor(
			grpcx.ChainUnaryServer(
				grpcx.ServerRecovery(), grpcx.ServerLogger(), grpcx.SevrerErrorx(), grpcx.ServerValidation()),
		),
	)
	apicassemdb.RegisterKVServer(s, srv)
	apicassemdb.RegisterClusterServer(s, srv)
	reflection.Register(s)

	return s
}

func (s grpcServer) GetKV(ctx context.Context, req *apicassemdb.GetKVReq) (*apicassemdb.GetKVResp, error) {
	v, err := s.coord.getKV(req.GetKey())
	if err != nil {
		return nil, err
	}

	resp := &apicassemdb.GetKVResp{
		Entity: v,
	}
	return resp, nil
}

func (s grpcServer) GetKVs(ctx context.Context, req *apicassemdb.GetKVsReq) (*apicassemdb.GetKVsResp, error) {
	entities := make([]*apicassemdb.Entity, 0, len(req.GetKeys()))
	for _, k := range req.GetKeys() {
		v, err := s.coord.getKV(k)
		if err != nil {
			continue
		}

		entities = append(entities, v)
	}

	resp := &apicassemdb.GetKVsResp{
		Entities: entities,
	}
	return resp, nil
}

func (s grpcServer) SetKV(ctx context.Context, req *apicassemdb.SetKVReq) (*apicassemdb.Empty, error) {
	err := s.coord.setKV(&setKVParam{
		key:       req.GetKey(),
		val:       req.GetVal(),
		isDir:     req.GetIsDir(),
		ttl:       req.GetTtl(),
		overwrite: req.GetOverwrite(),
	})

	return _empty, err
}

var _empty = new(apicassemdb.Empty)

func (s grpcServer) UnsetKV(ctx context.Context, req *apicassemdb.UnsetKVReq) (*apicassemdb.Empty, error) {
	err := s.coord.unsetKV(&unsetKVParam{
		key:   req.GetKey(),
		isDir: req.GetIsDir(),
	})
	return _empty, err
}

func (s grpcServer) Watch(req *apicassemdb.WatchReq, stream apicassemdb.KV_WatchServer) (err error) {
	ob, cancel := s.coord.watch(req.GetKeys()...)
	defer cancel()

	for {
		select {
		case change, ok := <-ob.Outbound():
			log.
				WithFields(log.Fields{
					"change": change,
					"ok":     ok,
				}).
				Debug("grpcServer.watch will be send to client")
			if !ok {
				return
			}

			// convert change from raw source into api.Change
			// DONE(@yeqown): use api.Change directly rather than convert it again an again.
			pbChange := translateChange(change)
			if pbChange == nil {
				continue
			}

			if err = stream.Send(pbChange); err != nil {
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

func (s grpcServer) TTL(ctx context.Context, req *apicassemdb.TtlReq) (*apicassemdb.TtlResp, error) {
	ttl, err := s.coord.ttl(req.GetKey())
	return &apicassemdb.TtlResp{Ttl: ttl}, err
}

func (s grpcServer) Expire(ctx context.Context, req *apicassemdb.ExpireReq) (*apicassemdb.Empty, error) {
	err := s.coord.expire(req.GetKey())
	return _empty, err
}

func (s grpcServer) Range(ctx context.Context, req *apicassemdb.RangeReq) (*apicassemdb.RangeResp, error) {
	result, err := s.coord.iterate(&rangeParam{
		key:   req.GetKey(),
		seek:  req.GetSeek(),
		limit: int(req.GetLimit()),
	})

	if err != nil {
		log.
			WithFields(log.Fields{
				"req":   req,
				"error": err,
			}).
			Error("grpcServer.Range failed")
		return nil, err
	}

	return result, nil
}

// translateChange construct an api.Change from watcher.IChange interface.
// DONE(@yeqown): use proto to ignore convert procedure.
func translateChange(change watcher.IChange) *apicassemdb.Change {
	var (
		c  *apicassemdb.Change
		ok bool
	)

	switch change.Type() {
	case watcher.ChangeType_KV:
		c, ok = change.(*apicassemdb.Change)
	case watcher.ChangeType_DIR:
		var pdc *apicassemdb.ParentDirectoryChange
		if pdc, ok = change.(*apicassemdb.ParentDirectoryChange); ok {
			c = pdc.Change
		}
	default:
	}

	if !ok || c == nil {
		log.
			WithField("change", change).
			Warn("cassemdb.translateChange skip the change")
		return nil
	}
	return c
}

func (s grpcServer) AddNode(
	ctx context.Context, req *apicassemdb.AddNodeRequest) (resp *apicassemdb.AddNodeResponse, err error) {
	resp = new(apicassemdb.AddNodeResponse)
	resp.NodeId, resp.Peers, err = s.coord.addNode(req.GetAddr())
	return
}

func (s grpcServer) RemoveNode(
	ctx context.Context, req *apicassemdb.RemoveNodeRequest) (resp *apicassemdb.RemoveNodeResponse, err error) {
	err = s.coord.removeNode(req.GetNodeId())
	resp = new(apicassemdb.RemoveNodeResponse)
	return
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
