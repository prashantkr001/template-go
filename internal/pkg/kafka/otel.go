package kafka

import (
	"context"

	"github.com/naughtygopher/errors"
	"github.com/twmb/franz-go/pkg/kgo"
	"github.com/twmb/franz-go/plugin/kotel"
	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/propagation"

	"github.com/prashantkr001/template-go/internal/pkg/apm"
)

func setupOtel() (*kotel.Tracer, kgo.Opt) { //nolint:ireturn // that's how otel sdk works
	tracerOpts := []kotel.TracerOpt{
		kotel.TracerProvider(apm.Global().GetTracerProvider()),
		kotel.TracerPropagator(propagation.TraceContext{}),
	}
	meterOpts := []kotel.MeterOpt{
		kotel.MeterProvider(apm.Global().GetMeterProvider()),
	}
	tracer := kotel.NewTracer(tracerOpts...)
	kmeter := kotel.NewMeter(meterOpts...)
	kotelOpts := []kotel.Opt{
		kotel.WithTracer(tracer),
		kotel.WithMeter(kmeter),
	}
	kotelsvc := kotel.NewKotel(kotelOpts...)
	return tracer, kgo.WithHooks(kotelsvc.Hooks()...)
}

func otelparams(opts ...kgo.Opt) ( //nolint:ireturn // that's how otel sdk works
	tracer *kotel.Tracer,
	latencyInstrument metric.Int64Histogram,
	oopts []kgo.Opt,
	err error,
) {
	tracer, kotelOpt := setupOtel()
	opts = append(opts, kotelOpt)
	latencyInstrument, err = apm.Global().AppMeter().Int64Histogram(
		"kafka.handler.duration",
		metric.WithUnit("ms"),
	)
	if err != nil {
		return nil, nil, nil, errors.Wrap(err, "meter.Int64Histogram")
	}
	return tracer, latencyInstrument, opts, nil
}

func withOTEL(ctx context.Context, cfg *Config, opts ...kgo.Opt) (*Kafka, error) {
	tracer, latencyInstrument, oopts, err := otelparams(opts...)
	if err != nil {
		return nil, err
	}
	kcli, err := newCli(ctx, cfg, oopts...)
	if err != nil {
		return nil, err
	}

	return &Kafka{
		cfg:               cfg,
		client:            kcli,
		tracer:            tracer,
		commitTimeout:     cfg.CommitTimeout,
		latencyInstrument: latencyInstrument,
	}, nil
}
