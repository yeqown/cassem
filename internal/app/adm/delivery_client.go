package adm

import (
	"context"
	"fmt"
	"time"

	pb "github.com/yeqown/cassem/internal/protogen"

	"github.com/yeqown/cassem/internal/pkg/grpcx"
	"google.golang.org/grpc"
	"google.golang.org/grpc/connectivity"
	"google.golang.org/grpc/credentials/insecure"
)

const deliveryConnectTimeout = 10 * time.Second

func dialDelivery(addr string) (pb.DeliveryClient, error) {
	timeout, cancel := context.WithTimeout(context.Background(), deliveryConnectTimeout)
	defer cancel()

	cc, err := grpc.NewClient(addr,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithChainUnaryInterceptor(grpcx.ClientRecovery(), grpcx.ClientErrorx(), grpcx.ClientValidation()),
	)
	if err != nil {
		return nil, fmt.Errorf("cassemagent.api.Dial: %w", err)
	}
	if err = waitForDeliveryConnReady(timeout, cc); err != nil {
		_ = cc.Close()
		return nil, fmt.Errorf("cassemagent.api.Dial: %w", err)
	}

	return pb.NewDeliveryClient(cc), nil
}

func waitForDeliveryConnReady(ctx context.Context, cc *grpc.ClientConn) error {
	cc.Connect()
	for {
		state := cc.GetState()
		if state == connectivity.Ready {
			return nil
		}
		if !cc.WaitForStateChange(ctx, state) {
			return ctx.Err()
		}
	}
}
