package concept

import (
	"github.com/pkg/errors"

	apicassemdb "github.com/yeqown/cassem/internal/cassemdb/api"
)

type admAggregate struct {
	kvReadOnly
	kvWriteOnly
	instanceHybrid
	agentInsHybrid
	aclImpl
}

func NewAdmAggregate(endpoints []string) (AdmAggregate, error) {
	cc, err := apicassemdb.DialWithMode(endpoints, apicassemdb.Mode_X)
	if err != nil {
		return nil, err
	}

	c := apicassemdb.NewKVClient(cc)

	acl, err := newRBAC(c)
	if err != nil {
		return nil, errors.Wrap(err, "NewAdmAggregate")
	}

	return admAggregate{
		kvReadOnly:     kvReadOnly{cassemdb: c},
		kvWriteOnly:    kvWriteOnly{cassemdb: c},
		instanceHybrid: instanceHybrid{cassemdb: c},
		agentInsHybrid: agentInsHybrid{cassemdb: c},
		aclImpl:        acl.(aclImpl),
	}, nil
}
