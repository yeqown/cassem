package raftimpl

import (
	"context"
	"errors"
	apikv "github.com/yeqown/cassem/api/kv"
	"testing"

	"github.com/stretchr/testify/require"

	errorx "github.com/yeqown/cassem/api/concept"
	"github.com/yeqown/cassem/internal/app/kv/storage"
)

type snapshotKV struct {
	snapshot []byte
	restored []byte
	snapErr  error
	recErr   error
}

type mutateKV struct {
	data     map[string]*apikv.Entity
	getErr   error
	setErr   error
	unsetErr error
}

func (m *mutateKV) GetKV(key string, _ bool) (*apikv.Entity, error) {
	if m.getErr != nil {
		return nil, m.getErr
	}
	entity, ok := m.data[key]
	if !ok {
		return nil, storage.ErrNotFound
	}
	return entity, nil
}

func (m *mutateKV) SetKV(key string, val *apikv.Entity, _ bool) error {
	if m.setErr != nil {
		return m.setErr
	}
	if m.data == nil {
		m.data = make(map[string]*apikv.Entity)
	}
	m.data[key] = val
	return nil
}

func (m *mutateKV) UnsetKV(key string, _ bool) error {
	if m.unsetErr != nil {
		return m.unsetErr
	}
	delete(m.data, key)
	return nil
}

func (m *mutateKV) Range(string, string, int) (*storage.RangeResult, error) {
	return &storage.RangeResult{}, nil
}

func (m *mutateKV) Snapshot() ([]byte, error) { return nil, nil }
func (m *mutateKV) RecoverSnapshot([]byte) error { return nil }

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

	err := (&raftNodeImpl{proposeC: make(chan *apikv.Propose)}).propose(ctx, &apikv.MutateCommand{Key: "k", Op: apikv.MutateCommand_SET})
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

func TestRaftNodeImplApplyMutateLogEntrySetResolvesAckAndEmitsChange(t *testing.T) {
	last := apikv.NewEntityWithCreated("root/key", []byte("old"), 0, 1)
	cur := apikv.NewEntityWithCreated("root/key", []byte("new"), 0, 1)
	ackCh := make(chan applyAck, 1)
	store := &mutateKV{data: map[string]*apikv.Entity{"root/key": last}}
	r := &raftNodeImpl{
		kvstore: store,
		changeC: make(chan errorx.Change, 4),
		pending: map[uint64]chan applyAck{1: ackCh},
	}

	err := r.applyMutateLogEntry(mutateEntry(t, &apikv.MutateCommand{RequestId: 1, Op: apikv.MutateCommand_SET, Key: "root/key", Value: cur}))
	require.NoError(t, err)
	require.Equal(t, cur.GetFingerprint(), store.data["root/key"].GetFingerprint())
	require.Equal(t, cur.GetKey(), store.data["root/key"].GetKey())
	require.NoError(t, (<-ackCh).err)
	require.Equal(t, "root/key", (<-r.changeC).Topic())
}

func TestRaftNodeImplApplyMutateLogEntryUnsetResolvesAckAndEmitsChange(t *testing.T) {
	last := apikv.NewEntityWithCreated("root/key", []byte("old"), 0, 1)
	ackCh := make(chan applyAck, 1)
	store := &mutateKV{data: map[string]*apikv.Entity{"root/key": last}}
	r := &raftNodeImpl{
		kvstore: store,
		changeC: make(chan errorx.Change, 4),
		pending: map[uint64]chan applyAck{2: ackCh},
	}

	err := r.applyMutateLogEntry(mutateEntry(t, &apikv.MutateCommand{RequestId: 2, Op: apikv.MutateCommand_UNSET, Key: "root/key"}))
	require.NoError(t, err)
	_, ok := store.data["root/key"]
	require.False(t, ok)
	require.NoError(t, (<-ackCh).err)
	require.Equal(t, "root/key", (<-r.changeC).Topic())
}

func TestRaftNodeImplApplyMutateLogEntryPropagatesStorageError(t *testing.T) {
	wantErr := errors.New("set failed")
	ackCh := make(chan applyAck, 1)
	r := &raftNodeImpl{
		kvstore: &mutateKV{setErr: wantErr},
		changeC: make(chan errorx.Change, 1),
		pending: map[uint64]chan applyAck{3: ackCh},
	}

	err := r.applyMutateLogEntry(mutateEntry(t, &apikv.MutateCommand{RequestId: 3, Op: apikv.MutateCommand_SET, Key: "root/key", Value: apikv.NewEntityWithCreated("root/key", []byte("new"), 0, 1)}))
	require.ErrorIs(t, err, wantErr)
	require.ErrorIs(t, (<-ackCh).err, wantErr)
	select {
	case <-r.changeC:
		t.Fatal("unexpected change emitted on storage failure")
	default:
	}
}

func TestRaftNodeImplShutdownFailsPendingRequests(t *testing.T) {
	ack1 := make(chan applyAck, 1)
	ack2 := make(chan applyAck, 1)
	r := &raftNodeImpl{
		proposeC: make(chan *apikv.Propose),
		pending:  map[uint64]chan applyAck{1: ack1, 2: ack2},
	}

	require.NoError(t, r.Shutdown())
	require.ErrorIs(t, (<-ack1).err, ErrShuttingDown)
	require.ErrorIs(t, (<-ack2).err, ErrShuttingDown)
}

func mutateEntry(t *testing.T, cmd *apikv.MutateCommand) *apikv.LogEntry {
	t.Helper()
	return &apikv.LogEntry{Action: apikv.LogEntry_Mutate, Command: apikv.Must(apikv.Marshal(cmd))}
}
