package repository

import (
	"encoding/json"
	"time"

	"github.com/yeqown/log"

	pb "github.com/yeqown/cassem/internal/cassemdb/api/gen"
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
	// TTL means Time to Live. -1: expired, -2: never expired. 0+ means normal time to live.
	TTL int32 `json:"ttl"`
}

const (
	NEVER_EXPIRED = -2
	EXPIRED       = -1
)

func (s StoreValue) Type() pb.EntityType {
	if s.Val == nil && s.Size == 0 {
		return pb.EntityType_DIR
	}

	return pb.EntityType_ELT
}

func (s *StoreValue) Expired() bool {
	switch s.TTL {
	case NEVER_EXPIRED:
		return false
	case EXPIRED:
		return true
	}

	return s.recalculateTTL() == EXPIRED
}

func (s *StoreValue) recalculateTTL() int32 {
	if s.TTL == NEVER_EXPIRED {
		return NEVER_EXPIRED
	}

	s.TTL -= int32(time.Now().Unix() - s.UpdatedAt)
	if s.TTL <= 0 {
		s.TTL = EXPIRED
	}

	return s.TTL
}

func (s *StoreValue) Unmarshal(bytes []byte) error {
	return json.Unmarshal(bytes, s)
}

func (s StoreValue) Marshal() ([]byte, error) {
	return json.Marshal(s)
}

//// NewKV construct a StoreKey, StoreValue pair from raw data.
//func NewKV(key string, val []byte, ttl uint32) (StoreKey, StoreValue) {
//	return NewKVWithCreatedAt(key, val, ttl, time.Now().Unix())
//}

func NewKVWithCreatedAt(key string, val []byte, ttl uint32, created int64) (StoreKey, StoreValue) {
	k := StoreKey(key)

	v := StoreValue{
		Fingerprint: hash.MD5(val),
		Key:         k,
		Val:         val,
		Size:        int64(len(val)),
		TTL:         calculateTTL(ttl),
		CreatedAt:   created,
		UpdatedAt:   time.Now().Unix(),
	}

	return k, v
}

func calculateTTL(ttl uint32) int32 {
	if ttl == 0 {
		return NEVER_EXPIRED
	}

	return int32(ttl)
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

type RangeResult struct {
	Items       []StoreValue
	HasMore     bool
	NextSeekKey string
	ExpiredKeys []string
}
