package telemetry

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"strings"
	"time"

	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetrichttp"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
	"go.opentelemetry.io/otel/metric"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.24.0"

	"github.com/SolaTyolo/herald/internal/config"
)

type ShutdownFunc func(context.Context) error

func Setup(ctx context.Context, cfg *config.Config) (ShutdownFunc, error) {
	level := parseLevel(cfg.LogLevel)
	opts := &slog.HandlerOptions{Level: level}
	var handler slog.Handler
	if strings.EqualFold(cfg.LogFormat, "json") {
		handler = slog.NewJSONHandler(os.Stdout, opts)
	} else {
		handler = slog.NewTextHandler(os.Stdout, opts)
	}
	slog.SetDefault(slog.New(handler))

	if !cfg.OTelEnabled {
		return func(context.Context) error { return nil }, nil
	}

	res, err := resource.Merge(
		resource.Default(),
		resource.NewWithAttributes(
			semconv.SchemaURL,
			semconv.ServiceName(cfg.OTelServiceName),
		),
	)
	if err != nil {
		return nil, err
	}

	var shutdowns []ShutdownFunc

	if cfg.OTelTracesEnabled {
		traceExp, err := otlptracehttp.New(ctx, otlptracehttp.WithEndpointURL(cfg.OTelEndpoint+"/v1/traces"))
		if err != nil {
			return nil, fmt.Errorf("trace exporter: %w", err)
		}
		tp := sdktrace.NewTracerProvider(
			sdktrace.WithBatcher(traceExp),
			sdktrace.WithResource(res),
		)
		otel.SetTracerProvider(tp)
		shutdowns = append(shutdowns, tp.Shutdown)
	}

	if cfg.OTelMetricsEnabled {
		metricExp, err := otlpmetrichttp.New(ctx, otlpmetrichttp.WithEndpointURL(cfg.OTelEndpoint+"/v1/metrics"))
		if err != nil {
			return nil, fmt.Errorf("metric exporter: %w", err)
		}
		mp := sdkmetric.NewMeterProvider(
			sdkmetric.WithReader(sdkmetric.NewPeriodicReader(metricExp, sdkmetric.WithInterval(15*time.Second))),
			sdkmetric.WithResource(res),
		)
		otel.SetMeterProvider(mp)
		shutdowns = append(shutdowns, mp.Shutdown)
	}

	return func(ctx context.Context) error {
		var err error
		for _, fn := range shutdowns {
			if e := fn(ctx); e != nil && err == nil {
				err = e
			}
		}
		return err
	}, nil
}

func parseLevel(s string) slog.Level {
	switch strings.ToLower(s) {
	case "debug":
		return slog.LevelDebug
	case "warn", "warning":
		return slog.LevelWarn
	case "error":
		return slog.LevelError
	default:
		return slog.LevelInfo
	}
}

func WrapHandler(operation string, h http.Handler) http.Handler {
	return otelhttp.NewHandler(h, operation)
}

func Meter(name string) metric.Meter {
	return otel.Meter(name)
}

func RecordHTTPRequest(ctx context.Context, method, route string, status int, duration time.Duration) {
	counter, _ := Meter("herald/http").Int64Counter("http.server.requests",
		metric.WithDescription("Total HTTP requests"))
	counter.Add(ctx, 1, metric.WithAttributes(
		attribute.String("http.method", method),
		attribute.String("http.route", route),
		attribute.Int("http.status_code", status),
	))
	hist, _ := Meter("herald/http").Float64Histogram("http.server.duration",
		metric.WithDescription("HTTP request duration"),
		metric.WithUnit("s"),
	)
	hist.Record(ctx, duration.Seconds(), metric.WithAttributes(
		attribute.String("http.method", method),
		attribute.String("http.route", route),
	))
}
