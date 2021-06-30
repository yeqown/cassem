package infras

import (
	"fmt"
	"net"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/yeqown/cassem/pkg/conf"
	"github.com/yeqown/cassem/pkg/runtime"
	"github.com/yeqown/cassem/pkg/types"
	"github.com/yeqown/cassem/pkg/watcher"

	"github.com/hashicorp/raft"
	raftboltdb "github.com/hashicorp/raft-boltdb"
	"github.com/pkg/errors"
	"github.com/yeqown/log"
)

var ErrNotLeader = errors.New("current node is not allow to write, should not be triggered normally")

// IMyRaft defines the ability of what raft component should act.
// TODO(@yeqown): implement a store interface for replication.
type IMyRaft interface {
	// GetKV get value of key
	GetKV(key string) (*types.StoreValue, error)
	// SetKV save key and value
	SetKV(key string, value []byte) error
	// UnsetKV save key and value
	UnsetKV(key string) error

	GetLeaderAddr() string // GetLeaderAddr

	Shutdown() error                    // Shutdown
	IsLeader() bool                     // IsLeader
	JoinedCluster() bool                // JoinedCluster
	RemoveNode(serveId string) error    // RemoveNode
	AddNode(serveId, addr string) error // AddNode
	ApplyRaw(data []byte) error         // ApplyRaw

	ChangeNotifyCh() <-chan watcher.IChange
}

type Conf struct {
	HTTP        *conf.HTTP
	Raft        *conf.Raft
	Persistence *conf.BBolt
}

type myraft struct {
	*raft.Raft

	conf *Conf

	repo Repository

	// fsm is the state machine to be used in raft.RAFT. In cassem, it's mainly used to store
	// caches those encoded bytes to containers who are requested and should be cached.
	//
	// It also be used to store and apply leaderAddr which indicates the address of the leader.
	// While a leadership changes happened, leader node calls raft.Apply() to commit a log that
	// will update slave nodes' leaderAddr. please checkout FSMWrapper for more information.
	fsm myFSM

	// The following properties are all about raft consensus algorithm.
	//
	// serverId indicates the unique identify in raft cluster,
	// also used to as the raft store directory name.
	serverId string

	// tryJoinIdx is a index to memory which address should be tried when
	// next tryJoinCluster called. This property helps to support that multiple
	// addresses of cluster nodes can be passed.
	tryJoinIdx int

	// joinedCluster indicates whether current node has joined to the cluster.
	// If current node is started as leader node, joinedCluster is true as default, otherwise
	// joinedCluster controls tryJoinCluster again and again in myraft.heartbeat.
	joinedCluster bool

	// changeCh
	changeCh chan watcher.IChange
}

func NewMyRaft(conf *Conf) (IMyRaft, error) {
	r := &myraft{
		Raft:          nil,
		conf:          conf,
		repo:          nil,
		fsm:           nil,
		serverId:      "",
		tryJoinIdx:    0,
		joinedCluster: false,
		changeCh:      make(chan watcher.IChange, _SIZE_CHANGE_BUF),
	}

	err := r.bootstrap()

	if err == nil {
		// leadership changes
		runtime.GoFunc("leadership-changes", r.watchLeaderChanges)

		// snapshot executor
		runtime.GoFunc("snapshot-strategy", r.doSnapshot)
	}

	return r, err
}

