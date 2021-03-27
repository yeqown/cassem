package http

import (
	"fmt"
	"net/http"
	"net/http/httputil"
	"os"

	"github.com/yeqown/cassem/internal/authorizer"
	"github.com/yeqown/cassem/pkg/runtime"

	"github.com/gin-gonic/gin"
	"github.com/yeqown/log"
)

const (
	_authorizationKey = "token"
)

// mapping indicates (1) current uri should match permissions? (2) how to match permissions?
// (3) should use specified matcher or not?
type mapping struct {
	action string
	object string
}

var (
	// uri:action
	_actionMappingURI = map[string]mapping{
		"/api/namespaces#GET": {
			action: authorizer.ACTION_READ,
			object: authorizer.OBJ_NAMESPACE,
		},
		"/api/namespaces/:ns/containers#GET": {
			action: authorizer.ACTION_READ,
			object: authorizer.OBJ_CONTAINER,
		},
		"/api/namespaces/:ns/containers/:key#GET": {
			action: authorizer.ACTION_WRITE,
			object: authorizer.OBJ_CONTAINER,
		},
		"/api/namespaces/:ns/containers/:key/dl#GET": {
			action: authorizer.ACTION_READ,
			object: authorizer.OBJ_CONTAINER,
		},
		"/api/namespaces/:ns/containers/:key#POST": {
			action: authorizer.ACTION_WRITE,
			object: authorizer.OBJ_CONTAINER,
		},
		"/api/namespaces/:ns/containers/:key#DELETE": {
			action: authorizer.ACTION_WRITE,
			object: authorizer.OBJ_CONTAINER,
		},
		"/api/namespaces/:ns/pairs#GET": {
			action: authorizer.ACTION_READ,
			object: authorizer.OBJ_PAIR,
		},
		"/api/namespaces/:ns/pairs/:key#GET": {
			action: authorizer.ACTION_WRITE,
			object: authorizer.OBJ_CONTAINER,
		},
		"/api/namespaces/:ns/pairs/:key#POST": {
			action: authorizer.ACTION_WRITE,
			object: authorizer.OBJ_CONTAINER,
		},
		"/api/users#GET": {
			action: authorizer.ACTION_READ,
			object: authorizer.OBJ_USER,
		},
		"/api/users/new#POST": {
			action: authorizer.ACTION_WRITE,
			object: authorizer.OBJ_USER,
		},
		"/api/users/reset-password#POST": {
			action: authorizer.ACTION_WRITE,
			object: authorizer.OBJ_USER,
		},
		"/api/users/:userid/policies#GET": {
			action: authorizer.ACTION_READ,
			object: authorizer.OBJ_POLICY,
		},
		"/api/users/:userid/policies/policy#PUT": {
			action: authorizer.ACTION_WRITE,
			object: authorizer.OBJ_POLICY,
		},
	}
)

func authorize(enforceFn func(req *authorizer.EnforceRequest) bool) gin.HandlerFunc {
	if enforceFn == nil {
		panic("could not initialize with nil enforceFn")
	}

	var skip bool
	if os.Getenv("IGNORE_AUTH") != "" {
		skip = true
	}

	return func(c *gin.Context) {
		if skip {
			return
		}

		// DONE(@yeqown): get resource mapping related to c.FullPath()
		p := c.FullPath()
		m, ok := _actionMappingURI[p+"#"+c.Request.Method]
		if !ok {
			// no need to enforce.
			c.Next()
			return
		}

		// token required
		tokenString := c.GetHeader("token")
		token, err := authorizer.Session(tokenString)
		if err != nil {
			log.
				WithFields(log.Fields{
					"token":       token,
					"tokenString": tokenString,
				}).
				Errorf("enforceFn.Session failed: %v", err)
			c.AbortWithStatus(http.StatusForbidden)

			return
		}

		// set into context, so that handler could use it.
		c.Set(_authorizationKey, token)

		// TODO(@yeqown): specified rules for namespaces
		//if m.protectedNamespace {
		//	ns := c.Param("ns")
		//	_ = ns
		//}

		if enforceFn(&authorizer.EnforceRequest{
			Subject: token.Subject(), // DONE(@yeqown): get subject from token
			Object:  m.object,
			Action:  m.action,
		}) {
			c.Next()

			return
		}

		c.AbortWithStatus(http.StatusForbidden)
	}
}

func clusterAuthorizeSimple() gin.HandlerFunc {

	return func(c *gin.Context) {
		if c.Query("clusterSecret") == "9520059dd167" {
			c.Next()
			return
		}

		// then forbidden the invalid request
		c.AbortWithStatus(http.StatusForbidden)
	}
}

func recovery() gin.HandlerFunc {
	return func(c *gin.Context) {
		panicked := true
		defer func() {
			if v := recover(); v != nil || panicked {
				dumpReq, _ := httputil.DumpRequest(c.Request, true)
				formatted := fmt.Sprintf("server panic: %v\n%s %s", v, dumpReq, runtime.Stack())
				_, _ = fmt.Fprint(os.Stderr, formatted)
				err := runtime.RecoverFrom(v)
				log.Errorf("server panic: %v", err)
				c.AbortWithStatusJSON(http.StatusInternalServerError, commonResponse{
					ErrCode:    FAILED,
					ErrMessage: err.Error(),
				})
			}
		}()

		c.Next()
		panicked = false
	}
}
