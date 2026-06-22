package coord

import (
	"fmt"
	apikv "github.com/yeqown/cassem/api/kv"

	"github.com/yeqown/cassem/api/concept"
)

type admAggregate struct {
	kvReadOnly
	kvWriteOnly
	instanceHybrid
	agentInsHybrid
	aclImpl
}

func NewAdmAggregate(endpoints []string) (concept.AdmAggregate, error) {
	cc, err := apikv.DialWithMode(endpoints, apikv.Mode_X)
	if err != nil {
		return nil, err
	}

	c := apikv.NewKVClient(cc)

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
