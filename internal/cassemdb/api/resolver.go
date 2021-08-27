package api

import (
	"strings"

	"github.com/yeqown/log"
	"google.golang.org/grpc/resolver"
)

var (
	_ resolver.Resolver = cassemdbResolver{}
	_ resolver.Builder  = cassemdbResolverBuilder{}
)

// cassemdbResolver endpoints comes from config and keep fixed, so cassemdbResolver.ResolveNow would never
// update resolver.ClientConn's state once resolver.Builder called.
type cassemdbResolver struct{}

func (c cassemdbResolver) ResolveNow(option resolver.ResolveNowOptions) {}
func (c cassemdbResolver) Close()                                       {}

type cassemdbResolverBuilder struct{}

func (c cassemdbResolverBuilder) Build(
	target resolver.Target, cc resolver.ClientConn, opts resolver.BuildOptions) (resolver.Resolver, error) {
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
			ServerName: "cassemdb:" + v,
			Attributes: nil,
		})
	}

	//scPlain := _SERVICE_CONFIG_JSON_WITH_HEALTH
	//switch target.Authority {
	//case "all":
	//	scPlain = _SERVICE_CONFIG_JSON_WITHOUT_HEALTH
	//default:
	//}

	// sc, _ := serviceconfig.Parse(scPlain)
	//log.
	//	WithFields(log.Fields{
	//		"sc": sc,
	//	}).
	//	Debug("cassemdbResolverBuilder parse service config")

	_ = cc.UpdateState(resolver.State{
		Addresses: addrs,
		// ServiceConfig: sc,
	})

	return cassemdbResolver{}, nil
}

var (
	// _SERVICE_CONFIG_JSON https://github.com/grpc/grpc/blob/master/doc/service_config.md
	_SERVICE_CONFIG_JSON_WITH_HEALTH    = `{"healthCheckConfig":{"serviceName": "cassemdb.RaftLeader"},"loadBalancingConfig":[{"round_robin":{}}]}`
	_SERVICE_CONFIG_JSON_WITHOUT_HEALTH = `{"loadBalancingConfig":[{"round_robin":{}}]}`
)

func (c cassemdbResolverBuilder) Scheme() string {
	return "cassemdb"
}
