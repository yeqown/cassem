package etcdio

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"sync"
	"sync/atomic"
	"time"

	"github.com/pkg/errors"
	"github.com/yeqown/log"
	"go.etcd.io/etcd/client/pkg/v3/fileutil"
	"go.etcd.io/etcd/client/pkg/v3/types"
	"go.etcd.io/etcd/raft/v3"
	"go.etcd.io/etcd/raft/v3/raftpb"
	"go.etcd.io/etcd/server/v3/etcdserver/api/rafthttp"
	"go.etcd.io/etcd/server/v3/etcdserver/api/snap"
	stats "go.etcd.io/etcd/server/v3/etcdserver/api/v2stats"
	"go.etcd.io/etcd/server/v3/wal"
	"go.etcd.io/etcd/server/v3/wal/walpb"
	"go.uber.org/zap"

	apicassemdb "github.com/yeqown/cassem/internal/cassemdb/api"
	"github.com/yeqown/cassem/pkg/retry"
	"github.com/yeqown/cassem/pkg/runtime"
)

type commit struct {
	data       []string
	applyDoneC chan<- struct{}
}

// A key-value stream backed by raft
type raftNode struct {
	proposeC    <-chan *apicassemdb.Propose // proposed messages (k,v)
	commitC     chan<- *commit              // entries committed to log (k,v)
	errorC      chan<- error                // errors from raft session
	leadershipC chan<- raft.StateType

	confChangeCount uint64
	id              uint64       // client ID for raft session
	bindAddress     string       // address of node [raft].
	peers           []string     // raft peer URLs
	muPeers         sync.RWMutex // mutex protects peers read and write access.
	join            bool         // node is joining an existing cluster
	waldir          string       // path to WAL directory
	snapdir         string       // path to snapshot directory
	getSnapshot     func() ([]byte, error)

	confState     raftpb.ConfState
	snapshotIndex uint64
	appliedIndex  uint64

	// raft backing for the commit/error channel
	node        raft.Node
	raftStorage *raft.MemoryStorage

	wal *wal.WAL

	snapshotter      *snap.Snapshotter
	snapshotterReady chan *snap.Snapshotter // signals when snapshotter is ready

	snapCount uint64
	transport *rafthttp.Transport
	stopc     chan struct{} // signals proposal channel closed
	httpstopc chan struct{} // signals http server to shutdown
	httpdonec chan struct{} // signals http server shutdown complete

	logger *zap.Logger

	proposalOnce sync.Once
}

var defaultSnapshotCount uint64 = 10000

type config struct {
	id          uint64
	peers       []string
	bindAddress string
	join        bool
	baseDir     string
	snapCount   uint64
	getSnapshot func() ([]byte, error)
}

type peerOperator interface {
	addPeer(peer string) (nodeID uint64, peers []string, err error)
	removePeer(nodeID uint64) error
	getPeers() []string
}

// newRaftNode initiates a raft instance and returns a committed log entry
// channel and error channel. Proposals for log updates are sent over the
// provided the proposal channel. All log entries are replayed over the
// commit channel, followed by a nil message (to indicate the channel is
// current), then new log entries. To shut it down, close proposeC and read errorC.
func newRaftNode(
	cfg *config,
	proposeC <-chan *apicassemdb.Propose,
	leadershipC chan<- raft.StateType,
) (<-chan *commit, <-chan error, <-chan *snap.Snapshotter, peerOperator) {

	commitC := make(chan *commit)
	errorC := make(chan error)

	if cfg.snapCount == 0 {
		cfg.snapCount = defaultSnapshotCount
	}

	rc := &raftNode{
		proposeC:    proposeC,
		commitC:     commitC,
		errorC:      errorC,
		leadershipC: leadershipC,
		id:          cfg.id,
		bindAddress: cfg.bindAddress,
		peers:       cfg.peers,
		join:        cfg.join,
		waldir:      fmt.Sprintf("%s/wal", cfg.baseDir),
		snapdir:     fmt.Sprintf("%s/snap", cfg.baseDir),
		getSnapshot: cfg.getSnapshot,
		snapCount:   cfg.snapCount,
		stopc:       make(chan struct{}),
		httpstopc:   make(chan struct{}),
		httpdonec:   make(chan struct{}),

		snapshotterReady: make(chan *snap.Snapshotter, 1),
		// rest of structure populated after WAL replay
		proposalOnce: sync.Once{},
	}
	go rc.startRaft()
	return commitC, errorC, rc.snapshotterReady, rc
}

