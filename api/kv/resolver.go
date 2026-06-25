package kv

import (
	"strings"

	"github.com/yeqown/log"
	"google.golang.org/grpc/resolver"
)

var (
	_ resolver.Resolver = cassemkvResolver{}
	_ resolver.Builder  = cassemkvResolverBuilder{}
)

// cassemkvResolver endpoints comes from config and keep fixed, so cassemkvResolver.ResolveNow would never
// update resolver.ClientConn's state once resolver.Builder called.
type cassemkvResolver struct{}

func (c cassemkvResolver) ResolveNow(option resolver.ResolveNowOptions) {}
func (c cassemkvResolver) Close()                                       {}

type cassemkvResolverBuilder struct{}

func (c cassemkvResolverBuilder) Build(
	target resolver.Target, cc resolver.ClientConn, opts resolver.BuildOptions) (resolver.Resolver, error) {
	log.
		WithFields(log.Fields{
			"target": target,
		}).
		Debug("cassemkvResolverBuilder called")

	endpoint := strings.TrimPrefix(target.Endpoint(), "all//")
	endpoints := strings.Split(endpoint, ",")
	addrs := make([]resolver.Address, 0, len(endpoints))
	for _, v := range endpoints {
		if v == "" {
			continue
		}
		addrs = append(addrs, resolver.Address{
			Addr:       v,
			ServerName: "cassemkv:" + v,
			Attributes: nil,
		})
	}

	// scPlain := _SERVICE_CONFIG_JSON_WITH_HEALTH
	// switch target.Authority {
	// case "all":
	//	scPlain = _SERVICE_CONFIG_JSON_WITHOUT_HEALTH
	// default:
	// }

	// sc, _ := serviceconfig.Parse(scPlain)
	// log.
	//	WithFields(log.Fields{
	//		"sc": sc,
	//	}).
	//	Debug("cassemkvResolverBuilder parse service config")

	_ = cc.UpdateState(resolver.State{
		Addresses: addrs,
		// ServiceConfig: sc,
	})

	return cassemkvResolver{}, nil
}

var (
	// _SERVICE_CONFIG_JSON https://github.com/grpc/grpc/blob/master/doc/service_config.md
	_SERVICE_CONFIG_JSON_WITH_HEALTH    = `{"healthCheckConfig":{"serviceName": "cassemkv.RaftLeader"},"loadBalancingConfig":[{"round_robin":{}}]}`
	_SERVICE_CONFIG_JSON_WITHOUT_HEALTH = `{"loadBalancingConfig":[{"round_robin":{}}]}`
)

func (c cassemkvResolverBuilder) Scheme() string {
	return "cassemkv"
}
