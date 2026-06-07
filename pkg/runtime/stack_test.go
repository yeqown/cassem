package runtime

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestStack(t *testing.T) {
	stack := Stack()
	assert.NotNil(t, stack)
	assert.Greater(t, len(stack), 0, "Stack should return non-empty bytes")
	// Stack trace should contain some common patterns
	stackStr := string(stack)
	assert.Contains(t, stackStr, "goroutine")
}

func TestRecoverFrom(t *testing.T) {
	tests := []struct {
		name    string
		panicVal any
	}{
		{"string panic", "test panic"},
		{"int panic", 42},
		{"nil panic", nil},
		{"error panic", assert.AnError},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := RecoverFrom(tt.panicVal)
			assert.Error(t, err)
			assert.Contains(t, err.Error(), "panic")
		})
	}
}
