package coordinator

import (
	"github.com/yeqown/cassem/api/concept"
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
		kvReadOnly:     kvReadOnly{cassemdb: c},
		instanceHybrid: instanceHybrid{cassemdb: c},
		agentInsHybrid: agentInsHybrid{cassemdb: c},
	}, nil
}
