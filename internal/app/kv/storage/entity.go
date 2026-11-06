package storage

import (
	"crypto/md5"
	"encoding/hex"
	"time"

	pb "github.com/yeqown/cassem/internal/protogen"
)

const (
	// NeverExpired marks an entity with no expiration deadline.
	NeverExpired int32 = -2
	// Expired marks an entity whose TTL has elapsed.
	Expired int32 = -1
)

// NewEntityWithCreated creates a stored entity, preserving its creation time.
func NewEntityWithCreated(key string, val []byte, ttl int32, created int64) *pb.Entity {
	fingerprint := md5.Sum(val)
	if ttl == 0 {
		ttl = NeverExpired
	}
	return &pb.Entity{
		Fingerprint: hex.EncodeToString(fingerprint[:]),
		Key:         key,
		Val:         val,
		CreatedAt:   created,
		UpdatedAt:   time.Now().Unix(),
		Ttl:         ttl,
		Typ:         pb.EntityType_ELT,
	}
}

// EntityExpired checks and updates the entity's remaining TTL.
// The caller owns the entity; this preserves the storage layer's TTL mutation.
func EntityExpired(entity *pb.Entity) bool {
	switch entity.GetTtl() {
	case NeverExpired:
		return false
	case Expired:
		return true
	}
	entity.Ttl -= int32(time.Now().Unix() - entity.UpdatedAt)
	if entity.Ttl <= 0 {
		entity.Ttl = Expired
	}
	return entity.Ttl == Expired
}
