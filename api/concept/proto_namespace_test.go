package concept

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestProtoNamespaces(t *testing.T) {
	assert.Equal(t, "cassem.api.concept", string(File_types_proto.Package()))
	assert.Equal(t, "cassem.api.concept", string(File_acl_proto.Package()))
}
