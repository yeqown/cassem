package core

import (
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/yeqown/cassem/internal/cache"
	"github.com/yeqown/cassem/pkg/httpc"

	"github.com/hashicorp/raft"
	raftboltdb "github.com/hashicorp/raft-boltdb"
	"github.com/pkg/errors"
	"github.com/yeqown/log"
)

const (
	_formServerId        = "serverId"
	_formAction          = "action"
	_formRaftBindAddress = "bind"

	_actionJoin = "join"
	_actionLeft = "left"
)

// tryJoinCluster only called by follower node, normally, conf.Config.Server.Raft.ClusterAddresses is
// the leader's address which is set manually. MAYBE~
func (c *Core) tryJoinCluster() (err error) {
	var base string

	if count := len(c.config.Server.Raft.ClusterAddresses); count != 0 {
		base = c.config.Server.Raft.ClusterAddresses[c.tryJoinIdx]
		c.tryJoinIdx = (c.tryJoinIdx + 1) % count
	}

	log.
		WithFields(log.Fields{
			"base":             base,
			"clusterAddresses": c.config.Server.Raft.ClusterAddresses,
			"tryJoinIdx":       c.tryJoinIdx,
		}).
		Debug("Core.tryJoinCluster called")

	if err = c.forwardToLeaderJoinLeft(_actionJoin, base); err != nil {
		log.
			Errorf("Core.tryJoinCluster calling c.forwardToLeaderJoinLeft failed: %v", err)

		return errors.Wrap(err, "Core.tryJoinCluster failed")
	}

	// FIXME(@yeqown): base maybe not leader, should get leader address from raft.
	// or forbid forwarding join request.
	c.fsm.setLeaderAddr(base)
	c.joinedCluster = true

	return
}

// tryLeaveCluster only called by follower node.
func (c *Core) tryLeaveCluster() (err error) {
	if err = c.forwardToLeaderJoinLeft(_actionLeft, ""); err != nil {
		log.
			Errorf("Core.tryLeaveCluster calling c.forwardToLeaderJoinLeft failed: %v", err)

		return errors.Wrap(err, "Core.tryLeaveCluster failed")
	}

	c.joinedCluster = false
	if err = c.raft.Shutdown().Error(); err != nil {
		err = errors.Wrap(err, "Core.tryLeaveCluster shutdown failed")
	}

	return
}

func (c *Core) bootstrapRaft() (err error) {
	defer func() {
		if err != nil {
			log.
				WithFields(log.Fields{
					"raftConfig": c.config.Server.Raft,
					"joined":     c.joinedCluster,
					"serverId":   c.serverId,
				}).
				Errorf("Core.bootstrapRaft failed: %v", err)
		}
	}()

	// prepare transport
	raftConf := c.config.Server.Raft
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
	config.LocalID = raft.ServerID(c.serverId)
	config.SnapshotThreshold = 1024

	c.fsm = newFSM(cache.NewNonCache())
	if c.raft, err = raft.NewRaft(config, c.fsm, boltDB, boltDB, snapshotStore, transport); err != nil {
		return errors.Wrap(err, "raft.NewRaft failed")
	}

	// DONE(@yeqown): allow node to bootstrap every time (IGNORING error), but only to join cluster when
	// raftConf.ClusterAddresses is not empty and raftConf.ClusterToJoin != raftConf.Listen
	var couldIgnore error

	// A cluster can only be bootstrapped once from a single participating Voter server.
	// Any further attempts to bootstrap will return an error that can be safely ignored.
	if couldIgnore = c.raft.BootstrapCluster(raft.Configuration{
		Servers: []raft.Server{
			{
				ID:      config.LocalID,
				Address: transport.LocalAddr(),
			},
		},
	}).Error(); couldIgnore != nil {
		log.Warnf("core.bootstrapRaft could not BootstrapCluster: %v", couldIgnore)
	}

	c.joinedCluster = true
	shouldJoinCluster := len(raftConf.ClusterAddresses) != 0
	if shouldJoinCluster {
		// FIXED(@yeqown) could not return error, tryJoinCluster will retry again.
		if couldIgnore = c.tryJoinCluster(); couldIgnore != nil {
			log.Warnf("tryJoinCluster cluster failed: %v", couldIgnore)
			c.joinedCluster = false
		}
	}

	return
}

