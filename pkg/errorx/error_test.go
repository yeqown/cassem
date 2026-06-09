package errorx

import (
	"errors"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

//func Test_errorx(t *testing.T) {
//	root := New(1, "mockCode")
//	err := Wrapf(root, "layer1")
//	err2 := Wrapf(err, "layer2")
//
//	t.Logf("root=%v, err2=%v, equal: %v", root, err2, Is(err2, root))
//	assert.True(t, Is(err2, root))
//
//	err3 := Unwrap(err2)
//	t.Logf("root=%v, err3=%v, equal: %v", root, err3, Is(err3, root))
//	assert.True(t, root == err3)
//}

func Test_errorx(t *testing.T) {
	err := New(1, "mockCode")
	err2 := fmt.Errorf("layer1: %w", err)
	err3 := fmt.Errorf("layer2: %w", err2)

	assert.True(t, errors.Is(err3, err))

	err4 := unwrapChain(err3)
	t.Logf("err4=%+v, err=%+v", err4, err)
	assert.True(t, err4 == err)
}

func Test_error_ToStatus(t *testing.T) {
	err := New(Code_UNKNOWN, "unknown")
	err2 := fmt.Errorf("wrap1: %w", err)

	err3, ok := FromError(err2)
	require.True(t, ok)
	assert.Equal(t, err, err3)
}

func TestToStatus(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		wantCode codes.Code
		wantMsg  string
	}{
		{name: "nil", err: nil, wantCode: codes.OK},
		{name: "plain error", err: errors.New("plain"), wantCode: codes.Unknown, wantMsg: "plain"},
		{name: "wrapped errorx", err: fmt.Errorf("wrap: %w", Err_ALREADY_EXISTS), wantCode: codes.AlreadyExists, wantMsg: "ALREADY_EXISTS"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ToStatus(tt.err)
			if tt.err == nil {
				assert.NoError(t, err)
				return
			}

			st, ok := status.FromError(err)
			require.True(t, ok)
			assert.Equal(t, tt.wantCode, st.Code())
			assert.Equal(t, tt.wantMsg, st.Message())
		})
	}
}

func TestFromStatus(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		wantCode Code
		wantMsg  string
	}{
		{name: "nil", err: nil, wantCode: Code_OK},
		{name: "grpc status", err: status.Error(codes.NotFound, "missing"), wantCode: Code_NOT_FOUND, wantMsg: "missing"},
		{name: "plain error", err: errors.New("plain"), wantCode: Code_UNKNOWN, wantMsg: "plain"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := FromStatus(tt.err)
			if tt.err == nil {
				assert.NoError(t, err)
				return
			}

			x, ok := FromError(err)
			require.True(t, ok)
			assert.Equal(t, tt.wantCode, x.Code)
			assert.Equal(t, tt.wantMsg, x.Message)
		})
	}
}
