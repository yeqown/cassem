package storage

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestEntityTTL(t *testing.T) {
	tests := []struct {
		name    string
		ttl     int32
		expired bool
		wantTTL int32
	}{
		{
			name:    "zero ttl never expires",
			ttl:     0,
			expired: false,
			wantTTL: NeverExpired,
		},
		{
			name:    "never expired sentinel",
			ttl:     NeverExpired,
			expired: false,
			wantTTL: NeverExpired,
		},
		{
			name:    "expired sentinel",
			ttl:     Expired,
			expired: true,
			wantTTL: Expired,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			entity := NewEntityWithCreated("ttl/key", []byte("value"), tt.ttl, time.Now().Unix())

			assert.Equal(t, tt.expired, EntityExpired(entity))
			assert.Equal(t, tt.wantTTL, entity.GetTtl())
		})
	}
}

func TestEntityPositiveTTLExpires(t *testing.T) {
	entity := NewEntityWithCreated("ttl/key", []byte("value"), 1, time.Now().Unix())
	entity.UpdatedAt = time.Now().Add(-2 * time.Second).Unix()

	assert.True(t, EntityExpired(entity))
	assert.Equal(t, Expired, entity.GetTtl())
}
