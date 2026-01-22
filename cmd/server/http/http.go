// Package http implements all the HTTP handlers exported by this application.
// It also has the health API check required for the app
package http

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/naughtygopher/errors"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
	"go.opentelemetry.io/otel/attribute"
	semconv "go.opentelemetry.io/otel/semconv/v1.26.0"

	"github.com/prashantkr001/template-go/internal/api"
	"github.com/prashantkr001/template-go/internal/pkg/apm"
)

type Config struct {
	Host              string
	Port              int
	ReadHeaderTimeout time.Duration
	ReadTimeout       time.Duration
	WriteTimeout      time.Duration
	IdleTimeout       time.Duration
	EnableAccesslog   bool
}

type HTTP struct {
	locker *sync.Mutex
	server *http.Server
	// apis has all the APIs, and respective HTTP handlers will call using this
	apis              *api.API
	shutdownInitiated bool
	serverStartTime   time.Time
}

func (ht *HTTP) Start() error {
	ht.serverStartTime = time.Now()
	err := ht.server.ListenAndServe()
	if err != nil {
		return errors.Wrap(err, "failed to start http server")
	}

	return nil
}

func (ht *HTTP) Shutdown(ctx context.Context) error {
	ht.locker.Lock()
	defer ht.locker.Unlock()

	ht.shutdownInitiated = true
	err := ht.server.Shutdown(ctx)
	if err != nil {
		return errors.Wrap(err, "failed to shutdown http server")
	}

	return nil
}

func (ht *HTTP) StartedAt() time.Time {
	return ht.serverStartTime
}

type HandlerFuncErr func(w http.ResponseWriter, req *http.Request) error

func (ht *HTTP) ErrorHandler(fn HandlerFuncErr) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		err := fn(w, r)
		if err == nil {
			return
		}

		status, message, _ := errors.HTTPStatusCodeMessage(err)
		w.WriteHeader(status)
		_, _ = w.Write([]byte(message))

		// log the full error here for troubleshooting.
		// *Assuming* we need not log 4xx errors since they're client side errors
		// maybe we just need internal errors to be logged.
		if status >= http.StatusInternalServerError {
			log.Println(errors.Stacktrace(err))
		}
	}
}

func chiURIPattern(router *chi.Mux, r *http.Request) string {
	cctx := chi.RouteContext(r.Context())
	uriPattern := "unmatched-path"
	if router.Match(cctx, r.Method, r.URL.Path) {
		uriPattern = cctx.RoutePattern()
	}
	return uriPattern
}

func newChiRouter(cfg *Config) chi.Router { //nolint:ireturn,nolintlint
	router := chi.NewRouter()
	router.Use(
		middleware.Recoverer,
		func(h http.Handler) http.Handler {
			wrapped := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				l := new(otelhttp.Labeler)
				l.Add(attribute.KeyValue{
					Key:   semconv.HTTPRouteKey,
					Value: attribute.StringValue(chiURIPattern(router, r)),
				})

				h.ServeHTTP(
					w,
					r.WithContext(otelhttp.ContextWithLabeler(r.Context(), l)),
				)
			})
			return wrapped
		},
		apm.NewHTTPMiddleware(&apm.HTTPOpts{
			OTEL: []otelhttp.Option{
				otelhttp.WithFilter(func(req *http.Request) bool {
					return !strings.HasPrefix(req.URL.Path, "/-/")
				}),
				otelhttp.WithSpanNameFormatter(func(_ string, req *http.Request) string {
					return chiURIPattern(router, req)
				}),
			},
		},
		),
	)

	if cfg.EnableAccesslog {
		router.Use(middleware.Logger)
	}

	return router
}

func New(apis *api.API, cfg *Config) *HTTP {
	ht := &HTTP{
		locker: &sync.Mutex{},
		apis:   apis,
		server: &http.Server{
			Addr:              fmt.Sprintf("%s:%d", cfg.Host, cfg.Port),
			ReadHeaderTimeout: cfg.ReadHeaderTimeout,
			ReadTimeout:       cfg.ReadTimeout,
			WriteTimeout:      cfg.WriteTimeout,
		},
	}

	router := newChiRouter(cfg)
	ht.itemRoutes(router)
	ht.server.Handler = router

	return ht
}
