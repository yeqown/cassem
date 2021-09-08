package app

import (
	"context"
	"math/rand"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	"github.com/pkg/errors"
	"github.com/yeqown/log"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"

	"github.com/yeqown/cassem/concept"
	apiagent "github.com/yeqown/cassem/internal/cassemagent/api"
	"github.com/yeqown/cassem/internal/cassemagent/domain"
	"github.com/yeqown/cassem/pkg/conf"
	"github.com/yeqown/cassem/pkg/grpcx"
	"github.com/yeqown/cassem/pkg/hash"
	"github.com/yeqown/cassem/pkg/httpx"
	"github.com/yeqown/cassem/pkg/runtime"
)

type app struct {
	uniqueId    string
	quit        chan struct{}
	regSuccessC chan struct{}

	actualRenewInterval int32
	conf                *conf.CassemAgentConfig

	aggregate concept.AgentAggregate

	cache        domain.Cache
	instancePool domain.InstancePool
}

func New(c *conf.CassemAgentConfig) (*app, error) {
	if err := c.Valid(); err != nil {
		return nil, errors.Wrap(err, "cassemagent.New failed")
	}

	agg, err := concept.NewAgentAggregate(c.CassemDBEndpoints)
	if err != nil {
		return nil, errors.Wrap(err, "cassemagent.New")
	}

	d := &app{
		uniqueId:     "",
		quit:         make(chan struct{}, 1),
		regSuccessC:  make(chan struct{}),
		conf:         c,
		aggregate:    agg,
		cache:        domain.NewCache(uint(c.ElementCacheSize)),
		instancePool: domain.NewInstancePool(),
	}

	return d, nil
}

func (d app) Run() {
	d.genUniqueId()
	d.startRoutines()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt, syscall.SIGHUP, syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT)

	for {
		select {
		case <-quit:
			log.Debug("app received one signal, then quit")
			// graceful shutdown and quit main goroutine
			d.shutdown()
			return
		}
	}
}

func (d app) shutdown() {
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

func (d app) serve() error {
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

	gate := httpx.NewGateway(d.conf.Server.Addr, nil, s)
	if err := gate.ListenAndServe(); err != nil {
		return err
	}

	return nil
}

// renew
func (d app) renew() error {
	renewSelf := func() error {
		timeoutCtx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		defer cancel()
		err := d.aggregate.Renew(timeoutCtx, &concept.AgentInstance{
			AgentId: d.uniqueId,
			Addr:    d.conf.Server.Addr,
			Annotations: map[string]string{
				"op":            "renew",
				"hostname":      runtime.Hostname(),
				"ttl":           strconv.Itoa(int(d.conf.TTL)),
				"renewInterval": strconv.Itoa(int(d.actualRenewInterval)),
				// "timestamp": time.Now().Format(time.RFC3339),
			},
		}, d.conf.TTL)
		if err != nil {
			return errors.Wrap(err, "cassemagent.app.renewSelf")
		}
		return err
	}

	// calculate renew interval
	rand.Seed(time.Now().UnixNano())
	d.actualRenewInterval = d.conf.RenewInterval + rand.Int31n(d.conf.TTL-d.conf.RenewInterval)
	timeoutCtx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
retryReg:
	err := d.aggregate.Register(timeoutCtx, &concept.AgentInstance{
		AgentId: d.uniqueId,
		Addr:    d.conf.Server.Addr,
		Annotations: map[string]string{
			"op":            "renew",
			"hostname":      runtime.Hostname(),
			"ttl":           strconv.Itoa(int(d.conf.TTL)),
			"renewInterval": strconv.Itoa(int(d.actualRenewInterval)),
			// "timestamp": time.Now().Format(time.RFC3339),
		},
	}, d.conf.TTL)
	if err != nil {
		log.
			WithFields(log.Fields{
				"error": err,
			}).
			Error("cassemagent.app.Register failed")
		goto retryReg
	}

	d.regSuccessC <- struct{}{}
	cancel()

	// actualRenewInterval = conf.renewInterval + int32n(conf.TTL - cond.RenewInterval)
	dur := time.Duration(d.actualRenewInterval) * time.Second
	ticker := time.NewTicker(dur)
	for {
		select {
		case ts := <-ticker.C:
			log.Info("cassemagent.app renewSelf")
			if err = renewSelf(); err != nil {
				log.
					WithFields(log.Fields{
						"error": err,
						"time":  ts.Format(time.RFC3339),
					}).
					Error("cassemagent.app.renewSelf failed")
			}
		case <-d.quit:
			log.Info("cassemagent.app receives a quit signal")
			timeoutCtx, cancel = context.WithTimeout(context.Background(), 3*time.Second)
			if err = d.aggregate.Unregister(timeoutCtx, d.uniqueId); err != nil {
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
