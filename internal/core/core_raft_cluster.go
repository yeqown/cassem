package core

import (
	"bytes"
	"encoding/json"
	"net"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/yeqown/cassem/internal/cache"

	"github.com/hashicorp/raft"
	raftboltdb "github.com/hashicorp/raft-boltdb"
	"github.com/pkg/errors"
	"github.com/yeqown/log"
)

var (
	client = &http.Client{}
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

	req := forwardRequest{
		forceBase: base,
		path:      "/cluster/nodes",
		method:    http.MethodGet,
		form: map[string]string{
			_formServerId:        c.serverId,
			_formAction:          _actionJoin,
			_formRaftBindAddress: c.config.Server.Raft.RaftBind,
		},
		body: nil,
	}

	// DONE(@yeqown): should send request to leader
	if err = c.forwardToLeader(&req); err != nil {
		log.Errorf("Core.tryJoinCluster calling c.forwardToLeader failed: %v", err)

		return errors.Wrap(err, "tryJoinCluster failed")
	}

	c.joinedCluster = true

	return
}

// tryLeaveCluster only called by follower node.
func (c *Core) tryLeaveCluster() (err error) {
	req := forwardRequest{
		path:   "/cluster/nodes",
		method: http.MethodGet,
		form: map[string]string{
			_formServerId: c.serverId,
			_formAction:   _actionLeft,
		},
		body: nil,
	}

	// DONE(@yeqown): should send request to leader
	if err = c.forwardToLeader(&req); err != nil {
		log.Errorf("Core.tryLeaveCluster calling c.forwardToLeader failed: %v", err)

		return errors.Wrap(err, "tryLeaveCluster failed")
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

// forwardToLeader only forward operations in core (apply, join, leave).
// this would send a request(HTTP) to leader contains what operation need to do, of course, it takes
// necessary external information.
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

	switch req.method {
	case http.MethodGet:
		err = handleGET(base, req.form)
	case http.MethodPost:
		err = handlePOST(base, req.body)
	}

	return
}

// operateNodeResp is a copy from internal/api/http.commonResponse, only be used to
// be unmarshalled from response of Core.tryJoinCluster.
type operateNodeResp struct {
	ErrCode    int         `json:"errcode"`
	ErrMessage string      `json:"errmsg,omitempty"`
	Data       interface{} `json:"data,omitempty"`
}

func handlePOST(base string, body interface{}) error {
	buf := bytes.NewBuffer(nil)
	if err := json.NewEncoder(buf).Encode(body); err != nil {
		return errors.Wrap(err, "could not encode body")
	}

	uri := base + "?" + "clusterSecret=9520059dd167"
	req, err := http.NewRequest(http.MethodPost, uri, buf)
	if err != nil {
		log.Errorf("invalid http.NewRequest: %v", err)
		return errors.Wrap(err, "invalid http.NewRequest")
	}
	req.Header.Set("Content-Type", "application/json")

	return execute(req)
}

func handleGET(base string, data map[string]string) error {
	// assemble form parameters
	form := url.Values{}
	for k, v := range data {
		form.Add(k, v)
	}
	form.Add("clusterSecret", "9520059dd167")

	uri := base + "?" + form.Encode()
	req, err := http.NewRequest(http.MethodGet, uri, nil)
	if err != nil {
		log.Errorf("invalid http.NewRequest: %v", err)
		return errors.Wrap(err, "invalid http.NewRequest")
	}

	return execute(req)
}

func execute(req *http.Request) error {
	r, err := client.Do(req)
	if err != nil {
		log.Errorf("invalid do: %v", err)
		return err
	}

	if r.StatusCode != http.StatusOK {
		err = errors.New("response code:" + strconv.Itoa(r.StatusCode))

		defer r.Body.Close()
		result := new(operateNodeResp)
		if err2 := json.NewDecoder(r.Body).Decode(result); err2 != nil {
			log.Errorf("executeOperateNodeRequest could not parse response: %v", err2)
			return errors.Wrap(err, err2.Error())
		}

		err = errors.Wrap(err, result.ErrMessage)
	}

	return err
}
