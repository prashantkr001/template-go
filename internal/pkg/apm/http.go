package apm

import (
	"net/http"

	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
)

type HTTPOpts struct {
	OperationName string
	OTEL          []otelhttp.Option
}

type HTTPMiddleware func(h http.Handler) http.Handler

func NewHTTPMiddleware(
	hopts *HTTPOpts,
) HTTPMiddleware {
	var opts []otelhttp.Option
	if hopts != nil {
		opts = hopts.OTEL
	}
	if len(opts) == 0 {
		const minOptions = 3
		opts = make([]otelhttp.Option, 0, minOptions)
	}

	gb := Global()
	opts = append(
		opts,
		otelhttp.WithMeterProvider(gb.GetMeterProvider()),
		otelhttp.WithTracerProvider(gb.GetTracerProvider()),
	)

	opName := hopts.OperationName
	if opName == "" {
		opName = "otelhttp"
	}

	return otelhttp.NewMiddleware(opName, opts...)
}
