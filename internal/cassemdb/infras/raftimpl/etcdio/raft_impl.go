package etcdio

import (
	"strings"
	"sync"
	"time"

	"github.com/golang/protobuf/proto"
	"github.com/pkg/errors"
	"github.com/yeqown/log"
	"go.etcd.io/etcd/raft/v3"
	"go.etcd.io/etcd/raft/v3/raftpb"
	"go.etcd.io/etcd/server/v3/etcdserver/api/snap"

	apicassemdb "github.com/yeqown/cassem/internal/cassemdb/api"
	"github.com/yeqown/cassem/internal/cassemdb/infras/raftimpl"
	"github.com/yeqown/cassem/internal/cassemdb/infras/storage"
	"github.com/yeqown/cassem/pkg/conf"
	"github.com/yeqown/cassem/pkg/errorx"
	"github.com/yeqown/cassem/pkg/runtime"
	"github.com/yeqown/cassem/pkg/watcher"
)

var (
	_ raftimpl.RaftNode = &raftNodeImpl{}
)

// raftNodeImpl implement raftimpl.RaftNode constraints.
type raftNodeImpl struct {
	// kvstore is backend store component.
	kvstore storage.KV
	// raftState of current raft node.
	raftState raft.StateType

	// proposeC to propose a commit to raft cluster.
	proposeC          chan *apicassemdb.Propose
	confChangeC       chan raftpb.ConfChange
	changeC           chan watcher.IChange
	leadershipFanOutC []chan<- bool
	snapshotter       *snap.Snapshotter

	mu sync.Mutex
}

func NewRaftNode(cfg *conf.CassemdbConfig) (rc *raftNodeImpl) {
	rc = new(raftNodeImpl)
	rc.proposeC = make(chan *apicassemdb.Propose, 4)
	rc.confChangeC = make(chan raftpb.ConfChange, 1)
	rc.changeC = make(chan watcher.IChange, 64)
	rc.leadershipFanOutC = make([]chan<- bool, 4)
	raftStateChangeC := make(chan raft.StateType, 4)
	var err error
	if rc.kvstore, err = storage.NewRepository(cfg.Bolt); err != nil {
		panic(err)
	}

	//start raft raftNode
	c := &config{
		id:          int(cfg.Raft.NodeId),
		peers:       cfg.Raft.Peers,
		join:        !cfg.Raft.BootstrapCluster,
		baseDir:     cfg.Raft.Base,
		snapCount:   uint64(cfg.Raft.SnapCount),
		getSnapshot: func() ([]byte, error) { return rc.getSnapshot() },
	}
	commitC, errorC, snapshotterReady := newRaftNode(c, rc.proposeC, rc.confChangeC, raftStateChangeC)
	rc.snapshotter = <-snapshotterReady

	rc.setup()

	// apply commits
	runtime.GoFunc("raftNodeImpl.applyCommits", func() error {
		return rc.applyCommits(commitC, errorC)
	})

	// fanout leader change signal.
	runtime.GoFunc("raftNodeImpl.leaderChangeFanOut", func() error {
		for {
			select {
			case state := <-raftStateChangeC:
				rc.raftState = state
				beLeader := state == raft.StateLeader
				log.Debugf("raftNodeImpl.leaderChangeFanOut: %v", beLeader)
				for _, ch := range rc.leadershipFanOutC {
					select {
					case ch <- beLeader:
					default:
					}
				}
			}
		}
	})

	return
}

func (r *raftNodeImpl) Shutdown() error {
	close(r.proposeC)
	close(r.confChangeC)

	return nil
}

func (r *raftNodeImpl) setup() {
	snapshot, err := r.loadSnapshot()
	if err != nil {
		log.Fatal(err)
	}
	if snapshot != nil {
		log.Infof("loading snapshot at term %d and index %d", snapshot.Metadata.Term, snapshot.Metadata.Index)
		if err := r.recoverFromSnapshot(snapshot.Data); err != nil {
			log.Fatal(err)
		}
	}
}

