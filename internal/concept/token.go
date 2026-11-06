package concept

import (
	"context"
	"time"
)

// TokenTarget is an application/environment whose configuration is requested.
type TokenTarget struct {
	App string
	Env string
}

// TokenAuthorizer checks a bearer token and the current user's READ permissions.
// With no targets it checks only token validity and the user's enabled status.
type TokenAuthorizer interface {
	AuthorizeToken(ctx context.Context, token string, targets ...TokenTarget) error
}

// TokenManager manages user-bound credentials. Durations of zero use the default
// lifetime; callers must keep the bearer returned by IssueToken themselves.
// Management contexts must carry an authenticated operator via WithOperator.
type TokenManager interface {
	IssueToken(ctx context.Context, account, name string, expiresIn time.Duration) (*IssuedToken, error)
	GetTokens(ctx context.Context, account, seek string, limit int) (*GetTokensResult, error)
	RevokeToken(ctx context.Context, account, id string) error
}

// TokenService provides management and authorization over shared token storage.
type TokenService interface {
	TokenManager
	TokenAuthorizer
}

// AccessToken contains public credential metadata, never the bearer value.
type AccessToken struct {
	ID        string `json:"id"`
	Account   string `json:"account"`
	Name      string `json:"name"`
	CreatedAt int64  `json:"createdAt"`
	ExpiresAt int64  `json:"expiresAt"`
}

// IssuedToken includes a bearer that is returned only at creation time.
type IssuedToken struct {
	AccessToken
	Token string `json:"token"`
}

// GetTokensResult is a page of token metadata for a single user.
type GetTokensResult struct {
	CommonPager
	Tokens []*AccessToken `json:"tokens"`
}
