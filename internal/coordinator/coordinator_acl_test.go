package coordinator

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"

	"github.com/yeqown/cassem/api/concept"
	apicassemdb "github.com/yeqown/cassem/internal/cassemdb/api"
	"github.com/yeqown/cassem/pkg/errorx"
	"github.com/yeqown/cassem/pkg/hash"
)

type aclTestKV struct {
	data map[string]*apicassemdb.Entity
}

func newACLTestKV() *aclTestKV {
	return &aclTestKV{data: make(map[string]*apicassemdb.Entity)}
}

func (f *aclTestKV) GetKV(_ context.Context, req *apicassemdb.GetKVReq, _ ...grpc.CallOption) (*apicassemdb.GetKVResp, error) {
	entity, ok := f.data[req.GetKey()]
	if !ok {
		return nil, errorx.Err_NOT_FOUND
	}
	return &apicassemdb.GetKVResp{Entity: entity}, nil
}

func (f *aclTestKV) GetKVs(context.Context, *apicassemdb.GetKVsReq, ...grpc.CallOption) (*apicassemdb.GetKVsResp, error) {
	return nil, errors.New("unused")
}

func (f *aclTestKV) SetKV(_ context.Context, req *apicassemdb.SetKVReq, _ ...grpc.CallOption) (*apicassemdb.Empty, error) {
	f.data[req.GetKey()] = apicassemdb.NewEntityWithCreated(req.GetKey(), req.GetVal(), 0, 1)
	return &apicassemdb.Empty{}, nil
}

func (f *aclTestKV) UnsetKV(context.Context, *apicassemdb.UnsetKVReq, ...grpc.CallOption) (*apicassemdb.Empty, error) {
	return nil, errors.New("unused")
}

func (f *aclTestKV) Watch(context.Context, *apicassemdb.WatchReq, ...grpc.CallOption) (apicassemdb.KV_WatchClient, error) {
	return nil, errors.New("unused")
}

func (f *aclTestKV) TTL(context.Context, *apicassemdb.TtlReq, ...grpc.CallOption) (*apicassemdb.TtlResp, error) {
	return nil, errors.New("unused")
}

func (f *aclTestKV) Expire(context.Context, *apicassemdb.ExpireReq, ...grpc.CallOption) (*apicassemdb.Empty, error) {
	return nil, errors.New("unused")
}

func (f *aclTestKV) Range(_ context.Context, req *apicassemdb.RangeReq, _ ...grpc.CallOption) (*apicassemdb.RangeResp, error) {
	if err := req.Validate(); err != nil {
		return nil, err
	}

	entities := make([]*apicassemdb.Entity, 0)
	for key, entity := range f.data {
		if req.GetKey() != "" && !strings.HasPrefix(key, req.GetKey()) {
			continue
		}
		if req.GetSeek() != "" && concept.ExtractPureKey(key) < req.GetSeek() {
			continue
		}
		entities = append(entities, entity)
	}
	slices.SortFunc(entities, func(a, b *apicassemdb.Entity) int { return strings.Compare(a.GetKey(), b.GetKey()) })
	if len(entities) > int(req.GetLimit()) {
		return &apicassemdb.RangeResp{
			Entities:    entities[:req.GetLimit()],
			HasMore:     true,
			NextSeekKey: concept.ExtractPureKey(entities[req.GetLimit()].GetKey()),
		}, nil
	}
	return &apicassemdb.RangeResp{Entities: entities}, nil
}

func (f *aclTestKV) CompactElementHistory(context.Context, *apicassemdb.CompactElementHistoryReq, ...grpc.CallOption) (*apicassemdb.CompactElementHistoryResp, error) {
	return nil, errors.New("unused")
}

func TestACLRejectsHardcodedSuperadminUser(t *testing.T) {
	acl, err := newRBAC(newACLTestKV())
	require.NoError(t, err)

	_, err = acl.GetUser("superadmin")
	require.ErrorIs(t, err, errorx.Err_NOT_FOUND)

	_, err = acl.GetUser("superadmin-anything")
	require.ErrorIs(t, err, errorx.Err_NOT_FOUND)
}

func TestACLDoesNotBypassBySuperadminSubject(t *testing.T) {
	acl, err := newRBAC(newACLTestKV())
	require.NoError(t, err)

	allowed, err := acl.Enforce("superadmin", concept.Domain_CLUSTER, concept.Object_APP, concept.Action_WRITE)
	require.NoError(t, err)
	require.False(t, allowed)
}

