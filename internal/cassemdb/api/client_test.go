package api

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	pb "github.com/yeqown/cassem/internal/cassemdb/api/gen"
)

func TestDialCassemDB(t *testing.T) {
	conn, err := DialWithMode([]string{"127.0.0.1:2021", "127.0.0.1:2022", "127.0.0.1:2023"}, Mode_R)
	require.NoError(t, err)
	assert.NotNil(t, conn)

	c := pb.NewKVClient(conn)

	for i := 1; i <= 100; i++ {
		_, err = c.GetKV(context.Background(), &pb.GetKVReq{
			//Key: "bench/" + strconv.Itoa(i),
			Key: "a/b",
		})
		assert.NoError(t, err)
	}
}