func (r *raftNodeImpl) applyCommits(commitCh <-chan *commit, errorCh <-chan error) error {
	for c := range commitCh {
		if c == nil {
			// signaled to load snapshot
			snapshot, err := r.loadSnapshot()
			if err != nil {
				log.Fatal(err)
			}
			if snapshot != nil {
				log.Infof("loading snapshot at term %d and index %d", snapshot.Metadata.Term, snapshot.Metadata.Index)
				if err = r.recoverFromSnapshot(snapshot.Data); err != nil {
					log.Fatal(err)
				}
			}
			continue
		}

		for _, data := range c.data {
			entry := new(apicassemdb.LogEntry)
			apicassemdb.MustUnmarshal(runtime.ToBytes(data), entry)

			log.Debug("raftNodeImpl.applyCommits recv one logEntry")

			switch entry.Action {
			case apicassemdb.LogEntry_Set:
				cmd := new(apicassemdb.SetCommand)
				apicassemdb.MustUnmarshal(entry.Command, cmd)
				if cmd.SetKey != "" {
					if err := r.kvstore.SetKV(cmd.GetSetKey(), cmd.GetValue(), cmd.GetIsDir()); err != nil {
						log.
							WithFields(log.Fields{"cmd": cmd, "error": err}).
							Error("raftNodeImpl.applyCommits failed to SetKV")
					}
				}
				if cmd.DeleteKey != "" {
					if err := r.kvstore.UnsetKV(cmd.GetDeleteKey(), cmd.GetIsDir()); err != nil {
						log.
							WithFields(log.Fields{"cmd": cmd, "error": err}).
							Error("raftNodeImpl.applyCommits failed to SetKV")
					}
				}

			case apicassemdb.LogEntry_ChangeSpread:
				if entry.Expired() {
					// skip change log entry.
					continue
				}
				cmd := new(apicassemdb.ChangeCommand)
				apicassemdb.MustUnmarshal(entry.Command, cmd)
				change := cmd.GetChange()
				select {
				case r.changeC <- change:
					paths, _ := storage.KeySplitter(change.GetKey())
					if len(paths) == 0 {
						break
					}
					parentDirectoryChange := &apicassemdb.ParentDirectoryChange{
						Change:        change,
						SpecificTopic: strings.Join(paths, "/"),
					}
					select {
					case r.changeC <- parentDirectoryChange:
					default:
					}
				default:
				}
			}
		}
		close(c.applyDoneC)
	}

	var (
		err error
		ok  bool
	)
	if err, ok = <-errorCh; ok {
		return err
	}

	return nil
}

func (r *raftNodeImpl) getSnapshot() ([]byte, error) {
	return []byte("empty snapshot"), nil
}

