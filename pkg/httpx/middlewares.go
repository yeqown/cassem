package httpx

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/httputil"
	"os"
	"strings"
	"time"

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

//type respBodyWriter struct {
//	gin.ResponseWriter
//	body *bytes.Buffer
//}
//
//func (w respBodyWriter) Write(b []byte) (int, error) {
//	w.body.Write(b)
//	return w.ResponseWriter.Write(b)
//}

func Logger() gin.HandlerFunc {
	return func(c *gin.Context) {
		//rbw := &respBodyWriter{
		//	body:           bytes.NewBufferString(""),
		//	ResponseWriter: c.Writer,
		//}
		//c.Writer = rbw
		body, err := c.GetRawData()
		if err == nil && len(body) != 0 {
			c.Request.Body = ioutil.NopCloser(bytes.NewBuffer(body))
		}

		start := time.Now()

		c.Next()

		latency := time.Since(start)
		fields := log.Fields{
			"clientIP": c.ClientIP(),
		}

		log.
			WithFields(fields).
			Infof("[%3d] [%v] %s '%s' [Body]: %s", c.Writer.Status(), latency,
				c.Request.Method, c.Request.URL, body)
	}
}

func formatHeader(header http.Header) string {
	buf := bytes.NewBuffer(nil)
	for k, v := range header {
		buf.WriteString(k + ":" + strings.Join(v, ";") + " ")
	}

	return buf.String()
}
