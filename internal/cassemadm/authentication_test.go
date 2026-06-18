package cassemadm

import (
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"

	"github.com/yeqown/cassem/api/concept"
)

func TestRetentionPolicyAuthMapping(t *testing.T) {
	def, ok := defMapping["GET/api/admin/retention"]

	require.True(t, ok)
	require.Equal(t, concept.Object_CLUSTER, def.object)
	require.Equal(t, concept.Action_READ, def.act)
}

func TestRequestDomainFromRouteParams(t *testing.T) {
	c := &gin.Context{}
	c.Params = gin.Params{
		{Key: "appId", Value: "payment-service"},
		{Key: "env", Value: "production"},
	}
	require.Equal(t, "payment-service/production", requestDomain(c))

	c = &gin.Context{}
	c.Params = gin.Params{{Key: "appId", Value: "payment-service"}}
	require.Equal(t, "payment-service/*", requestDomain(c))

	c = &gin.Context{}
	require.Equal(t, concept.Domain_CLUSTER, requestDomain(c))
}
