package concept

import (
	apicassemdb "github.com/yeqown/cassem/internal/cassemdb/api"
)

type agentAggregate struct {
	kvReadOnly
	instanceHybrid
}

func NewAgentAggregate(endpoints []string) (AgentAggregate, error) {
	cc, err := apicassemdb.DialWithMode(endpoints, apicassemdb.Mode_X)
	if err != nil {
		return nil, err
	}

	c := apicassemdb.NewKVClient(cc)
	return agentAggregate{
		kvReadOnly:     kvReadOnly{cassemdb: c},
		instanceHybrid: instanceHybrid{cassemdb: c},
	}, nil
}
