package middleware

import (
	"log/slog"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"go.opentelemetry.io/otel/trace"
)

type statusRecorder struct {
	http.ResponseWriter
	status int
}

func (r *statusRecorder) WriteHeader(status int) {
	r.status = status
	r.ResponseWriter.WriteHeader(status)
}

func (r *statusRecorder) Write(body []byte) (int, error) {
	if r.status == 0 {
		r.WriteHeader(http.StatusOK)
	}
	return r.ResponseWriter.Write(body)
}

func Logging(logger *slog.Logger) func(http.Handler) http.Handler {
	if logger == nil {
		logger = slog.Default()
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()
			recorder := &statusRecorder{
				ResponseWriter: w,
			}

			next.ServeHTTP(recorder, r)

			status := recorder.status
			if status == 0 {
				status = http.StatusOK
			}

			route := ""
			if routeContext := chi.RouteContext(r.Context()); routeContext != nil {
				route = routeContext.RoutePattern()
			}
			if route == "" {
				route = r.URL.Path
			}

			attrs := []any{
				"method", r.Method,
				"route", route,
				"status", status,
				"duration_ms", time.Since(start).Milliseconds(),
				"remote_addr", r.RemoteAddr,
			}

			spanContext := trace.SpanContextFromContext(r.Context())
			if spanContext.IsValid() {
				attrs = append(attrs,
					"trace_id", spanContext.TraceID().String(),
					"span_id", spanContext.SpanID().String(),
				)
			}

			logger.Info("http request", attrs...)
		})
	}
}
