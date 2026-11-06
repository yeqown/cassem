package protogen

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestModelProtoNamespaces(t *testing.T) {
	assert.Equal(t, "cassem.api.concept", string(File_cassem_internal_models_proto.Package()))
	assert.Equal(t, "cassem.api.concept", string(File_cassem_internal_acl_proto.Package()))
}