func (rc *raftNode) saveSnap(snap raftpb.Snapshot) error {
	log.Debug("raftNode.saveSnap called")
	walSnap := walpb.Snapshot{
		Index:     snap.Metadata.Index,
		Term:      snap.Metadata.Term,
		ConfState: &snap.Metadata.ConfState,
	}
	// save the snapshot file before writing the snapshot to the wal.
	// This makes it possible for the snapshot file to become orphaned, but prevents
	// a WAL snapshot entry from having no corresponding snapshot file.
	if err := rc.snapshotter.SaveSnap(snap); err != nil {
		return err
	}
	if err := rc.wal.SaveSnapshot(walSnap); err != nil {
		return err
	}
	return rc.wal.ReleaseLockTo(snap.Metadata.Index)
}

func (rc *raftNode) entriesToApply(ents []raftpb.Entry) (nents []raftpb.Entry) {
	if len(ents) == 0 {
		return ents
	}
	firstIdx := ents[0].Index
	if firstIdx > rc.appliedIndex+1 {
		log.Fatalf("first index of committed entry[%d] should <= progress.appliedIndex[%d]+1", firstIdx, rc.appliedIndex)
	}
	if rc.appliedIndex-firstIdx+1 < uint64(len(ents)) {
		nents = ents[rc.appliedIndex-firstIdx+1:]
	}
	return nents
}

// publishEntries writes committed log entries to commit channel and returns
// whether all entries could be published.
func (rc *raftNode) publishEntries(ents []raftpb.Entry) (<-chan struct{}, bool) {
	if len(ents) == 0 {
		return nil, true
	}

	data := make([]string, 0, len(ents))
	for i := range ents {
		switch ents[i].Type {
		case raftpb.EntryNormal:
			if len(ents[i].Data) == 0 {
				// ignore empty messages
				break
			}
			s := string(ents[i].Data)
			data = append(data, s)
		case raftpb.EntryConfChange:
			var cc raftpb.ConfChange
			_ = cc.Unmarshal(ents[i].Data)
			rc.confState = *rc.node.ApplyConfChange(cc)
			switch cc.Type {
			case raftpb.ConfChangeAddNode:
				if len(cc.Context) > 0 {
					rc.transport.AddPeer(types.ID(cc.NodeID), []string{string(cc.Context)})
				}
			case raftpb.ConfChangeRemoveNode:
				if cc.NodeID == rc.id {
					log.Info("I've been removed from the cluster! Shutting down.")
					return nil, false
				}
				rc.transport.RemovePeer(types.ID(cc.NodeID))
			}
		}
	}

	var applyDoneC chan struct{}

	if len(data) > 0 {
		applyDoneC = make(chan struct{}, 1)
		select {
		case rc.commitC <- &commit{data, applyDoneC}:
		case <-rc.stopc:
			return nil, false
		}
	}

	// after commit, update appliedIndex
	rc.appliedIndex = ents[len(ents)-1].Index

	return applyDoneC, true
}

func (rc *raftNode) loadSnapshot() *raftpb.Snapshot {
	if wal.Exist(rc.waldir) {
		walSnaps, err := wal.ValidSnapshotEntries(rc.logger, rc.waldir)
		if err != nil {
			log.Fatalf("raftNode: error listing snapshots (%v)", err)
		}
		snapshot, err := rc.snapshotter.LoadNewestAvailable(walSnaps)
		if err != nil && err != snap.ErrNoSnapshot {
			log.Fatalf("raftNode: error loading snapshot (%v)", err)
		}
		return snapshot
	}
	return &raftpb.Snapshot{}
}

// openWAL returns a WAL ready for reading.
func (rc *raftNode) openWAL(snapshot *raftpb.Snapshot) *wal.WAL {
	if !wal.Exist(rc.waldir) {
		if err := os.Mkdir(rc.waldir, 0750); err != nil {
			log.Fatalf("raftNode: cannot create dir for wal (%v)", err)
		}

		w, err := wal.Create(zap.NewExample(), rc.waldir, nil)
		if err != nil {
			log.Fatalf("raftNode: create wal error (%v)", err)
		}
		_ = w.Close()
	}

	walsnap := walpb.Snapshot{}
	if snapshot != nil {
		walsnap.Index, walsnap.Term = snapshot.Metadata.Index, snapshot.Metadata.Term
	}
	log.Infof("loading WAL at term %d and index %d", walsnap.Term, walsnap.Index)
	w, err := wal.Open(zap.NewExample(), rc.waldir, walsnap)
	if err != nil {
		log.Fatalf("raftNode: error loading wal (%v)", err)
	}

	return w
}