func TestACLAutoMigratePersistsPolicies(t *testing.T) {
	store := newACLTestKV()
	rbac, err := newRBAC(store)
	require.NoError(t, err)
	acl := rbac.(aclImpl)
	require.NoError(t, acl.AutoMigrate())
	require.Contains(t, store.data, concept.GenAclPolicyKey())

	reloaded, err := newRBAC(store)
	require.NoError(t, err)
	allowed, err := reloaded.Enforce("alice", concept.Domain_CLUSTER, concept.Object_APP, concept.Action_WRITE)
	require.NoError(t, err)
	require.False(t, allowed)

	require.NoError(t, reloaded.AssignRole("alice", concept.Role_ADMIN, concept.Domain_ALL))
	allowed, err = reloaded.Enforce("alice", concept.Domain_CLUSTER, concept.Object_APP, concept.Action_WRITE)
	require.NoError(t, err)
	require.True(t, allowed)
}

func TestNewRBACSeedsBuiltinPolicies(t *testing.T) {
	store := newACLTestKV()
	rbac, err := newRBAC(store)
	require.NoError(t, err)
	require.NoError(t, rbac.AssignRole("alice", "admin", concept.Domain_ALL))

	allowed, err := rbac.Enforce("alice", concept.Domain_CLUSTER, concept.Object_APP, concept.Action_READ)
	require.NoError(t, err)
	require.True(t, allowed)
}

func TestAutoMigrateRemovesLegacyRoleSubjectBypass(t *testing.T) {
	store := newACLTestKV()
	legacy := &concept.Casbin{Policies: []*concept.Casbin_Policy{
		{Ptype: "p", V0: concept.Role_SUPERADMIN, V1: concept.Domain_ALL, V2: concept.Object_ALL, V3: concept.Action_ANY},
		{Ptype: "g", V0: "alice", V1: concept.Role_SUPERADMIN, V2: concept.Domain_ALL},
	}}
	store.data[concept.GenAclPolicyKey()] = apicassemdb.NewEntityWithCreated(
		concept.GenAclPolicyKey(), apicassemdb.Must(apicassemdb.Marshal(legacy)), 0, 1)

	rbac, err := newRBAC(store)
	require.NoError(t, err)

	allowed, err := rbac.Enforce("superadmin", concept.Domain_CLUSTER, concept.Object_APP, concept.Action_WRITE)
	require.NoError(t, err)
	require.False(t, allowed)

	allowed, err = rbac.Enforce("alice", concept.Domain_CLUSTER, concept.Object_APP, concept.Action_WRITE)
	require.NoError(t, err)
	require.True(t, allowed)
}

func TestResetUserAllowsLoginWithNewPassword(t *testing.T) {
	store := newACLTestKV()
	rbac, err := newRBAC(store)
	require.NoError(t, err)
	acl := rbac.(aclImpl)

	require.NoError(t, acl.AddUser(&concept.User{
		Account:        "alice@example.com",
		Nickname:       "Alice",
		HashedPassword: "old-secret",
		Status:         concept.User_FORBIDDEN,
	}))

	require.NoError(t, acl.ResetUser("alice@example.com", "new-secret"))
	u, err := acl.GetUser("alice@example.com")
	require.NoError(t, err)
	require.Equal(t, concept.User_NORMAL, u.GetStatus())
	require.Equal(t, u.GetHashedPassword(), hash.WithSalt("new-secret", u.GetSalt()))
	require.NotEqual(t, u.GetHashedPassword(), hash.WithSalt("old-secret", u.GetSalt()))
}

func TestAssignRoleRejectsSuperadmin(t *testing.T) {
	store := newACLTestKV()
	rbac, err := newRBAC(store)
	require.NoError(t, err)
	acl := rbac.(aclImpl)

	require.NoError(t, acl.AddUser(&concept.User{Account: "alice@example.com", Nickname: "Alice", HashedPassword: "secret", Status: concept.User_NORMAL}))

	err = acl.AssignRole("alice@example.com", concept.Role_SUPERADMIN, concept.Domain_ALL)
	require.ErrorIs(t, err, errorx.Err_PERMISSION_DENIED)
}

func TestBootstrapAdminCanAssignSuperadmin(t *testing.T) {
	store := newACLTestKV()
	rbac, err := newRBAC(store)
	require.NoError(t, err)
	acl := rbac.(aclImpl)

	require.NoError(t, acl.BootstrapAdmin("root@example.com", "Root", "secret"))

	roles, err := acl.GetUserRoles("root@example.com")
	require.NoError(t, err)
	require.Contains(t, roles, concept.Role_SUPERADMIN)
}

