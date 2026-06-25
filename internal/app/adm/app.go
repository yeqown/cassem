package adm

import (
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"

	"github.com/yeqown/cassem/api/concept"
	"github.com/yeqown/cassem/internal/coord"
	"github.com/yeqown/cassem/pkg/conf"
	"github.com/yeqown/cassem/pkg/httpx"
)

func isDebug() bool {
	v := os.Getenv("DEBUG")
	return v == "1" || v == "TRUE" || v == "true"
}

type app struct {
	conf *conf.CassemAdminConfig

	// aggregate is special methods interface customized form adm component which
	// can only be used by cassemadm.app.
	aggregate concept.AdmAggregate

	// ap type agentPool is a pool contains all ap nodes, and agentPool will update
	// agent nodes  automatically.
	ap *agentPool

	retention *retentionGC
}

func New(c *conf.CassemAdminConfig) (*app, error) {
	if err := c.Valid(); err != nil {
		return nil, fmt.Errorf("adm.New failed: %w", err)
	}

	agg, err := coord.NewAdmAggregate(c.CassemKVEndpoints)
	if err != nil {
		return nil, fmt.Errorf("adm.New: %w", err)
	}
	if err = agg.AutoMigrate(); err != nil {
		return nil, fmt.Errorf("adm.New.AutoMigrate: %w", err)
	}
	if c.Auth.HasBootstrapAdmin() {
		if err = agg.BootstrapAdmin(c.Auth.BootstrapAccount, c.Auth.BootstrapNickname, c.Auth.BootstrapPassword); err != nil {
			return nil, fmt.Errorf("adm.New.BootstrapAdmin: %w", err)
		}
	}

	retention, err := newRetentionGC(c.CassemKVEndpoints, c.Retention)
	if err != nil {
		return nil, fmt.Errorf("adm.New.RetentionGC: %w", err)
	}

	d := &app{
		aggregate: agg,
		conf:      c,
		ap:        newAgentPool(agg),
		retention: retention,
	}

	return d, nil
}

func (d app) Run() {
	r := chi.NewRouter()
	d.initialHTTP(r)
	listener, err := net.Listen("tcp", d.conf.HTTP.Addr)
	if err != nil {
		log.Fatal(err)
	}
	if d.retention != nil {
		d.retention.run()
	}

	if err = http.Serve(listener, r); err != nil {
		log.Fatal(err)
	}
}

func (d app) initialHTTP(r chi.Router) {
	r.Use(httpx.RecoveryHTTP)
	r.Use(httpx.LoggerHTTP)
	r.Use(cors.Handler(cors.Options{
		AllowedOrigins:   []string{"*"},
		AllowedMethods:   []string{"GET", "POST", "PUT", "PATCH", "DELETE", "HEAD", "OPTIONS"},
		AllowedHeaders:   []string{"Origin", "Content-Length", "Content-Type", "X-CASSEM-SESSION"},
		AllowCredentials: false,
		MaxAge:           int((12 * time.Hour).Seconds()),
	}))

	if isDebug() {
		r.Mount("/debug/pprof", middleware.Profiler())
	}

	if err := mountUI(r); err != nil {
		log.Fatalf("mount UI: %v", err)
	}

	d.mountAdminRoutes(r)
}
