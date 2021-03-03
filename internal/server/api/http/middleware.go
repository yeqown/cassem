package http

import (
	"fmt"
	"net/http"
	"net/http/httputil"

	"github.com/yeqown/cassem/pkg/runtime"

	"github.com/gin-gonic/gin"
	"github.com/yeqown/log"
)

func authorize() gin.HandlerFunc {

	return func(c *gin.Context) {
		c.Next()
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
				formatted := fmt.Sprintf("server panic: %v %v %s", dumpReq, v, runtime.Stack())
				log.Errorf(formatted)
				responseError(c, runtime.RecoverFrom(v))
				c.Abort()
			}
		}()

		c.Next()
		panicked = false
	}
}
