package agent

import (
	"context"
	"errors"
	"fmt"
	"math/rand"
	"net"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	"github.com/yeqown/log"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"

	apiagent "github.com/yeqown/cassem/api/agent"
	"github.com/yeqown/cassem/api/concept"
	"github.com/yeqown/cassem/internal/coord"
	"github.com/yeqown/cassem/pkg/conf"
	"github.com/yeqown/cassem/pkg/grpcx"
	"github.com/yeqown/cassem/pkg/hash"
	"github.com/yeqown/cassem/pkg/runtime"
)

// getHostname returns the hostname or "unknown" if error occurs.
func getHostname() string {
	hostname, err := os.Hostname()
	if err != nil {
		return "unknown"
	}
	return hostname
}

func advertiseAddr(bindAddr, hostname string) string {
	host, port, err := net.SplitHostPort(bindAddr)
	if err != nil {
		return bindAddr
	}
	if host == "" || host == "0.0.0.0" || host == "::" {
		return net.JoinHostPort(hostname, port)
	}
	return bindAddr
}

type app struct {
	apiagent.UnimplementedAgentServer
	apiagent.UnimplementedDeliveryServer

	uniqueId    string
	quit        chan struct{}
	regSuccessC chan struct{}
	ctx         context.Context
	cancel      context.CancelFunc

	actualRenewInterval int32
	conf                *conf.CassemAgentConfig

	aggregate concept.AgentAggregate

	cache        Cache
	instancePool InstancePool
}

func New(c *conf.CassemAgentConfig) (*app, error) {
	if err := c.Valid(); err != nil {
		return nil, fmt.Errorf("cassemagent.New failed: %w", err)
	}

	agg, err := coord.NewAgentAggregate(c.CassemKVEndpoints)
	if err != nil {
		return nil, fmt.Errorf("cassemagent.New: %w", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	d := &app{
		uniqueId:     "",
		quit:         make(chan struct{}, 1),
		regSuccessC:  make(chan struct{}),
		ctx:          ctx,
		cancel:       cancel,
		conf:         c,
		aggregate:    agg,
		cache:        NewCache(uint(c.ElementCacheSize)),
		instancePool: NewInstancePool(),
	}

	return d, nil
}

func (d *app) Run() {
	d.genUniqueId()
	d.startRoutines()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt, syscall.SIGHUP, syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT)

	for range quit {
		log.Debug("app received one signal, then quit")
		// graceful shutdown and quit main goroutine
		d.shutdown()
		return
	}
}

func (d *app) lifecycleContext() context.Context {
	if d.ctx != nil {
		return d.ctx
	}
	return context.Background()
}

func (d *app) shutdown() {
	if d.cancel != nil {
		d.cancel()
	}
	select {
	case d.quit <- struct{}{}:
		time.Sleep(5 * time.Second)
	default:
	}
}

func (d *app) startRoutines() {
	runtime.GoFunc("app.serve", d.serve)
	runtime.GoFunc("app.renewSelf", d.renew)
}

func (d *app) serve() error {
	// blocked here until app register itself success
	<-d.regSuccessC

	s := grpc.NewServer(
		grpc.UnaryInterceptor(grpcx.ChainUnaryServer(
			grpcx.ServerRecovery(), grpcx.ServerLogger(), grpcx.SevrerErrorx(), grpcx.ServerValidation())),
	)

	// register service and rpcs
	apiagent.RegisterAgentServer(s, d)
	apiagent.RegisterDeliveryServer(s, d)
	reflection.Register(s)

	lis, err := net.Listen("tcp", d.conf.Server.Addr)
	if err != nil {
		return err
	}

	return s.Serve(lis)
}

// renew
func (d *app) renew() error {
	renewSelf := func() error {
		timeoutCtx, cancel := context.WithTimeout(d.lifecycleContext(), 3*time.Second)
		defer cancel()
		err := d.aggregate.Renew(timeoutCtx, &concept.AgentInstance{
			AgentId: d.uniqueId,
			Addr:    advertiseAddr(d.conf.Server.Addr, getHostname()),
			Annotations: map[string]string{
				"op":            "renew",
				"hostname":      getHostname(),
				"ttl":           strconv.Itoa(int(d.conf.TTL)),
				"renewInterval": strconv.Itoa(int(d.actualRenewInterval)),
				// "timestamp": time.Now().Format(time.RFC3339),
			},
		}, d.conf.TTL)
		if err != nil {
			return fmt.Errorf("cassemagent.app.renewSelf: %w", err)
		}
		return err
	}

	// calculate renew interval
	d.actualRenewInterval = d.conf.RenewInterval + rand.Int31n(d.conf.TTL-d.conf.RenewInterval)
	for {
		timeoutCtx, cancel := context.WithTimeout(d.lifecycleContext(), 3*time.Second)
		err := d.aggregate.Register(timeoutCtx, &concept.AgentInstance{
			AgentId: d.uniqueId,
			Addr:    advertiseAddr(d.conf.Server.Addr, getHostname()),
			Annotations: map[string]string{
				"op":            "renew",
				"hostname":      getHostname(),
				"ttl":           strconv.Itoa(int(d.conf.TTL)),
				"renewInterval": strconv.Itoa(int(d.actualRenewInterval)),
				// "timestamp": time.Now().Format(time.RFC3339),
			},
		}, d.conf.TTL)
		cancel()
		if err == nil {
			break
		}
		if errors.Is(err, context.Canceled) {
			return err
		}
		log.
			WithFields(log.Fields{
				"error": err,
			}).
			Error("cassemagent.app.Register failed")
	}

	d.regSuccessC <- struct{}{}

	// actualRenewInterval = conf.renewInterval + int32n(conf.TTL - cond.RenewInterval)
	dur := time.Duration(d.actualRenewInterval) * time.Second
	ticker := time.NewTicker(dur)
	for {
		select {
		case ts := <-ticker.C:
			log.Info("cassemagent.app renewSelf")
			if err := renewSelf(); err != nil {
				log.
					WithFields(log.Fields{
						"error": err,
						"time":  ts.Format(time.RFC3339),
					}).
					Error("cassemagent.app.renewSelf failed")
			}
		case <-d.quit:
			log.Info("cassemagent.app receives a quit signal")
			timeoutCtx, cancel := context.WithTimeout(d.lifecycleContext(), 3*time.Second)
			if err := d.aggregate.Unregister(timeoutCtx, d.uniqueId); err != nil {
				log.
					WithFields(log.Fields{
						"error": err,
					}).
					Error("cassemagent.app.Unregister failed")
			}
			cancel()
			// quit routine
			return nil
		}
	}
}

// genUniqueId panics if any error encountered during apply unique id.
func (d *app) genUniqueId() string {
	d.uniqueId = hash.RandKey(8)
	return d.uniqueId
}
