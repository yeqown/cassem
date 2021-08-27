package storage

import apicassemdb "github.com/yeqown/cassem/internal/cassemdb/api"

type empty struct{}

func (e empty) GetKV(key string, isDir bool) (*apicassemdb.Entity, error) {
	return &apicassemdb.Entity{
		Fingerprint: "empty",
		Key:         key,
		Val:         []byte("empty to test"),
		Size:        32,
		CreatedAt:   1629968602,
		UpdatedAt:   1629968602,
		Ttl:         12,
	}, nil
}

func NewEmptyRepository() KV {
	return empty{}
}

func (e empty) SetKV(key string, value *apicassemdb.Entity, isDir bool) error {
	return nil
}

func (e empty) UnsetKV(key string, isDir bool) error {
	return nil
}

func (e empty) Range(key string, seek string, limit int) (*RangeResult, error) {
	return &RangeResult{
		Items:       nil,
		HasMore:     false,
		NextSeekKey: "",
		ExpiredKeys: nil,
	}, nil
}
