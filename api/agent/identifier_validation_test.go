package agent

import (
	"testing"

	"buf.build/go/protovalidate"
	"github.com/stretchr/testify/require"
)

func TestGetElementReqIdentifierValidation(t *testing.T) {
	valid := &GetElementReq{App: "demo_app", Env: "prod-1", Keys: []string{"db_url", "feature_flag"}}
	require.NoError(t, protovalidate.Validate(valid))

	for _, req := range []*GetElementReq{
		{App: "demo.app", Env: "prod", Keys: []string{"db_url"}},
		{App: "demo", Env: "prod env", Keys: []string{"db_url"}},
		{App: "demo", Env: "prod", Keys: []string{"db.url"}},
		{App: "demo", Env: "prod", Keys: []string{"feature,flag"}},
	} {
		require.Error(t, protovalidate.Validate(req))
	}
}
