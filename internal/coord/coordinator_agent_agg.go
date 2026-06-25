package coord

import (
	"github.com/yeqown/cassem/api/concept"
	apikv "github.com/yeqown/cassem/api/kv"
)

type agentAggregate struct {
	kvReadOnly
	instanceHybrid
	agentInsHybrid
}

func NewAgentAggregate(endpoints []string) (concept.AgentAggregate, error) {
	cc, err := apikv.DialWithMode(endpoints, apikv.Mode_X)
	if err != nil {
		return nil, err
	}

	c := apikv.NewKVClient(cc)
	return agentAggregate{
		kvReadOnly:     kvReadOnly{cassemkv: c},
		instanceHybrid: instanceHybrid{cassemkv: c},
		agentInsHybrid: agentInsHybrid{cassemkv: c},
	}, nil
}
