// Package config reads config required for the entire application
package config

import (
	"errors"
	"fmt"
	"time"

	"github.com/caarlos0/env/v6"
	"github.com/spf13/viper"
)

const (
	EnvLive        = "live"
	EnvDevelopment = "development"
	EnvCI          = "ci"
)

/*
Config holds all the configurations required for the application to function.
Most drivers like Redis, SQL etc. have optional "client name" field. Make use of the
`cfg.AppFullname()` to set these. It helps us easily identify which version of the app is
communicating with the respective dependency. Especially when we have multiple versions of
deployed, connected to the same dependencies. It would also help us forcefully remove connections
from dependencies if required.
*/
type Config struct {
	AppName      string `json:"appName,omitempty" env:"APP_NAME" envDefault:"template-go"`
	Version      string `json:"version,omitempty" env:"APP_VERSION" envDefault:"v0.0.0"`
	AppBuildDate string `json:"appBuild,omitempty" env:"APP_BUILT_AT" envDefault:"0000-00-00"`
	Environment  string `json:"environment,omitempty" env:"ENVIRONMENT" envDefault:""`

	HTTP struct {
		Host              string        `json:"host,omitempty" env:"APP_HTTP_HOST" envDefault:""`
		Port              int           `json:"port,omitempty" env:"APP_HTTP_PORT" envDefault:"5001"`
		ReadHeaderTimeout time.Duration `json:"readHeaderTimeout,omitempty" env:"HTTP_READ_HEADER_TIMEOUT" envDefault:"5s"`
		ReadTimeout       time.Duration `json:"readTimeout,omitempty" env:"HTTP_READ_TIMEOUT" envDefault:"60s"`
		WriteTimeout      time.Duration `json:"writeTimeout,omitempty" env:"HTTP_WRITE_TIMEOUT" envDefault:"60s"`
		IdleTimeout       time.Duration `json:"idleTimeout,omitempty" env:"HTTP_IDLE_TIMEOUT" envDefault:"60s"`
		EnableAccesslog   bool
	} `json:"http,omitempty"`
	GRPC struct {
		Host            string        `json:"grpcHost,omitempty" env:"APP_GRPC_HOST" envDefault:""`
		Port            int           `json:"grpcPort,omitempty" env:"APP_PORT_PORT" envDefault:"5002"`
		ConnTimeout     time.Duration `json:"grpcTimeout,omitempty" env:"APP_GRPC_TIMEOUT" envDefault:"15s"`
		EnableAccesslog bool
	}
	MongoDB struct {
		Hosts     []string `json:"hosts,omitempty" env:"MONGODB_HOSTS" envDefault:"localhost"`
		Port      int      `json:"port,omitempty" env:"MONGODB_PORT"`
		Database  string   `json:"database,omitempty" env:"MONGODB_DATABASE"`
		Namespace string   `json:"namespace,omitempty" env:"MONGODB_NAMESPACE"`
		Username  string   `json:"username,omitempty" env:"MONGODB_USERNAME"`
		Password  string   `json:"password,omitempty" env:"MONGODB_PASSWORD"`
		// AuthMechanism for MongoDB should be one of "SCRAM-SHA-256", "SCRAM-SHA-1", "MONGODB-CR", "PLAIN", "GSSAPI", "MONGODB-X509",
		AuthMechanism   string        `json:"authMechanism,omitempty" env:"MONGODB_AUTH_MECHANISM" envDefault:"SCRAM-SHA-1"`
		AuthDatabase    string        `json:"authDatabase,omitempty" env:"MONGODB_AUTH_DATABASE"`
		ReadTimeout     time.Duration `json:"readTimeout,omitempty" env:"MONGODB_READTIMEOUT" envDefault:"3m"`
		WriteTimeout    time.Duration `json:"writeTimeout,omitempty" env:"MONGODB_WRITETIMEOUT" envDefault:"3m"`
		MaxConnIdleTime time.Duration `json:"maxIdleTimeout,omitempty" env:"MONGODB_IDLE_TIMEOUT" envDefault:"4m"`
		PingTimeout     time.Duration `json:"pingTimeout,omitempty" env:"MONGODB_PING_TIMEOUT" envDefault:"3s"`
	} `json:"mongoDB,omitempty"`

	Kafka struct {
		LogLevel int8 `json:"logLevel,omitempty" env:"KAFKA_LOG_LEVEL" envDefault:"1"` // loglevel 1 is >= error

		Seeds         []string `json:"seeds,omitempty" env:"KAFKA_SEEDS" envDefault:"localhost:9092"`
		Topics        []string `json:"topics,omitempty" env:"KAFKA_TOPICS" envDefault:"item_create"`
		ConsumerGroup string   `json:"consumerGroup,omitempty" env:"KAFKA_CONSUMERGROUP" envDefault:""`

		IdleTimeout            time.Duration `json:"idleTimeout,omitempty" env:"KAFKA_IDLETIMEOUT" envDefault:"3s"`
		RequestTimeoutOverhead time.Duration `json:"requestTimeoutOverhead,omitempty" env:"KAFKA_REQTIMEOUT" envDefault:"3s"`
		RetryTimeout           time.Duration `json:"retryTimeout,omitempty" env:"KAFKA_RETTIMEOUT" envDefault:"3s"`
		TxnTimeout             time.Duration `json:"txnTimeout,omitempty" env:"KAFKA_TXNTIMEOUT" envDefault:"3s"`
		RecordTimeout          time.Duration `json:"recordTimeout,omitempty" env:"KAFKA_RECTIMEOUT" envDefault:"3s"`
		SessionTimeout         time.Duration `json:"sessionTimeout,omitempty" env:"KAFKA_SESSTIMEOUT" envDefault:"60s"`
		CommitTimeout          time.Duration `json:"CommitTimeout,omitempty" env:"KAFKA_COMMTIMEOUT" envDefault:"5s"`

		AuthMechanism string `json:"authMechanism,omitempty" env:"KAFKA_AUTH_MECHANISM" envDefault:""`
		SASLUsername  string `json:"saslUsername,omitempty" env:"KAFKA_SASL_USERNAME" envDefault:""`
		SASLPassword  string `json:"saslPassword,omitempty" env:"KAFKA_SASL_PASSWORD" envDefault:""`
		CACertificate string `json:"caCertificate,omitempty" env:"KAFKA_CA_CERT" envDefault:""`

		FetchMaxBytes int32 `json:"fetchMaxBytes,omitempty" env:"KAFKA_FETCH_MAXBYTES" envDefault:"1048576"` // 1MiB

		EnableAutoCommit bool `json:"enableAutoCommit,omitempty" env:"KAFKA_AUTO_COMMIT" envDefault:"true"`
		EnableTLSDialer  bool `json:"enableTLSDialer,omitempty" env:"KAFKA_ENABLE_TLSDIALER" envDefault:"false"`
	}
	APM struct {
		Debug              bool    `json:"debug" env:"TRACES_DEBUG"`
		TracesSampleRate   float64 `json:"tracesSampleRate" env:"TRACES_SAMPLE_RATE"`
		TracesCollectorURL string  `json:"collectorUrl" env:"TRACES_COLLECTOR_URL"`
		MetricScrapePort   uint16  `json:"metricScrapePort" env:"METRIC_SCRAPE_PORT" envDefault:"2223"`
		EnableStdout       string
	} `json:"apm"`
}

func (cfg *Config) AppFullname() string {
	return fmt.Sprintf("%s%s", cfg.AppName, cfg.Version)
}

func Load(path, fileName string) (*Config, error) {
	configs := Config{}

	viper.AddConfigPath(path)
	viper.SetConfigName(fileName)
	viper.SetConfigType("yaml")
	viper.AutomaticEnv()

	if err := viper.ReadInConfig(); err != nil {
		if !errors.As(err, &viper.ConfigFileNotFoundError{}) {
			panic(err)
		} else if err = env.Parse(&configs); err != nil {
			panic(fmt.Sprintf("%+v\n", err))
		}
	}

	err := viper.Unmarshal(&configs)
	if err != nil {
		panic(fmt.Errorf("fatal error when unrmashaling config file: %w", err))
	}

	return &configs, nil
}
