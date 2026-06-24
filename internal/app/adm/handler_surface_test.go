package adm

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestAdminHTTPHandlerSurface(t *testing.T) {
	d := app{}
	handlers := []http.HandlerFunc{
		d.UserLoginHTTP,
		d.GetUsersHTTP,
		d.GetUserACLHTTP,
		d.AddUserHTTP,
		d.DisableUserHTTP,
		d.ResetUserHTTP,
		d.GetACLDomainOptionsHTTP,
		d.AssignRoleHTTP,
		d.RevokeRoleHTTP,
		d.GetRetentionPolicyHTTP,
		d.GetAppsHTTP,
		d.GetAppHTTP,
		d.CreateAppHTTP,
		d.DeleteAppHTTP,
		d.GetAppEnvironmentsHTTP,
		d.CreateAppEnvironmentHTTP,
		d.DeleteAppEnvironmentHTTP,
		d.GetAppEnvElementsHTTP,
		d.GetAppEnvElementHTTP,
		d.CreateAppEnvElementHTTP,
		d.UpdateAppEnvElementHTTP,
		d.DeleteAppEnvElementHTTP,
		d.GetAppEnvElementAllVersionsHTTP,
		d.RollbackAppEnvElementHTTP,
		d.PublishAppEnvElementHTTP,
		d.GetAppEnvElementOperationsHTTP,
		d.GetClusterTopologyHTTP,
		d.GetAgentsHTTP,
		d.GetInstancesHTTP,
		d.GetInstanceHTTP,
		d.GetInstancesByElementHTTP,
	}

	require.Len(t, handlers, 31)
}
