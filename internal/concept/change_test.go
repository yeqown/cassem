package concept

import (
	"testing"

	"github.com/stretchr/testify/require"
	pb "github.com/yeqown/cassem/internal/protogen"
)

func TestChangeAdapters(t *testing.T) {
	wire := &pb.Change{Key: "app/default/key", Op: pb.Change_Set}
	kv := &KVChange{Change: wire}
	require.Equal(t, "app/default/key", kv.Topic())
	require.Equal(t, pb.ChangeType_KV, kv.Type())
	require.Same(t, wire, kv.Change)

	dir := &DirectoryChange{ParentDirectoryChange: &pb.ParentDirectoryChange{Change: wire, SpecificTopic: "app/default"}}
	require.Equal(t, "app/default", dir.Topic())
	require.Equal(t, pb.ChangeType_DIR, dir.Type())
	require.Same(t, wire, dir.GetChange())
}
