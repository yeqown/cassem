package grpcx

import (
	"strings"

	"github.com/yeqown/log"
	"google.golang.org/grpc/resolver"
	"google.golang.org/grpc/serviceconfig"
)

var (
	_ resolver.Resolver = cassemdbResolver{}
	_ resolver.Builder  = cassemdbResolverBuilder{}
)

// cassemdbResolver endpoints comes from config and keep fixed, so cassemdbResolver.ResolveNow would never
// update resolver.ClientConn's state once resolver.Builder called.
type cassemdbResolver struct{}

func (c cassemdbResolver) ResolveNow(option resolver.ResolveNowOption) {}
func (c cassemdbResolver) Close()                                      {}

type cassemdbResolverBuilder struct{}

func (c cassemdbResolverBuilder) Build(
	target resolver.Target, cc resolver.ClientConn, opts resolver.BuildOption) (resolver.Resolver, error) {
	log.
		WithFields(log.Fields{
			"target": target,
		}).
		Debug("cassemdbResolverBuilder called")

	endpoints := strings.Split(target.Endpoint, ",")
	addrs := make([]resolver.Address, 0, len(endpoints))
	for _, v := range endpoints {
		addrs = append(addrs, resolver.Address{
			Addr:       v,
			Type:       resolver.Backend,
			ServerName: "cassemdb:" + v,
			Metadata:   nil,
		})
	}

	sc, _ := serviceconfig.Parse(_SERVICE_CONFIG_JSON)
	log.
		WithFields(log.Fields{
			"sc": sc,
		}).
		Debug("cassemdbResolverBuilder parse service config")

	cc.UpdateState(resolver.State{
		Addresses:     addrs,
		ServiceConfig: sc,
	})

	return cassemdbResolver{}, nil
}

var (

	// _SERVICE_CONFIG_JSON https://github.com/grpc/grpc/blob/master/doc/service_config.md
	_SERVICE_CONFIG_JSON = `{"loadBalancingConfig":[{"round_robin":{}}]}`
)

func (c cassemdbResolverBuilder) Scheme() string {
	return "cassemdb"
}
