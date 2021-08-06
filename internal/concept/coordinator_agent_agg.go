package concept

import (
	apicassemdb "github.com/yeqown/cassem/internal/cassemdb/api"
	pbcassemdb "github.com/yeqown/cassem/internal/cassemdb/api/gen"
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

	c := pbcassemdb.NewKVClient(cc)
	return agentAggregate{
		kvReadOnly:     kvReadOnly{cassemdb: c},
		instanceHybrid: instanceHybrid{cassemdb: c},
	}, nil
}
