package concept

import pb "github.com/yeqown/cassem/internal/protogen"

// KVChange adapts a wire change to the internal watcher contract.
type KVChange struct {
	*pb.Change
}

func (c *KVChange) Topic() string     { return c.GetKey() }
func (*KVChange) Type() pb.ChangeType { return pb.ChangeType_KV }

// DirectoryChange routes a change to observers of its parent directory.
type DirectoryChange struct {
	*pb.ParentDirectoryChange
}

func (c *DirectoryChange) Topic() string     { return c.GetSpecificTopic() }
func (*DirectoryChange) Type() pb.ChangeType { return pb.ChangeType_DIR }

var (
	_ Change = (*KVChange)(nil)
	_ Change = (*DirectoryChange)(nil)
)
