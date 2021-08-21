package domain

import (
	"fmt"
	"net"
	"os"
	"path/filepath"
	"sync/atomic"
	"time"

	"github.com/yeqown/cassem/internal/cassemdb/infras/repository"
	"github.com/yeqown/cassem/pkg/conf"
	"github.com/yeqown/cassem/pkg/hash"
	"github.com/yeqown/cassem/pkg/runtime"
	"github.com/yeqown/cassem/pkg/watcher"

	"github.com/hashicorp/raft"
	raftboltdb "github.com/hashicorp/raft-boltdb"
	"github.com/pkg/errors"
	"github.com/yeqown/log"
)

// IMyRaft defines the ability of what raft component should act.
type IMyRaft interface {
	GetKV(key string) (*repository.StoreValue, error)                       // GetKV get value of key
	SetKV(key string, value []byte, isDir, overwrite bool, ttl int32) error // SetKV save key and value
	UnsetKV(key string, isDir bool) error                                   // UnsetKV save key and value
	Range(key, seek string, limit int) (*repository.RangeResult, error)
	Expire(key string) error

	ChangeNotifyCh() <-chan watcher.IChange

	// IsLeader
	// TODO(@yeqown): replace IsLeader() into Stat()
	IsLeader() bool // IsLeader

	Shutdown() error

	RAFT() *raft.Raft
}

type Conf struct {
	HTTP        *conf.Server
	Raft        *conf.Raft
	Persistence *conf.Bolt
}

type myraft struct {
	*raft.Raft

	conf *conf.CassemdbConfig

	repo repository.KV

	// fsm is the state machine to be used in raft.RAFT. In cassem, it's mainly used to store
	// caches those encoded bytes to containers who are requested and should be cached.
	//
	// It also be used to store an"github.com/yeqown/cassem/pkg/types"ly leaderAddr which indicates the address of the leader.
	// While a leadership changes happened, leader node calls raft.Apply() to commit a log that
	// will update slave nodes' leaderAddr. please checkout FSMWrapper for more information.
	fsm myFSM

	// executionSinceLastSnapshot records the count how many times has fsm.Apply been called since
	// last time fsm.Snapshot called. It helps Core.doSnapshot to judge that should Core trigger snapshot or not.
	executionSinceLastSnapshot int32

	// changeCh
	changeCh chan watcher.IChange
}

func NewMyRaft(c *conf.CassemdbConfig) (IMyRaft, error) {
	r := &myraft{
		Raft:     nil,
		conf:     c,
		repo:     nil,
		fsm:      nil,
		changeCh: make(chan watcher.IChange, _SIZE_CHANGE_BUF),
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

func mallocServerId(addr string) raft.ServerID {
	return raft.ServerID(hash.MD5(runtime.ToBytes(addr)))
}

func (r *myraft) RAFT() *raft.Raft {
	return r.Raft
}

func (r *myraft) bootstrap() (err error) {
	defer func() {
		if err != nil {
			log.
				WithFields(log.Fields{
					"raftConfig": r.conf.Raft,
				}).
				Errorf("myraft.bootstrapRaft failed: %v", err)
		}
	}()

	// prepare transport
	raftConf := r.conf.Raft
	addr, err := net.ResolveTCPAddr("tcp", raftConf.Bind)
	if err != nil {
		return errors.Wrap(err, "ResolveTCPAddr failed")
	}
	transport, err := raft.NewTCPTransport(raftConf.Bind, addr, 3, 10*time.Second, os.Stderr)
	if err != nil {
		return errors.Wrap(err, "NewTCPTransport failed")
	}

	// prepare snapshot store
	snapshotStore, err := raft.NewFileSnapshotStore(raftConf.Base, 2, os.Stderr)
	if err != nil {
		return errors.Wrap(err, "NewFileSnapshotStore failed")
	}

	// boltStore implement log store and stable store interface
	boltStore, err := raftboltdb.NewBoltStore(filepath.Join(raftConf.Base, "raft.db"))
	if err != nil {
		return errors.Wrap(err, "NewBoltStore failed")
	}

	// fsm loading
	r.repo, err = repository.NewRepository(r.conf.Bolt)
	if err != nil {
		return errors.Wrap(err, "bbolt.NewMyRaft failed")
	}
	log.Debug("myraft.repo loaded")

	r.fsm = newFSM(r.repo, r.changeCh)
	log.Debug("myraft.fsm loaded")

	// construct raft system
	config := raft.DefaultConfig()
	config.LocalID = mallocServerId(raftConf.Bind)
	config.SnapshotThreshold = 1024

	if r.Raft, err = raft.NewRaft(config, r.fsm, boltStore, boltStore, snapshotStore, transport); err != nil {
		return errors.Wrap(err, "raft.NewRaft failed")
	}

	// DONE(@yeqown): allow node to bootstrap every time (IGNORING error), but only to join cluster when
	// raftConf.ClusterAddresses is not empty and raftConf.ClusterToJoin != raftConf.Bind
	var couldIgnore error

	log.
		WithFields(log.Fields{
			"shouldBootstrapCluster": raftConf.BootstrapCluster,
		}).
		Debug("myraft.bootstrap called")
	if raftConf.BootstrapCluster {
		servers := make([]raft.Server, 0, len(raftConf.ClusterAddresses))
		for _, address := range raftConf.ClusterAddresses {
			servers = append(servers, raft.Server{
				Suffrage: raft.Voter,
				ID:       mallocServerId(address),
				Address:  raft.ServerAddress(address),
			})
		}

		// A cluster can only be bootstrapped once from a single participating Voter server.
		// Any further attempts to bootstrap will return an error that can be safely ignored.
		if couldIgnore = r.
			BootstrapCluster(raft.Configuration{Servers: servers}).Error(); couldIgnore != nil {

			log.Warnf("myraft.bootstrap could not BootstrapCluster: %v", couldIgnore)
		}
	}

	return
}

func (r myraft) Shutdown() error {
	return r.Raft.Shutdown().Error()
}

func (r myraft) printStat() {
	log.
		WithFields(log.Fields{
			"leaderAddr": r.Raft.Leader(),
			"stats":      r.Raft.Stats(),
		}).
		Debug("myraft.stat")
}

// DONE(@yeqown): let node be notified while leader changes, and also mark current node is leader or not?
func (r myraft) watchLeaderChanges() error {
	isLeaderCh := r.Raft.LeaderCh()

	for isLeader := range isLeaderCh {
		log.
			WithField("toBeLeader", isLeader).
			Debug("myraft.watchLeaderChanges got a signal")

		// FIXED(@yeqown): reset leader address when leadership transition has occurred.
		//r.fsm.setLeaderAddr("")

		if !isLeader {
			continue
		}
	}

	return nil
}

const (
	// _SIZE_EXECUTIONS is a value which limit the minimum count of logs
	// must be executed since last snapshot action.
	_SIZE_EXECUTIONS = 100

	// _SIZE_CHANGE_BUF
	_SIZE_CHANGE_BUF = 1024

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

			if n := r.executionSinceLastSnapshot; n > _SIZE_EXECUTIONS {
				// if the state machine has received log over than 10
				// after last snapshot.
				reason = fmt.Sprintf("_SIZE_EXECUTIONS(%d) reached", _SIZE_EXECUTIONS)
				needSnapshot = true
			}
		}

		log.WithFields(log.Fields{"needSnapshot": needSnapshot, "reason": reason}).
			Debug("myraft.doSnapshot called")

		if !needSnapshot {
			continue
		}
		// execute snapshot
		err := r.Snapshot().Error()
		if err == nil {
			atomic.StoreInt32(&(r.executionSinceLastSnapshot), 0)
		}
		if err != nil {
			log.Errorf("myraft.doSnapshot failed to snapshot: %v", err)
		}
	}
}

