package main

import (
	"context"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/naughtygopher/proberesponder"
	"go.mongodb.org/mongo-driver/mongo"

	"github.com/prashantkr001/template-go/cmd/server/grpc"
	xhttp "github.com/prashantkr001/template-go/cmd/server/http"
	kafkaSubs "github.com/prashantkr001/template-go/cmd/subscriber/kafka"
	"github.com/prashantkr001/template-go/internal/pkg/apm"
	"github.com/prashantkr001/template-go/internal/pkg/logger"
)

// shutdownItemHTTPServer is used to shutdown the HTTP server
// Similarly, we should implement shutdown for any other long-standing actions. e.g. gcppubsub listener
// kafka listener etc.
func shutdownItemHTTPServer(ctx context.Context, hserver *xhttp.HTTP) {
	err := hserver.Shutdown(ctx)
	if err != nil {
		logger.ErrWithStacktrace(err)
		return
	}
}

func shutdownItemKafkaSubscriber(ctx context.Context, ksub *kafkaSubs.Kafka) {
	err := ksub.Shutdown(ctx)
	if err != nil {
		logger.ErrWithStacktrace(err)
		return
	}
}

func shutdown(
	pResp *proberesponder.ProbeResponder,
	healthResp *http.Server,
	httpServer *xhttp.HTTP,
	grpcServer *grpc.GRPC,
	ksub *kafkaSubs.Kafka,
	mongoCli *mongo.Client,
	apmHandler *apm.APM,
) {
	// the time should be decided based on the K8s grace period allowed for shutdown
	// ref: terminationGracePeriodSeconds, https://kubernetes.io/docs/concepts/containers/container-lifecycle-hooks/
	const shutdownTimeout = time.Second * 60
	pResp.AppendHealthResponse("shutdown", fmt.Sprintf("initiated %s", time.Now().Format(time.RFC3339)))
	ctx, cancel := context.WithTimeout(context.Background(), shutdownTimeout)
	defer cancel()

	/*
		Note: Though there is no mandate to do healthcheck via HTTP, it is important to keep healthcheck
		endpoint available as long as possible to provide Kubernetes probes as much context as possible.
		Esepcially during the graceful shutdown period. Hence it is recommended to setup an independent
		server for health checks alone.
	*/
	defer func() {
		_ = healthResp.Shutdown(ctx)
	}()

	wgroup := &sync.WaitGroup{}

	shutdownAPIs(ctx, wgroup, pResp, httpServer, grpcServer, ksub)

	// after all the APIs of the application are shutdown (e.g. HTTP, gRPC, Pubsub listener etc.)
	// we should close connections to dependencies like database, cache etc.
	// This should only be done after the APIs are shutdown completely
	shutdownDependencies(ctx, wgroup, pResp, mongoCli, apmHandler)

	wgroup.Wait()
}

func shutdownAPIs(
	ctx context.Context,
	wgroup *sync.WaitGroup,
	pResp *proberesponder.ProbeResponder,
	httpServer *xhttp.HTTP,
	grpcServer *grpc.GRPC,
	ksub *kafkaSubs.Kafka,
) {
	wgroup.Add(1)
	go func() {
		defer func() {
			wgroup.Done()
			pResp.AppendHealthResponse(
				"shutdown/http-itemserver",
				fmt.Sprintf("completed %s", time.Now().Format(time.RFC3339)),
			)
		}()
		pResp.AppendHealthResponse(
			"shutdown/http-itemserver",
			fmt.Sprintf("initiated %s", time.Now().Format(time.RFC3339)),
		)
		shutdownItemHTTPServer(ctx, httpServer)
	}()

	wgroup.Add(1)
	go func() {
		defer func() {
			wgroup.Done()
			pResp.AppendHealthResponse(
				"shutdown/grpc-itemserver",
				fmt.Sprintf("completed %s", time.Now().Format(time.RFC3339)),
			)
		}()
		pResp.AppendHealthResponse(
			"shutdown/grpc-itemserver",
			fmt.Sprintf("initiated %s", time.Now().Format(time.RFC3339)),
		)
		grpcServer.Shutdown()
	}()

	wgroup.Add(1)
	go func() {
		defer func() {
			wgroup.Done()
			pResp.AppendHealthResponse(
				"shutdown/kafka-subscriber",
				fmt.Sprintf("completed %s", time.Now().Format(time.RFC3339)),
			)
		}()
		pResp.AppendHealthResponse(
			"shutdown/kafka-subscriber",
			fmt.Sprintf("initiated %s", time.Now().Format(time.RFC3339)),
		)
		shutdownItemKafkaSubscriber(ctx, ksub)
	}()

	wgroup.Wait()
}

func shutdownDependencies(
	ctx context.Context,
	wgroup *sync.WaitGroup,
	pResp *proberesponder.ProbeResponder,
	mongoCli *mongo.Client,
	apmHandler *apm.APM,
) {
	wgroup.Add(1)
	go func() {
		defer func() {
			wgroup.Done()
			pResp.AppendHealthResponse(
				"shutdown/mongodb-driver",
				fmt.Sprintf("completed %s", time.Now().Format(time.RFC3339)),
			)
		}()
		pResp.AppendHealthResponse(
			"shutdown/mongodb-driver",
			fmt.Sprintf("initiated %s", time.Now().Format(time.RFC3339)),
		)
		_ = mongoCli.Disconnect(ctx)
	}()

	wgroup.Add(1)
	go func() {
		defer func() {
			wgroup.Done()
			pResp.AppendHealthResponse(
				"shutdown/apm-server",
				fmt.Sprintf("completed %s", time.Now().Format(time.RFC3339)),
			)
		}()
		pResp.AppendHealthResponse(
			"shutdown/apm-server",
			fmt.Sprintf("initiated %s", time.Now().Format(time.RFC3339)),
		)
		err := apmHandler.Shutdown(ctx)
		if err != nil {
			logger.ErrWithStacktrace(err)
		}
	}()
}
