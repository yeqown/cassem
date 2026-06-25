package kv

// import "google.golang.org/protobuf/proto"
import (
	"context"
	"crypto/md5"
	"encoding/hex"
	"time"

	"google.golang.org/protobuf/proto"
)

func Marshal(m proto.Message) ([]byte, error) {
	return proto.Marshal(m)
}

func Must(d []byte, err error) []byte {
	if err != nil {
		panic(err)
	}

	return d
}

func Unmarshal(data []byte, m proto.Message) error {
	return proto.Unmarshal(data, m)
}

func MustUnmarshal(data []byte, m proto.Message) {
	if err := Unmarshal(data, m); err != nil {
		panic(err)
	}
}

func (*MutateCommand) Action() LogEntry_Action { return LogEntry_Mutate }

func NewEntityWithCreated(key string, val []byte, ttl int32, created int64) *Entity {
	h := md5.New()
	h.Write(val)
	return &Entity{
		Fingerprint: hex.EncodeToString(h.Sum(nil)),
		Key:         key,
		Val:         val,
		CreatedAt:   created,
		UpdatedAt:   time.Now().Unix(),
		Ttl:         calculateTTL(ttl),
		Typ:         EntityType_ELT,
	}
}

const (
	NEVER_EXPIRED = -2
	EXPIRED       = -1
)

func (m *Entity) Type() EntityType {
	if m.Val == nil && m.Size == 0 {
		return EntityType_DIR
	}

	return EntityType_ELT
}

func (m *Entity) Expired() bool {
	switch m.GetTtl() {
	case NEVER_EXPIRED:
		return false
	case EXPIRED:
		return true
	}

	return m.recalculateTTL() == EXPIRED
}

func (m *Entity) recalculateTTL() int32 {
	switch m.Ttl {
	case NEVER_EXPIRED, EXPIRED:
		return m.Ttl
	}

	m.Ttl -= int32(time.Now().Unix() - m.UpdatedAt)
	if m.Ttl <= 0 {
		m.Ttl = EXPIRED
	}

	return m.Ttl
}

func calculateTTL(ttl int32) int32 {
	switch ttl {
	case EXPIRED, NEVER_EXPIRED:
		return ttl
	case 0:
		return NEVER_EXPIRED
	default:
		return ttl
	}
}

const (
	// _expiredInterval means how long the log entry could live.
	_expiredInterval = 10
)

// Expired represents the LogEntry has expired, could not be applied by raft node.
// this method should only be used in some case which cares about duplicate log entries applied.
func (m *LogEntry) Expired() bool {
	now := time.Now().Unix()
	return now-m.CreatedAt > _expiredInterval
}

// Propose is wrapper of log entry, and only used by node internal.
type Propose struct {
	Ctx   context.Context
	Entry *LogEntry
	ErrC  chan<- error
}

func NewPropose(entry *LogEntry, errC chan<- error) *Propose {
	return NewProposeWithContext(context.Background(), entry, errC)
}

func NewProposeWithContext(ctx context.Context, entry *LogEntry, errC chan<- error) *Propose {
	if ctx == nil || entry == nil || errC == nil {
		panic("invalid parameters for commit")
	}

	return &Propose{
		Ctx:   ctx,
		Entry: entry,
		ErrC:  errC,
	}
}
