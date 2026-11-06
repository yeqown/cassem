package coord

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"net/mail"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/casbin/casbin/v2"
	"github.com/casbin/casbin/v2/model"
	"github.com/casbin/casbin/v2/persist"
	"github.com/yeqown/cassem/internal/concept"
	"github.com/yeqown/cassem/internal/kvclient"
	pb "github.com/yeqown/cassem/internal/protogen"
)

const (
	tokenLifetimeDefault = 30 * 24 * time.Hour
	tokenLifetimeMax     = 365 * 24 * time.Hour
	tokenPrefix          = "cassem_"
)

var _ concept.TokenService = (*tokenService)(nil)

type tokenService struct {
	kv pb.KVClient
}

// NewTokenService constructs a credential service over the shared KV leader.
// Authorization always reads current token, user and policy state; construction
// does not migrate or write ACL policies on an agent's behalf.
func NewTokenService(endpoints []string) (concept.TokenService, error) {
	cc, err := kvclient.DialWithMode(endpoints, kvclient.Mode_X)
	if err != nil {
		return nil, fmt.Errorf("NewTokenService: %w", err)
	}
	return &tokenService{kv: pb.NewKVClient(cc)}, nil
}

func validTokenAccount(account string) bool {
	if len(account) > 254 || strings.ContainsAny(account, "/\\\x00") {
		return false
	}
	addr, err := mail.ParseAddress(account)
	return err == nil && addr.Address == account
}

func validTokenID(id string) bool {
	if len(id) != sha256.Size*2 {
		return false
	}
	for _, ch := range id {
		switch {
		case ch >= '0' && ch <= '9', ch >= 'a' && ch <= 'f':
		default:
			return false
		}
	}
	return true
}

// The account component permits per-user pagination with one authoritative
// record per credential. It is only a lookup hint: the digest covers the entire
// bearer, and the stored account must match before it can become an identity.
func tokenDirectory(account string) string {
	return "cassem/acl/tokens/" + base64.RawURLEncoding.EncodeToString([]byte(account))
}

func tokenKey(account, id string) string { return tokenDirectory(account) + "/" + id }

func tokenDigest(raw string) string {
	digest := sha256.Sum256([]byte(raw))
	return hex.EncodeToString(digest[:])
}

func tokenAccount(raw string) (string, bool) {
	if !strings.HasPrefix(raw, tokenPrefix) || len(raw) > 512 {
		return "", false
	}
	encodedAccount, secret, ok := strings.Cut(strings.TrimPrefix(raw, tokenPrefix), ".")
	if !ok {
		return "", false
	}
	account, err := base64.RawURLEncoding.Strict().DecodeString(encodedAccount)
	if err != nil || !validTokenAccount(string(account)) {
		return "", false
	}
	random, err := base64.RawURLEncoding.Strict().DecodeString(secret)
	return string(account), err == nil && len(random) == 32
}

func (s *tokenService) validateUser(ctx context.Context, account string) error {
	resp, err := s.kv.GetKV(ctx, &pb.GetKVReq{Key: concept.GenUserKey(account)})
	if err != nil {
		return err
	}
	user := new(pb.User)
	if resp.GetEntity() == nil || concept.UnmarshalProto(resp.GetEntity().GetVal(), user) != nil || user.GetAccount() != account {
		return fmt.Errorf("invalid token user record: %w", concept.Err_INTERNAL)
	}
	if user.GetStatus() != pb.User_NORMAL {
		return concept.Err_UNAUTHENTICATED
	}
	return nil
}

