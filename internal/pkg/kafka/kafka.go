package kafka

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"encoding/base64"
	"log"
	"net"
	"os"
	"strings"
	"time"

	"github.com/naughtygopher/errors"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
	semconv "go.opentelemetry.io/otel/semconv/v1.26.0"

	"github.com/twmb/franz-go/pkg/kgo"
	"github.com/twmb/franz-go/pkg/sasl/plain"
	"github.com/twmb/franz-go/plugin/kotel"
)

type Config struct {
	LogLevel int8

	Seeds         []string
	Topics        []string
	ConsumerGroup string

	IdleTimeout            time.Duration
	RequestTimeoutOverhead time.Duration
	RetryTimeout           time.Duration
	TxnTimeout             time.Duration
	RecordTimeout          time.Duration
	SessionTimeout         time.Duration
	CommitTimeout          time.Duration

	AuthMechanism string
	SASLUsername  string
	SASLPassword  string
	CACertificate string

	FetchMaxBytes int32

	EnableAutoCommit bool
	EnableTLSDialer  bool
}

type kafkaHandler func(ctx context.Context, payload []byte) error
type Kafka struct {
	cfg               *Config
	client            *kgo.Client
	tracer            *kotel.Tracer
	commitTimeout     time.Duration
	latencyInstrument metric.Int64Histogram
}

func (kfk *Kafka) Ping(ctx context.Context) error {
	err := kfk.client.Ping(ctx)
	if err != nil {
		return errors.Wrap(err, "kafka ping failed")
	}
	return nil
}

func (kfk *Kafka) Flush(ctx context.Context) error {
	err := kfk.client.Flush(ctx)
	if err != nil {
		return errors.Wrap(err, "failed to flush Kafka client")
	}
	return nil
}

func (kfk *Kafka) Shutdown(ctx context.Context) error {
	err := kfk.client.Flush(ctx)
	if err != nil {
		return errors.Wrap(err, "failed flushing Kafka client")
	}

	_ = kfk.client.PauseFetchTopics(kfk.cfg.Topics...)
	kfk.Close()

	return nil
}
func (kfk *Kafka) Close() {
	kfk.client.Close()
}

func (kfk *Kafka) PollFetches(ctx context.Context) kgo.Fetches {
	return kfk.client.PollFetches(ctx)
}

func (kfk *Kafka) CommitRecords(ctx context.Context, records ...*kgo.Record) error {
	if len(records) == 0 {
		return nil
	}

	ctx, cancel := context.WithTimeout(ctx, kfk.commitTimeout)
	defer cancel()
	err := kfk.client.CommitRecords(ctx, records...)
	if err != nil {
		return errors.Wrap(err, "kafka commit failed")
	}

	return nil
}

func (kfk *Kafka) ProduceSync(ctx context.Context, rec *kgo.Record) error {
	results := kfk.client.ProduceSync(ctx, rec)
	err := results.FirstErr()
	if err != nil {
		return errors.Wrap(err, "kafka produce sync failed")
	}
	return nil
}

func (kfk *Kafka) HandleTopic(
	ctx context.Context,
	commitRecords *[]*kgo.Record,
	record *kgo.Record,
	fn kafkaHandler,
) {
	childCtx, span := kfk.tracer.WithProcessSpan(record)

	if deadLine, ok := ctx.Deadline(); ok {
		var cancel context.CancelFunc
		childCtx, cancel = context.WithDeadline(childCtx, deadLine)
		defer cancel()
	}

	attr := []attribute.KeyValue{
		{Key: semconv.MessagingKafkaConsumerGroupKey, Value: attribute.StringValue(kfk.cfg.ConsumerGroup)},
		{Key: "kafka.topic", Value: attribute.StringValue(record.Topic)},
	}
	defer func(t time.Time) {
		span.SetAttributes(attr...)
		elapsedTime := time.Duration(time.Since(t).Milliseconds())
		kfk.latencyInstrument.Record(childCtx, int64(elapsedTime), metric.WithAttributes(attr...))
		span.End()
	}(time.Now())

	err := fn(childCtx, record.Value)
	if err != nil {
		log.Println(err)
		return
	}
	// only records which are successfully handled should be committed
	*commitRecords = append(*commitRecords, record)
}

func (kfk *Kafka) Client() *kgo.Client {
	return kfk.client
}

