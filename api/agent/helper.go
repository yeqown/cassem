package agent

import (
	"context"

	"github.com/pkg/errors"
	"google.golang.org/grpc"

	"github.com/yeqown/cassem/pkg/grpcx"
)

func DialDelivery(addr string) (DeliveryClient, error) {
	timeout, cancel := context.WithTimeout(context.Background(), _CLIENT_INIT_TIMEOUT)
	defer cancel()

	cc, err := grpc.DialContext(timeout, addr,
		grpc.WithInsecure(),
		grpc.WithBlock(),
		grpc.WithChainUnaryInterceptor(grpcx.ClientRecovery(), grpcx.ClientErrorx(), grpcx.ClientValidation()),
	)
	if err != nil {
		return nil, errors.Wrap(err, "cassemagent.api.Dial")
	}

	return NewDeliveryClient(cc), nil
}
