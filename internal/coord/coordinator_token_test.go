package coord

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/yeqown/cassem/internal/concept"
	pb "github.com/yeqown/cassem/internal/protogen"
	"google.golang.org/grpc"
)

type tokenTestKV struct {
	*aclTestKV
	readErr  error
	deadline time.Time
}

func (s *tokenTestKV) GetKV(ctx context.Context, req *pb.GetKVReq, opts ...grpc.CallOption) (*pb.GetKVResp, error) {
	s.deadline, _ = ctx.Deadline()
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if s.readErr != nil {
		return nil, s.readErr
	}
	return s.aclTestKV.GetKV(ctx, req, opts...)
}

func TestTokenAuthorizationBoundsStreamContext(t *testing.T) {
	service, store, _ := tokenFixture(t)
	issued, err := service.IssueToken(tokenManagementContext(), "reader@example.com", "worker", time.Hour)
	require.NoError(t, err)
	require.NoError(t, service.AuthorizeToken(context.Background(), issued.Token, concept.TokenTarget{App: "checkout", Env: "prod"}))
	require.False(t, store.deadline.IsZero(), "authorization on a long-lived stream must bound storage calls")
	require.LessOrEqual(t, time.Until(store.deadline), 3*time.Second)
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	deadline, _ := ctx.Deadline()
	require.NoError(t, service.AuthorizeToken(ctx, issued.Token))
	require.Equal(t, deadline, store.deadline)
}

func (s *tokenTestKV) UnsetKV(_ context.Context, req *pb.UnsetKVReq, _ ...grpc.CallOption) (*pb.Empty, error) {
	if _, ok := s.data[req.GetKey()]; !ok {
		return nil, concept.Err_NOT_FOUND
	}
	delete(s.data, req.GetKey())
	return &pb.Empty{}, nil
}

func tokenFixture(t *testing.T) (*tokenService, *tokenTestKV, concept.RBAC) {
	t.Helper()
	store := &tokenTestKV{aclTestKV: newACLTestKV()}
	rbac, err := newRBAC(store)
	require.NoError(t, err)
	require.NoError(t, rbac.BootstrapAdmin("issuer@example.com", "Issuer", "password"))
	require.NoError(t, rbac.AddUser(&pb.User{Account: "reader@example.com", Status: pb.User_NORMAL, HashedPassword: "password"}))
	require.NoError(t, rbac.AssignRole("reader@example.com", concept.Role_VISITOR, "checkout/prod"))
	return &tokenService{kv: store}, store, rbac
}

func tokenManagementContext() context.Context {
	return concept.WithOperator(context.Background(), "issuer@example.com")
}

func TestTokenManagementRejectsProtectedSuperadminTarget(t *testing.T) {
	service, _, rbac := tokenFixture(t)
	require.NoError(t, rbac.BootstrapAdmin("root@example.com", "Root", "password"))
	require.NoError(t, rbac.AddUser(&pb.User{Account: "admin@example.com", Status: pb.User_NORMAL, HashedPassword: "password"}))
	require.NoError(t, rbac.AssignRole("admin@example.com", concept.Role_ADMIN, concept.Domain_CLUSTER))
	adminCtx := concept.WithOperator(context.Background(), "admin@example.com")
	issued, err := service.IssueToken(adminCtx, "root@example.com", "escalation", time.Hour)
	require.ErrorIs(t, err, concept.Err_PERMISSION_DENIED)
	require.Nil(t, issued)
	rootCtx := concept.WithOperator(context.Background(), "root@example.com")
	issued, err = service.IssueToken(rootCtx, "root@example.com", "root-worker", time.Hour)
	require.NoError(t, err)
	require.ErrorIs(t, service.RevokeToken(adminCtx, "root@example.com", issued.ID), concept.Err_PERMISSION_DENIED)
	require.NoError(t, service.AuthorizeToken(context.Background(), issued.Token, concept.TokenTarget{App: "arbitrary", Env: "prod"}))
	require.NoError(t, service.RevokeToken(rootCtx, "root@example.com", issued.ID))
}

