package raftimpl

import (
	"context"
	"errors"
	apikv "github.com/yeqown/cassem/api/kv"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/yeqown/cassem/internal/app/kv/storage"
)

type snapshotKV struct {
	snapshot []byte
	restored []byte
	snapErr  error
	recErr   error
}

func (s *snapshotKV) GetKV(string, bool) (*apikv.Entity, error) {
	return nil, storage.ErrNotFound
}

func (s *snapshotKV) SetKV(string, *apikv.Entity, bool) error { return nil }
func (s *snapshotKV) UnsetKV(string, bool) error              { return nil }
func (s *snapshotKV) Range(string, string, int) (*storage.RangeResult, error) {
	return &storage.RangeResult{}, nil
}
func (s *snapshotKV) Snapshot() ([]byte, error) { return s.snapshot, s.snapErr }
func (s *snapshotKV) RecoverSnapshot(snapshot []byte) error {
	s.restored = append([]byte(nil), snapshot...)
	return s.recErr
}

func TestRaftNodeImplProposeReturnsCanceledContext(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	err := (&raftNodeImpl{proposeC: make(chan *apikv.Propose)}).propose(ctx, &apikv.SetCommand{SetKey: "k"})
	require.ErrorIs(t, err, context.Canceled)
}

func TestRaftNodeImplApplyCommitsReturnsDecodeErrors(t *testing.T) {
	commitCh := make(chan *commit, 1)
	applyDoneC := make(chan struct{})
	commitCh <- &commit{data: []string{"not protobuf"}, applyDoneC: applyDoneC}
	close(commitCh)
	errorCh := make(chan error)
	close(errorCh)

	var err error
	require.NotPanics(t, func() {
		err = (&raftNodeImpl{kvstore: &snapshotKV{}}).applyCommits(commitCh, errorCh)
	})
	require.Error(t, err)
	require.Contains(t, err.Error(), "decode log entry")
}

func TestRaftNodeImplSnapshotDelegatesToStorage(t *testing.T) {
	store := &snapshotKV{snapshot: []byte("storage snapshot")}
	r := &raftNodeImpl{kvstore: store}

	snapshot, err := r.getSnapshot()
	require.NoError(t, err)
	require.Equal(t, store.snapshot, snapshot)
}

func TestRaftNodeImplSnapshotPropagatesStorageErrors(t *testing.T) {
	wantErr := errors.New("snapshot failed")
	r := &raftNodeImpl{kvstore: &snapshotKV{snapErr: wantErr}}

	_, err := r.getSnapshot()
	require.ErrorIs(t, err, wantErr)
}

func TestRaftNodeImplRecoverSnapshotDelegatesToStorage(t *testing.T) {
	store := &snapshotKV{}
	r := &raftNodeImpl{kvstore: store}

	snapshot := []byte("storage snapshot")
	require.NoError(t, r.recoverFromSnapshot(snapshot))
	require.Equal(t, snapshot, store.restored)
}

func TestRaftNodeImplRecoverSnapshotPropagatesStorageErrors(t *testing.T) {
	wantErr := errors.New("recover failed")
	r := &raftNodeImpl{kvstore: &snapshotKV{recErr: wantErr}}

	err := r.recoverFromSnapshot([]byte("storage snapshot"))
	require.ErrorIs(t, err, wantErr)
}
