// Package main is your entry point, which is probably the "ugliest" package.
// Since you'll have to initialize all your dependencies like drivers; read configs etc.
package main

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/naughtygopher/errors"
	"github.com/naughtygopher/proberesponder"

	"github.com/prashantkr001/template-go/internal/config"
	"github.com/prashantkr001/template-go/internal/pkg/apm"
	"github.com/prashantkr001/template-go/internal/pkg/logger"
	"github.com/prashantkr001/template-go/internal/pkg/sysignals"
)

// recoverer is used for panic recovery of the application (note: this is not for the HTTP/gRPC servers).
// So that even if the main function panics we can produce required logs for troubleshooting.
var errExit error

func recoverer() {
	exitCode := 0
	var exitInfo any
	rec := recover()
	err, _ := rec.(error)
	switch {
	case err != nil:
		exitCode = 1
		exitInfo = err
	case rec != nil:
		exitCode = 2
		exitInfo = rec
	case errExit != nil:
		exitCode = 3
		exitInfo = errExit
	default:
		break
	}

	// exiting after receiving a quit signal can be considered a *clean/successful* exit
	if errors.Is(errExit, sysignals.ErrSigQuit) {
		exitCode = 0
	}

	// logging this because we have info logs saying "listening to" various port numbers
	// based on the server type (gRPC, HTTP etc.). But it's unclear *from the logs*
	// if the server is up and running, if it exits for any reason
	if exitCode == 0 {
		logger.Info(fmt.Sprintf("shutdown complete: %+v", exitInfo))
	} else {
		logger.Error(fmt.Sprintf("shutdown complete (exit: %d): %+v", exitCode, exitInfo))
	}

	os.Exit(exitCode)
}

func main() {
	defer recoverer()

	var (
		ctx      = context.Background()
		fatalErr = make(chan error, 1)
		// by default all probe responses are negative.
		probestatus = proberesponder.New()
	)

	healthResponder, err := startHealthResponder(ctx, probestatus, fatalErr)
	if err != nil {
		panic(err)
	}

	go sysignals.NotifyErrorOnQuit(fatalErr)

	cfg, err := config.Load("", "")
	if err != nil {
		panic(err)
	}
	ctx = context.WithValue(ctx, CtxKeyEnv, cfg.Environment)

	probestatus.AppendHealthResponse("app->version", cfg.AppFullname())
	probestatus.AppendHealthResponse("app->built", cfg.AppBuildDate)

	initLogger(cfg)

	mongoClient, kafkaClient, hserver, gserver, ksub := start(ctx, cfg, probestatus, fatalErr)

	const probeInterval = time.Second * 30
	var depProbeStopper = healthStatus(
		probeInterval,
		probestatus,
		mongoClient,
		kafkaClient,
	)

	defer func() {
		// probestatus update should be done as soon as the service is shutting down for any reason.
		// set the service as Not ready as soon as it's exiting main
		probestatus.SetNotReady(true)
		probestatus.SetNotStarted(true)
		probestatus.SetNotLive(true)

		depProbeStopper.Stop()

		probestatus.AppendHealthResponse(
			"shutdown",
			fmt.Sprintf("initiated: %s", time.Now().Format(time.RFC3339)),
		)

		/*
			When a server begins its shutdown process, it first signals Kubernetes by changing its
			readiness state to "not ready". This ensures that the server stops receiving new traffic.

			However, there is typically a delay before Kubernetes detects this readiness change because
			it relies on periodic probing to check the status. During this delay, Kubernetes may still
			route new requests to the server, unaware that shutdown has started.

			To handle this, a deliberate pause is introduced between changing the readiness state to
			"not ready" and initiating the full shutdown. This pause should be longer than Kubernetes'
			readiness probe interval. This way, Kubernetes has enough time to notice the readiness change
			and stop sending new requests before the server begins rejecting them.
		*/
		// in this case, the Kuberenetes probe interval is assumed to be 2 seconds
		const k8sProbeInterval = time.Second * 3
		time.Sleep(k8sProbeInterval)

		logger.Info("initiating shutdown")
		shutdown(
			probestatus,
			healthResponder,
			hserver,
			gserver,
			ksub,
			mongoClient,
			apm.Global(),
		)
	}()

	// by now all the intended servers, subscribers etc. are up and running.
	probestatus.SetNotStarted(false)
	probestatus.SetNotReady(false)
	probestatus.SetNotLive(false)

	errExit = <-fatalErr
}
