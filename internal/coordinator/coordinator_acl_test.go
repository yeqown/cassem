package coordinator

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"

	"github.com/yeqown/cassem/api/concept"
	apicassemdb "github.com/yeqown/cassem/internal/cassemdb/api"
	"github.com/yeqown/cassem/pkg/errorx"
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

func (f *aclTestKV) Range(context.Context, *apicassemdb.RangeReq, ...grpc.CallOption) (*apicassemdb.RangeResp, error) {
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

	require.NoError(t, reloaded.AssignRole("alice", "superadmin", concept.Domain_ALL))
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

func TestResetUserRejectsBootstrapSuperadmin(t *testing.T) {
	store := newACLTestKV()
	rbac, err := newRBAC(store)
	require.NoError(t, err)
	acl := rbac.(aclImpl)

	require.NoError(t, acl.AddUser(&concept.User{
		Account:        "superadmin@example.com",
		Nickname:       "superadmin",
		HashedPassword: "cassem",
		Status:         concept.User_NORMAL,
	}))
	require.NoError(t, acl.AssignRole("superadmin@example.com", concept.Role_SUPERADMIN, concept.Domain_ALL))

	err = acl.ResetUser("superadmin@example.com", "new-password")
	require.ErrorIs(t, err, errorx.Err_PERMISSION_DENIED)
}
