package main

import (
	"context"
	"time"

	"github.com/naughtygopher/proberesponder"
	"github.com/naughtygopher/proberesponder/extensions/depprober"
	"go.mongodb.org/mongo-driver/mongo"

	"github.com/prashantkr001/template-go/internal/pkg/kafka"
)

const (
	dependencyIDKafka = "kafka"
	dependencyIDMongo = "mongodb"
)

func healthStatus( //nolint:ireturn // returning interface because that's what's exposed by the package
	delay time.Duration,
	pstatus *proberesponder.ProbeResponder,
	mongoCli *mongo.Client,
	kafkaCli *kafka.Kafka,
) depprober.Stopper {
	/*
		Important: having regular pings would keep the respective clients "active".
		This may or may not be a desirable behavior.
		e.g. it might be better to let all connections of MongoDB be disconnected
		if there's no activity, so that the server would only need to deal with fewer connections.
	*/
	probes := []depprober.Prober{
		&depprober.Probe{
			ID:               dependencyIDMongo,
			AffectedStatuses: []proberesponder.Statuskey{proberesponder.StatusReady},
			Checker: depprober.CheckerFunc(func(ctx context.Context) error {
				return mongoCli.Ping(ctx, nil)
			}),
		},
		&depprober.Probe{
			ID:               dependencyIDKafka,
			AffectedStatuses: []proberesponder.Statuskey{proberesponder.StatusReady},
			Checker: depprober.CheckerFunc(func(ctx context.Context) error {
				return kafkaCli.Ping(ctx)
			}),
		},
	}

	return depprober.Start(delay, pstatus, probes...)
}
