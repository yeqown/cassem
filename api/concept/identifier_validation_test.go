package concept

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestInstanceWatchingIdentifierValidation(t *testing.T) {
	valid := &Instance_Watching{App: "demo_app", Env: "prod-1", WatchKeys: []string{"db_url", "feature_flag"}}
	require.NoError(t, valid.Validate())

	for _, watching := range []*Instance_Watching{
		{App: "demo.app", Env: "prod", WatchKeys: []string{"db_url"}},
		{App: "demo", Env: "prod env", WatchKeys: []string{"db_url"}},
		{App: "demo", Env: "prod", WatchKeys: []string{"db.url"}},
		{App: "demo", Env: "prod", WatchKeys: []string{"feature,flag"}},
	} {
		require.Error(t, watching.Validate())
	}
}
