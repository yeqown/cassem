package agent

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestGetElementReqIdentifierValidation(t *testing.T) {
	valid := &GetElementReq{App: "demo_app", Env: "prod-1", Keys: []string{"db_url", "feature_flag"}}
	require.NoError(t, valid.Validate())

	for _, req := range []*GetElementReq{
		{App: "demo.app", Env: "prod", Keys: []string{"db_url"}},
		{App: "demo", Env: "prod env", Keys: []string{"db_url"}},
		{App: "demo", Env: "prod", Keys: []string{"db.url"}},
		{App: "demo", Env: "prod", Keys: []string{"feature,flag"}},
	} {
		require.Error(t, req.Validate())
	}
}
