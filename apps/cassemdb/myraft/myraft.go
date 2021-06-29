package myraft

import (
	"net"
	"os"
	"path/filepath"
	"time"

	"github.com/yeqown/cassem/apps/cassemdb/persistence"
	"github.com/yeqown/cassem/internal/conf"
	"github.com/yeqown/cassem/pkg/runtime"

	"github.com/hashicorp/raft"
	raftboltdb "github.com/hashicorp/raft-boltdb"
	"github.com/pkg/errors"
	"github.com/yeqown/log"
)

var ErrNotLeader = errors.New("current node is not allow to write, should not be triggered normally")

// IMyRaft defines the ability of what raft component should act.
// TODO(@yeqown): implement a store interface for replication.
type IMyRaft interface {
	GetLeaderAddr() string           // GetLeaderAddr
	SetLeaderAddr(leaderAddr string) // SetLeaderAddr

	Shutdown() error                    // Shutdown
	IsLeader() bool                     // IsLeader
	JoinedCluster() bool                // JoinedCluster
	RemoveNode(serveId string) error    // RemoveNode
	AddNode(serveId, addr string) error // AddNode

	SetKV(key string, val []byte) error
	UnsetKV(key string) error
	GetKV(key string) ([]byte, error)

	ApplyRaw(data []byte) error
}

type Conf struct {
	HTTP *conf.HTTP
	Raft *conf.Raft
	Repo persistence.Repository
}

type myraft struct {
	*raft.Raft

	conf *Conf

	repo persistence.Repository

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
	// joinedCluster controls tryJoinCluster again and again in myraft.Heartbeat.
	joinedCluster bool
}

func New(conf *Conf) (IMyRaft, error) {
	r := &myraft{
		conf: conf,
		repo: conf.Repo,
		fsm:  newFSM(conf.Repo),
	}

	err := r.bootstrap()

	if err == nil {
		// leadership changes
		go runtime.GoFunc("leadership-changes", r.watchLeaderChanges)

		// snapshot executor
		//go runtime.GoFunc("snapshot-strategy", r.doSnapshot)
	}

	return r, err
}

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

	// boltDB implement log store and stable store interface
	boltDB, err := raftboltdb.NewBoltStore(filepath.Join(raftConf.RaftBase, "raft.db"))
	if err != nil {
		return errors.Wrap(err, "NewBoltStore failed")
	}

	// construct raft system
	config := raft.DefaultConfig()
	config.LocalID = raft.ServerID(r.serverId)
	config.SnapshotThreshold = 1024

	if r.Raft, err = raft.NewRaft(config, r.fsm, boltDB, boltDB, snapshotStore, transport); err != nil {
		return errors.Wrap(err, "raft.NewRaft failed")
	}

	// DONE(@yeqown): allow node to bootstrap every time (IGNORING error), but only to join cluster when
	// raftConf.ClusterAddresses is not empty and raftConf.ClusterToJoin != raftConf.Listen
	var couldIgnore error

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

	r.joinedCluster = true
	shouldJoinCluster := len(raftConf.ClusterAddresses) != 0
	if shouldJoinCluster {
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
	for {
		select {
		case isLeader := <-isLeaderCh:
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
			fsmLog, _ := NewFsmLog(ActionSetLeaderAddr, &SetLeaderAddrCommand{
				LeaderAddr: r.conf.HTTP.Addr,
			})
			if err := r.propagateToSlaves(fsmLog); err != nil {
				log.
					WithFields(log.Fields{
						"addr":   r.conf.HTTP.Addr,
						"fsmLog": fsmLog,
					}).
					Errorf("myraft.watchLeaderChanges applyTo raft failed: %v", err)
			}
		}
	}
}

const (
	// _SIZE_EXECUTIONS is a value which limit the minimum count of logs
	// must be executed since last snapshot action.
	_SIZE_EXECUTIONS = 100
)

//
//// DoSnapshot to execute snapshot of state machine with specified strategy:
////
//// 1. just do snapshot periodically.
//// 2. if state machine has executed logs more than specified size.
////
//func (r myraft) doSnapshot() error {
//	ticker := time.NewTicker(30 * time.Minute)
//	sizeTicker := time.NewTicker(10 * time.Second)
//
//	for {
//		needSnapshot := false
//		select {
//		case <-ticker.C:
//			needSnapshot = true
//
//		case <-sizeTicker.C:
//			if !r.IsLeader() {
//				continue
//			}
//			if r.fsm.getExecutionSinceLastSnapshot() > _SIZE_EXECUTIONS {
//				// if the state machine has received log over than 10
//				// after last snapshot.
//				needSnapshot = true
//			}
//
//			log.
//				WithField("needSnapshot", needSnapshot).
//				Debug("myraft.doSnapshot called")
//
//			if needSnapshot {
//				if err := r.Snapshot().Error(); err != nil {
//					log.Errorf("myraft.doSnapshot failed to snapshot: %v", err)
//				}
//			}
//			// case done
//		}
//	}
//}

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

// tryJoinCluster only called by follower node, normally, conf.Config.Server.Raft.ClusterAddresses is
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

func (r *myraft) SetKV(key string, val []byte) error {
	l, err := NewFsmLog(
		ActionSet,
		&SetCommand{SetKey: key, DeleteKey: "", NeedSetData: val},
	)
	if err != nil {
		return errors.Wrap(err, "myraft.NewFsmLog failed")
	}

	return r.applyLog(l)
}

func (r *myraft) UnsetKV(key string) error {
	l, err := NewFsmLog(
		ActionSet,
		&SetCommand{SetKey: "", DeleteKey: key, NeedSetData: nil},
	)
	if err != nil {
		return errors.Wrap(err, "myraft.NewFsmLog failed")
	}

	return r.applyLog(l)
}

func (r *myraft) GetKV(key string) ([]byte, error) {
	return r.repo.Get(key)
}

func (r myraft) applyLog(fsmLog *CoreFSMLog) (err error) {
	err = r.propagateToSlaves(fsmLog)
	if err != nil {
		log.
			WithFields(log.Fields{
				"fsmLog": fsmLog,
			}).
			Errorf("myraft.delContainerCache propagateToSlaves failed: %v", err)
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
func (r myraft) propagateToSlaves(fsmLog *CoreFSMLog) (err error) {
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

func (r myraft) GetLeaderAddr() string {
	return r.fsm.getLeaderAddr()
}

func (r myraft) SetLeaderAddr(addr string) {
	r.fsm.setLeaderAddr(addr)
}
