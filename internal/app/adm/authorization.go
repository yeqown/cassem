package adm

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/yeqown/log"

	"github.com/yeqown/cassem/api/concept"
	"github.com/yeqown/cassem/pkg/errorx"
	"github.com/yeqown/cassem/pkg/httpx"
)

// Session represents user info who has login.
type Session struct {
	Account   string
	Salt      string
	ExpiredAt int64
}

func Authorization(rbac concept.RBAC, sessionSecret string) gin.HandlerFunc {
	return func(c *gin.Context) {
		s := c.GetHeader("x-cassem-session")
		log.
			WithFields(log.Fields{"sess": s}).
			Debug("Authorization called")
		if s == "" {
			httpx.ResponseErrorStatusAndAbort(c, http.StatusUnauthorized, errorx.Err_UNAUTHENTICATED)
			return
		}

		sess, err := parseSession(s, sessionSecret)
		if err != nil {
			httpx.ResponseErrorStatusAndAbort(c, http.StatusUnauthorized, errorx.Err_UNAUTHENTICATED)
			return
		}

		user, err := rbac.GetUser(sess.Account)
		if err != nil {
			log.Warnf("Authentication get user failed: %v", err)
			httpx.ResponseErrorAndAbort(c, fmt.Errorf("authentication get user: %w", errors.Join(err, errorx.Err_INTERNAL)))
			return
		}

		if err = validSession(sess, user); err != nil {
			httpx.ResponseErrorAndAbort(c, fmt.Errorf("valid session: %w", errors.Join(err, errorx.Err_UNAUTHENTICATED)))
			return
		}

		c.Set("sess", sess)
		c.Request = c.Request.WithContext(concept.WithOperator(c.Request.Context(), sess.Account))
		c.Next()
	}
}

func GetSessionFromContext(c *gin.Context) (*Session, bool) {
	v, ok := c.Get("sess")
	if !ok {
		return nil, false
	}

	sess, ok := v.(*Session)
	return sess, ok
}

func validSession(sess *Session, user *concept.User) error {
	// valid session status
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
