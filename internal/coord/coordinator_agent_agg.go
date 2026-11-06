package coord

import (
	"github.com/yeqown/cassem/internal/concept"
	"github.com/yeqown/cassem/internal/kvclient"
	pb "github.com/yeqown/cassem/internal/protogen"
)

type agentAggregate struct {
	kvReadOnly
	instanceHybrid
	agentInsHybrid
}

func NewAgentAggregate(endpoints []string) (concept.AgentAggregate, error) {
	cc, err := kvclient.DialWithMode(endpoints, kvclient.Mode_X)
	if err != nil {
		return nil, err
	}

	c := pb.NewKVClient(cc)
	return agentAggregate{
		kvReadOnly:     kvReadOnly{cassemkv: c},
		instanceHybrid: instanceHybrid{cassemkv: c},
		agentInsHybrid: agentInsHybrid{cassemkv: c},
	}, nil
}
