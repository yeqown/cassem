package agent

import (
	"testing"

	pb "github.com/yeqown/cassem/internal/protogen"
	clientpb "github.com/yeqown/cassem/sdk/go/protogen"

	"github.com/stretchr/testify/assert"
)

func TestProtoNamespaces(t *testing.T) {
	assert.Equal(t, "cassem.api.agent", string(pb.File_cassem_internal_delivery_proto.Package()))
}

func TestGRPCServiceNamespaces(t *testing.T) {
	assert.Equal(t, "cassem.api.agent.agent", clientpb.Agent_ServiceDesc.ServiceName)
	assert.Equal(t, "cassem.api.agent.delivery", pb.Delivery_ServiceDesc.ServiceName)
	assert.Equal(t, "/cassem.api.agent.agent/GetElement", clientpb.Agent_GetElement_FullMethodName)
	assert.Equal(t, "/cassem.api.agent.delivery/Dispatch", pb.Delivery_Dispatch_FullMethodName)
}
