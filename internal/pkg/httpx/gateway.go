// Package httpx gateway.go contains two main API module: HTTP and gRPC. The reason for designing a gateway to
// serve request both HTTP and gRPC is that DO NOT want to listen on another TCP port so that client
// could build connections to only one server address.
//
// This design referred:
// https://eddycjy.com/posts/go/grpc-gateway/2019-06-22-grpc-gateway-tls/
//
// I hope them can help you too.
package httpx

import (
	"net/http"
	"strings"
	"time"

	"github.com/yeqown/log"
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

func (g gateway) Addr() string {
	return g.addr
}

func (g gateway) server() *http.Server {
	protocols := new(http.Protocols)
	protocols.SetHTTP1(true)
	protocols.SetUnencryptedHTTP2(true)
	return &http.Server{
		Addr:        g.Addr(),
		Handler:     g,
		ReadTimeout: 10 * time.Second,
		Protocols:   protocols,
	}
}

func (g gateway) ListenAndServe() error {
	return g.server().ListenAndServe()
}
