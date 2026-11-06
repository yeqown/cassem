package adm

import (
	"encoding/json"
	"fmt"
	"net/http"
	"regexp"
	"strings"
	"time"

	"github.com/yeqown/cassem/internal/concept"
	"github.com/yeqown/cassem/internal/pkg/httpx"
)

const defaultTokenLifetime = 30 * 24 * time.Hour

var tokenDigestPattern = regexp.MustCompile(`^[0-9a-f]{64}$`)

type issueTokenRequest struct {
	Account          string          `uri:"account" json:"-" binding:"required,email"`
	Name             string          `json:"name" binding:"required,max=100"`
	ExpiresInSeconds json.RawMessage `json:"expiresInSeconds"`
}

type listTokensRequest struct {
	Account string `uri:"account" json:"-" binding:"required,email"`
	Limit   int    `form:"limit,default=100" json:"-" binding:"min=1,max=100"`
	Seek    string `form:"seek" json:"-"`
}

type revokeTokenRequest struct {
	Account string `uri:"account" json:"-" binding:"required,email"`
	TokenID string `uri:"tokenId" json:"-"`
}

func tokenNoStore(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Cache-Control", "no-store")
		next.ServeHTTP(w, r)
	})
}

func bindTokenRequest(r *http.Request, req any) error {
	if err := bindRequest(r, req); err != nil {
		return fmt.Errorf("invalid token request: %v: %w", err, concept.Err_INVALID_ARGUMENT)
	}
	return nil
}

// IssueTokenHTTP returns a newly issued access token. Its secret is only returned here.
func (d app) IssueTokenHTTP(w http.ResponseWriter, r *http.Request) {
	req := new(issueTokenRequest)
	if err := bindTokenRequest(r, req); err != nil {
		httpx.WriteError(w, err)
		return
	}
	if strings.TrimSpace(req.Name) == "" {
		httpx.WriteError(w, concept.New(concept.Code_INVALID_ARGUMENT, "token name must not be empty"))
		return
	}
	lifetime := defaultTokenLifetime
	if len(req.ExpiresInSeconds) != 0 {
		var seconds int64
		if err := json.Unmarshal(req.ExpiresInSeconds, &seconds); err != nil || seconds < 1 || seconds > 31536000 {
			httpx.WriteError(w, concept.New(concept.Code_INVALID_ARGUMENT, "expiresInSeconds must be an integer between 1 and 31536000"))
			return
		}
		lifetime = time.Duration(seconds) * time.Second
	}
	out, err := d.tokens.IssueToken(r.Context(), req.Account, req.Name, lifetime)
	if err != nil {
		httpx.WriteError(w, err)
		return
	}
	httpx.WriteJSON(w, out)
}

// GetTokensHTTP lists token metadata for the account, without bearer secrets.
func (d app) GetTokensHTTP(w http.ResponseWriter, r *http.Request) {
	req := new(listTokensRequest)
	if err := bindTokenRequest(r, req); err != nil {
		httpx.WriteError(w, err)
		return
	}
	if req.Seek != "" && !tokenDigestPattern.MatchString(req.Seek) {
		httpx.WriteError(w, concept.New(concept.Code_INVALID_ARGUMENT, "seek must be a token ID"))
		return
	}
	out, err := d.tokens.GetTokens(r.Context(), req.Account, req.Seek, req.Limit)
	if err != nil {
		httpx.WriteError(w, err)
		return
	}
	httpx.WriteJSON(w, out)
}

// RevokeTokenHTTP revokes the token scoped to the path account.
func (d app) RevokeTokenHTTP(w http.ResponseWriter, r *http.Request) {
	req := new(revokeTokenRequest)
	if err := bindTokenRequest(r, req); err != nil {
		httpx.WriteError(w, err)
		return
	}
	if !tokenDigestPattern.MatchString(req.TokenID) {
		httpx.WriteError(w, concept.New(concept.Code_INVALID_ARGUMENT, "tokenId must be a token ID"))
		return
	}
	if err := d.tokens.RevokeToken(r.Context(), req.Account, req.TokenID); err != nil {
		httpx.WriteError(w, err)
		return
	}
	httpx.WriteJSON(w, nil)
}
