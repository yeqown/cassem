package adm

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/yeqown/cassem/api/concept"
	errorx "github.com/yeqown/cassem/api/concept"
	"github.com/yeqown/cassem/pkg/httpx"
)

func requestDomainHTTP(r *http.Request) string {
	return domainFromParams(chi.URLParam(r, "appId"), chi.URLParam(r, "env"))
}

func domainFromParams(appID, env string) string {
	if appID != "" && env != "" {
		return appID + "/" + env
	}
	if appID != "" {
		return appID + "/*"
	}
	return concept.Domain_CLUSTER
}

type objectDef struct {
	object string
	act    string
}

var defMapping = map[string]objectDef{
	"GET/api/apps":           {object: concept.Object_APP, act: concept.Action_READ},
	"GET/api/apps/:appId":    {object: concept.Object_APP, act: concept.Action_READ},
	"POST/api/apps/:appId":   {object: concept.Object_APP, act: concept.Action_WRITE},
	"DELETE/api/apps/:appId": {object: concept.Object_APP, act: concept.Action_DELETE},

	"GET/api/apps/:appId/envs":                       {object: concept.Object_APP, act: concept.Action_READ},
	"POST/api/apps/:appId/envs/:env":                 {object: concept.Object_APP, act: concept.Action_WRITE},
	"DELETE/api/apps/:appId/envs/:env":               {object: concept.Object_APP, act: concept.Action_WRITE},
	"GET/api/apps/:appId/envs/:env/elements":         {object: concept.Object_APP, act: concept.Action_READ},
	"GET/api/apps/:appId/envs/:env/elements/:key":    {object: concept.Object_ELEMENT, act: concept.Action_READ},
	"POST/api/apps/:appId/envs/:env/elements/:key":   {object: concept.Object_ELEMENT, act: concept.Action_WRITE},
	"PUT/api/apps/:appId/envs/:env/elements/:key":    {object: concept.Object_ELEMENT, act: concept.Action_WRITE},
	"DELETE/api/apps/:appId/envs/:env/elements/:key": {object: concept.Object_ELEMENT, act: concept.Action_DELETE},

	"GET/api/apps/:appId/envs/:env/elements/:key/versions":   {object: concept.Object_ELEMENT, act: concept.Action_READ},
	"GET/api/apps/:appId/envs/:env/elements/:key/operations": {object: concept.Object_ELEMENT, act: concept.Action_READ},
	"POST/api/apps/:appId/envs/:env/elements/:key/rollback":  {object: concept.Object_ELEMENT, act: concept.Action_PUBLISH},
	"POST/api/apps/:appId/envs/:env/elements/:key/publish":   {object: concept.Object_ELEMENT, act: concept.Action_PUBLISH},

	"GET/api/account/users":              {object: concept.Object_USER, act: concept.Action_READ},
	"GET/api/account/users/:account/acl": {object: concept.Object_ACL, act: concept.Action_READ},
	"POST/api/account/add":               {object: concept.Object_USER, act: concept.Action_WRITE},
	"GET/api/account/disable":            {object: concept.Object_USER, act: concept.Action_WRITE},
	"GET/api/account/reset":              {object: concept.Object_USER, act: concept.Action_WRITE},
	"POST/api/account/reset":             {object: concept.Object_USER, act: concept.Action_WRITE},
	"GET/api/account/acl/domains":        {object: concept.Object_ACL, act: concept.Action_READ},
	"GET/api/account/acl/assign":         {object: concept.Object_ACL, act: concept.Action_WRITE},
	"GET/api/account/acl/revoke":         {object: concept.Object_ACL, act: concept.Action_WRITE},

	"GET/api/admin/retention": {object: concept.Object_CLUSTER, act: concept.Action_READ},

	"GET/api/cluster/agents":                  {object: concept.Object_CLUSTER, act: concept.Action_READ},
	"GET/api/cluster/instances":               {object: concept.Object_CLUSTER, act: concept.Action_READ},
	"GET/api/cluster/instances/filter":        {object: concept.Object_CLUSTER, act: concept.Action_READ},
	"GET/api/cluster/instances/detail/:insId": {object: concept.Object_CLUSTER, act: concept.Action_READ},
}

type Session struct {
	Account   string
	Salt      string
	ExpiredAt int64
}

type sessionContextKey struct{}

func withSession(ctx context.Context, sess *Session) context.Context {
	return context.WithValue(ctx, sessionContextKey{}, sess)
}

