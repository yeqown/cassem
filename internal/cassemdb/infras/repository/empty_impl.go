package repository

type empty struct{}

func (e empty) GetKV(key StoreKey, isDir bool) (*StoreValue, error) {
	return &StoreValue{
		Fingerprint: "empty",
		Key:         key,
		Val:         []byte("empty to test"),
		Size:        32,
		CreatedAt:   1629968602,
		UpdatedAt:   1629968602,
		TTL:         12,
	}, nil
}

func NewEmptyRepository() KV {
	return empty{}
}

func (e empty) SetKV(key StoreKey, value *StoreValue, isDir bool) error {
	return nil
}

func (e empty) UnsetKV(key StoreKey, isDir bool) error {
	return nil
}

func (e empty) Range(key StoreKey, seek string, limit int) (*RangeResult, error) {
	return &RangeResult{
		Items:       nil,
		HasMore:     false,
		NextSeekKey: "",
		ExpiredKeys: nil,
	}, nil
}
