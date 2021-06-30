package api

import (
	"github.com/yeqown/log"

	apigrpc "github.com/yeqown/cassem/internal/cassemdb/api/grpc"
	apihtp "github.com/yeqown/cassem/internal/cassemdb/api/http"
	"github.com/yeqown/cassem/internal/cassemdb/app"
	"github.com/yeqown/cassem/pkg/conf"
	"github.com/yeqown/cassem/pkg/httpc"
	"github.com/yeqown/cassem/pkg/runtime"
)

func Run(config *conf.CassemdbConfig) {
	var (
		d   app.ICoordinator
		err error
	)
	if d, err = app.New(config); err != nil {
		panic(err)
	}

	// gate contains HTTP and gRPC protocol server. HTTP server provides all PUBLIC managing API and
	// internal cluster API. The duty of gRPC server is serving cassem's components for watching changes.
	//
	// Notice that HTTP server and gRPC server use backend of gateway, so there is only one TCP port to
	// listen on.
	gate := httpc.NewGateway(config.Server.HTTP.Addr, apihtp.New(d), apigrpc.New(d))
	log.Info("app: Gate server loaded")

	runtime.GoFunc("cassemdb.app.gate", func() (err error) {
		if err = gate.ListenAndServe(); err != nil {
			log.Errorf("cassemdb.app.gate quited: %v", err)
		}

		return
	})

	d.Heartbeat()
}
