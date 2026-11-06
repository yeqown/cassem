package coord

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"strings"
	"testing"

	storage "github.com/yeqown/cassem/internal/app/kv/storage"
	pb "github.com/yeqown/cassem/internal/protogen"

	"buf.build/go/protovalidate"
	"github.com/stretchr/testify/require"
	"github.com/yeqown/cassem/internal/concept"
	"github.com/yeqown/cassem/internal/pkg/hash"
	"google.golang.org/grpc"
)

type aclTestKV struct {
	data   map[string]*pb.Entity
	setErr error
}

func newACLTestKV() *aclTestKV {
	return &aclTestKV{data: make(map[string]*pb.Entity)}
}

func (f *aclTestKV) GetKV(_ context.Context, req *pb.GetKVReq, _ ...grpc.CallOption) (*pb.GetKVResp, error) {
	entity, ok := f.data[req.GetKey()]
	if !ok {
		return nil, concept.Err_NOT_FOUND
	}
	return &pb.GetKVResp{Entity: entity}, nil
}

func (f *aclTestKV) GetKVs(context.Context, *pb.GetKVsReq, ...grpc.CallOption) (*pb.GetKVsResp, error) {
	return nil, errors.New("unused")
}

func (f *aclTestKV) SetKV(_ context.Context, req *pb.SetKVReq, _ ...grpc.CallOption) (*pb.Empty, error) {
	if f.setErr != nil {
		return nil, f.setErr
	}
	f.data[req.GetKey()] = storage.NewEntityWithCreated(req.GetKey(), req.GetVal(), 0, 1)
	return &pb.Empty{}, nil
}

func (f *aclTestKV) UnsetKV(context.Context, *pb.UnsetKVReq, ...grpc.CallOption) (*pb.Empty, error) {
	return nil, errors.New("unused")
}

func (f *aclTestKV) Watch(context.Context, *pb.WatchReq, ...grpc.CallOption) (pb.KV_WatchClient, error) {
	return nil, errors.New("unused")
}

func (f *aclTestKV) TTL(context.Context, *pb.TtlReq, ...grpc.CallOption) (*pb.TtlResp, error) {
	return nil, errors.New("unused")
}

func (f *aclTestKV) Expire(context.Context, *pb.ExpireReq, ...grpc.CallOption) (*pb.Empty, error) {
	return nil, errors.New("unused")
}

func (f *aclTestKV) Range(_ context.Context, req *pb.RangeReq, _ ...grpc.CallOption) (*pb.RangeResp, error) {
	if err := protovalidate.Validate(req); err != nil {
		return nil, err
	}

	entities := make([]*pb.Entity, 0)
	for key, entity := range f.data {
		if req.GetKey() != "" && !strings.HasPrefix(key, req.GetKey()) {
			continue
		}
		if req.GetSeek() != "" && concept.ExtractPureKey(key) < req.GetSeek() {
			continue
		}
		entities = append(entities, entity)
	}
	slices.SortFunc(entities, func(a, b *pb.Entity) int { return strings.Compare(a.GetKey(), b.GetKey()) })
	if len(entities) > int(req.GetLimit()) {
		return &pb.RangeResp{
			Entities:    entities[:req.GetLimit()],
			HasMore:     true,
			NextSeekKey: concept.ExtractPureKey(entities[req.GetLimit()].GetKey()),
		}, nil
	}
	return &pb.RangeResp{Entities: entities}, nil
}

func (f *aclTestKV) CompactElementHistory(context.Context, *pb.CompactElementHistoryReq, ...grpc.CallOption) (*pb.CompactElementHistoryResp, error) {
	return nil, errors.New("unused")
}

func TestACLRejectsHardcodedSuperadminUser(t *testing.T) {
	acl, err := newRBAC(newACLTestKV())
	require.NoError(t, err)

	_, err = acl.GetUser("superadmin")
	require.ErrorIs(t, err, concept.Err_NOT_FOUND)

	_, err = acl.GetUser("superadmin-anything")
	require.ErrorIs(t, err, concept.Err_NOT_FOUND)
}

