package infras

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/pkg/errors"

	"github.com/yeqown/log"

	"github.com/yeqown/cassem/concept"
	"github.com/yeqown/cassem/pkg/errorx"
	"github.com/yeqown/cassem/pkg/httpx"
)

// Session represents user info who has login.
type Session struct {
	Account string
	Salt    string
}

func Authorization(rbac concept.RBAC) gin.HandlerFunc {
	return func(c *gin.Context) {
		account, salt := c.GetHeader("x-cassem-user"), c.GetHeader("x-cassem-hash")
		log.
			WithFields(log.Fields{
				"account": account,
				"salt":    salt,
			}).
			Debug("Authorization called")
		if account == "" || salt == "" {
			_ = c.AbortWithError(http.StatusUnauthorized, errorx.Err_PERMISSION_DENIED)
			return
		}

		user, err := rbac.GetUser(account)
		if err != nil {
			log.Warnf("Authentication get user failed: %v", err)
			httpx.ResponseErrorAndAbort(c, errors.Wrap(errorx.Err_INTERNAL, err.Error()))
			return
		}

		// valid session status
		if user.GetStatus() != concept.User_NORMAL {
			httpx.ResponseErrorAndAbort(c, errors.Wrap(errorx.Err_PERMISSION_DENIED, "status disabled"))
			return
		}
		if user.GetSalt() != salt {
			httpx.ResponseErrorAndAbort(c, errors.Wrap(errorx.Err_PERMISSION_DENIED, "invalid session header"))
			return
		}

		sess := &Session{Account: account, Salt: salt}
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
