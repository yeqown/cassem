package storage

import (
	pb "github.com/yeqown/cassem/internal/protogen"
)

type empty struct{}

func (e empty) GetKV(key string, isDir bool) (*pb.Entity, error) {
	return &pb.Entity{
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

func (e empty) SetKV(key string, value *pb.Entity, isDir bool) error {
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

func (e empty) Snapshot() ([]byte, error) {
	return nil, nil
}

func (e empty) RecoverSnapshot([]byte) error {
	return nil
}