// FIXME(@yeqown): could not bootstrap the cluster after leadership changed or
// config `server.raft.join` is a wrong node.
func (r *myraft) bootstrap() (err error) {
	defer func() {
		if err != nil {
			log.
				WithFields(log.Fields{
					"raftConfig": r.conf.Raft,
					"joined":     r.joinedCluster,
					"serverId":   r.serverId,
				}).
				Errorf("myraft.bootstrapRaft failed: %v", err)
		}
	}()

	// prepare transport
	raftConf := r.conf.Raft
	r.serverId = raftConf.ServerId
	addr, err := net.ResolveTCPAddr("tcp", raftConf.RaftBind)
	if err != nil {
		return errors.Wrap(err, "ResolveTCPAddr failed")
	}
	transport, err := raft.NewTCPTransport(raftConf.RaftBind, addr, 3, 10*time.Second, os.Stderr)
	if err != nil {
		return errors.Wrap(err, "NewTCPTransport failed")
	}

	// prepare snapshot store
	snapshotStore, err := raft.NewFileSnapshotStore(raftConf.RaftBase, 2, os.Stderr)
	if err != nil {
		return errors.Wrap(err, "NewFileSnapshotStore failed")
	}

	// boltStore implement log store and stable store interface
	boltStore, err := raftboltdb.NewBoltStore(filepath.Join(raftConf.RaftBase, "raft.db"))
	if err != nil {
		return errors.Wrap(err, "NewBoltStore failed")
	}

	// fsm loading
	r.repo, err = newRepository(r.conf.Persistence)
	if err != nil {
		return errors.Wrap(err, "bbolt.NewMyRaft failed")
	}
	log.Debug("myraft.repo loaded")

	r.fsm = newFSM(r.repo, r.changeCh)
	log.Debug("myraft.fsm loaded")

	// construct raft system
	config := raft.DefaultConfig()
	config.LocalID = raft.ServerID(r.serverId)
	config.SnapshotThreshold = 1024

	if r.Raft, err = raft.NewRaft(config, r.fsm, boltStore, boltStore, snapshotStore, transport); err != nil {
		return errors.Wrap(err, "raft.NewRaft failed")
	}

	// DONE(@yeqown): allow node to bootstrap every time (IGNORING error), but only to join cluster when
	// raftConf.ClusterAddresses is not empty and raftConf.ClusterToJoin != raftConf.Listen
	var couldIgnore error

	shouldBootstrapCluster := len(raftConf.ClusterAddresses) == 0
	r.joinedCluster = true

	log.
		WithFields(log.Fields{
			"shouldBootstrapCluster": shouldBootstrapCluster,
		}).
		Debug("myraft.bootstrap called")
	if shouldBootstrapCluster {
		// A cluster can only be bootstrapped once from a single participating Voter server.
		// Any further attempts to bootstrap will return an error that can be safely ignored.
		if couldIgnore = r.BootstrapCluster(raft.Configuration{
			Servers: []raft.Server{
				{
					ID:      config.LocalID,
					Address: transport.LocalAddr(),
				},
			},
		}).Error(); couldIgnore != nil {
			log.Warnf("myraft.bootstrap could not BootstrapCluster: %v", couldIgnore)
		}
	} else {
		r.joinedCluster = false
		go func() {
			for !r.joinedCluster {
				// FIXED(@yeqown) could not return error, tryJoinCluster will retry again.
				if couldIgnore = r.tryJoinCluster(); couldIgnore != nil {
					log.Warnf("tryJoinCluster cluster failed: %v", couldIgnore)
					time.Sleep(3 * time.Second)
				} else {
					r.joinedCluster = true
				}
			}
		}()
	}

	return
}

// DONE(@yeqown): let node be notified while leader changes, and also mark current node is leader or not?
func (r myraft) watchLeaderChanges() error {
	isLeaderCh := r.LeaderCh()

	for isLeader := range isLeaderCh {

		log.
			WithField("toBeLeader", isLeader).
			Debug("myraft.watchLeaderChanges got a signal")

		// FIXED(@yeqown): reset leader address when leadership transition has occurred.
		r.fsm.setLeaderAddr("")

		if !isLeader {
			continue
		}

		// broadcast leader itself address to nodes.
		// DONE(@yeqown): should broadcast to other nodes of leaders
		fsmlog, _ := newFsmLog(actionSetLeader, &setLeaderCommand{
			LeaderAddr: r.conf.HTTP.Addr,
		})
		if err := r.propagateToSlaves(fsmlog); err != nil {
			log.
				WithFields(log.Fields{
					"addr": r.conf.HTTP.Addr,
					"log":  fsmlog,
				}).
				Errorf("myraft.watchLeaderChanges applyTo raft failed: %v", err)
		}
	}

	return nil
}

const (
	// _SIZE_EXECUTIONS is a value which limit the minimum count of logs
	// must be executed since last snapshot action.
	_SIZE_EXECUTIONS = 100

	// _SIZE_CHANGE_BUF
	_SIZE_CHANGE_BUF = 100

	_SNAPSHOT_INTERVAL = time.Minute
)

// DoSnapshot to execute snapshot of state machine with specified strategy:
//
// 1. just do snapshot periodically.
// 2. if state machine has executed logs more than specified size.
//
func (r myraft) doSnapshot() error {
	t1 := time.NewTicker(_SNAPSHOT_INTERVAL)
	t2 := time.NewTicker(10 * time.Second)

	var (
		needSnapshot bool
		reason       string
	)

	for {
		needSnapshot = false
		reason = "no condition reached(max_execution/time_interval)"
		select {
		case <-t1.C:
			needSnapshot = true
			reason = "time interval reached"
		case <-t2.C:
			if !r.IsLeader() {
				continue
			}

			if n := r.fsm.getExecutionSinceLastSnapshot(); n > _SIZE_EXECUTIONS {
				// if the state machine has received log over than 10
				// after last snapshot.
				reason = fmt.Sprintf("_SIZE_EXECUTIONS(%d) reached", _SIZE_EXECUTIONS)
				needSnapshot = true
			}
		}
		// case done

		log.
			WithFields(log.Fields{
				"needSnapshot": needSnapshot,
				"reason":       reason,
			}).
			Debug("myraft.doSnapshot called")

		if needSnapshot {
			if err := r.Snapshot().Error(); err != nil {
				log.Errorf("myraft.doSnapshot failed to snapshot: %v", err)
			}
		}
	}
}