type forwardRequest struct {
	forceBase string
	path      string
	method    string
	form      map[string]string
	body      interface{}
}

// operateNodeResp is a copy from internal/api/http.commonResponse, only be used to
// be unmarshalled from response of Core.tryJoinCluster.
type operateNodeResp struct {
	ErrCode    int    `json:"errcode"`
	ErrMessage string `json:"errmsg,omitempty"`
}

// forwardToLeader only forward operations in core (apply, join, leave).
// this would send a request(HTTP) to leader contains what operation need to do, of course, it takes
// necessary external information.
//
// Only slaves should call this.
func (c *Core) forwardToLeader(req *forwardRequest) (err error) {
	base := c.fsm.getLeaderAddr()
	if req.forceBase != "" {
		base = req.forceBase
	}

	// detection base empty or not, fix schema and assemble path to base
	if base == "" {
		log.Warn("forwardToLeader could not be executed with empty RAFT bind address, skip")
		return nil
	}

	if !strings.HasPrefix(base, "http://") && !strings.HasPrefix(base, "https://") {
		base = "http://" + base
	}
	if strings.HasSuffix(base, "/") {
		base = strings.TrimRight(base, "/")
	}
	base += req.path

	var resp = new(operateNodeResp)
	switch req.method {
	case http.MethodGet:
		req.form["clusterSecret"] = "9520059dd167"
		err = httpc.GET(base, req.form, resp)
	case http.MethodPost:
		base = base + "?" + "clusterSecret=9520059dd167"
		err = httpc.POST(base, req.body, resp)
	}

	if resp != nil && resp.ErrCode != 0 {
		err = errors.New(resp.ErrMessage)
	}

	return
}

func (c Core) forwardToLeaderJoinLeft(action string, forceBase string) (err error) {
	form := map[string]string{
		_formServerId: c.serverId,
		_formAction:   action,
	}

	switch action {
	case _actionJoin:
		form[_formRaftBindAddress] = c.config.Server.Raft.RaftBind
	case _actionLeft:
	}

	req := forwardRequest{
		forceBase: forceBase,
		path:      "/cluster/nodes",
		method:    http.MethodGet,
		form:      form,
		body:      nil,
	}

	// DONE(@yeqown): should send request to leader
	if err = c.forwardToLeader(&req); err != nil {
		log.
			Errorf("Core.forwardToLeaderJoinLeft calling c.forwardToLeader failed: %v", err)

		return errors.Wrap(err, "forwardToLeaderJoinLeft failed")
	}

	return err
}

func (c Core) forwardToLeaderApply(fsmLog *coreFSMLog) error {
	data, err := fsmLog.serialize()
	if err != nil {
		return errors.Wrap(err, "Core.forwardToLeaderApply failed to serialize log")
	}

	req := &forwardRequest{
		path:   "/cluster/apply",
		method: http.MethodPost,
		form:   nil,
		body: struct {
			ApplyData []byte `json:"Data"`
		}{
			ApplyData: data,
		},
	}

	if err = c.forwardToLeader(req); err != nil {
		log.
			Errorf("Core.setContainerCache forwardToLeader failed: %v", err)
	}

	return err
}

// propagateToSlaves calls raft.Apply to distribute changes to FSM or propagate signal to all slaves.
// Importantly, DO NOT call c.raft.Apply directly, all operations those need to be propagated should call this.
// Only leader should call this.
func (c Core) propagateToSlaves(fsmLog *coreFSMLog) error {
	if !c.isLeader() {
		log.
			Warn("Core.propagateToSlaves Apply request should not be executed by non leader node")

		return ErrNotLeader
	}

	fsmLog.CreatedAt = time.Now().Unix()
	data, err := fsmLog.serialize()
	if err != nil {
		return errors.Wrap(err, "Core.propagateToSlaves failed to serialize log")
	}

	future := c.raft.Apply(data, 10*time.Second)
	if err = future.Error(); err != nil {
		return err
	}

	return nil
}
