package coordinator

import (
	"github.com/yeqown/cassem/api/concept"
	apicassemdb "github.com/yeqown/cassem/internal/cassemdb/api"
)

type agentAggregate struct {
	kvReadOnly
	instanceHybrid
	agentInsHybrid
}

func NewAgentAggregate(endpoints []string) (concept.AgentAggregate, error) {
	cc, err := apicassemdb.DialWithMode(endpoints, apicassemdb.Mode_X)
	if err != nil {
		return nil, err
	}

	c := apicassemdb.NewKVClient(cc)
	return agentAggregate{
		kvReadOnly:     kvReadOnly{cassemdb: c},
		instanceHybrid: instanceHybrid{cassemdb: c},
		agentInsHybrid: agentInsHybrid{cassemdb: c},
	}, nil
}
