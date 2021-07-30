package concept

import (
	apicassemdb "github.com/yeqown/cassem/internal/cassemdb/api"
	pbcassemdb "github.com/yeqown/cassem/internal/cassemdb/api/gen"
)

type admAggregate struct {
	kvReadOnly
	kvWriteOnly
	instanceHybrid
}

func NewAdmAggregate(endpoints []string) (AdmAggregate, error) {
	cc, err := apicassemdb.DialWithMode(endpoints, apicassemdb.Mode_X)
	if err != nil {
		return nil, err
	}

	c := pbcassemdb.NewKVClient(cc)
	return admAggregate{
		kvReadOnly:     kvReadOnly{cassemdb: c},
		kvWriteOnly:    kvWriteOnly{cassemdb: c},
		instanceHybrid: instanceHybrid{cassemdb: c},
	}, nil
}
