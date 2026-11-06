package protogen

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestProtoNamespaces(t *testing.T) {
	assert.Equal(t, "cassem.api.kv", string(File_cassem_internal_kv_proto.Package()))
	assert.Equal(t, "cassem.api.kv", string(File_cassem_internal_raft_proto.Package()))
}

func TestGRPCServiceNamespaces(t *testing.T) {
	assert.Equal(t, "cassem.api.kv.KV", _KV_serviceDesc.ServiceName)
	assert.Equal(t, "cassem.api.kv.Cluster", _Cluster_serviceDesc.ServiceName)
}
