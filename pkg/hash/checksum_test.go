package hash_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/yeqown/cassem/pkg/hash"
)

func Test_CheckSum(t *testing.T) {
	content := []byte("this is a conent to test checksum")
	sum1 := hash.CheckSum(content)
	sum2 := hash.CheckSum(content)

	assert.Equal(t, sum1, sum2)
	t.Log(sum1)
	t.Log(sum2)

	conent2 := append(content, []byte("append value")...)
	sum3 := hash.CheckSum(conent2)
	assert.NotEqual(t, sum2, sum3)
	t.Log(sum3)
}
