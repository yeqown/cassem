package adm

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/yeqown/cassem/api/concept"
)

func TestAssignRoleRequestAllowsVisitor(t *testing.T) {
	r := httptest.NewRequest(http.MethodGet, "/api/account/acl/assign?account=viewer@example.com&role=visitor&domain=payment-service/production", nil)

	var req assignOrRevokeRoleReq
	require.NoError(t, bindRequest(r, &req))
	require.Equal(t, concept.Role_VISITOR, req.Role)
	require.Equal(t, []string{"payment-service/production"}, req.Domains)
}
