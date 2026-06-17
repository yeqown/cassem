package infras

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
