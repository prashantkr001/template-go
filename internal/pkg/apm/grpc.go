package apm

import (
	"context"
	"fmt"
	"time"

	"github.com/naughtygopher/errors"
	"go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/keepalive"
	"google.golang.org/grpc/stats"
)

type grpcHandler struct {
	stats.Handler
	customHandler func(ctx context.Context, rs stats.RPCStats)
}

func (gh *grpcHandler) HandleRPC(ctx context.Context, rs stats.RPCStats) {
	gh.customHandler(ctx, rs)
}

func OtelGRPCNewServerHandler(ignoredMethods ...string) stats.Handler { //nolint:ireturn // that's how otel sdk works
	checkList := map[string]struct{}{}
	for _, m := range ignoredMethods {
		checkList[m] = struct{}{}
	}

	handler := otelgrpc.NewServerHandler(
		otelgrpc.WithTracerProvider(Global().GetTracerProvider()),
		otelgrpc.WithMeterProvider(Global().GetMeterProvider()),
	)

	gh := &grpcHandler{}
	gh.Handler = handler
	gh.customHandler = func(ctx context.Context, rs stats.RPCStats) {
		methodName, ok := grpc.Method(ctx)
		if !ok {
			return
		}

		_, skip := checkList[methodName]
		if skip {
			return
		}

		handler.HandleRPC(ctx, rs)
	}

	return gh
}

func NewGrpcClient(address string, port int) (*grpc.ClientConn, error) {
	const (
		keepAlivetime = time.Second * 30
		timeout       = time.Second * 10
	)
	dialOpts := []grpc.DialOption{
		grpc.WithUnaryInterceptor(nil),
		grpc.WithStreamInterceptor(nil),
		grpc.WithKeepaliveParams(keepalive.ClientParameters{
			Time:                keepAlivetime,
			Timeout:             timeout,
			PermitWithoutStream: true,
		}),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	}

	conn, err := grpc.NewClient(fmt.Sprintf("%s:%d", address, port), dialOpts...)
	if err != nil {
		return nil, errors.Wrap(err, "failed to create new gRPC client")
	}

	return conn, nil
}
