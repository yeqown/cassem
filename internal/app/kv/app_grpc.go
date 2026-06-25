package kv

import (
	"context"
	"net"

	"github.com/yeqown/log"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/reflection"
	"google.golang.org/grpc/status"

	errorx "github.com/yeqown/cassem/api/concept"
	apikv "github.com/yeqown/cassem/api/kv"
	"github.com/yeqown/cassem/pkg/grpcx"
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
	apikv.RegisterKVServer(s, srv)
	apikv.RegisterClusterServer(s, srv)
	reflection.Register(s)

	return s
}

func (s grpcServer) GetKV(ctx context.Context, req *apikv.GetKVReq) (*apikv.GetKVResp, error) {
	v, err := s.coord.getKV(req.GetKey())
	if err != nil {
		return nil, err
	}

	resp := &apikv.GetKVResp{
		Entity: v,
	}
	return resp, nil
}

func (s grpcServer) GetKVs(ctx context.Context, req *apikv.GetKVsReq) (*apikv.GetKVsResp, error) {
	entities := make([]*apikv.Entity, 0, len(req.GetKeys()))
	errors := make([]*apikv.KeyError, 0)
	for _, k := range req.GetKeys() {
		v, err := s.coord.getKV(k)
		if err != nil {
			errors = append(errors, newKeyError(k, err))
			continue
		}

		entities = append(entities, v)
	}

	resp := &apikv.GetKVsResp{
		Entities: entities,
		Errors:   errors,
	}
	return resp, nil
}

func newKeyError(key string, err error) *apikv.KeyError {
	code := codes.Unknown
	message := err.Error()
	if x, ok := errorx.FromError(err); ok {
		code = x.Code.Code()
		message = x.Message
	} else if s, ok := status.FromError(err); ok {
		code = s.Code()
		message = s.Message()
	}

	return &apikv.KeyError{
		Key:     key,
		Code:    code.String(),
		Message: message,
	}
}

func (s grpcServer) SetKV(ctx context.Context, req *apikv.SetKVReq) (*apikv.Empty, error) {
	err := s.coord.setKV(ctx, &setKVParam{
		key:       req.GetKey(),
		val:       req.GetVal(),
		isDir:     req.GetIsDir(),
		ttl:       req.GetTtl(),
		overwrite: req.GetOverwrite(),
	})

	return _empty, err
}

var _empty = new(apikv.Empty)

func (s grpcServer) UnsetKV(ctx context.Context, req *apikv.UnsetKVReq) (*apikv.Empty, error) {
	err := s.coord.unsetKV(ctx, &unsetKVParam{
		key:   req.GetKey(),
		isDir: req.GetIsDir(),
	})
	return _empty, err
}

func (s grpcServer) Watch(req *apikv.WatchReq, stream apikv.KV_WatchServer) (err error) {
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

func (s grpcServer) TTL(ctx context.Context, req *apikv.TtlReq) (*apikv.TtlResp, error) {
	ttl, err := s.coord.ttl(req.GetKey())
	return &apikv.TtlResp{Ttl: ttl}, err
}

func (s grpcServer) Expire(ctx context.Context, req *apikv.ExpireReq) (*apikv.Empty, error) {
	err := s.coord.expire(req.GetKey())
	return _empty, err
}

func (s grpcServer) Range(ctx context.Context, req *apikv.RangeReq) (*apikv.RangeResp, error) {
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

func (s grpcServer) CompactElementHistory(ctx context.Context, req *apikv.CompactElementHistoryReq) (*apikv.CompactElementHistoryResp, error) {
	return s.coord.compactElementHistory(req)
}

// translateChange construct an kv.Change from concept.Change interface.
// DONE(@yeqown): use proto to ignore convert procedure.
func translateChange(change errorx.Change) *apikv.Change {
	var (
		c  *apikv.Change
		ok bool
	)

	switch change.Type() {
	case errorx.ChangeType_KV:
		c, ok = change.(*apikv.Change)
	case errorx.ChangeType_DIR:
		var pdc *apikv.ParentDirectoryChange
		if pdc, ok = change.(*apikv.ParentDirectoryChange); ok {
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
	ctx context.Context, req *apikv.AddNodeRequest) (resp *apikv.AddNodeResponse, err error) {
	resp = new(apikv.AddNodeResponse)
	resp.NodeId, resp.Peers, err = s.coord.addNode(ctx, req.GetRaftAddr(), req.GetGrpcEndpoint())
	return
}

func (s grpcServer) RemoveNode(
	ctx context.Context, req *apikv.RemoveNodeRequest) (resp *apikv.RemoveNodeResponse, err error) {
	err = s.coord.removeNode(ctx, req.GetNodeId())
	resp = new(apikv.RemoveNodeResponse)
	return
}

func (s grpcServer) ListMembers(
	ctx context.Context, req *apikv.ListMembersRequest) (*apikv.ListMembersResponse, error) {
	members, err := s.coord.listMembers()
	if err != nil {
		return nil, err
	}
	return &apikv.ListMembersResponse{Members: members}, nil
}

// // isClientClosed check whether the error contains any code which indicates client is offline.
// // These codes includes: codes.Unavailable
// func isClientClosed(err error) bool {
//	return status.Convert(err).Code() == codes.Unavailable
// }

func serve(s *grpc.Server, addr string) error {
	lis, err := net.Listen("tcp", addr)
	if err != nil {
		return err
	}

	return s.Serve(lis)
}
