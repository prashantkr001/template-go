package grpc

import (
	"context"
	"fmt"
	"time"

	"github.com/naughtygopher/errors"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"

	"github.com/prashantkr001/template-go/internal/pkg/logger"
)

func MwAccessLog( //nolint:nonamedreturns //nolint:nolintlint
	ctx context.Context,
	req any,
	info *grpc.UnaryServerInfo,
	handler grpc.UnaryHandler,
) (resp any, err error) {
	start := time.Now()
	msg := logger.Cyan(info.FullMethod)
	resp, err = handler(ctx, req)
	msg = fmt.Sprintf("%s %s", msg, time.Since(start))
	status := logger.Green("[grpc]::✔")
	if err != nil {
		code, _ := errors.GRPCStatusCode(err)
		status = logger.Red(fmt.Sprintf("[grpc]::%s", code))
	}

	logger.Info(fmt.Sprintf("%s %s %s\n", status, time.Now().Format(time.RFC3339Nano), msg))

	return resp, err
}

func MwErrWrapper( //nolint:nonamedreturns //nolint:nolintlint
	ctx context.Context, req any,
	_ *grpc.UnaryServerInfo,
	handler grpc.UnaryHandler,
) (resp any, err error) {
	resp, err = handler(ctx, req)
	if err == nil {
		return resp, nil
	}

	return nil, responseErrWithLogs(ctx, err)
}

func responseErrWithLogs(ctx context.Context, err error) error {
	code, _ := errors.GRPCStatusCode(err)
	emsg := fmt.Sprintf("%+v", err)
	switch code {
	case codes.InvalidArgument,
		codes.AlreadyExists,
		codes.NotFound:
		logger.WarnCtx(ctx, emsg)
	default:
		logger.ErrorCtx(ctx, emsg)
	}

	return responseError(err)
}