func (r myraft) IsLeader() bool {
	return r.State() == raft.Leader
}

func (r myraft) JoinedCluster() bool {
	return r.joinedCluster
}

func (r myraft) Shutdown() error {
	if r.IsLeader() {
		// FIXED(@yeqown): if there is no more nodes in cluster, just let leader quit.
		if len(r.GetConfiguration().Configuration().Servers) < 2 {
			return nil
		}

		if err := r.RemoveNode(r.serverId); err != nil {
			log.
				Errorf("myraft.Shutdown (leader) could not remove from cluster: %v", err)
			return err
		}

		return nil
	}

	if err := r.tryLeaveCluster(); err != nil {
		log.
			Errorf("myraft.Shutdown (node) could not remove from cluster: %v", err)
		return err
	}

	return nil
}

const (
	_formServerId        = "serverId"
	_formAction          = "action"
	_formRaftBindAddress = "bind"

	_actionJoin = "join"
	_actionLeft = "left"
)

// tryJoinCluster only called by follower node, normally, conf.CassemdbConfig.Server.Raft.ClusterAddresses is
// the leader's address which is set manually. MAYBE~
func (r *myraft) tryJoinCluster() (err error) {
	var base string

	if count := len(r.conf.Raft.ClusterAddresses); count != 0 {
		base = r.conf.Raft.ClusterAddresses[r.tryJoinIdx]
		r.tryJoinIdx = (r.tryJoinIdx + 1) % count
	}

	log.
		WithFields(log.Fields{
			"base":             base,
			"clusterAddresses": r.conf.Raft.ClusterAddresses,
			"tryJoinIdx":       r.tryJoinIdx,
		}).
		Debug("myraft.tryJoinCluster called")

	if err = r.forwardToLeaderJoinLeft(_actionJoin, base); err != nil {
		log.
			Errorf("myraft.tryJoinCluster calling c.forwardToLeaderJoinLeft failed: %v", err)

		return errors.Wrap(err, "myraft.tryJoinCluster failed")
	}

	// FIXME(@yeqown): base maybe not leader, should get leader address from raft.
	// or forbid forwarding join request.
	r.fsm.setLeaderAddr(base)
	r.joinedCluster = true

	return
}

// tryLeaveCluster only called by follower node.
func (r myraft) tryLeaveCluster() (err error) {
	if err = r.forwardToLeaderJoinLeft(_actionLeft, ""); err != nil {
		log.
			Errorf("myraft.tryLeaveCluster calling c.forwardToLeaderJoinLeft failed: %v", err)

		return errors.Wrap(err, "myraft.tryLeaveCluster failed")
	}

	r.joinedCluster = false
	if err = r.Raft.Shutdown().Error(); err != nil {
		err = errors.Wrap(err, "myraft.tryLeaveCluster shutdown failed")
	}

	return
}

func (r myraft) ApplyRaw(data []byte) error {
	future := r.Apply(data, 10*time.Second)
	return future.Error()
}

// propagateToSlaves calls raft.Apply to distribute changes to FSM or propagate signal to all slaves.
// Importantly, DO NOT call r.raft.Apply directly, all operations those need to be propagated should call this.
// Only leader should call this.
func (r myraft) propagateToSlaves(fsmLog *fsmLog) (err error) {
	if !r.IsLeader() {
		if err = r.forwardToLeaderApply(fsmLog); err != nil {
			err = errors.Wrap(err, "myraft.propagateToSlaves failed to forwardToLeaderApply")
		}

		return
	}

	fsmLog.CreatedAt = time.Now().Unix()
	data, err := fsmLog.Serialize()
	if err != nil {
		return errors.Wrap(err, "myraft.propagateToSlaves failed to Serialize log")
	}

	future := r.Apply(data, 10*time.Second)
	if err = future.Error(); err != nil {
		return err
	}

	return nil
}

// RemoveNode only leader node would receive such request.
func (r myraft) RemoveNode(nodeID string) error {
	log.Infof("received RemoveNode request for remote node %s", nodeID)

	if !r.IsLeader() {
		log.
			Warn("RemoveNode request should not be executed by nonleader node")

		return ErrNotLeader
	}

	cf := r.GetConfiguration()
	if err := cf.Error(); err != nil {
		log.Errorf("failed to get raft configuration: %v", err)
		return err
	}

	for _, srv := range cf.Configuration().Servers {
		if srv.ID == raft.ServerID(nodeID) {
			f := r.RemoveServer(srv.ID, 0, 0)
			if err := f.Error(); err != nil {
				log.Errorf("failed to remove srv %s, err: %v", nodeID, err)
				return err
			}

			log.Infof("node %s left successfully", nodeID)
			return nil
		}
	}

	log.Infof("node %s not exists in raft group", nodeID)
	return nil
}

