package kv

import (
	"context"
	"fmt"
	"sync"
	"testing"

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

func (f *fakeLockKV) GetKV(context.Context, *GetKVReq, ...grpc.CallOption) (*GetKVResp, error) {
	panic("not implemented")
}

func (f *fakeLockKV) GetKVs(context.Context, *GetKVsReq, ...grpc.CallOption) (*GetKVsResp, error) {
	panic("not implemented")
}

func (f *fakeLockKV) SetKV(_ context.Context, req *SetKVReq, _ ...grpc.CallOption) (*Empty, error) {
	f.mu.Lock()
	defer f.mu.Unlock()

	if _, exists := f.keys[req.GetKey()]; exists && !req.GetOverwrite() {
		return nil, fmt.Errorf("key exists")
	}
	f.keys[req.GetKey()] = struct{}{}
	return &Empty{}, nil
}

func (f *fakeLockKV) UnsetKV(_ context.Context, req *UnsetKVReq, _ ...grpc.CallOption) (*Empty, error) {
	f.mu.Lock()
	defer f.mu.Unlock()

	delete(f.keys, req.GetKey())
	return &Empty{}, nil
}

func (f *fakeLockKV) Watch(context.Context, *WatchReq, ...grpc.CallOption) (KV_WatchClient, error) {
	panic("not implemented")
}

func (f *fakeLockKV) TTL(context.Context, *TtlReq, ...grpc.CallOption) (*TtlResp, error) {
	panic("not implemented")
}

func (f *fakeLockKV) Expire(context.Context, *ExpireReq, ...grpc.CallOption) (*Empty, error) {
	panic("not implemented")
}

func (f *fakeLockKV) Range(context.Context, *RangeReq, ...grpc.CallOption) (*RangeResp, error) {
	panic("not implemented")
}

func (f *fakeLockKV) CompactElementHistory(context.Context, *CompactElementHistoryReq, ...grpc.CallOption) (*CompactElementHistoryResp, error) {
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
	_, err := kv.SetKV(context.Background(), &SetKVReq{Key: "locks/TestWithLockPanicsWhenLockAlreadyHeld"})
	assert.NoError(t, err)

	assert.Panics(t, func() {
		WithLock(kv, "locks/TestWithLockPanicsWhenLockAlreadyHeld", 10, func() {})
	})
}
