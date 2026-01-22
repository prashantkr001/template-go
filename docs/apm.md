# APM setup

## Components

- Metrics (Opentelemetry + Prometheus)
- Tracing (Opentelemetry + Jaeger/OTEL Collector)
- Error tracking (Sentry)
- Log visualization (Grafana, Loki)

We rely on OTEL stack for all our observability requirements.

## Codebase organization

Steps to setup apm

1. Init apm module in main

```go
	apmModule, err := apm.New(&apm.Options{
		Debug:                cfg.APM.Debug,
		Environment:          cfg.Environment,
		ServiceName:          cfg.AppFullname(),
		ServiceVersion:       cfg.AppVersion(),
		TracesSampleRate:     cfg.APM.TracesSampleRate, // 0.1
		CollectorURL:         cfg.APM.TracesCollectorURL,
		PrometheusScrapePort: cfg.APM.MetricScrapePort, // 2223
		UseStdOut:       cfg.APM.Debug,
	})
	if err != nil {
		return err
	}
	apm.SetGlobal(apmModule)
```

2. Init apm middleware/hooks, depends on your inbound/outbound traffic

### Http

- Use middleware from pkg/http

```go
import (
    pkghttp "github.com/prashantkr001/template-go
/internal/pkg/http"
)

router := chi.NewRouter()
router.Use(pkghttp.ChiOtelMW(router))
```

### Grpc

- Use interceptors (unary/stream) from pkg/grpc, provides a list of method you would like to ignore (healthcheck/ping for example)

```go
import (
	pkggrpc "github.com/prashantkr001/template-go
/internal/pkg/grpc"
)

grpcServer := grpc.NewServer(
    grpc.ChainUnaryInterceptor(pkggrpc.OtelUnaryInterceptor(nil)),
)
```

## Local development

We have some docker containers as followed. To enable them, add this docker env variable, because they are disabled by default
`COMPOSE_PROFILES=monitoring`

```yaml
  jaeger-all-in-one:
    image: jaegertracing/all-in-one:1.42
    restart: always
    networks:
      - template-go
    profiles: ["monitoring"]
  prometheus:
    container_name: prometheus
    image: prom/prometheus:v2.42.0
    restart: always
    ...
```

If configuration is correct, you will be able to see the metrics/tracing info in your local Prometheus/Jaeger
