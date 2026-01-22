package main

import (
	"context"
	"fmt"
	"net/http"
	"slices"
	"time"

	"github.com/naughtygopher/proberesponder"
	proberespHTTP "github.com/naughtygopher/proberesponder/extensions/http"
	"go.mongodb.org/mongo-driver/mongo"

	"github.com/prashantkr001/template-go/cmd/server/grpc"
	xhttp "github.com/prashantkr001/template-go/cmd/server/http"
	kafkaSubs "github.com/prashantkr001/template-go/cmd/subscriber/kafka"
	"github.com/prashantkr001/template-go/internal/api"
	"github.com/prashantkr001/template-go/internal/config"
	"github.com/prashantkr001/template-go/internal/item"
	"github.com/prashantkr001/template-go/internal/pkg/kafka"
	"github.com/prashantkr001/template-go/internal/pkg/logger"
)

func startItemHTTPServer(
	ctx context.Context,
	pResp *proberesponder.ProbeResponder,
	fatalErr chan<- error,
	apis *api.API,
	cfg *xhttp.Config,
) (*xhttp.HTTP, error) { //nolint:unparam,nolintlint
	itemServer := xhttp.New(apis, cfg)
	go func() {
		defer logger.InfoCtx(ctx, fmt.Sprintf("[http] %s:%d shutdown complete", cfg.Host, cfg.Port))
		logger.InfoCtx(ctx, fmt.Sprintf("[http] listening on %s:%d", cfg.Host, cfg.Port))
		pResp.AppendHealthResponse(
			"http/itemserver",
			fmt.Sprintf("OK: %s", time.Now().Format(time.RFC3339)),
		)
		fatalErr <- itemServer.Start()
	}()

	return itemServer, nil
}

func startItemGrpcServer(
	ctx context.Context,
	pResp *proberesponder.ProbeResponder,
	fatalErr chan<- error,
	apis *api.API,
	cfg *grpc.Config,
) (*grpc.GRPC, error) { //nolint:unparam,nolintlint
	itemServer := grpc.New(apis, cfg)
	go func() {
		defer logger.InfoCtx(ctx, fmt.Sprintf("[grpc] %s:%d shutdown complete", cfg.Host, cfg.Port))
		logger.InfoCtx(ctx, fmt.Sprintf("[grpc] listening on %s:%d", cfg.Host, cfg.Port))
		pResp.AppendHealthResponse(
			"grpc/itemserver",
			fmt.Sprintf("OK: %s", time.Now().Format(time.RFC3339)),
		)
		fatalErr <- itemServer.Start()
	}()

	return itemServer, nil
}

func startHealthResponder(
	ctx context.Context,
	ps *proberesponder.ProbeResponder,
	fatalErr chan<- error,
) (*http.Server, error) { //nolint:unparam,nolintlint
	const port = uint16(2000)
	srv := proberespHTTP.Server(ps, "", port)
	go func() {
		defer logger.InfoCtx(ctx, fmt.Sprintf("[http/healthresponder] :%d shutdown complete", port))
		logger.InfoCtx(ctx, fmt.Sprintf("[http/healthresponder] listening on :%d", port))
		fatalErr <- srv.ListenAndServe()
	}()
	return srv, nil
}

func startItemSubscriber(
	ctx context.Context,
	pResp *proberesponder.ProbeResponder,
	fatalErr chan<- error,
	kafkaClient *kafka.Kafka,
	apiService *api.API,
	cfg *kafkaSubs.Config,
) (*kafkaSubs.Kafka, error) {
	ksub, err := kafkaSubs.NewService(kafkaClient, apiService, cfg)
	if err != nil {
		return nil, err
	}

	go func() {
		logger.InfoCtx(
			ctx,
			fmt.Sprintf("[kafka] subscribing to topic(s): '%s'", cfg.TopicItemCreate),
		)
		pResp.AppendHealthResponse(
			"kafka/susbcriber",
			fmt.Sprintf("OK: %s", time.Now().Format(time.RFC3339)),
		)
		fatalErr <- ksub.Subscribe(context.Background())
	}()

	return ksub, nil
}

func startServices(
	ctx context.Context,
	pResp *proberesponder.ProbeResponder,
	fatalErr chan<- error,
	cfg *config.Config,
	kafkaClient *kafka.Kafka,
	apiService *api.API,
) (ksub *kafkaSubs.Kafka, hserver *xhttp.HTTP, gserver *grpc.GRPC, err error) {
	ksub, err = startItemSubscriber(
		ctx,
		pResp,
		fatalErr,
		kafkaClient,
		apiService,
		&kafkaSubs.Config{TopicItemCreate: cfg.Kafka.Topics[0]},
	)
	if err != nil {
		return nil, nil, nil, err
	}

	// start below service(s) based on command line arguments or os.Env
	// e.g. if services=item,grpcserver,something_else etc. it should start all 3
	hConfig := xhttp.Config(cfg.HTTP)
	hConfig.EnableAccesslog = slices.Contains(
		[]string{config.EnvDevelopment, config.EnvCI},
		cfg.Environment,
	)
	hserver, err = startItemHTTPServer(ctx, pResp, fatalErr, apiService, &hConfig)
	if err != nil {
		return nil, nil, nil, err
	}

	gcfg := grpc.Config(cfg.GRPC)
	gcfg.EnableAccesslog = slices.Contains(
		[]string{config.EnvDevelopment, config.EnvCI},
		cfg.Environment,
	)
	gserver, err = startItemGrpcServer(ctx, pResp, fatalErr, apiService, &gcfg)
	if err != nil {
		return nil, nil, nil, err
	}

	return ksub, hserver, gserver, nil
}

func start(
	ctx context.Context,
	cfg *config.Config,
	probestatus *proberesponder.ProbeResponder,
	fatalErr chan<- error,
) (
	mongoClient *mongo.Client,
	kafkaClient *kafka.Kafka,
	hserver *xhttp.HTTP,
	gserver *grpc.GRPC,
	ksub *kafkaSubs.Kafka,
) {
	err := initAPM(ctx, cfg)
	if err != nil {
		panic(err)
	}

	mongoClient, mongoDB, err := initializeMongoDB(ctx, cfg)
	if err != nil {
		panic(err)
	}

	kafkaClient, _, err = initKafka(ctx, cfg)
	if err != nil {
		panic(err)
	}

	itemPersistence, err := item.NewMongoPersistentStore(mongoDB)
	if err != nil {
		panic(err)
	}

	itemPublisher, err := item.NewKafkaItemPublisher(kafkaClient, "template-item-created")
	if err != nil {
		panic(err)
	}

	itemService, err := item.NewService(itemPersistence, itemPublisher)
	if err != nil {
		panic(err)
	}

	apiService := api.NewService(itemService)

	ksub, hserver, gserver, err = startServices(
		ctx,
		probestatus,
		fatalErr,
		cfg,
		kafkaClient,
		apiService,
	)
	if err != nil {
		panic(err)
	}

	return mongoClient, kafkaClient, hserver, gserver, ksub
}