func TestTokenManagementUsesCurrentIssuerPermissions(t *testing.T) {
	service, _, rbac := tokenFixture(t)
	require.NoError(t, rbac.AddUser(&pb.User{Account: "admin@example.com", Status: pb.User_NORMAL, HashedPassword: "password"}))
	require.NoError(t, rbac.AssignRole("admin@example.com", concept.Role_ADMIN, concept.Domain_CLUSTER))
	ctx := concept.WithOperator(context.Background(), "admin@example.com")
	issued, err := service.IssueToken(ctx, "reader@example.com", "reader-worker", time.Hour)
	require.NoError(t, err)
	require.NoError(t, rbac.RevokeRole("admin@example.com", concept.Role_ADMIN, concept.Domain_CLUSTER))
	_, err = service.IssueToken(ctx, "reader@example.com", "after-revoke", time.Hour)
	require.ErrorIs(t, err, concept.Err_PERMISSION_DENIED)
	require.ErrorIs(t, service.RevokeToken(ctx, "reader@example.com", issued.ID), concept.Err_PERMISSION_DENIED)
	_, err = service.GetTokens(ctx, "reader@example.com", "", 100)
	require.ErrorIs(t, err, concept.Err_PERMISSION_DENIED)
	_, err = service.IssueToken(context.Background(), "reader@example.com", "anonymous", time.Hour)
	require.ErrorIs(t, err, concept.Err_UNAUTHENTICATED)
}

func TestTokenLifecycleStoresNoBearerAndRevokesAcrossServices(t *testing.T) {
	service, store, _ := tokenFixture(t)
	ctx := tokenManagementContext()
	issued, err := service.IssueToken(ctx, "reader@example.com", "worker", time.Hour)
	require.NoError(t, err)
	require.Len(t, issued.ID, 64)
	require.NotEmpty(t, issued.Token)
	require.Equal(t, "reader@example.com", issued.Account)
	require.InDelta(t, time.Now().Add(time.Hour).Unix(), issued.ExpiresAt, 2)
	for _, entity := range store.data {
		require.NotContains(t, string(entity.GetVal()), issued.Token)
	}
	otherNode := &tokenService{kv: store}
	require.NoError(t, otherNode.AuthorizeToken(ctx, issued.Token, concept.TokenTarget{App: "checkout", Env: "prod"}))
	page, err := service.GetTokens(ctx, "reader@example.com", "", 100)
	require.NoError(t, err)
	require.Len(t, page.Tokens, 1)
	require.Equal(t, issued.ID, page.Tokens[0].ID)
	raw, err := json.Marshal(page)
	require.NoError(t, err)
	require.NotContains(t, string(raw), issued.Token)
	require.NoError(t, service.RevokeToken(ctx, "reader@example.com", issued.ID))
	require.NoError(t, service.RevokeToken(ctx, "reader@example.com", issued.ID))
	require.ErrorIs(t, otherNode.AuthorizeToken(ctx, issued.Token), concept.Err_UNAUTHENTICATED)
	page, err = service.GetTokens(ctx, "reader@example.com", "", 100)
	require.NoError(t, err)
	require.Empty(t, page.Tokens)
}

func TestTokenEnforcesCurrentUserDomainsAndPolicyChanges(t *testing.T) {
	service, _, rbac := tokenFixture(t)
	ctx := tokenManagementContext()
	issued, err := service.IssueToken(ctx, "reader@example.com", "worker", time.Hour)
	require.NoError(t, err)
	for _, target := range []concept.TokenTarget{
		{App: "checkout", Env: "staging"},
		{App: "payments", Env: "prod"},
	} {
		require.ErrorIs(t, service.AuthorizeToken(ctx, issued.Token, target), concept.Err_PERMISSION_DENIED)
	}
	require.ErrorIs(t, service.AuthorizeToken(ctx, issued.Token,
		concept.TokenTarget{App: "checkout", Env: "prod"},
		concept.TokenTarget{App: "payments", Env: "prod"}), concept.Err_PERMISSION_DENIED)
	require.NoError(t, rbac.RevokeRole("reader@example.com", concept.Role_VISITOR, "checkout/prod"))
	require.ErrorIs(t, service.AuthorizeToken(ctx, issued.Token, concept.TokenTarget{App: "checkout", Env: "prod"}), concept.Err_PERMISSION_DENIED)
	require.NoError(t, rbac.AssignRole("reader@example.com", concept.Role_VISITOR, "checkout/*"))
	require.NoError(t, service.AuthorizeToken(ctx, issued.Token, concept.TokenTarget{App: "checkout", Env: "staging"}))
	require.NoError(t, rbac.DisableUser("reader@example.com"))
	require.ErrorIs(t, service.AuthorizeToken(ctx, issued.Token), concept.Err_UNAUTHENTICATED)
}

