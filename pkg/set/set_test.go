package set

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewStringSet(t *testing.T) {
	tests := []struct {
		name string
		size int
	}{
		{"zero size", 0},
		{"negative size", -5},
		{"positive size", 10},
		{"large size", 100},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := NewStringSet(tt.size)
			assert.NotNil(t, s)
			// Map capacity is not directly testable through len()
			// but we can verify the map is functional
			assert.Equal(t, 0, len(s), "New set should be empty")
		})
	}
}

func TestStringSet_Add(t *testing.T) {
	s := NewStringSet(8)

	// Test adding new key
	evicted := s.Add("key1")
	assert.False(t, evicted, "Adding new key should return false")
	assert.Equal(t, 1, len(s), "Set should have 1 element")

	// Test adding duplicate key
	evicted = s.Add("key1")
	assert.True(t, evicted, "Adding duplicate key should return true")
	assert.Equal(t, 1, len(s), "Set should still have 1 element")

	// Test adding multiple keys
	s.Add("key2")
	s.Add("key3")
	assert.Equal(t, 3, len(s), "Set should have 3 elements")

	// Test adding duplicate of different key
	evicted = s.Add("key2")
	assert.True(t, evicted)
	assert.Equal(t, 3, len(s))
}

func TestStringSet_Del(t *testing.T) {
	s := NewStringSet(8)

	s.Add("key1")
	s.Add("key2")
	s.Add("key3")

	// Test deleting existing key
	s.Del("key2")
	assert.Equal(t, 2, len(s))
	assert.Equal(t, StringSet{"key1": {}, "key3": {}}, s)

	// Test deleting non-existing key (should be safe)
	s.Del("key4")
	assert.Equal(t, 2, len(s))

	// Test deleting from empty set
	empty := NewStringSet(0)
	empty.Del("anything")
	assert.Equal(t, 0, len(empty), "Empty set should remain empty")
}

func TestStringSet_Adds(t *testing.T) {
	s := NewStringSet(8)

	// Test adding multiple keys at once
	keys := []string{"key1", "key2", "key3"}
	s.Adds(keys)
	assert.Equal(t, 3, len(s))

	// Test that all keys are present
	for _, key := range keys {
		_, exists := s[key]
		assert.True(t, exists, "Key %s should exist in set", key)
	}

	// Test adding empty slice
	s.Adds([]string{})
	assert.Equal(t, 3, len(s))

	// Test adding with duplicates
	s.Adds([]string{"key1", "key4"})
	assert.Equal(t, 4, len(s))
}

func TestStringSet_Keys(t *testing.T) {
	s := NewStringSet(8)

	// Test empty set
	keys := s.Keys()
	assert.NotNil(t, keys)
	assert.Equal(t, 0, len(keys))

	// Test with elements
	s.Adds([]string{"key1", "key2", "key3"})
	keys = s.Keys()
	assert.Equal(t, 3, len(keys))

	// Test that all keys are present
	keyMap := make(map[string]struct{})
	for _, key := range keys {
		keyMap[key] = struct{}{}
	}
	for _, key := range []string{"key1", "key2", "key3"} {
		_, exists := keyMap[key]
		assert.True(t, exists, "Key %s should be in keys", key)
	}
}

func TestStringSet_Integration(t *testing.T) {
	s := NewStringSet(8)

	// Test a complete workflow
	assert.False(t, s.Add("apple"))
	assert.False(t, s.Add("banana"))
	assert.False(t, s.Add("cherry"))

	assert.Equal(t, 3, len(s))

	keys := s.Keys()
	assert.Equal(t, 3, len(keys))

	s.Del("banana")
	assert.Equal(t, 2, len(s))

	keys = s.Keys()
	assert.Equal(t, 2, len(keys))

	// Test that deleted key can be added again (not evicted)
	assert.False(t, s.Add("banana"), "Re-adding deleted key should return false")
	assert.Equal(t, 3, len(s))
}
