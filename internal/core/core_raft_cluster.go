package core

import (
	"encoding/json"
	"net"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/hashicorp/raft"
	raftboltdb "github.com/hashicorp/raft-boltdb"
	"github.com/pkg/errors"
	"github.com/yeqown/log"
)

var (
	client = &http.Client{}
)

// operateNodeResp is a copy from internal/api/http.commonResponse, only be used to
// be unmarshalled from response of Core.tryJoinCluster.
type operateNodeResp struct {
	ErrCode    int         `json:"errcode"`
	ErrMessage string      `json:"errmsg,omitempty"`
	Data       interface{} `json:"data,omitempty"`
}

func operateNodeRequest(base string, data map[string]string) error {
	if base == "" {
		log.Warn("operateNodeRequest could not be executed with empty RAFT bind address, skip")
		return nil
	}
	// detection and fix schema
	if !strings.HasPrefix(base, "http://") && !strings.HasPrefix(base, "https://") {
		base = "http://" + base
	}

	// assemble form parameters
	form := url.Values{}
	for k, v := range data {
		form.Add(k, v)
	}

	uri := base + "?" + form.Encode()
	req, err := http.NewRequest(http.MethodGet, uri, nil)
	if err != nil {
		log.Errorf("invalid http.NewRequest: %v", err)
		return errors.Wrap(err, "invalid http.NewRequest")
	}

	r, err := client.Do(req)
	if err != nil {
		log.Errorf("invalid do: %v", err)
		return err
	}

	if r.StatusCode != http.StatusOK {
		defer r.Body.Close()
		result := new(operateNodeResp)
		if err = json.NewDecoder(r.Body).Decode(result); err != nil {
			log.Errorf("executeOperateNodeRequest could not parse response: %v", err)
			return err
		}

		err = errors.New(result.ErrMessage)
	}

	return err
}

const (
	_formServerId        = "serverId"
	_formAction          = "action"
	_formRaftBindAddress = "bind"

	_actionJoin = "join"
	_actionLeft = "left"
)

func (c *Core) tryJoinCluster() (err error) {
	base := c.cfg.Server.Raft.ClusterAddrToJoin
	if err = operateNodeRequest(base, map[string]string{
		_formServerId:        c.serverId,
		_formAction:          _actionJoin,
		_formRaftBindAddress: c.cfg.Server.Raft.RaftBind,
	}); err != nil {
		log.Errorf("invalid request: %v", err)

		return errors.Wrap(err, "invalid http.NewRequest")
	}

	c.joinedCluster = true

	return
}

func (c *Core) tryLeaveCluster() (err error) {
	base := c.cfg.Server.Raft.ClusterAddrToJoin
	if err = operateNodeRequest(base, map[string]string{
		_formServerId: c.serverId,
		_formAction:   _actionLeft,
	}); err != nil {
		log.Errorf("invalid request: %v", err)

		return errors.Wrap(err, "invalid http.NewRequest")
	}

	c.joinedCluster = false

	return
}

func (c *Core) bootstrapRaft() (err error) {
	defer func() {
		if err != nil {
			log.
				WithFields(log.Fields{
					"raftConfig": c.cfg.Server.Raft,
					"joined":     c.joinedCluster,
					"serverId":   c.serverId,
				}).
				Errorf("Core.bootstrapRaft failed: %v", err)
		}
	}()

	// prepare transport
	raftConf := c.cfg.Server.Raft
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

	raftFSM := newFSM(c._containerCache)
	if c.raft, err = raft.NewRaft(config, raftFSM, boltDB, boltDB, snapshotStore, transport); err != nil {
		return errors.Wrap(err, "raft.NewRaft failed")
	}

	// DONE(@yeqown): allow node to bootstrap every time (IGNORING error), but only to join cluster when
	// raftConf.ClusterAddrToJoin is not empty and raftConf.ClusterToJoin != raftConf.Listen
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
	shouldJoinCluster := raftConf.ClusterAddrToJoin != ""
	if shouldJoinCluster {
		// FIXED(@yeqown) could not return error, tryJoinCluster will retry again.
		if couldIgnore = c.tryJoinCluster(); couldIgnore != nil {
			log.Warnf("tryJoinCluster cluster failed: %v", couldIgnore)
			c.joinedCluster = false
		}
	}

	return
}