func TestTokenRejectsExpiredTamperedAndDeletedUsers(t *testing.T) {
	service, store, _ := tokenFixture(t)
	ctx := tokenManagementContext()
	issued, err := service.IssueToken(ctx, "reader@example.com", "worker", time.Hour)
	require.NoError(t, err)
	for _, raw := range []string{"", "not-a-token", issued.Token + "x", strings.Replace(issued.Token, ".", "/", 1)} {
		require.ErrorIs(t, service.AuthorizeToken(ctx, raw), concept.Err_UNAUTHENTICATED)
	}
	for _, entity := range store.data {
		var token concept.AccessToken
		if json.Unmarshal(entity.GetVal(), &token) == nil && token.ID == issued.ID {
			token.CreatedAt = time.Now().Add(-2 * time.Hour).Unix()
			token.ExpiresAt = time.Now().Add(-time.Second).Unix()
			entity.Val, err = json.Marshal(token)
			require.NoError(t, err)
		}
	}
	require.ErrorIs(t, service.AuthorizeToken(ctx, issued.Token), concept.Err_UNAUTHENTICATED)
	issued, err = service.IssueToken(ctx, "reader@example.com", "second", time.Hour)
	require.NoError(t, err)
	delete(store.data, concept.GenUserKey("reader@example.com"))
	require.ErrorIs(t, service.AuthorizeToken(ctx, issued.Token), concept.Err_UNAUTHENTICATED)
}

func TestTokenListAndRevokeAreScopedToUser(t *testing.T) {
	service, _, rbac := tokenFixture(t)
	ctx := tokenManagementContext()
	require.NoError(t, rbac.AddUser(&pb.User{Account: "other@example.com", Status: pb.User_NORMAL, HashedPassword: "password"}))
	first, err := service.IssueToken(ctx, "reader@example.com", "one", time.Hour)
	require.NoError(t, err)
	second, err := service.IssueToken(ctx, "reader@example.com", "two", time.Hour)
	require.NoError(t, err)
	other, err := service.IssueToken(ctx, "other@example.com", "other", time.Hour)
	require.NoError(t, err)
	page, err := service.GetTokens(ctx, "reader@example.com", "", 1)
	require.NoError(t, err)
	require.Len(t, page.Tokens, 1)
	require.True(t, page.HasMore)
	require.NotEmpty(t, page.NextSeek)
	rest, err := service.GetTokens(ctx, "reader@example.com", page.NextSeek, 1)
	require.NoError(t, err)
	require.Len(t, rest.Tokens, 1)
	require.False(t, rest.HasMore)
	require.ElementsMatch(t, []string{first.ID, second.ID}, []string{page.Tokens[0].ID, rest.Tokens[0].ID})
	require.NoError(t, service.RevokeToken(ctx, "reader@example.com", other.ID))
	require.NoError(t, service.AuthorizeToken(ctx, other.Token))
}

func TestTokenServiceFailsClosedOnStorageFailureAndCancellation(t *testing.T) {
	service, store, _ := tokenFixture(t)
	issued, err := service.IssueToken(tokenManagementContext(), "reader@example.com", "worker", time.Hour)
	require.NoError(t, err)
	store.readErr = concept.Err_UNAVAILABLE
	require.ErrorIs(t, service.AuthorizeToken(context.Background(), issued.Token), concept.Err_UNAVAILABLE)
	store.readErr = nil
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	require.ErrorIs(t, service.AuthorizeToken(ctx, issued.Token), context.Canceled)
	store.setErr = errors.New("write unavailable")
	_, err = service.IssueToken(tokenManagementContext(), "reader@example.com", "worker", time.Hour)
	require.ErrorIs(t, err, store.setErr)
}

func TestTokenIssueRejectsInvalidInputs(t *testing.T) {
	service, _, rbac := tokenFixture(t)
	for _, tc := range []struct {
		name, account, label string
		lifetime             time.Duration
	}{
		{"blank name", "reader@example.com", " ", time.Hour},
		{"long name", "reader@example.com", strings.Repeat("x", 101), time.Hour},
		{"negative expiry", "reader@example.com", "name", -time.Second},
		{"overlong expiry", "reader@example.com", "name", 366 * 24 * time.Hour},
		{"path traversal", "../reader@example.com", "name", time.Hour},
	} {
		t.Run(tc.name, func(t *testing.T) {
			_, err := service.IssueToken(tokenManagementContext(), tc.account, tc.label, tc.lifetime)
			require.ErrorIs(t, err, concept.Err_INVALID_ARGUMENT)
		})
	}
	issued, err := service.IssueToken(tokenManagementContext(), "reader@example.com", "default", 0)
	require.NoError(t, err)
	require.InDelta(t, time.Now().Add(30*24*time.Hour).Unix(), issued.ExpiresAt, 2)
	require.NoError(t, rbac.DisableUser("reader@example.com"))
	_, err = service.IssueToken(tokenManagementContext(), "reader@example.com", "disabled", time.Hour)
	require.ErrorIs(t, err, concept.Err_UNAUTHENTICATED)
}
