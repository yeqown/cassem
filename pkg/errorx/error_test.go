package errorx

import (
	"testing"

	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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
	err2 := errors.Wrap(err, "layer1")
	err3 := errors.Wrap(err2, "layer2")

	assert.True(t, errors.Is(err3, err))

	err4 := errors.Cause(err3)
	t.Logf("err4=%+v, err=%+v", err4, err)
	assert.True(t, err4 == err)
}

func Test_error_ToStatus(t *testing.T) {
	err := New(Code_UNKNOWN, "unknown")
	err2 := errors.Wrapf(err, "wrap1")

	err3, ok := FromError(err2)
	require.True(t, ok)
	assert.Equal(t, err, err3)
}
