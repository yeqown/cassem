package coordinator

import (
	"fmt"

	"github.com/yeqown/cassem/api/concept"
	apicassemdb "github.com/yeqown/cassem/internal/cassemdb/api"
)

type admAggregate struct {
	kvReadOnly
	kvWriteOnly
	instanceHybrid
	agentInsHybrid
	aclImpl
}

func NewAdmAggregate(endpoints []string) (concept.AdmAggregate, error) {
	cc, err := apicassemdb.DialWithMode(endpoints, apicassemdb.Mode_X)
	if err != nil {
		return nil, err
	}

	c := apicassemdb.NewKVClient(cc)

	acl, err := newRBAC(c)
	if err != nil {
		return nil, fmt.Errorf("NewAdmAggregate: %w", err)
	}
	if err = acl.AutoMigrate(); err != nil {
		return nil, fmt.Errorf("NewAdmAggregate.AutoMigrate: %w", err)
	}

	return admAggregate{
		kvReadOnly:     kvReadOnly{cassemdb: c},
		kvWriteOnly:    kvWriteOnly{cassemdb: c},
		instanceHybrid: instanceHybrid{cassemdb: c},
		agentInsHybrid: agentInsHybrid{cassemdb: c},
		aclImpl:        acl.(aclImpl),
	}, nil
}
