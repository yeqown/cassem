package api

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestDialWithModeContextUsesCallerContext(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	startedAt := time.Now()
	_, err := DialWithModeContext(ctx, []string{"127.0.0.1:0"}, Mode_R)
	require.Error(t, err)
	require.ErrorContains(t, err, "DialWithModeContext failed")
	require.ErrorIs(t, err, context.Canceled)
	require.Less(t, time.Since(startedAt), time.Second)
}
