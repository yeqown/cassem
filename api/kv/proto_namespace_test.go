package kv

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestProtoNamespaces(t *testing.T) {
	assert.Equal(t, "cassem.api.kv", string(File_cassemkv_api_proto.Package()))
	assert.Equal(t, "cassem.api.kv", string(File_cassemkv_raft_proto.Package()))
}

func TestGRPCServiceNamespaces(t *testing.T) {
	assert.Equal(t, "cassem.api.kv.KV", _KV_serviceDesc.ServiceName)
	assert.Equal(t, "cassem.api.kv.Cluster", _Cluster_serviceDesc.ServiceName)
}
