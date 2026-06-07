package runtime

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestBytesToString(t *testing.T) {
	tests := []struct {
		name     string
		input    []byte
		expected string
	}{
		{"empty bytes", []byte{}, ""},
		{"simple string", []byte("hello"), "hello"},
		{"with spaces", []byte("hello world"), "hello world"},
		{"with special chars", []byte("hello\nworld"), "hello\nworld"},
		{"unicode", []byte("hello世界"), "hello世界"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ToString(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestStringToBytes(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected []byte
	}{
		{"empty string", "", []byte{}},
		{"simple string", "hello", []byte("hello")},
		{"with spaces", "hello world", []byte("hello world")},
		{"with special chars", "hello\nworld", []byte("hello\nworld")},
		{"unicode", "hello世界", []byte("hello世界")},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ToBytes(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestBytesStringRoundTrip(t *testing.T) {
	original := "test string with unicode: 你好世界"
	bytes := ToBytes(original)
	backToString := ToString(bytes)
	assert.Equal(t, original, backToString)
}

func TestIndexOf(t *testing.T) {
	tests := []struct {
		name        string
		target      string
		arr         []string
		expectedPos int
	}{
		{"found at beginning", "apple", []string{"apple", "banana", "cherry"}, 0},
		{"found in middle", "banana", []string{"apple", "banana", "cherry"}, 1},
		{"found at end", "cherry", []string{"apple", "banana", "cherry"}, 2},
		{"not found", "dragon", []string{"apple", "banana", "cherry"}, -1},
		{"empty array", "apple", []string{}, -1},
		{"empty target", "", []string{"", "apple", "banana"}, 0},
		{"duplicate - first occurrence", "apple", []string{"apple", "banana", "apple"}, 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pos := IndexOf(tt.target, tt.arr)
			assert.Equal(t, tt.expectedPos, pos)
		})
	}
}
