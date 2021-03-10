package httpc_test

import (
	"testing"

	"github.com/yeqown/cassem/pkg/httpc"

	"github.com/stretchr/testify/assert"
)

func Test_GET(t *testing.T) {
	resp := make(map[string]interface{})
	form := map[string]string{}
	err := httpc.GET("http://172.16.30.18:2022/api/namespaces", form, &resp)
	assert.Nil(t, err)
	t.Logf("%+v", resp)
}