func (r myraft) AddNode(serverId, addr string) error {
	if !r.IsLeader() {
		log.
			Warn("RemoveNode request should not be executed by nonleader node")

		return ErrNotLeader
	}

	cf := r.GetConfiguration()
	if err := cf.Error(); err != nil {
		log.Errorf("failed to get raft configuration: %v", err)
		return err
	}

	for _, server := range cf.Configuration().Servers {
		if server.ID == raft.ServerID(serverId) {
			log.Infof("node %s already joinedCluster raft cluster", serverId)
			return nil
		}
	}

	f := r.AddVoter(raft.ServerID(serverId), raft.ServerAddress(addr), 0, 0)
	if err := f.Error(); err != nil {
		return err
	}

	log.Infof("node %s at %s joinedCluster successfully", serverId, addr)
	return nil
}

func (r *myraft) SetKV(key string, val []byte) (err error) {
	log.
		WithFields(log.Fields{
			"key": key,
			"val": runtime.ToString(val),
		}).
		Debug("myraft.SetKV called")

	k, v := types.NewStoreKV(key, val)
	l, err := newFsmLog(actionSetKV, &setKVCommand{
		SetKey:    k,
		DeleteKey: "",
		Data:      &v,
	})
	if err != nil {
		err = errors.Wrap(err, "myraft.newFsmLog failed")
		return
	}

	last, err := r.repo.GetKV(types.StoreKey(key))
	if err != nil {
		log.
			WithFields(log.Fields{
				"key":   key,
				"error": err,
			}).
			Warn("myraft.touchOffChange could to load last value of key")
	}

	// DONE(@yeqown): apply change log
	if err = r.propagateToSlaves(l); err != nil {
		return errors.Wrap(err, "myraft.propagateToSlaves failed")
	}

	// touch off change signal to cassemdb cluster.
	r.touchOffChange(types.OpSet, key, last, &v)

	return nil
}

func (r *myraft) UnsetKV(key string) error {
	l, err := newFsmLog(actionSetKV, &setKVCommand{
		SetKey:    "",
		DeleteKey: types.StoreKey(key),
		Data:      nil,
	})
	if err != nil {
		return errors.Wrap(err, "myraft.newFsmLog failed")
	}

	last, err := r.repo.GetKV(types.StoreKey(key))
	if err != nil {
		log.
			WithFields(log.Fields{
				"key":   key,
				"error": err,
			}).
			Warn("myraft.touchOffChange could to load last value of key")
	}

	// DONE(@yeqown): apply change log
	if err = r.propagateToSlaves(l); err != nil {
		return errors.Wrap(err, "myraft.propagateToSlaves failed")
	}

	// touch off change signal to cassemdb cluster.
	r.touchOffChange(types.OpUnset, key, last, nil)

	return nil
}

// touchOffChange only touch off a notification while:
// 1. delete a kv.
// 2. really update a existed kv.
//
func (r myraft) touchOffChange(op types.ChangeOp, key string, last, newVal *types.StoreValue) {
	if last == nil {
		// last == nil means that the key is new, there's no observer;
		return
	}

	if newVal != nil && strings.Compare(last.Fingerprint, newVal.Fingerprint) == 0 {
		// set kv but newVal is same to old value, so no need to touch off a change notification.
		return
	}

	go func() {
		log.
			WithFields(log.Fields{
				"key":    key,
				"newVal": newVal,
			}).
			Debug("myraft.touchOffChange called")

		l, err := newFsmLog(actionChange, &changeCommand{
			Change: &types.Change{
				Op:      op,
				Key:     types.StoreKey(key),
				Last:    last,
				Current: newVal,
			}})

		if err = r.propagateToSlaves(l); err != nil {
			log.
				WithFields(log.Fields{
					"key":    key,
					"newVal": newVal,
				}).
				Error("myraft.touchOffChange called")
		}
	}()
}

func (r *myraft) GetKV(key string) (*types.StoreValue, error) {
	val, err := r.repo.GetKV(types.StoreKey(key))
	if err != nil {
		log.
			WithFields(log.Fields{
				"key":   key,
				"error": err,
			}).
			Error("repo.GetKV failed")
	}

	return val, err
}

func (r myraft) GetLeaderAddr() string {
	return r.fsm.getLeaderAddr()
}

func (r *myraft) ChangeNotifyCh() <-chan watcher.IChange {
	return r.changeCh
}
