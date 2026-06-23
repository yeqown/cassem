package agent

import (
	"context"

	"fmt"
	"google.golang.org/grpc"

	apiinternal "github.com/yeqown/cassem/api/internal"
)

func DialDelivery(addr string) (DeliveryClient, error) {
	timeout, cancel := context.WithTimeout(context.Background(), _CLIENT_INIT_TIMEOUT)
	defer cancel()

	cc, err := grpc.DialContext(timeout, addr,
		grpc.WithInsecure(),
		grpc.WithBlock(),
		grpc.WithChainUnaryInterceptor(apiinternal.ClientRecovery(), apiinternal.ClientErrorx(), apiinternal.ClientValidation()),
	)
	if err != nil {
		return nil, fmt.Errorf("cassemagent.api.Dial: %w", err)
	}

	return NewDeliveryClient(cc), nil
}