func (s *tokenService) IssueToken(ctx context.Context, account, name string, expiresIn time.Duration) (*concept.IssuedToken, error) {
	name = strings.TrimSpace(name)
	if expiresIn == 0 {
		expiresIn = tokenLifetimeDefault
	}
	if !validTokenAccount(account) || name == "" || !utf8.ValidString(name) || utf8.RuneCountInString(name) > 100 ||
		expiresIn < time.Second || expiresIn > tokenLifetimeMax {
		return nil, concept.Err_INVALID_ARGUMENT
	}
	if err := s.authorizeManagement(ctx, account, concept.Action_WRITE); err != nil {
		return nil, err
	}
	if err := s.validateUser(ctx, account); err != nil {
		return nil, fmt.Errorf("IssueToken.user: %w", err)
	}
	var secret [32]byte
	if _, err := rand.Read(secret[:]); err != nil {
		return nil, fmt.Errorf("IssueToken.random: %w", concept.Err_INTERNAL)
	}
	raw := tokenPrefix + base64.RawURLEncoding.EncodeToString([]byte(account)) + "." + base64.RawURLEncoding.EncodeToString(secret[:])
	now := time.Now()
	metadata := concept.AccessToken{
		ID: tokenDigest(raw), Account: account, Name: name,
		CreatedAt: now.Unix(), ExpiresAt: now.Add(expiresIn).Unix(),
	}
	data, err := json.Marshal(metadata)
	if err != nil {
		return nil, fmt.Errorf("IssueToken.encode: %w", err)
	}
	if _, err = s.kv.SetKV(ctx, &pb.SetKVReq{Key: tokenKey(account, metadata.ID), Val: data, Overwrite: false}); err != nil {
		return nil, fmt.Errorf("IssueToken.store: %w", err)
	}
	return &concept.IssuedToken{AccessToken: metadata, Token: raw}, nil
}

func decodeToken(data []byte, account, id string) (*concept.AccessToken, error) {
	token := new(concept.AccessToken)
	if json.Unmarshal(data, token) != nil || !validTokenID(token.ID) || token.Account != account ||
		(id != "" && token.ID != id) || token.CreatedAt <= 0 || token.ExpiresAt <= token.CreatedAt {
		return nil, fmt.Errorf("invalid token record: %w", concept.Err_INTERNAL)
	}
	return token, nil
}

func (s *tokenService) GetTokens(ctx context.Context, account, seek string, limit int) (*concept.GetTokensResult, error) {
	if !validTokenAccount(account) || (seek != "" && !validTokenID(seek)) || limit < 1 || limit > 100 {
		return nil, concept.Err_INVALID_ARGUMENT
	}
	if err := s.authorizeManagement(ctx, account, concept.Action_READ); err != nil {
		return nil, err
	}
	result := &concept.GetTokensResult{Tokens: make([]*concept.AccessToken, 0)}
	resp, err := s.kv.Range(ctx, &pb.RangeReq{Key: tokenDirectory(account), Seek: seek, Limit: int32(limit)})
	if errors.Is(err, concept.Err_NOT_FOUND) {
		return result, nil
	}
	if err != nil {
		return nil, fmt.Errorf("GetTokens: %w", err)
	}
	for _, entity := range resp.GetEntities() {
		token, err := decodeToken(entity.GetVal(), account, concept.ExtractPureKey(entity.GetKey()))
		if err != nil {
			return nil, err
		}
		result.Tokens = append(result.Tokens, token)
	}
	result.HasMore = resp.GetHasMore()
	result.NextSeek = resp.GetNextSeekKey()
	return result, nil
}

func (s *tokenService) RevokeToken(ctx context.Context, account, id string) error {
	if !validTokenAccount(account) || !validTokenID(id) {
		return concept.Err_INVALID_ARGUMENT
	}
	if err := s.authorizeManagement(ctx, account, concept.Action_WRITE); err != nil {
		return err
	}
	_, err := s.kv.UnsetKV(ctx, &pb.UnsetKVReq{Key: tokenKey(account, id)})
	if errors.Is(err, concept.Err_NOT_FOUND) {
		return nil
	}
	if err != nil {
		return fmt.Errorf("RevokeToken: %w", err)
	}
	return nil
}