func TestACLDoesNotBypassBySuperadminSubject(t *testing.T) {
	acl, err := newRBAC(newACLTestKV())
	require.NoError(t, err)

	allowed, err := acl.Enforce("superadmin", concept.Domain_CLUSTER, concept.Object_APP, concept.Action_WRITE)
	require.NoError(t, err)
	require.False(t, allowed)
}

func TestACLEnforceReturnsEngineErrors(t *testing.T) {
	acl, err := newRBAC(newACLTestKV())
	require.NoError(t, err)
	impl := acl.(aclImpl)
	impl.e.GetModel()["m"]["m"].Value = "missing_func(r.sub)"

	allowed, err := impl.Enforce("alice", concept.Domain_CLUSTER, concept.Object_APP, concept.Action_READ)
	require.Error(t, err)
	require.False(t, allowed)
	require.Contains(t, err.Error(), "missing_func")
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
	legacy := &pb.Casbin{Policies: []*pb.Casbin_Policy{
		{Ptype: "p", V0: concept.Role_SUPERADMIN, V1: concept.Domain_ALL, V2: concept.Object_ALL, V3: concept.Action_ANY},
		{Ptype: "g", V0: "alice", V1: concept.Role_SUPERADMIN, V2: concept.Domain_ALL},
	}}
	store.data[concept.GenAclPolicyKey()] = storage.NewEntityWithCreated(
		concept.GenAclPolicyKey(), concept.MustMarshalProto(legacy), 0, 1)

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

	require.NoError(t, acl.AddUser(&pb.User{
		Account:        "alice@example.com",
		Nickname:       "Alice",
		HashedPassword: "old-secret",
		Status:         pb.User_FORBIDDEN,
	}))

	require.NoError(t, acl.ResetUser("alice@example.com", "new-secret"))
	u, err := acl.GetUser("alice@example.com")
	require.NoError(t, err)
	require.Equal(t, pb.User_NORMAL, u.GetStatus())
	require.Equal(t, u.GetHashedPassword(), hash.WithSalt("new-secret", u.GetSalt()))
	require.NotEqual(t, u.GetHashedPassword(), hash.WithSalt("old-secret", u.GetSalt()))
}

func TestAssignRoleRejectsSuperadmin(t *testing.T) {
	store := newACLTestKV()
	rbac, err := newRBAC(store)
	require.NoError(t, err)
	acl := rbac.(aclImpl)

	require.NoError(t, acl.AddUser(&pb.User{Account: "alice@example.com", Nickname: "Alice", HashedPassword: "secret", Status: pb.User_NORMAL}))

	err = acl.AssignRole("alice@example.com", concept.Role_SUPERADMIN, concept.Domain_ALL)
	require.ErrorIs(t, err, concept.Err_PERMISSION_DENIED)
}

func TestAssignAndRevokeRoleReturnSavePolicyErrors(t *testing.T) {
	store := newACLTestKV()
	rbac, err := newRBAC(store)
	require.NoError(t, err)
	acl := rbac.(aclImpl)
	store.setErr = concept.Err_INTERNAL

	err = acl.AssignRole("alice@example.com", concept.Role_ADMIN, concept.Domain_ALL)
	require.ErrorIs(t, err, concept.Err_INTERNAL)
	require.Contains(t, err.Error(), "aclImpl.AssignRole")

	store.setErr = nil
	require.NoError(t, acl.AssignRole("bob@example.com", concept.Role_ADMIN, concept.Domain_ALL))
	store.setErr = concept.Err_INTERNAL

	err = acl.RevokeRole("bob@example.com", concept.Role_ADMIN, concept.Domain_ALL)
	require.ErrorIs(t, err, concept.Err_INTERNAL)
	require.Contains(t, err.Error(), "aclImpl.RevokeRole")
}

