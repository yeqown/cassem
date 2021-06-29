// Package delivery container two main API module: HTTP and gRPC. The reason for designing a Gateway to
// serve request both HTTP and gRPC is that DO NOT want to listen on another TCP port so that client
// could build connections to only one server address.
//
// This design referred:
// https://eddycjy.com/posts/go/grpc-gateway/2019-06-22-grpc-gateway-tls/
//
// I hope them can help you too.
//
package delivery

import (
	"net/http"
	"strings"
	"time"

	"github.com/yeqown/log"
	"golang.org/x/net/http2"
	"golang.org/x/net/http2/h2c"
	"google.golang.org/grpc"

	"github.com/yeqown/cassem/apps/cassemdb/coord"
	apihtp "github.com/yeqown/cassem/apps/cassemdb/delivery/http"

	"github.com/yeqown/cassem/internal/conf"
)

// Gateway is the gate to all cassem API. It provides both HTTP and gRPC protocol applications at once.
// The purpose is a solution what could serve them on one TCP port, this makes easily for client to
// build connections to cassem server.
type Gateway struct {
	addr         string
	api          *apihtp.Server
	notifyServer *grpc.Server
}

func New(cfg *conf.HTTP, coord coord.ICoordinator) *Gateway {
	api := apihtp.New(cfg, coord)
	// notifyServer := notifier.New()

	return &Gateway{
		addr: cfg.Addr,
		api:  api,
		// notifyServer: notifyServer,
	}
}

// ServeHTTP implement http.Handler, so Gateway.http2Wrapper could wrap with it.
// This method is not allowed to use directly, unless you DO use HTTP only.
func (gate Gateway) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	log.
		WithFields(log.Fields{
			"req.ProtoMajor":   req.ProtoMajor,
			"req.Content-Type": req.Header.Get("Content-Type"),
		}).
		Debug("Gateway.ServeHTTP called")

	if req.ProtoMajor == 2 && strings.Contains(req.Header.Get("Content-Type"), "application/grpc") {
		gate.notifyServer.ServeHTTP(w, req)
	} else {
		gate.api.ServeHTTP(w, req)
	}
}

// ServeHTTP implements http.Handler
func (gate Gateway) http2Wrapper() http.Handler {
	return h2c.NewHandler(gate, &http2.Server{})
}

func (gate Gateway) Addr() string {
	return gate.addr
}

func (gate Gateway) ListenAndServe() error {
	srv := http.Server{
		Addr:         gate.Addr(),
		Handler:      gate.http2Wrapper(),
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
		//IdleTimeout:  0,
	}

	return srv.ListenAndServe()
}
