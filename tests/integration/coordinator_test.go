//go:build integration

package integration_test

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/yeqown/cassem/api/concept"
	"github.com/yeqown/cassem/internal/coordinator"
	"github.com/yeqown/cassem/tests/testutil"
)

func TestCoordinatorAdmAggregateElementLifecycle(t *testing.T) {
	cluster := testutil.UseDBCluster(t)
	agg, err := coordinator.NewAdmAggregate(cluster.DBEndpoints)
	require.NoError(t, err)

	ctx := context.Background()
	app, env, key := uniqueNames(t)

	require.NoError(t, agg.CreateApp(ctx, &concept.AppMetadata{
		Id:          app,
		Description: "integration test app",
		Creator:     "integration",
		Owner:       "integration",
		Status:      concept.AppMetadata_INUSE,
	}))
	require.NoError(t, agg.CreateEnvironment(ctx, app, env))
	require.NoError(t, agg.CreateElement(ctx, app, env, key, []byte("value-v1"), concept.ContentType_PLAINTEXT))

	created, err := agg.GetElementWithVersion(ctx, app, env, key, 1)
	require.NoError(t, err)
	require.Equal(t, []byte("value-v1"), created.GetRaw())
	require.False(t, created.GetPublished())

	published, err := agg.PublishElementVersion(ctx, app, env, key, 1)
	require.NoError(t, err)
	require.True(t, published.GetPublished())

	require.NoError(t, agg.UpdateElement(ctx, app, env, key, []byte("value-v2")))
	updated, err := agg.GetElementWithVersion(ctx, app, env, key, 2)
	require.NoError(t, err)
	require.Equal(t, []byte("value-v2"), updated.GetRaw())

	missing, err := agg.GetElementWithVersion(ctx, app, env, key, 99)
	require.Error(t, err)
	require.Nil(t, missing)

	require.NoError(t, agg.DeleteElement(ctx, app, env, key))
}

func TestCoordinatorAdmAggregateInstanceLifecycle(t *testing.T) {
	cluster := testutil.UseDBCluster(t)
	agg, err := coordinator.NewAdmAggregate(cluster.DBEndpoints)
	require.NoError(t, err)

	ctx := context.Background()
	app, env, key := uniqueNames(t)
	ins := &concept.Instance{
		ClientId: fmt.Sprintf("client-%d", time.Now().UnixNano()),
		ClientIp: "127.0.0.1",
		Watching: []*concept.Instance_Watching{{
			App:       app,
			Env:       env,
			WatchKeys: []string{key},
		}},
	}

	require.NoError(t, agg.RegisterInstance(ctx, ins))
	got, err := agg.GetInstance(ctx, ins.Id())
	require.NoError(t, err)
	require.Equal(t, ins.Id(), got.Id())

	require.NoError(t, agg.RenewInstance(ctx, ins))
	require.NoError(t, agg.UnregisterInstance(ctx, ins.Id()))
}

func TestCoordinatorAdmAggregateRBAC(t *testing.T) {
	cluster := testutil.UseDBCluster(t)
	agg, err := coordinator.NewAdmAggregate(cluster.DBEndpoints)
	require.NoError(t, err)

	account := fmt.Sprintf("user-%d@example.com", time.Now().UnixNano())
	require.NoError(t, agg.AddUser(&concept.User{
		Account:        account,
		Nickname:       "integration user",
		HashedPassword: "password",
		Status:         concept.User_NORMAL,
	}))

	user, err := agg.GetUser(account)
	require.NoError(t, err)
	require.Equal(t, account, user.GetAccount())
	require.NotEqual(t, "password", user.GetHashedPassword())

	require.NoError(t, agg.AssignRole(account, "admin", concept.Domain_ALL))
	allow, err := agg.Enforce(account, concept.Domain_CLUSTER, concept.Object_APP, concept.Action_READ)
	require.NoError(t, err)
	require.True(t, allow)

	require.NoError(t, agg.RevokeRole(account, "admin", concept.Domain_ALL))
	allow, err = agg.Enforce(account, concept.Domain_CLUSTER, concept.Object_APP, concept.Action_READ)
	require.NoError(t, err)
	require.False(t, allow)

	require.NoError(t, agg.DisableUser(account))
	disabled, err := agg.GetUser(account)
	require.NoError(t, err)
	require.Equal(t, concept.User_FORBIDDEN, disabled.GetStatus())
}