func TestACLVisitorRoleIsReadOnlyWithinAppEnvironment(t *testing.T) {
	store := newACLTestKV()
	rbac, err := newRBAC(store)
	require.NoError(t, err)
	acl := rbac.(aclImpl)

	require.NoError(t, acl.AddUser(&pb.User{
		Account:        "viewer@example.com",
		Nickname:       "Viewer",
		HashedPassword: "secret",
		Status:         pb.User_NORMAL,
	}))
	require.NoError(t, acl.AssignRole("viewer@example.com", concept.Role_VISITOR, "payment-service/production"))

	allow, err := acl.Enforce("viewer@example.com", "payment-service/production", concept.Object_APP, concept.Action_READ)
	require.NoError(t, err)
	require.True(t, allow)

	allow, err = acl.Enforce("viewer@example.com", "payment-service/production", concept.Object_ELEMENT, concept.Action_READ)
	require.NoError(t, err)
	require.True(t, allow)

	allow, err = acl.Enforce("viewer@example.com", "payment-service/production", concept.Object_ELEMENT, concept.Action_WRITE)
	require.NoError(t, err)
	require.False(t, allow)

	allow, err = acl.Enforce("viewer@example.com", "payment-service/production", concept.Object_ELEMENT, concept.Action_PUBLISH)
	require.NoError(t, err)
	require.False(t, allow)
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
	require.ErrorIs(t, err, concept.Err_PERMISSION_DENIED)
}

func TestDisableUserRejectsBootstrapSuperadmin(t *testing.T) {
	store := newACLTestKV()
	rbac, err := newRBAC(store)
	require.NoError(t, err)
	acl := rbac.(aclImpl)

	require.NoError(t, acl.BootstrapAdmin("root@example.com", "Root", "secret"))

	err = acl.DisableUser("root@example.com")
	require.ErrorIs(t, err, concept.Err_PERMISSION_DENIED)
}

func TestRevokeRoleRejectsBootstrapSuperadmin(t *testing.T) {
	store := newACLTestKV()
	rbac, err := newRBAC(store)
	require.NoError(t, err)
	acl := rbac.(aclImpl)

	require.NoError(t, acl.BootstrapAdmin("root@example.com", "Root", "secret"))

	err = acl.RevokeRole("root@example.com", concept.Role_SUPERADMIN, concept.Domain_ALL)
	require.ErrorIs(t, err, concept.Err_PERMISSION_DENIED)
}

func TestACLListDomainOptionsPagesAppsAndEnvironments(t *testing.T) {
	store := newACLTestKV()
	for i := range 101 {
		app := fmt.Sprintf("app-%03d", i)
		store.data[concept.GenAppKey(app)] = storage.NewEntityWithCreated(concept.GenAppKey(app), []byte("app"), 0, 1)
		store.data[concept.GenAppElementEnvKey(app, "prod")] = storage.NewEntityWithCreated(concept.GenAppElementEnvKey(app, "prod"), []byte("env"), 0, 1)
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
	store.data[concept.GenAppKey("demo")] = storage.NewEntityWithCreated(concept.GenAppKey("demo"), []byte("app"), 0, 1)
	store.data[concept.GenAppElementEnvKey("demo", "prod")] = storage.NewEntityWithCreated(concept.GenAppElementEnvKey("demo", "prod"), []byte("env"), 0, 1)

	rbac, err := newRBAC(store)
	require.NoError(t, err)
	acl := rbac.(aclImpl)

	require.NoError(t, acl.AddUser(&pb.User{Account: "alice@example.com", Nickname: "Alice", HashedPassword: "secret-1", Status: pb.User_NORMAL}))
	require.NoError(t, acl.AddUser(&pb.User{Account: "bob@example.com", Nickname: "Bob", HashedPassword: "secret-2", Status: pb.User_FORBIDDEN}))
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
