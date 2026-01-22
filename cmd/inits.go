package main

import (
	"context"
	"fmt"
	"slices"
	"time"

	mongoprom "github.com/globocom/mongo-go-prometheus"
	"github.com/naughtygopher/errors"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.opentelemetry.io/otel/trace"
	"go.uber.org/zap"

	"github.com/prashantkr001/template-go/internal/config"
	"github.com/prashantkr001/template-go/internal/pkg/apm"
	"github.com/prashantkr001/template-go/internal/pkg/kafka"
	"github.com/prashantkr001/template-go/internal/pkg/logger"
)

type ctxKey string

const (
	CtxKeyEnv ctxKey = "env"
)

func initLogger(cfg *config.Config) {
	ctxKeys := []any{CtxKeyEnv}
	if slices.Contains([]string{config.EnvDevelopment, config.EnvCI}, cfg.Environment) {
		lh, _ := zap.NewDevelopment(zap.AddCallerSkip(1))
		logger.SetGlobal(lh)
	}

	logger.SetContextFieldsSetter(func(ctx context.Context) []zap.Field {
		fields := make([]zap.Field, 0, 1)
		for _, key := range ctxKeys {
			fields = append(
				fields,
				zap.Any(fmt.Sprintf("%v", key), ctx.Value(key)),
			)
		}

		traceID := trace.SpanContextFromContext(ctx).TraceID()
		if traceID.IsValid() {
			fields = append(fields, zap.String("trace_id", traceID.String()))
		}

		return fields
	})
}

func initAPM(ctx context.Context, cfg *config.Config) error {
	ins, err := apm.New(ctx, &apm.Options{
		Environment:          cfg.Environment,
		Debug:                cfg.APM.Debug,
		ServiceName:          cfg.AppName,
		ServiceVersion:       cfg.Version,
		TracesSampleRate:     cfg.APM.TracesSampleRate,
		CollectorURL:         cfg.APM.TracesCollectorURL,
		PrometheusScrapePort: cfg.APM.MetricScrapePort,
		UseStdOut:            slices.Contains([]string{config.EnvDevelopment, config.EnvCI}, cfg.Environment),
	})
	if err != nil {
		return errors.Wrap(err, "failed to initialize APM")
	}
	apm.SetGlobal(ins)
	return nil
}

func initializeMongoDB(ctx context.Context, cfg *config.Config) (*mongo.Client, *mongo.Database, error) {
	// setting up monitoring using Prometheus. #TODO: not verified/tested
	monitor := mongoprom.NewCommandMonitor(
		mongoprom.WithInstanceName(cfg.MongoDB.Database),
		mongoprom.WithNamespace(cfg.MongoDB.Namespace),
		mongoprom.WithDurationBuckets([]float64{.001, .005, .01}),
	)

	opts := options.Client().SetMonitor(monitor)
	opts.Hosts = cfg.MongoDB.Hosts
	opts.Auth = &options.Credential{
		AuthMechanism:           cfg.MongoDB.AuthMechanism,
		AuthMechanismProperties: nil,
		AuthSource:              cfg.MongoDB.AuthDatabase,
		Username:                cfg.MongoDB.Username,
		Password:                cfg.MongoDB.Password,

		PasswordSet:         false,
		OIDCMachineCallback: nil,
		OIDCHumanCallback:   nil,
	}
	opts.MaxConnIdleTime = &cfg.MongoDB.MaxConnIdleTime
	opts.SetAppName(cfg.AppFullname())

	mongoClient, err := mongo.Connect(ctx, opts)
	if err != nil {
		return nil, nil, errors.Wrap(err, "failed to connect to MongoDB")
	}
	const pingTimeout = 3 * time.Second
	ctx, cancel := context.WithTimeout(ctx, pingTimeout)
	defer cancel()

	err = mongoClient.Ping(ctx, nil)
	if err != nil {
		return nil, nil, errors.Wrap(err, "failed to ping MongoDB")
	}

	return mongoClient, mongoClient.Database(cfg.MongoDB.Database), nil
}

func initKafka(
	ctx context.Context,
	cfg *config.Config,
) (*kafka.Kafka, *kafka.Config, error) {
	// initialize ingestor service layers
	if cfg.Kafka.ConsumerGroup == "" {
		cfg.Kafka.ConsumerGroup = cfg.AppFullname()
	}

	kfCfg := kafka.Config(cfg.Kafka)
	kfkClient, err := kafka.New(ctx, &kfCfg)
	if err != nil {
		return nil, nil, err
	}

	return kfkClient, &kfCfg, nil
}
