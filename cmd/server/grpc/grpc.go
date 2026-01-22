// Package grpc implements all the GRPC based APIs. It initializes a GRPC
// server with monitoring, error handling etc. enabled.
package grpc

import (
	"fmt"
	"net"
	"time"

	"github.com/naughtygopher/errors"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/keepalive"
	"google.golang.org/grpc/reflection"
	"google.golang.org/grpc/status"

	"github.com/prashantkr001/template-go/cmd/server/grpc/proto/v1/pbitems"
	"github.com/prashantkr001/template-go/internal/api"
	"github.com/prashantkr001/template-go/internal/pkg/apm"
)

type Config struct {
	Host            string
	Port            int
	ConnTimeout     time.Duration
	EnableAccesslog bool
}
type GRPC struct {
	hostaddress string
	grpcServer  *grpc.Server
	port        int
	apis        *api.API
	startedAt   time.Time
	pbitems.ItemsServiceServer
}

// Start will start the grpc server.
func (grp *GRPC) Start() error {
	lis, err := net.Listen("tcp", fmt.Sprintf(":%d", grp.port))
	if err != nil {
		return errors.Wrap(err, "failed to create listener")
	}
	grp.startedAt = time.Now()
	err = grp.grpcServer.Serve(lis)
	if err != nil {
		return errors.Wrap(err, "failed to serve")
	}

	return nil
}

func (grp *GRPC) StartedAt() time.Time {
	return grp.startedAt
}

func (grp *GRPC) Address() string {
	return grp.hostaddress
}

// Implementor is used if the underlying grpc server is required for any specific usecase.
func (grp *GRPC) Implementor() *grpc.Server {
	return grp.grpcServer
}

// Shutdown will shutdown the grpc server
func (grp *GRPC) Shutdown() {
	grp.grpcServer.GracefulStop()
}

// New makes new grpc server.
func New(apis *api.API, cfg *Config) *GRPC {
	const graceShutdownTime = time.Second * 5
	opts := []grpc.ServerOption{
		grpc.KeepaliveParams(keepalive.ServerParameters{
			MaxConnectionIdle: time.Minute,
			// MaxConnectionAgeGrace is the graceful period for outstanding connections
			MaxConnectionAgeGrace: graceShutdownTime,

			Time:             0,
			Timeout:          0,
			MaxConnectionAge: 0,
		}),
		// insecure option is added by default to ease development,
		// you should reconisder this before deploying to production.
		grpc.Creds(insecure.NewCredentials()),
		grpc.ConnectionTimeout(cfg.ConnTimeout),
		grpc.StatsHandler(apm.OtelGRPCNewServerHandler()),
		grpc.ChainUnaryInterceptor(MwErrWrapper),
	}

	if cfg.EnableAccesslog {
		opts = append(opts, grpc.ChainUnaryInterceptor(MwAccessLog))
	}

	grpcServer := grpc.NewServer(opts...)

	grp := &GRPC{
		hostaddress: fmt.Sprintf("%d", cfg.Port), //nolint:perfsprint // this is more readable and there's no performance penalty
		grpcServer:  grpcServer,
		apis:        apis,
		port:        cfg.Port,
	}
	pbitems.RegisterItemsServiceServer(grp.grpcServer, grp)

	// this would expose an API which returns all the gRPC API contracts.
	// it can be used by grpc clients, in which case the proto file
	// is not required while making calls from the clients
	// this may or may not be a desired feature considering security
	// e.g. for services which are internal (not accessible from public internet) only
	// it shouldn't cause any issues
	// grpcurl -plaintext localhost:5002 list
	reflection.Register(grpcServer)

	return grp
}

func responseError(err error) error {
	code, message, _ := errors.GRPCStatusCodeMessage(err)
	return status.Error(code, message) //nolint:wrapcheck // raw unwrapped error is expected
}
