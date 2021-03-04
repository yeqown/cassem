package http

import (
	"fmt"
	"net/http"
	"net/http/httputil"
	"os"

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
