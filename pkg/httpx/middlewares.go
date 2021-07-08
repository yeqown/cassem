package httpx

import (
	"fmt"
	"net/http"
	"net/http/httputil"
	"os"

	"github.com/gin-gonic/gin"
	"github.com/yeqown/log"

	"github.com/yeqown/cassem/pkg/runtime"
)

func Recovery() gin.HandlerFunc {
	return func(c *gin.Context) {
		panicked := true
		defer func() {
			if v := recover(); v != nil || panicked {
				dumpReq, _ := httputil.DumpRequest(c.Request, true)
				formatted := fmt.Sprintf("server panic: %v\n%s %s", v, dumpReq, runtime.Stack())
				_, _ = fmt.Fprint(os.Stderr, formatted)
				err := runtime.RecoverFrom(v)
				log.Errorf("server panic: %v", err)
				c.AbortWithStatusJSON(http.StatusInternalServerError, CommonResponse{
					ErrCode:    FAILED,
					ErrMessage: err.Error(),
				})
			}
		}()

		c.Next()
		panicked = false
	}
}
