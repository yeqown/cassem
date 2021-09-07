package infras

import (
	"encoding/base64"
	"encoding/json"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/pkg/errors"
	"github.com/yeqown/log"

	"github.com/yeqown/cassem/concept"
	"github.com/yeqown/cassem/pkg/errorx"
	"github.com/yeqown/cassem/pkg/httpx"
)

// Session represents user info who has login.
type Session struct {
	Account   string
	Salt      string
	ExpiredAt int64
}

func Authorization(rbac concept.RBAC) gin.HandlerFunc {
	return func(c *gin.Context) {
		s := c.GetHeader("x-cassem-session")
		log.
			WithFields(log.Fields{"sess": s}).
			Debug("Authorization called")
		if s == "" {
			httpx.ResponseErrorStatusAndAbort(c, http.StatusUnauthorized, errorx.Err_UNAUTHENTICATED)
			return
		}

		sess, err := parseSession(s)
		if err != nil {
			httpx.ResponseErrorStatusAndAbort(c, http.StatusUnauthorized, errorx.Err_UNAUTHENTICATED)
			return
		}

		user, err := rbac.GetUser(sess.Account)
		if err != nil {
			log.Warnf("Authentication get user failed: %v", err)
			httpx.ResponseErrorAndAbort(c, errors.Wrap(errorx.Err_INTERNAL, err.Error()))
			return
		}

		if err = validSession(sess, user); err != nil {
			httpx.ResponseErrorAndAbort(c, errors.Wrap(errorx.Err_UNAUTHENTICATED, err.Error()))
			return
		}

		c.Set("sess", sess)
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
		return errors.Wrap(errorx.Err_UNAUTHENTICATED, "status disabled")
	}
	if user.GetSalt() != sess.Salt {
		return errors.Wrap(errorx.Err_UNAUTHENTICATED, "invalid session header")
	}

	if sub := time.Now().Unix() - sess.ExpiredAt; sub >= 0 {
		return errors.Wrap(errorx.Err_UNAUTHENTICATED, "session expired")
	}

	return nil
}

func parseSession(s string) (*Session, error) {
	if s == "" {
		return nil, errorx.Err_INVALID_ARGUMENT
	}

	val, err := base64.StdEncoding.DecodeString(s)
	if err != nil {
		return nil, errors.Wrap(errorx.Err_INVALID_ARGUMENT, err.Error())
	}

	sess := new(Session)
	if err = json.Unmarshal(val, sess); err != nil {
		return nil, errors.Wrap(errorx.Err_INVALID_ARGUMENT, err.Error())
	}

	return sess, nil
}

func EncodeSession(sess *Session) (string, error) {
	val, err := json.Marshal(sess)
	if err != nil {
		return "", errors.Wrap(err, "EncodeSession")
	}

	//out := make([]byte, base64.StdEncoding.EncodedLen(len(val)))
	//base64.StdEncoding.Encode(out, val)
	return base64.StdEncoding.EncodeToString(val), nil
}