func (s *tokenService) AuthorizeToken(ctx context.Context, raw string, targets ...concept.TokenTarget) error {
	ctx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()
	account, ok := tokenAccount(raw)
	if !ok {
		return concept.Err_UNAUTHENTICATED
	}
	id := tokenDigest(raw)
	resp, err := s.kv.GetKV(ctx, &pb.GetKVReq{Key: tokenKey(account, id)})
	if errors.Is(err, concept.Err_NOT_FOUND) {
		return concept.Err_UNAUTHENTICATED
	}
	if err != nil {
		return fmt.Errorf("AuthorizeToken.lookup: %w", err)
	}
	token, err := decodeToken(resp.GetEntity().GetVal(), account, id)
	if err != nil {
		return err
	}
	if time.Now().Unix() >= token.ExpiresAt {
		return concept.Err_UNAUTHENTICATED
	}
	if err = s.validateUser(ctx, account); err != nil {
		if errors.Is(err, concept.Err_NOT_FOUND) {
			return concept.Err_UNAUTHENTICATED
		}
		return fmt.Errorf("AuthorizeToken.user: %w", err)
	}
	if len(targets) == 0 {
		return nil
	}
	enforcer, err := s.currentPolicy(ctx)
	if err != nil {
		return err
	}
	for _, target := range targets {
		if target.App == "" || target.Env == "" || strings.ContainsAny(target.App+target.Env, "/\\*") {
			return concept.Err_INVALID_ARGUMENT
		}
		allow, err := enforcer.Enforce(account, target.App+"/"+target.Env, concept.Object_ELEMENT, concept.Action_READ)
		if err != nil {
			return fmt.Errorf("AuthorizeToken.policy: %w", concept.Err_INTERNAL)
		}
		if !allow {
			return concept.Err_PERMISSION_DENIED
		}
	}
	return nil
}

func (s *tokenService) currentPolicy(ctx context.Context) (*casbin.Enforcer, error) {
	resp, err := s.kv.GetKV(ctx, &pb.GetKVReq{Key: concept.GenAclPolicyKey()})
	if errors.Is(err, concept.Err_NOT_FOUND) {
		return nil, concept.Err_PERMISSION_DENIED
	}
	if err != nil {
		return nil, fmt.Errorf("AuthorizeToken.loadPolicy: %w", err)
	}
	policy := new(pb.Casbin)
	if resp.GetEntity() == nil || concept.UnmarshalProto(resp.GetEntity().GetVal(), policy) != nil {
		return nil, fmt.Errorf("invalid ACL policy record: %w", concept.Err_INTERNAL)
	}
	m, err := model.NewModelFromString(_casbinModel)
	if err != nil {
		return nil, fmt.Errorf("AuthorizeToken.model: %w", concept.Err_INTERNAL)
	}
	for _, rule := range policy.GetPolicies() {
		values := []string{rule.GetPtype(), rule.GetV0(), rule.GetV1(), rule.GetV2(), rule.GetV3(), rule.GetV4(), rule.GetV5()}
		switch values[0] {
		case "p":
			values[1] = rbacRole(values[1])
		case "g":
			values[2] = rbacRole(values[2])
		default:
			return nil, fmt.Errorf("invalid ACL policy type: %w", concept.Err_INTERNAL)
		}
		for len(values) > 1 && values[len(values)-1] == "" {
			values = values[:len(values)-1]
		}
		if err = persist.LoadPolicyArray(values, m); err != nil {
			return nil, fmt.Errorf("invalid ACL policy rule: %w", concept.Err_INTERNAL)
		}
	}
	e, err := casbin.NewEnforcer(m)
	if err != nil {
		return nil, fmt.Errorf("AuthorizeToken.enforcer: %w", concept.Err_INTERNAL)
	}
	configureRBACDomains(e)
	if err = e.BuildRoleLinks(); err != nil {
		return nil, fmt.Errorf("AuthorizeToken.roleLinks: %w", concept.Err_INTERNAL)
	}
	return e, nil
}
