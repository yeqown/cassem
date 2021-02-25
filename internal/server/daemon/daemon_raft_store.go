package daemon

import (
	"net"
	"os"
	"path/filepath"
	"time"

	"github.com/yeqown/cassem/pkg/fs"

	"github.com/hashicorp/raft"
	raftboltdb "github.com/hashicorp/raft-boltdb"
	"github.com/pkg/errors"
	"github.com/yeqown/log"
)

//// TODO(@yeqown): store intersects to the coordinator, maybe them two should be merged?
//type store struct {
//	c    *conf.Raft
//	raft *raft.Raft
//	fsm  raft.FSM
//}
//
//func newStore(c *conf.Raft) *store {
//	return &store{
//		c:   c,
//		fsm: newFSM(),
//	}
//}

func (d *Daemon) bootstrapRaft() (err error) {
	defer func() {
		if err != nil {
			log.
				WithFields(log.Fields{
					"raftConfig": d.cfg.Server.Raft,
					"joined":     d.joinedCluster,
					"serverId":   d.serverId,
				}).
				Errorf("Daemon.bootstrapRaft failed: %v", err)
		}
	}()

	c := d.cfg.Server.Raft
	config := raft.DefaultConfig()
	config.LocalID = raft.ServerID(d.serverId)
	config.SnapshotThreshold = 1024

	// prepare transport
	addr, err := net.ResolveTCPAddr("tcp", c.Bind)
	if err != nil {
		return errors.Wrap(err, "ResolveTCPAddr failed")
	}
	transport, err := raft.NewTCPTransport(c.Bind, addr, 3, 10*time.Second, os.Stderr)
	if err != nil {
		return errors.Wrap(err, "NewTCPTransport failed")
	}

	// prepare snapshot store
	ss, err := raft.NewFileSnapshotStore(c.Base, 2, os.Stderr)
	if err != nil {
		return errors.Wrap(err, "NewFileSnapshotStore failed")
	}

	// boltDB implement log store and stable store interface
	boltDB, err := raftboltdb.NewBoltStore(filepath.Join(c.Base, "raft.db"))
	if err != nil {
		return errors.Wrap(err, "NewBoltStore failed")
	}

	// construct raft system
	if d.raft, err = raft.NewRaft(config, d.fsm, boltDB, boltDB, ss, transport); err != nil {
		return errors.Wrap(err, "raft.NewRaft failed")
	}

	// FIXED: BootstrapCluster only executed at first time without any store.
	bootstrapCluster := d.cfg.Server.Raft.Join == "" && !fs.Exists(d.cfg.Server.Raft.Base)
	if bootstrapCluster {
		d.joinedCluster = true
		configuration := raft.Configuration{
			Servers: []raft.Server{
				{
					ID:      config.LocalID,
					Address: transport.LocalAddr(),
				},
			},
		}
		if err = d.raft.BootstrapCluster(configuration).Error(); err != nil {
			err = errors.Wrap(err, "d.raft.BootstrapCluster failed")
		}

		return
	}

	// restart
	if fs.Exists(d.cfg.Server.Raft.Base) && d.cfg.Server.Raft.Join == "" {
		// local store file exists
		// pass: do nothing
	} else {
		// FIXED(@yeqown) could not return error, join could retry
		if err2 := d.join(); err2 != nil {
			log.Errorf("join cluster failed: %v", err2)
		}
	}

	return
}

func (d Daemon) addNode(serverId, addr string) error {
	log.Infof("received join request for remote node %s, addr %s", serverId, addr)

	cf := d.raft.GetConfiguration()
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

	f := d.raft.AddVoter(raft.ServerID(serverId), raft.ServerAddress(addr), 0, 0)
	if err := f.Error(); err != nil {
		return err
	}

	log.Infof("node %s at %s joinedCluster successfully", serverId, addr)
	return nil
}

func (d Daemon) removeNode(nodeID string) error {
	log.Infof("received join request for remote node %s", nodeID)

	cf := d.raft.GetConfiguration()
	if err := cf.Error(); err != nil {
		log.Errorf("failed to get raft configuration: %v", err)
		return err
	}

	for _, srv := range cf.Configuration().Servers {
		if srv.ID == raft.ServerID(nodeID) {
			f := d.raft.RemoveServer(srv.ID, 0, 0)
			if err := f.Error(); err != nil {
				log.Errorf("failed to remove srv %s, err: ", nodeID, err)
				return err
			}

			log.Infof("node %s left successfully", nodeID)
			return nil
		}
	}

	log.Infof("node %s not exists in raft group", nodeID)
	return nil
}
