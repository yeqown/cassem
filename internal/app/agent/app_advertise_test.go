package agent

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestAdvertiseAddrUsesHostnameForBindOnlyAddr(t *testing.T) {
	require.Equal(t, "cassemagent:20219", advertiseAddr(":20219", "cassemagent"))
}

func TestAdvertiseAddrKeepsReachableHostPort(t *testing.T) {
	require.Equal(t, "127.0.0.1:20219", advertiseAddr("127.0.0.1:20219", "cassemagent"))
}