func kgoDialer(cfg *Config) (*tls.Dialer, error) {
	tlsDialer := &tls.Dialer{
		NetDialer: &net.Dialer{Timeout: cfg.RetryTimeout},
	}

	if cfg.CACertificate == "" {
		return tlsDialer, nil
	}

	cACert, err := base64.StdEncoding.DecodeString(cfg.CACertificate)
	if err != nil {
		return nil, errors.Wrap(err, "failed to decode base64 ca certificate")
	}

	caCertPool := x509.NewCertPool()
	ok := caCertPool.AppendCertsFromPEM(cACert)
	if !ok {
		return nil, errors.New("invalid ca certificated provided")
	}

	tlsDialer.Config = &tls.Config{
		RootCAs:    caCertPool,
		MinVersion: tls.VersionTLS12,
	}

	return tlsDialer, nil
}
func kgoOptsFromCfg(cfg *Config, extra ...kgo.Opt) ([]kgo.Opt, error) {
	logLevel := kgo.LogLevel(cfg.LogLevel)
	opts := []kgo.Opt{kgo.SeedBrokers(cfg.Seeds...),
		kgo.ConsumeTopics(cfg.Topics...),
		kgo.ConnIdleTimeout(cfg.IdleTimeout),
		kgo.RetryTimeout(cfg.RetryTimeout),
		kgo.RequestTimeoutOverhead(cfg.RequestTimeoutOverhead),
		kgo.TransactionTimeout(cfg.TxnTimeout),
		kgo.RecordDeliveryTimeout(cfg.RecordTimeout),
		kgo.SessionTimeout(cfg.SessionTimeout),
		kgo.ConsumerGroup(cfg.ConsumerGroup),
		kgo.WithLogger(kgo.BasicLogger(os.Stdout, logLevel, nil)),
	}

	if !cfg.EnableAutoCommit {
		// DisableAutoCommit is required to handle usecases where we have to NACK a message
		// if the processing fails.
		opts = append(opts, kgo.DisableAutoCommit())
	}

	if cfg.FetchMaxBytes > 0 {
		const maxSizeMultiplier = 2
		opts = append(
			opts,
			kgo.FetchMaxBytes(cfg.FetchMaxBytes),
			kgo.BrokerMaxReadBytes(maxSizeMultiplier*cfg.FetchMaxBytes),
		)
	}

	if strings.EqualFold(cfg.AuthMechanism, "SASL") {
		opts = append(opts, kgo.SASL(plain.Auth{
			User: cfg.SASLUsername,
			Pass: cfg.SASLPassword,
		}.AsMechanism()))
	}

	if cfg.EnableTLSDialer {
		tlsDialer, err := kgoDialer(cfg)
		if err != nil {
			return nil, err
		}
		opts = append(opts, kgo.Dialer(tlsDialer.DialContext))
	}

	opts = append(opts, extra...)

	return opts, nil
}

func newCli(ctx context.Context, cfg *Config, opts ...kgo.Opt) (*kgo.Client, error) {
	if opts == nil {
		const minOptions = 10
		opts = make([]kgo.Opt, 0, minOptions)
	}

	kgoOts, err := kgoOptsFromCfg(cfg)
	if err != nil {
		return nil, err
	}

	opts = append(opts, kgoOts...)

	cli, err := kgo.NewClient(opts...)
	if err != nil {
		return nil, errors.Wrap(err, "kafka client initialization failed")
	}
	failure := 0
	for {
		const (
			pingTimeout = time.Second * 3
			sleepTime   = time.Second * 3
			maxTries    = 3
		)
		pingCtx, cancel := context.WithTimeout(ctx, pingTimeout)
		err = cli.Ping(pingCtx)
		cancel()
		if err != nil {
			if failure < maxTries {
				failure++
				time.Sleep(sleepTime)
				continue
			}
			return nil, errors.Wrap(err, "kafka ping failed")
		}
		break
	}

	return cli, nil
}

func New(ctx context.Context, cfg *Config, opts ...kgo.Opt) (*Kafka, error) {
	kcli, err := withOTEL(ctx, cfg, opts...)
	if err != nil {
		return nil, err
	}

	const pingTimeout = time.Second * 3
	ctx, cancel := context.WithTimeout(ctx, pingTimeout)
	defer cancel()

	err = kcli.Ping(ctx)
	if err != nil {
		return nil, errors.Wrap(err, "kafka ping failed")
	}

	return kcli, nil
}
