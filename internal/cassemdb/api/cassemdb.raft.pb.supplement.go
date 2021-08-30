package api

//import "google.golang.org/protobuf/proto"
import (
	"time"

	"github.com/golang/protobuf/proto"

	"github.com/yeqown/cassem/pkg/hash"
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
	return
}

func (*SetCommand) Action() LogEntry_Action    { return LogEntry_Set }
func (*ChangeCommand) Action() LogEntry_Action { return LogEntry_ChangeSpread }

func NewEntityWithCreated(key string, val []byte, ttl int32, created int64) *Entity {
	return &Entity{
		Fingerprint: hash.MD5(val),
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

func (m Entity) Type() EntityType {
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
	if m.Ttl == NEVER_EXPIRED {
		return NEVER_EXPIRED
	}

	m.Ttl -= int32(time.Now().Unix() - m.UpdatedAt)
	if m.Ttl <= 0 {
		m.Ttl = EXPIRED
	}

	return m.Ttl
}

func calculateTTL(ttl int32) int32 {
	if ttl <= 0 {
		return NEVER_EXPIRED
	}

	return ttl
}

const (
	// _expiredInterval means how long the log entry could live.
	_expiredInterval = 10
)

// Expired represents the LogEntry has expired, could not be applied by raft node.
// this method should only be used in some case which cares about duplicate log entries applied.
func (m *LogEntry) Expired() bool {
	now := time.Now().Unix()
	if now-m.CreatedAt > _expiredInterval {
		return true
	}

	return false
}

// Propose is wrapper of log entry, and only used by node internal.
type Propose struct {
	Entry *LogEntry
	ErrC  chan<- error
}

func NewPropose(entry *LogEntry, errC chan<- error) *Propose {
	if entry == nil || errC == nil {
		panic("invalid parameters for commit")
	}

	return &Propose{
		Entry: entry,
		ErrC:  errC,
	}
}
