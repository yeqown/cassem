package adm

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"

	"github.com/yeqown/cassem/api/concept"
)

func TestAssignRoleRequestAllowsVisitor(t *testing.T) {
	gin.SetMode(gin.TestMode)
	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	c.Request = httptest.NewRequest(http.MethodGet, "/api/account/acl/assign?account=viewer@example.com&role=visitor&domain=payment-service/production", nil)

	var req assignOrRevokeRoleReq
	require.NoError(t, c.ShouldBind(&req))
	require.Equal(t, concept.Role_VISITOR, req.Role)
	require.Equal(t, []string{"payment-service/production"}, req.Domains)
}