func GetSessionFromRequest(r *http.Request) (*Session, bool) {
	v := r.Context().Value(sessionContextKey{})
	sess, ok := v.(*Session)
	return sess, ok
}

func withRouteAuth(rbac concept.RBAC, sessionSecret, method, authPattern string, next http.Handler) http.Handler {
	def, ok := defMapping[method+authPattern]
	if !ok {
		return next
	}

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		sess, err := authorizeRequest(r, rbac, sessionSecret)
		if err != nil {
			httpx.WriteErrorStatus(w, http.StatusUnauthorized, errorx.Err_UNAUTHENTICATED)
			return
		}

		allow, err := rbac.Enforce(sess.Account, requestDomainHTTP(r), def.object, def.act)
		if err != nil {
			httpx.WriteError(w, err)
			return
		}
		if !allow {
			httpx.WriteError(w, fmt.Errorf("not allowed: %w", errorx.Err_PERMISSION_DENIED))
			return
		}

		ctx := withSession(r.Context(), sess)
		ctx = concept.WithOperator(ctx, sess.Account)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func authorizeRequest(r *http.Request, rbac concept.RBAC, sessionSecret string) (*Session, error) {
	s := r.Header.Get("x-cassem-session")
	if s == "" {
		return nil, errorx.Err_UNAUTHENTICATED
	}

	sess, err := parseSession(s, sessionSecret)
	if err != nil {
		return nil, errorx.Err_UNAUTHENTICATED
	}

	user, err := rbac.GetUser(sess.Account)
	if err != nil {
		return nil, fmt.Errorf("authentication get user: %w", errors.Join(err, errorx.Err_INTERNAL))
	}

	if err = validSession(sess, user); err != nil {
		return nil, fmt.Errorf("valid session: %w", errors.Join(err, errorx.Err_UNAUTHENTICATED))
	}

	return sess, nil
}

func validSession(sess *Session, user *concept.User) error {
	if user.GetStatus() != concept.User_NORMAL {
		return fmt.Errorf("status disabled: %w", errorx.Err_UNAUTHENTICATED)
	}
	if user.GetSalt() != sess.Salt {
		return fmt.Errorf("invalid session header: %w", errorx.Err_UNAUTHENTICATED)
	}
	if sub := time.Now().Unix() - sess.ExpiredAt; sub >= 0 {
		return fmt.Errorf("session expired: %w", errorx.Err_UNAUTHENTICATED)
	}
	return nil
}

func parseSession(s string, sessionSecret string) (*Session, error) {
	if s == "" || sessionSecret == "" {
		return nil, errorx.Err_INVALID_ARGUMENT
	}

	parts := strings.Split(s, ".")
	if len(parts) != 2 {
		return nil, fmt.Errorf("invalid session token: %w", errorx.Err_INVALID_ARGUMENT)
	}

	payload, err := base64.RawURLEncoding.DecodeString(parts[0])
	if err != nil {
		return nil, fmt.Errorf("parse session: %w", errors.Join(err, errorx.Err_INVALID_ARGUMENT))
	}
	signature, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return nil, fmt.Errorf("parse session: %w", errors.Join(err, errorx.Err_INVALID_ARGUMENT))
	}
	if !hmac.Equal(signature, sessionSignature(payload, sessionSecret)) {
		return nil, fmt.Errorf("invalid session signature: %w", errorx.Err_INVALID_ARGUMENT)
	}

	sess := new(Session)
	if err = json.Unmarshal(payload, sess); err != nil {
		return nil, fmt.Errorf("parse session: %w", errors.Join(err, errorx.Err_INVALID_ARGUMENT))
	}
	return sess, nil
}

func EncodeSession(sess *Session, sessionSecret string) (string, error) {
	if sessionSecret == "" {
		return "", errorx.Err_INVALID_ARGUMENT
	}
	val, err := json.Marshal(sess)
	if err != nil {
		return "", fmt.Errorf("EncodeSession: %w", err)
	}

	payload := base64.RawURLEncoding.EncodeToString(val)
	signature := base64.RawURLEncoding.EncodeToString(sessionSignature(val, sessionSecret))
	return payload + "." + signature, nil
}

func sessionSignature(payload []byte, sessionSecret string) []byte {
	mac := hmac.New(sha256.New, []byte(sessionSecret))
	mac.Write(payload)
	return mac.Sum(nil)
}
