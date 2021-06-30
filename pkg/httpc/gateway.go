// Package httpc gateway.go contains two main API module: HTTP and gRPC. The reason for designing a gateway to
// serve request both HTTP and gRPC is that DO NOT want to listen on another TCP port so that client
// could build connections to only one server address.
//
// This design referred:
// https://eddycjy.com/posts/go/grpc-gateway/2019-06-22-grpc-gateway-tls/
//
// I hope them can help you too.
//
package httpc

import (
	"net/http"
	"strings"
	"time"

	"github.com/yeqown/log"
	"golang.org/x/net/http2"
	"golang.org/x/net/http2/h2c"
	"google.golang.org/grpc"
)

// gateway is the gate to all cassem API. It provides both HTTP and gRPC protocol applications at once.
// The purpose is a solution what could serve them on one TCP port, this makes easily for client to
// build connections to cassem server.
type gateway struct {
	addr string
	http http.Handler
	grpc *grpc.Server
}

func NewGateway(addr string, s1 http.Handler, s2 *grpc.Server) *gateway {
	return &gateway{
		addr: addr,
		http: s1,
		grpc: s2,
	}
}

// ServeHTTP implement http.Handler, so gateway.http2Wrapper could wrap with it.
// This method is not allowed to use directly, unless you DO use HTTP only.
func (g gateway) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	log.
		WithFields(log.Fields{
			"req.ProtoMajor":   req.ProtoMajor,
			"req.Content-Type": req.Header.Get("Content-Type"),
		}).
		Debug("gateway.ServeHTTP called")

	if req.ProtoMajor == 2 && strings.Contains(req.Header.Get("Content-Type"), "application/grpc") {
		g.grpc.ServeHTTP(w, req)
	} else {
		g.http.ServeHTTP(w, req)
	}
}

// ServeHTTP implements http.Handler
func (g gateway) http2Wrapper() http.Handler {
	return h2c.NewHandler(g, &http2.Server{})
}

func (g gateway) Addr() string {
	return g.addr
}

func (g gateway) ListenAndServe() error {
	srv := http.Server{
		Addr:         g.Addr(),
		Handler:      g.http2Wrapper(),
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 30 * time.Second,
		//IdleTimeout:  0,
	}

	return srv.ListenAndServe()
}
