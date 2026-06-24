package adm

import (
	"testing"

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
	require.Equal(t, "payment-service/production", domainFromParams("payment-service", "production"))
	require.Equal(t, "payment-service/*", domainFromParams("payment-service", ""))
	require.Equal(t, concept.Domain_CLUSTER, domainFromParams("", ""))
}
