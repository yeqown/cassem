package app

import (
	"context"
	"math/rand"
	"strconv"
	"time"

	"github.com/hashicorp/go-uuid"
	"github.com/pkg/errors"
	"github.com/yeqown/log"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"

	apiagent "github.com/yeqown/cassem/internal/cassemagent/api"
	"github.com/yeqown/cassem/internal/cassemagent/domain"
	"github.com/yeqown/cassem/internal/concept"
	"github.com/yeqown/cassem/pkg/conf"
	"github.com/yeqown/cassem/pkg/grpcx"
	"github.com/yeqown/cassem/pkg/httpx"
	"github.com/yeqown/cassem/pkg/runtime"
)

type app struct {
	uniqueId string
	// TODO(@yeqown): trigger quit from TERMINATED/KILL signal.
	quit chan struct{}

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
		uniqueId:     uniqueId(),
		quit:         make(chan struct{}, 1),
		conf:         c,
		aggregate:    agg,
		cache:        domain.NewCache(1000), // TODO(@yeqown): measure the parameter of 1000
		instancePool: domain.NewInstancePool(),
	}

	return d, nil
}

func (d app) Run() {
	d.startRoutines()

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
		d.shutdown()
		log.Fatal(err)
	}
}

func (d app) shutdown() {
	select {
	case d.quit <- struct{}{}:
	default:
	}
}

func (d *app) startRoutines() {
	d.actualRenewInterval = d.conf.RenewInterval + rand.Int31n(d.conf.TTL-d.conf.RenewInterval)
	timeoutCtx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
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
	}
	cancel()

	runtime.GoFunc("renew", func() error {
		// actualRenewInterval = conf.renewInterval + int32n(conf.TTL - cond.RenewInterval)
		dur := time.Duration(d.actualRenewInterval) * time.Second
		ticker := time.NewTicker(dur)

		for {
			select {
			case ts := <-ticker.C:
				log.Info("cassemagent.app renewSelf")
				if err = d.renewSelf(); err != nil {
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
	})
}

func (d app) renewSelf() error {
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

// uniqueId panics if any error encountered during apply unique id.
func uniqueId() string {
	buf, err := uuid.GenerateRandomBytes(16)
	if err != nil {
		panic(err)
	}

	uid, err2 := uuid.FormatUUID(buf)
	if err2 != nil {
		panic(err2)
	}

	return uid
}