func TestResetUserRejectsBootstrapSuperadmin(t *testing.T) {
	store := newACLTestKV()
	rbac, err := newRBAC(store)
	require.NoError(t, err)
	acl := rbac.(aclImpl)

	require.NoError(t, acl.BootstrapAdmin("root@example.com", "Root", "secret"))

	err = acl.ResetUser("root@example.com", "new-password")
	require.ErrorIs(t, err, errorx.Err_PERMISSION_DENIED)
}

func TestDisableUserRejectsBootstrapSuperadmin(t *testing.T) {
	store := newACLTestKV()
	rbac, err := newRBAC(store)
	require.NoError(t, err)
	acl := rbac.(aclImpl)

	require.NoError(t, acl.BootstrapAdmin("root@example.com", "Root", "secret"))

	err = acl.DisableUser("root@example.com")
	require.ErrorIs(t, err, errorx.Err_PERMISSION_DENIED)
}

func TestRevokeRoleRejectsBootstrapSuperadmin(t *testing.T) {
	store := newACLTestKV()
	rbac, err := newRBAC(store)
	require.NoError(t, err)
	acl := rbac.(aclImpl)

	require.NoError(t, acl.BootstrapAdmin("root@example.com", "Root", "secret"))

	err = acl.RevokeRole("root@example.com", concept.Role_SUPERADMIN, concept.Domain_ALL)
	require.ErrorIs(t, err, errorx.Err_PERMISSION_DENIED)
}

func TestACLListDomainOptionsPagesAppsAndEnvironments(t *testing.T) {
	store := newACLTestKV()
	for i := range 101 {
		app := fmt.Sprintf("app-%03d", i)
		store.data[concept.GenAppKey(app)] = apicassemdb.NewEntityWithCreated(concept.GenAppKey(app), []byte("app"), 0, 1)
		store.data[concept.GenAppElementEnvKey(app, "prod")] = apicassemdb.NewEntityWithCreated(concept.GenAppElementEnvKey(app, "prod"), []byte("env"), 0, 1)
	}

	rbac, err := newRBAC(store)
	require.NoError(t, err)
	acl := rbac.(aclImpl)

	domains, err := acl.ListDomainOptions()
	require.NoError(t, err)
	require.Contains(t, domains, concept.Domain_CLUSTER)
	require.Contains(t, domains, "app-100/*")
	require.Contains(t, domains, "app-100/prod")
}

func TestACLGetUsersAndRoles(t *testing.T) {
	store := newACLTestKV()
	store.data[concept.GenAppKey("demo")] = apicassemdb.NewEntityWithCreated(concept.GenAppKey("demo"), []byte("app"), 0, 1)
	store.data[concept.GenAppElementEnvKey("demo", "prod")] = apicassemdb.NewEntityWithCreated(concept.GenAppElementEnvKey("demo", "prod"), []byte("env"), 0, 1)

	rbac, err := newRBAC(store)
	require.NoError(t, err)
	acl := rbac.(aclImpl)

	require.NoError(t, acl.AddUser(&concept.User{Account: "alice@example.com", Nickname: "Alice", HashedPassword: "secret-1", Status: concept.User_NORMAL}))
	require.NoError(t, acl.AddUser(&concept.User{Account: "bob@example.com", Nickname: "Bob", HashedPassword: "secret-2", Status: concept.User_FORBIDDEN}))
	require.NoError(t, acl.AssignRole("alice@example.com", concept.Role_ADMIN, concept.Domain_ALL))
	require.NoError(t, acl.AssignRole("alice@example.com", concept.Role_APPOWNER, concept.Domain_CLUSTER))
	require.NoError(t, acl.AssignRole("alice@example.com", "developer", "demo/prod"))

	out, err := acl.GetUsers("", 100)
	require.NoError(t, err)
	require.Len(t, out.Users, 2)

	roles, err := acl.GetUserRoles("alice@example.com")
	require.NoError(t, err)
	require.ElementsMatch(t, []string{concept.Role_ADMIN, concept.Role_APPOWNER, concept.Role_DEVELOPER}, roles)

	bindings, err := acl.GetUserRoleBindings("alice@example.com")
	require.NoError(t, err)
	require.ElementsMatch(t, []concept.RoleBinding{
		{Role: concept.Role_ADMIN, Domain: concept.Domain_ALL},
		{Role: concept.Role_APPOWNER, Domain: concept.Domain_CLUSTER},
		{Role: concept.Role_DEVELOPER, Domain: "demo/prod"},
	}, bindings)

	domains, err := acl.ListDomainOptions()
	require.NoError(t, err)
	require.Contains(t, domains, concept.Domain_CLUSTER)
	require.Contains(t, domains, "demo/*")
	require.Contains(t, domains, "demo/prod")
}
