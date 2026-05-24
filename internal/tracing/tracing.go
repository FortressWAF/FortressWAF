package tracing

import (
	"context"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.24.0"
	"go.opentelemetry.io/otel/trace"
)

type Config struct {
	Enabled        bool
	ServiceName    string
	Exporter       string
	Endpoint       string
	SampleRate     float64
	IncludeBody    bool
	Attributes     map[string]string
}

type Provider struct {
	mu       sync.RWMutex
	tp       *sdktrace.TracerProvider
	config   *Config
	tracer   trace.Tracer
Shutdown func() error
}

var (
	defaultProvider *Provider
	once            sync.Once
)

func Init(cfg Config) (*Provider, error) {
	var initErr error

	once.Do(func() {
		if !cfg.Enabled {
			defaultProvider = &Provider{config: &cfg}
			return
		}

		ctx := context.Background()

		exporter, err := newExporter(cfg.Exporter, cfg.Endpoint)
		if err != nil {
			initErr = fmt.Errorf("create exporter: %w", err)
			return
		}

		res, err := resource.New(ctx,
			resource.WithAttributes(
				semconv.ServiceName(cfg.ServiceName),
				semconv.ServiceVersion("1.0.0"),
			),
		)
		if err != nil {
			initErr = fmt.Errorf("create resource: %w", err)
			return
		}

		tp := sdktrace.NewTracerProvider(
			sdktrace.WithBatcher(exporter),
			sdktrace.WithResource(res),
			sdktrace.WithSampler(sdktrace.TraceIDRatioBased(cfg.SampleRate)),
		)

		otel.SetTracerProvider(tp)
		otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(
			propagation.TraceContext{},
			propagation.Baggage{},
		))

		defaultProvider = &Provider{
			tp:      tp,
			config:  &cfg,
			tracer:  tp.Tracer(cfg.ServiceName),
			Shutdown: tp.Shutdown,
		}
	})

	return defaultProvider, initErr
}

func newExporter(exporterType, endpoint string) (sdktrace.SpanExporter, error) {
	switch exporterType {
	case "otlp":
		client := otlptracegrpc.NewClient(
			otlptracegrpc.WithEndpoint(endpoint),
			otlptracegrpc.WithInsecure(),
		)
		return otlptrace.New(context.Background(), client)
	case "stdout":
		return &stdoutExporter{}, nil
	default:
		return &noopExporter{}, nil
	}
}

type stdoutExporter struct{}

func (e *stdoutExporter) ExportSpans(ctx context.Context, spans []sdktrace.ReadOnlySpan) error {
	for _, span := range spans {
		slog.Debug("trace span",
			"trace_id", span.TraceID().String(),
			"span_id", span.SpanID().String(),
			"name", span.Name(),
			"duration", span.EndTime().Sub(span.StartTime()),
		)
	}
	return nil
}

func (e *stdoutExporter) Shutdown(ctx context.Context) error {
	return nil
}

type noopExporter struct{}

func (e *noopExporter) ExportSpans(ctx context.Context, spans []sdktrace.ReadOnlySpan) error {
	return nil
}

func (e *noopExporter) Shutdown(ctx context.Context) error {
	return nil
}

func Get() *Provider {
	return defaultProvider
}

func (p *Provider) Tracer(name string) trace.Tracer {
	if p == nil || p.tracer == nil {
		return otel.Tracer(name)
	}
	return p.tracer
}

func StartSpan(ctx context.Context, name string, opts ...trace.SpanStartOption) (context.Context, trace.Span) {
	if defaultProvider == nil {
		return ctx, trace.SpanFromContext(ctx)
	}

	tracer := defaultProvider.Tracer("fortresswaf")
	return tracer.Start(ctx, name, opts...)
}

func AddSpanAttributes(ctx context.Context, attrs ...attribute.KeyValue) {
	span := trace.SpanFromContext(ctx)
	if span != nil {
		span.SetAttributes(attrs...)
	}
}

func RecordError(ctx context.Context, err error) {
	span := trace.SpanFromContext(ctx)
	if span != nil {
		span.RecordError(err)
	}
}

func WithRequestAttributes(method, path, host, clientIP string) []attribute.KeyValue {
	return []attribute.KeyValue{
		attribute.String("http.method", method),
		attribute.String("http.url", path),
		attribute.String("http.host", host),
		attribute.String("http.client_ip", clientIP),
	}
}

func WithWAFAttributes(ruleID, action string, score float64) []attribute.KeyValue {
	return []attribute.KeyValue{
		attribute.String("waf.rule_id", ruleID),
		attribute.String("waf.action", action),
		attribute.Float64("waf.threat_score", score),
	}
}

var _ = time.Duration
