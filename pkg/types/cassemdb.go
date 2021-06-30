package types

import (
	"encoding/json"
	"time"

	"github.com/yeqown/log"

	"github.com/yeqown/cassem/pkg/hash"
)

type StoreKey string

func (k StoreKey) String() string {
	return string(k)
}

type StoreValue struct {
	Fingerprint string   `json:"fingerprint"`
	Key         StoreKey `json:"key"`
	Val         []byte   `json:"val"`
	Size        int64    `json:"size"`
	CreatedAt   int64    `json:"createdAt"`
	UpdatedAt   int64    `json:"updatedAt"`
}

func (s *StoreValue) Unmarshal(bytes []byte) error {
	return json.Unmarshal(bytes, s)
}

func (s StoreValue) Marshal() ([]byte, error) {
	return json.Marshal(s)
}

// NewStoreKV construct a StoreKey, StoreValue pair from raw data.
func NewStoreKV(key string, val []byte) (StoreKey, StoreValue) {
	return NewStoreKVWith(key, val, time.Now().Unix())
}

func NewStoreKVWith(key string, val []byte, created int64) (StoreKey, StoreValue) {
	k := StoreKey(key)
	v := StoreValue{
		Fingerprint: hash.MD5(val),
		Key:         k,
		Val:         val,
		Size:        int64(len(val)),
		CreatedAt:   created,
		UpdatedAt:   time.Now().Unix(),
	}

	return k, v
}

//go:generate stringer -type=Op
type ChangeOp uint8

const (
	OpSet ChangeOp = iota + 1
	OpUnset
)

type Change struct {
	Op      ChangeOp    `json:"op"`
	Key     StoreKey    `json:"key"`
	Last    *StoreValue `json:"last"`
	Current *StoreValue `json:"current"`

	data []byte
}

func (c *Change) Topic() string {
	return c.Key.String()
}

func (c *Change) Data() []byte {
	if c.data != nil {
		return c.data
	}

	var err error
	c.data, err = json.Marshal(c)
	if err != nil {
		log.
			WithFields(log.Fields{
				"key":     c.Key,
				"last":    c.Last,
				"current": c.Current,
				"error":   err,
			}).
			Error("Change.Data failed")
	}

	return c.data
}