func (r myraft) IsLeader() bool {
	return r.State() == raft.Leader
}

func (r myraft) propagateCommand(c command) error {
	atomic.AddInt32(&(r.executionSinceLastSnapshot), 1)

	l, err := newLog(c.action(), c)
	if err != nil {
		return errors.Wrap(err, "myraft.propagateCommand failed to generate fsmLog")
	}

	// DONE(@yeqown): apply change log
	if err = r.propagateToSlaves(l); err != nil {
		return errors.Wrap(err, "myraft.propagateCommand failed to propagateToSlaves")
	}

	return nil
}

// propagateToSlaves calls raft.Apply to distribute changes to FSM or propagate signal to all slaves.
// Importantly, DO NOT call r.raft.Apply directly, all operations those need to be propagated should call this.
// Only leader should call this.
func (r myraft) propagateToSlaves(fsmLog *fsmLog) (err error) {
	if !r.IsLeader() {
		panic("impossible")
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

//
//// removeNode only leader node would receive such request.
//func (r myraft) removeNode(nodeID string) error {
//	log.Infof("received removeNode request for remote node %s", nodeID)
//
//	if !r.IsLeader() {
//		log.
//			Warn("removeNode request should not be executed by nonleader node")
//
//		return ErrNotLeader
//	}
//
//	cf := r.GetConfiguration()
//	if err := cf.Error(); err != nil {
//		log.Errorf("failed to get raft configuration: %v", err)
//		return err
//	}
//
//	for _, srv := range cf.Configuration().Servers {
//		if srv.ID == raft.ServerID(nodeID) {
//			f := r.RemoveServer(srv.ID, 0, 0)
//			if err := f.Error(); err != nil {
//				log.Errorf("failed to remove srv %s, err: %v", nodeID, err)
//				return err
//			}
//
//			log.Infof("node %s left successfully", nodeID)
//			return nil
//		}
//	}
//
//	log.Infof("node %s not exists in raft group", nodeID)
//	return nil
//}
//
//func (r myraft) addNode(serverId, addr string) error {
//	if !r.IsLeader() {
//		log.
//			Warn("removeNode request should not be executed by nonleader node")
//
//		return ErrNotLeader
//	}
//
//	cf := r.GetConfiguration()
//	if err := cf.Error(); err != nil {
//		log.Errorf("failed to get raft configuration: %v", err)
//		return err
//	}
//
//	for _, server := range cf.Configuration().Servers {
//		if server.ID == raft.ServerID(serverId) {
//			log.Infof("node %s already joinedCluster raft cluster", serverId)
//			return nil
//		}
//	}
//
//	f := r.AddVoter(raft.ServerID(serverId), raft.ServerAddress(addr), 0, 0)
//	if err := f.Error(); err != nil {
//		return err
//	}
//
//	log.Infof("node %s at %s joinedCluster successfully", serverId, addr)
//	return nil
//}

func (r *myraft) ChangeNotifyCh() <-chan watcher.IChange {
	return r.changeCh
}
