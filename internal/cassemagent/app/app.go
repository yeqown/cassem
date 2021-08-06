package app

import (
	"log"

	"github.com/pkg/errors"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"

	apiagent "github.com/yeqown/cassem/internal/cassemagent/api"
	"github.com/yeqown/cassem/internal/cassemagent/domain"
	"github.com/yeqown/cassem/internal/concept"
	"github.com/yeqown/cassem/pkg/conf"
	"github.com/yeqown/cassem/pkg/httpx"
)

type app struct {
	conf *conf.CassemAgentConfig

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
		conf:         c,
		aggregate:    agg,
		cache:        domain.NewCache(1000), // TODO(@yeqown): measure the parameter of 1000
		instancePool: domain.NewInstancePool(),
	}

	return d, nil
}

func (d app) Run() {
	s := grpc.NewServer()
	apiagent.RegisterAgentServer(s, d)
	reflection.Register(s)
	gate := httpx.NewGateway(d.conf.Server.Addr, nil, s)
	if err := gate.ListenAndServe(); err != nil {
		log.Fatal(err)
	}
}

// id returns the unique string of agent.
func (d app) id() string {
	return "uniqueId(todo@yeqown)"
}
