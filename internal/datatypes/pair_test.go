package datatypes

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func Test_NewPair(t *testing.T) {
	p := NewPair("ns-pair", "pair-int", WithInt(123123))

	assert.Equal(t, INT_DATATYPE_, p.Value().Datatype())
	assert.Equal(t, IntDT(123123), p.Value().Data())
	assert.Equal(t, "pair-int", p.Key())
	assert.Equal(t, "ns-pair", p.NS())
}