// replayWAL replays WAL entries into the raft instance.
func (rc *raftNode) replayWAL() *wal.WAL {
	log.Infof("replaying WAL of member %d", rc.id)
	snapshot := rc.loadSnapshot()
	w := rc.openWAL(snapshot)
	_, st, ents, err := w.ReadAll()
	if err != nil {
		log.Fatalf("raftNode: failed to read WAL (%v)", err)
	}
	rc.raftStorage = raft.NewMemoryStorage()
	if snapshot != nil {
		_ = rc.raftStorage.ApplySnapshot(*snapshot)
	}
	_ = rc.raftStorage.SetHardState(st)

	// append to storage so raft starts at the right place in log
	_ = rc.raftStorage.Append(ents)

	return w
}

func (rc *raftNode) writeError(err error) {
	rc.stopHTTP()
	close(rc.commitC)
	rc.errorC <- err
	close(rc.errorC)
	rc.node.Stop()
}

func (rc *raftNode) startRaft() {
	if !fileutil.Exist(rc.snapdir) {
		if err := os.Mkdir(rc.snapdir, 0750); err != nil {
			log.Fatalf("raftNode: cannot create dir for snapshot (%v)", err)
		}
	}
	rc.snapshotter = snap.New(zap.NewExample(), rc.snapdir)
	oldwal := wal.Exist(rc.waldir)
	rc.wal = rc.replayWAL()
	// signal replay has finished
	rc.snapshotterReady <- rc.snapshotter

	c := &raft.Config{
		ID:                        rc.id,
		ElectionTick:              10,
		HeartbeatTick:             1,
		Storage:                   rc.raftStorage,
		MaxSizePerMsg:             1024 * 1024,
		MaxInflightMsgs:           256,
		MaxUncommittedEntriesSize: 1 << 30,
	}

	log.
		WithFields(log.Fields{
			"oldwal":                  oldwal,
			"join":                    rc.join,
			"isRestart(oldwal||join)": oldwal || rc.join,
		}).
		Debug("startRaft")

	if oldwal || rc.join {
		rc.node = raft.RestartNode(c)
	} else {
		rpeers := make([]raft.Peer, len(rc.peers))
		for i := range rpeers {
			rpeers[i] = raft.Peer{ID: uint64(i + 1)}
		}
		rc.node = raft.StartNode(c, rpeers)
	}

	rc.transport = &rafthttp.Transport{
		Logger:      rc.logger,
		ID:          types.ID(rc.id),
		ClusterID:   0x1000,
		Raft:        rc,
		ServerStats: stats.NewServerStats("", ""),
		LeaderStats: stats.NewLeaderStats(zap.NewExample(), strconv.FormatUint(rc.id, 10)),
		ErrorC:      make(chan error),
	}
	_ = rc.transport.Start()
	for i := range rc.peers {
		if uint64(i+1) != rc.id {
			rc.transport.AddPeer(types.ID(i+1), []string{rc.peers[i]})
		}
	}

	runtime.GoFunc("serveRaft", func() error {
		r := retry.DefaultExponential()
		return r.Do(context.TODO(), rc.serveRaft)
	})
	runtime.GoFunc("serveChannels", func() error { rc.serveChannels(); return nil })

	log.Debug("raftNode.startRaft setup done, starting raft")
}

// stop closes http, closes all channels, and stops raft.
func (rc *raftNode) stop() {
	rc.stopHTTP()
	close(rc.commitC)
	close(rc.errorC)
	rc.node.Stop()
}

func (rc *raftNode) stopHTTP() {
	rc.transport.Stop()
	close(rc.httpstopc)
	<-rc.httpdonec
}

