package coord

import (
	"fmt"

	"github.com/yeqown/cassem/internal/concept"
	"github.com/yeqown/cassem/internal/kvclient"
	pb "github.com/yeqown/cassem/internal/protogen"
)

type admAggregate struct {
	kvReadOnly
	kvWriteOnly
	instanceHybrid
	agentInsHybrid
	aclImpl
}

func NewAdmAggregate(endpoints []string) (concept.AdmAggregate, error) {
	cc, err := kvclient.DialWithMode(endpoints, kvclient.Mode_X)
	if err != nil {
		return nil, err
	}

	c := pb.NewKVClient(cc)

	acl, err := newRBAC(c)
	if err != nil {
		return nil, fmt.Errorf("NewAdmAggregate: %w", err)
	}
	if err = acl.AutoMigrate(); err != nil {
		return nil, fmt.Errorf("NewAdmAggregate.AutoMigrate: %w", err)
	}
	aclAgg, ok := acl.(aclImpl)
	if !ok {
		return nil, fmt.Errorf("NewAdmAggregate: unexpected ACL implementation %T", acl)
	}

	return admAggregate{
		kvReadOnly:     kvReadOnly{cassemkv: c},
		kvWriteOnly:    kvWriteOnly{cassemkv: c},
		instanceHybrid: instanceHybrid{cassemkv: c},
		agentInsHybrid: agentInsHybrid{cassemkv: c},
		aclImpl:        aclAgg,
	}, nil
}
