package apm

import (
	"fmt"
	"net/http"
	"time"

	"github.com/naughtygopher/errors"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"go.opentelemetry.io/otel/exporters/prometheus"
	"go.uber.org/zap"

	"github.com/prashantkr001/template-go/internal/pkg/logger"
)

func prometheusExporter() (*prometheus.Exporter, error) {
	exporter, err := prometheus.New()
	if err != nil {
		return nil, errors.Wrap(err, "promexporter.New")
	}

	return exporter, nil
}

func prometheusScraper(opts *Options) {
	if opts.PrometheusScrapePort == 0 {
		return
	}
	const readTimeout = time.Second * 5
	mux := http.NewServeMux()
	mux.Handle("/-/metrics", promhttp.Handler())
	server := &http.Server{
		Handler:           mux,
		Addr:              fmt.Sprintf(":%d", opts.PrometheusScrapePort),
		ReadHeaderTimeout: readTimeout,
	}

	logger.Info(
		"[http/otel] starting prometheus scrape endpoint",
		zap.String(
			"addr",
			fmt.Sprintf("localhost:%d/-/metrics", opts.PrometheusScrapePort),
		),
	)

	err := server.ListenAndServe()
	if err != nil {
		logger.Error(
			"[http/otel] failed to serve metrics at:",
			zap.Error(err),
			zap.Uint16("port", opts.PrometheusScrapePort),
		)
		return
	}
}