func (rc *raftNode) publishSnapshot(snapshotToSave raftpb.Snapshot) {
	if raft.IsEmptySnap(snapshotToSave) {
		return
	}

	log.Infof("publishing snapshot at index %d", rc.snapshotIndex)
	defer log.Infof("finished publishing snapshot at index %d", rc.snapshotIndex)

	if snapshotToSave.Metadata.Index <= rc.appliedIndex {
		log.Fatalf("snapshot index [%d] should > progress.appliedIndex [%d]", snapshotToSave.Metadata.Index, rc.appliedIndex)
	}
	rc.commitC <- nil // trigger store to load snapshot

	rc.confState = snapshotToSave.Metadata.ConfState
	rc.snapshotIndex = snapshotToSave.Metadata.Index
	rc.appliedIndex = snapshotToSave.Metadata.Index
}

var snapshotCatchUpEntriesN uint64 = 10000

func (rc *raftNode) maybeTriggerSnapshot(applyDoneC <-chan struct{}) {
	if rc.appliedIndex-rc.snapshotIndex <= rc.snapCount {
		return
	}

	// wait until all committed entries are applied (or server is closed)
	if applyDoneC != nil {
		select {
		case <-applyDoneC:
		case <-rc.stopc:
			return
		}
	}

	log.Infof("start snapshot [applied index: %d | last snapshot index: %d]", rc.appliedIndex, rc.snapshotIndex)
	data, err := rc.getSnapshot()
	if err != nil {
		log.Fatal(err)
	}
	snapshot, err := rc.raftStorage.CreateSnapshot(rc.appliedIndex, &rc.confState, data)
	if err != nil {
		// panic(err)
		log.Error(err)
	}
	if err := rc.saveSnap(snapshot); err != nil {
		panic(err)
	}

	compactIndex := uint64(1)
	if rc.appliedIndex > snapshotCatchUpEntriesN {
		compactIndex = rc.appliedIndex - snapshotCatchUpEntriesN
	}
	if err := rc.raftStorage.Compact(compactIndex); err != nil {
		//panic(err)
		log.Error(err)
	}

	log.Infof("compacted log at index %d", compactIndex)
	rc.snapshotIndex = rc.appliedIndex
}

func (rc *raftNode) serveChannels() {
	snapshot, err := rc.raftStorage.Snapshot()
	if err != nil {
		panic(err)
	}
	rc.confState = snapshot.Metadata.ConfState
	rc.snapshotIndex = snapshot.Metadata.Index
	rc.appliedIndex = snapshot.Metadata.Index

	defer rc.wal.Close()

	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()

	rc.proposalOnce.Do(func() {
		// send proposals over raft
		runtime.GoFunc("sendProposals", func() error {
			for rc.proposeC != nil {
				select {
				case prop, ok := <-rc.proposeC:
					log.
						WithFields(log.Fields{"ok": ok, "propose": prop}).
						Debug("raftNode.serveChannels proposeC called")
					if !ok {
						rc.proposeC = nil
					} else {
						// blocks until accepted by raft state machine
						// DONE(@yeqown): return error to request (synchronized?)
						prop.ErrC <- rc.node.Propose(context.TODO(), apicassemdb.Must(apicassemdb.Marshal(prop.Entry)))
					}
				}
			}
			// client closed channel; shutdown raft if not already
			close(rc.stopc)

			return nil
		})
	})

	// event loop on raft state machine updates
	for {
		select {
		case <-ticker.C:
			rc.node.Tick()

		// store raft entries to wal, then publish over commit channel
		case rd := <-rc.node.Ready():
			if rd.SoftState != nil {
				log.
					WithFields(log.Fields{"rc.raftState": rd.RaftState}).
					Debug("membership changed")
				rc.leadershipC <- rd.RaftState
			}

			_ = rc.wal.Save(rd.HardState, rd.Entries)
			if !raft.IsEmptySnap(rd.Snapshot) {
				_ = rc.saveSnap(rd.Snapshot)
				_ = rc.raftStorage.ApplySnapshot(rd.Snapshot)
				rc.publishSnapshot(rd.Snapshot)
			}
			_ = rc.raftStorage.Append(rd.Entries)
			rc.transport.Send(rd.Messages)
			applyDoneC, ok := rc.publishEntries(rc.entriesToApply(rd.CommittedEntries))
			if !ok {
				rc.stop()
				return
			}
			rc.maybeTriggerSnapshot(applyDoneC)
			rc.node.Advance()

		case err2 := <-rc.transport.ErrorC:
			rc.writeError(err2)
			return

		case <-rc.stopc:
			rc.stop()
			return
		}
	}
}

