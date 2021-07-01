package grpcx

import (
	"context"
	"strconv"
	"testing"

	pb "github.com/yeqown/cassem/internal/cassemdb/api/grpc/gen"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDialCassemDB(t *testing.T) {
	target := "cassemdb://noauth/127.0.0.1:2021,127.0.0.1:2022,127.0.0.1:2023"
	conn, err := DialCassemDB(target)
	require.NoError(t, err)
	assert.NotNil(t, conn)

	c := pb.NewApiClient(conn)

	for i := 1; i <= 100; i++ {
		_, err = c.GetKV(context.Background(), &pb.GetKVReq{
			Key: "bench/" + strconv.Itoa(i),
		})
		assert.NoError(t, err)
	}
}
