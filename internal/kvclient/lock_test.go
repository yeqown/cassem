package kvclient

import (
	"context"
	"fmt"
	"sync"
	"testing"

	pb "github.com/yeqown/cassem/internal/protogen"

	"github.com/stretchr/testify/assert"
	"google.golang.org/grpc"
)

type fakeLockKV struct {
	mu   sync.Mutex
	keys map[string]struct{}
}

func newFakeLockKV() *fakeLockKV {
	return &fakeLockKV{keys: make(map[string]struct{})}
}

func (f *fakeLockKV) GetKV(context.Context, *pb.GetKVReq, ...grpc.CallOption) (*pb.GetKVResp, error) {
	panic("not implemented")
}

func (f *fakeLockKV) GetKVs(context.Context, *pb.GetKVsReq, ...grpc.CallOption) (*pb.GetKVsResp, error) {
	panic("not implemented")
}

func (f *fakeLockKV) SetKV(_ context.Context, req *pb.SetKVReq, _ ...grpc.CallOption) (*pb.Empty, error) {
	f.mu.Lock()
	defer f.mu.Unlock()

	if _, exists := f.keys[req.GetKey()]; exists && !req.GetOverwrite() {
		return nil, fmt.Errorf("key exists")
	}
	f.keys[req.GetKey()] = struct{}{}
	return &pb.Empty{}, nil
}

func (f *fakeLockKV) UnsetKV(_ context.Context, req *pb.UnsetKVReq, _ ...grpc.CallOption) (*pb.Empty, error) {
	f.mu.Lock()
	defer f.mu.Unlock()

	delete(f.keys, req.GetKey())
	return &pb.Empty{}, nil
}

func (f *fakeLockKV) Watch(context.Context, *pb.WatchReq, ...grpc.CallOption) (pb.KV_WatchClient, error) {
	panic("not implemented")
}

func (f *fakeLockKV) TTL(context.Context, *pb.TtlReq, ...grpc.CallOption) (*pb.TtlResp, error) {
	panic("not implemented")
}

func (f *fakeLockKV) Expire(context.Context, *pb.ExpireReq, ...grpc.CallOption) (*pb.Empty, error) {
	panic("not implemented")
}

func (f *fakeLockKV) Range(context.Context, *pb.RangeReq, ...grpc.CallOption) (*pb.RangeResp, error) {
	panic("not implemented")
}

func (f *fakeLockKV) CompactElementHistory(context.Context, *pb.CompactElementHistoryReq, ...grpc.CallOption) (*pb.CompactElementHistoryResp, error) {
	panic("not implemented")
}

func TestWithLockReleasesLock(t *testing.T) {
	kv := newFakeLockKV()

	assert.NotPanics(t, func() {
		WithLock(kv, "locks/TestWithLockReleasesLock", 10, func() {})
	})
	assert.NotPanics(t, func() {
		WithLock(kv, "locks/TestWithLockReleasesLock", 10, func() {})
	})
}

func TestWithLockPanicsWhenLockAlreadyHeld(t *testing.T) {
	kv := newFakeLockKV()
	_, err := kv.SetKV(context.Background(), &pb.SetKVReq{Key: "locks/TestWithLockPanicsWhenLockAlreadyHeld"})
	assert.NoError(t, err)

	assert.Panics(t, func() {
		WithLock(kv, "locks/TestWithLockPanicsWhenLockAlreadyHeld", 10, func() {})
	})
}