func (rc *raftNode) serveRaft() (err error) {
	defer func() {
		if err != nil {
			log.Errorf("raftNode.serveRaft() error: %v", err)
		}
	}()
	log.Info("serving raft called")

	var u *url.URL
	u, err = url.Parse(rc.bindAddress)
	if err != nil {
		err = errors.Wrap(err, "raftNode: Failed parsing URL")
		return err
	}

	var ln *stoppableListener
	ln, err = newStoppableListener(u.Host, rc.httpstopc)
	if err != nil {
		err = errors.Wrap(err, "raftNode: Failed to listen rafthttp")
		return err
	}

	err = (&http.Server{
		Handler: rc.transport.Handler(),
	}).Serve(ln)

	select {
	case <-rc.httpstopc:
	default:
		err = errors.Wrap(err, "raftNode: Failed to serve rafthttp")
	}
	close(rc.httpdonec)

	return err
}

func (rc *raftNode) addPeer(peer string) (nodeID uint64, peers []string, err error) {
	log.
		WithFields(log.Fields{
			"addr":    peer,
			"current": rc.peers,
		}).
		Infof("raftNodeImpl.AddNode received a request from remote node")

	tryAdd := func(peer string) (tryNodeID uint64, tryPeers []string) {
		rc.muPeers.Lock()
		for idx, p := range rc.peers {
			// ignore duplicate peer
			if peer != p {
				continue
			}

			// hit duplicated
			tryNodeID = uint64(idx + 1)
			tryPeers = rc.peers
			rc.muPeers.Unlock()
			return
		}

		// not found in before peers
		rc.peers = append(rc.peers, peer)
		tryPeers = rc.peers
		tryNodeID = uint64(len(rc.peers))
		rc.muPeers.Unlock()

		return
	}

	rollback := func(idx uint64) {
		rc.muPeers.Lock()
		rc.peers = append(rc.peers[:idx], rc.peers[idx+1:]...)
		rc.muPeers.RLock()
	}

	nodeID, peers = tryAdd(peer)
	if err = rc.changeConf(raftpb.ConfChange{
		Type:    raftpb.ConfChangeAddNode,
		NodeID:  nodeID,
		Context: []byte(peer),
		ID:      0,
	}); err != nil {
		rollback(nodeID)
		err = errors.Wrap(err, "raftNode failed to add peer")
	}

	return
}

func (rc *raftNode) removePeer(nodeID uint64) error {
	rc.muPeers.RUnlock()
	length := len(rc.peers)
	rc.muPeers.RUnlock()

	if length <= int(nodeID) {
		return errors.New("invalid nodeID")
	}

	// TODO(@yeqown): remove peer from peers.

	log.
		WithFields(log.Fields{
			"nodeID":   nodeID,
			"nodeAddr": rc.peers[nodeID],
			"current":  rc.peers,
		}).
		Infof("raftNodeImpl.RemoveNode received a request from remote node")

	if err := rc.changeConf(raftpb.ConfChange{
		Type:    raftpb.ConfChangeRemoveNode,
		NodeID:  nodeID,
		Context: nil,
		ID:      0,
	}); err != nil {
		return errors.Wrap(err, "raftNode failed to remove peer")

	}

	return nil
}

func (rc *raftNode) getPeers() []string {
	rc.muPeers.RLock()
	peers := rc.peers
	rc.muPeers.RUnlock()
	return peers
}

func (rc *raftNode) changeConf(cc raftpb.ConfChange) error {
	cc.ID = atomic.AddUint64(&rc.confChangeCount, 1)
	if err := rc.node.ProposeConfChange(context.TODO(), cc); err != nil {
		return errors.Wrap(err, "raftNode failed to changeConf")
	}

	return nil
}

func (rc *raftNode) Process(ctx context.Context, m raftpb.Message) error {
	return rc.node.Step(ctx, m)
}
func (rc *raftNode) IsIDRemoved(id uint64) bool  { return false }
func (rc *raftNode) ReportUnreachable(id uint64) { rc.node.ReportUnreachable(id) }
func (rc *raftNode) ReportSnapshot(id uint64, status raft.SnapshotStatus) {
	rc.node.ReportSnapshot(id, status)
}