func (r *raftNodeImpl) loadSnapshot() (*raftpb.Snapshot, error) {
	snapshot, err := r.snapshotter.Load()
	if err == snap.ErrNoSnapshot {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return snapshot, nil
}

func (r *raftNodeImpl) recoverFromSnapshot(snapshot []byte) error {
	return nil
}

type LogEntryCommand interface {
	proto.Message

	Action() apicassemdb.LogEntry_Action
}

// propose log to commit
func (r *raftNodeImpl) propose(cmd LogEntryCommand) error {
	if cmd == nil {
		return errors.New("empty logEntryCommand")
	}

	entry := &apicassemdb.LogEntry{
		Action:    cmd.Action(),
		Command:   apicassemdb.Must(apicassemdb.Marshal(cmd)),
		CreatedAt: time.Now().Unix(),
	}
	errC := make(chan error)
	r.proposeC <- apicassemdb.NewPropose(entry, errC)
	log.WithFields(log.Fields{"entry": entry}).Debug("raftNodeImpl.propose done")
	return <-errC
}

// SetKV set a KV or directory into db storage with other parameters.
// isDir parameter indicates key means a kv or directory, if it's ture val will be ignored,
// overwrite indicates the operation MUST BE failed if key exists with storage.ErrExists,
// ttl means Time To Live, which will only be stored in file and recalculated in memory to use.
func (r *raftNodeImpl) SetKV(req *apicassemdb.SetKVReq) (err error) {
	log.
		WithFields(log.Fields{
			"key":       req.GetKey(),
			"val":       runtime.ToString(req.GetVal()),
			"isDir":     req.GetIsDir(),
			"overwrite": req.GetOverwrite(),
			"ttl":       req.GetTtl(),
		}).
		Debug("raftNodeImpl.setKV called")

	// get preview value
	last, err := r.kvstore.GetKV(req.GetKey(), req.GetIsDir())
	if err != nil {
		log.
			WithFields(log.Fields{
				"key":   req.GetKey(),
				"error": err.Error(),
			}).
			Warn("raftNodeImpl.SetKV could to load last value of key")
	}

	// remove expired value automatically.
	if r.probeRemoveExpired(last) {
		last = nil
	}

	if !req.GetOverwrite() && last != nil {
		return storage.ErrExists
	}

	var createdAt = time.Now().Unix()
	if last != nil && !last.Expired() {
		createdAt = last.CreatedAt
	}

	v := apicassemdb.NewEntityWithCreated(req.GetKey(), req.GetVal(), req.GetTtl(), createdAt)
	if err = r.propose(&apicassemdb.SetCommand{
		DeleteKey: "",
		IsDir:     false,
		SetKey:    req.GetKey(),
		Value:     v,
	}); err != nil {
		return errors.Wrap(err, "raftNodeImpl.SetKV")
	}

	// touch off change signal to cassemdb cluster.
	r.triggerWatchingMechanism(apicassemdb.Change_Set, req.GetKey(), last, v)

	return nil
}

func (r *raftNodeImpl) UnsetKV(req *apicassemdb.UnsetKVReq) error {
	last, err := r.kvstore.GetKV(req.GetKey(), req.GetIsDir())
	if err != nil {
		log.
			WithFields(log.Fields{
				"req":   req,
				"error": err,
			}).
			Warn("raftNodeImpl.triggerWatchingMechanism could to load last value of key")
	}

	if err = r.propose(&apicassemdb.SetCommand{
		DeleteKey: req.GetKey(),
		IsDir:     req.GetIsDir(),
		SetKey:    "",
		Value:     nil,
	}); err != nil {
		return errors.Wrap(err, "raftNodeImpl.UnsetKV")
	}

	// touch off change signal to cassemdb cluster.
	r.triggerWatchingMechanism(apicassemdb.Change_Unset, req.GetKey(), last, nil)

	return nil
}

// triggerWatchingMechanism only trigger a change notification while:
// 1. delete a kv.
// 2. really update an existed kv.
//
func (r *raftNodeImpl) triggerWatchingMechanism(op apicassemdb.Change_Op, key string, last, cur *apicassemdb.Entity) {
	log.
		WithFields(log.Fields{
			"key": key,
			"op":  op,
		}).
		Debug("raftNodeImpl.triggerWatchingMechanism called")

	// FIXED(@yeqown): new value should also notify watchers.
	//if last == nil || last.Expired() {
	//	// last == nil means that the key is new, there's no observer;
	//	return
	//}

	if last != nil && cur != nil && strings.Compare(last.Fingerprint, cur.Fingerprint) == 0 {
		// set kv but cur is same to old value, so no need to touch off a change notification.
		return
	}

	go func() {
		log.
			WithFields(log.Fields{"key": key, "cur": cur}).
			Debug("raftNodeImpl.triggerWatchingMechanism called")

		if err := r.propose(&apicassemdb.ChangeCommand{
			Change: &apicassemdb.Change{
				Op:      op,
				Key:     key,
				Last:    last,
				Current: cur,
			}}); err != nil {
			log.
				WithFields(log.Fields{
					"key": key,
					"cur": cur,
				}).
				Error("raftNodeImpl.triggerWatchingMechanism called")
		}
	}()
}

func (r *raftNodeImpl) GetKV(req *apicassemdb.GetKVReq) (*apicassemdb.Entity, error) {
	val, err := r.kvstore.GetKV(req.GetKey(), false)
	if err != nil {
		log.
			WithFields(log.Fields{
				"req":   req,
				"error": err,
			}).
			Error("kvstore.getKV failed")
		return nil, err
	}

	if r.probeRemoveExpired(val) {
		return nil, storage.ErrNotFound
	}

	return val, nil
}

// probeRemoveExpired returns true while val.Expired() is true.
func (r *raftNodeImpl) probeRemoveExpired(val *apicassemdb.Entity) (removed bool) {
	if val == nil {
		return false
	}

	if val.Expired() {
		if err := r.UnsetKV(&apicassemdb.UnsetKVReq{Key: val.GetKey()}); err != nil {
			log.
				WithFields(log.Fields{"key": val.Key, "error": err}).
				Error("kvstore.GetKV failed to remove expired key")
		}
		return true
	}

	return false
}

var (
	emptyRangeResp = &apicassemdb.RangeResp{
		Entities:    make([]*apicassemdb.Entity, 0, 0),
		HasMore:     false,
		NextSeekKey: "",
	}
)

func (r *raftNodeImpl) Range(req *apicassemdb.RangeReq) (*apicassemdb.RangeResp, error) {
	// DONE(@yeqown): return expired keys and trigger probeRemoveExpired methods
	result, err := r.kvstore.Range(req.GetKey(), req.GetSeek(), int(req.GetLimit()))
	if err != nil {
		if errors.Is(err, errorx.Err_NOT_FOUND) {
			return emptyRangeResp, nil
		}

		return nil, errors.Wrap(err, "raftNodeImpl.Range")
	}

	if len(result.ExpiredKeys) != 0 {
		// DONE(@yeqown): delete the expired keys while got expired keys.
		go func() {
			log.
				WithFields(log.Fields{
					"keys": result.ExpiredKeys,
				}).
				Debug("raftNodeImpl.Range trigger remove expired keys")

			for _, k := range result.ExpiredKeys {
				_ = r.UnsetKV(&apicassemdb.UnsetKVReq{Key: k})
			}
		}()
	}

	resp := &apicassemdb.RangeResp{
		Entities:    make([]*apicassemdb.Entity, 0, len(result.Items)),
		HasMore:     result.HasMore,
		NextSeekKey: result.NextSeekKey,
	}

	for _, v := range result.Items {
		resp.Entities = append(resp.Entities, v)
	}

	return resp, err
}

func (r *raftNodeImpl) Expire(req *apicassemdb.ExpireReq) error {
	v, err := r.kvstore.GetKV(req.GetKey(), false)
	if err != nil {
		if errors.Is(err, errorx.Err_NOT_FOUND) {
			return nil
		}

		return errors.Wrap(err, "cassemdb.raftNodeImpl.Expire")
	}

	switch v.GetTtl() {
	case apicassemdb.NEVER_EXPIRED:
		return nil
	}

	// unset the key value directly or update it's TTL, choose update it's TTL
	// so that the expiry(expire) operation is same to method's meaning.
	return r.SetKV(&apicassemdb.SetKVReq{
		Key:       req.GetKey(),
		IsDir:     false,
		Ttl:       apicassemdb.EXPIRED,
		Val:       v.Val,
		Overwrite: true,
	})
}

func (r *raftNodeImpl) IsLeader() bool {
	return r.raftState == raft.StateLeader
}

func (r *raftNodeImpl) LeaderChangeCh(c chan<- bool) {
	// lock
	r.mu.Lock()
	defer r.mu.Unlock()

	r.leadershipFanOutC = append(r.leadershipFanOutC, c)
}

func (r *raftNodeImpl) ChangeNotifyCh() <-chan watcher.IChange {
	return r.changeC
}
