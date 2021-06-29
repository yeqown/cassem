package cassemdb

import (
	"github.com/pkg/errors"
	"github.com/yeqown/log"

	"github.com/yeqown/cassem/apps/cassemdb/coord"
)

var _ coord.ICoordinator = &cassemdb{}

var (
	ErrNotLeader = errors.New("current node is not allow to write, should not be triggered normally")
)

// AddNode only leader node would receive such request. MAYBE?
func (c cassemdb) AddNode(serverId, addr string) error {
	log.Infof("received AddNode request for remote node %s, addr %s", serverId, addr)
	return c.raft.AddNode(serverId, addr)
}

// RemoveNode only leader node would receive such request.
func (c cassemdb) RemoveNode(nodeID string) error {
	return c.raft.RemoveNode(nodeID)
}

//func (c cassemdb) Apply(msg []byte) (err error) {
//	return c.raft.ApplyFromMessage(msg)
//}

// isLeader only return true if current node is leader.
func (c cassemdb) isLeader() bool {
	return c.raft.IsLeader()
}

func (c cassemdb) ShouldForwardToLeader() (shouldForward bool, leadAddr string) {
	return !c.isLeader(), c.raft.GetLeaderAddr()
}

func (c *cassemdb) GetKV(key string) ([]byte, error) {
	return c.raft.GetKV(key)
}

func (c *cassemdb) SetKV(key string, val []byte) error {
	return c.raft.SetKV(key, val)
}

func (c *cassemdb) UnsetKV(key string) error {
	return c.raft.UnsetKV(key)
}
