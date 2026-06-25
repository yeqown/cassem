package agent

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestProtoNamespaces(t *testing.T) {
	assert.Equal(t, "cassem.api.agent", string(File_cassemagent_api_proto.Package()))
}

func TestGRPCServiceNamespaces(t *testing.T) {
	assert.Equal(t, "cassem.api.agent.agent", Agent_ServiceDesc.ServiceName)
	assert.Equal(t, "cassem.api.agent.delivery", Delivery_ServiceDesc.ServiceName)
	assert.Equal(t, "/cassem.api.agent.agent/GetElement", Agent_GetElement_FullMethodName)
	assert.Equal(t, "/cassem.api.agent.delivery/Dispatch", Delivery_Dispatch_FullMethodName)
}
