package httpx

import (
	"bytes"
	"fmt"
	"net/http"
	"net/http/httputil"
	"os"
	"strings"
	"time"

	"github.com/yeqown/log"

	"github.com/yeqown/cassem/internal/pkg/runtime"
)

type statusRecorder struct {
	http.ResponseWriter
	status int
}

func (w *statusRecorder) WriteHeader(status int) {
	w.status = status
	w.ResponseWriter.WriteHeader(status)
}

func (w *statusRecorder) Status() int {
	if w.status == 0 {
		return http.StatusOK
	}
	return w.status
}

func RecoveryHTTP(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		panicked := true
		defer func() {
			if v := recover(); v != nil || panicked {
				diagnosticRequest := r.Clone(r.Context())
				for name := range diagnosticRequest.Header {
					if strings.EqualFold(name, "Authorization") || strings.EqualFold(name, "Proxy-Authorization") || strings.EqualFold(name, "X-CASSEM-SESSION") {
						diagnosticRequest.Header[name] = []string{"[REDACTED]"}
					}
				}
				dumpReq, _ := httputil.DumpRequest(diagnosticRequest, true)
				// DumpRequest restores the body it reads; retain that behavior on the original request.
				r.Body = diagnosticRequest.Body
				formatted := fmt.Sprintf("server panic: %v\n%s %s", v, dumpReq, runtime.Stack())
				_, _ = fmt.Fprint(os.Stderr, formatted)
				err := runtime.RecoverFrom(v)
				log.Errorf("server panic: %v", err)
				WriteErrorStatus(w, http.StatusInternalServerError, err)
			}
		}()

		next.ServeHTTP(w, r)
		panicked = false
	})
}

func LoggerHTTP(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// var body []byte
		// if r.Body != nil {
		// 	var err error
		// 	body, err = io.ReadAll(r.Body)
		// 	if err == nil && len(body) != 0 {
		// 		r.Body = io.NopCloser(bytes.NewBuffer(body))
		// 	}
		// }

		start := time.Now()
		recorder := &statusRecorder{ResponseWriter: w}
		next.ServeHTTP(recorder, r)

		latency := time.Since(start)
		fields := log.Fields{}
		if host, _, found := strings.Cut(r.RemoteAddr, ":"); found && host != "" {
			fields["clientIP"] = host
		}

		log.
			WithFields(fields).
			Infof("%3d %v %s %s", recorder.Status(), latency, r.Method, r.URL)
	})
}

func formatHeader(header http.Header) string {
	buf := bytes.NewBuffer(nil)
	for k, v := range header {
		buf.WriteString(k + ":" + strings.Join(v, ";") + " ")
	}
	return buf.String()
}
