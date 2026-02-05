/*
Copyright 2025 The Scion Authors.
*/

package telemetry

import (
	"context"
	"crypto/tls"
	"fmt"

	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
	"go.opentelemetry.io/otel/sdk/trace"
	coltracepb "go.opentelemetry.io/proto/otlp/collector/trace/v1"
	tracepb "go.opentelemetry.io/proto/otlp/trace/v1"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/credentials/insecure"
)

// CloudExporter exports traces to a cloud OTLP endpoint.
type CloudExporter struct {
	traceExporter trace.SpanExporter
	grpcClient    coltracepb.TraceServiceClient
	grpcConn      *grpc.ClientConn
	protocol      string
	endpoint      string
}

// NewCloudExporter creates a new cloud trace exporter.
// Returns nil if cloud export is not configured.
func NewCloudExporter(config *Config) (*CloudExporter, error) {
	if !config.IsCloudConfigured() {
		return nil, nil
	}

	exporter := &CloudExporter{
		protocol: config.Protocol,
		endpoint: config.Endpoint,
	}

	var err error
	switch config.Protocol {
	case "grpc":
		err = exporter.initGRPC(config)
	case "http":
		err = exporter.initHTTP(config)
	default:
		err = exporter.initGRPC(config) // default to gRPC
	}

	if err != nil {
		return nil, err
	}

	return exporter, nil
}

// initGRPC initializes the gRPC exporter.
func (e *CloudExporter) initGRPC(config *Config) error {
	opts := []otlptracegrpc.Option{
		otlptracegrpc.WithEndpoint(config.Endpoint),
	}

	if config.Insecure {
		opts = append(opts, otlptracegrpc.WithInsecure())
	}

	exporter, err := otlptracegrpc.New(context.Background(), opts...)
	if err != nil {
		return fmt.Errorf("failed to create gRPC trace exporter: %w", err)
	}

	e.traceExporter = exporter

	// Also create a raw gRPC client for proto forwarding
	var creds credentials.TransportCredentials
	if config.Insecure {
		creds = insecure.NewCredentials()
	} else {
		creds = credentials.NewTLS(&tls.Config{})
	}

	conn, err := grpc.NewClient(config.Endpoint, grpc.WithTransportCredentials(creds))
	if err != nil {
		// Continue without raw client - we can still use SDK exporter
		return nil
	}

	e.grpcConn = conn
	e.grpcClient = coltracepb.NewTraceServiceClient(conn)

	return nil
}

// initHTTP initializes the HTTP exporter.
func (e *CloudExporter) initHTTP(config *Config) error {
	opts := []otlptracehttp.Option{
		otlptracehttp.WithEndpoint(config.Endpoint),
	}

	if config.Insecure {
		opts = append(opts, otlptracehttp.WithInsecure())
	}

	exporter, err := otlptracehttp.New(context.Background(), opts...)
	if err != nil {
		return fmt.Errorf("failed to create HTTP trace exporter: %w", err)
	}

	e.traceExporter = exporter
	return nil
}

// ExportSpans exports a batch of SDK spans to the cloud endpoint.
func (e *CloudExporter) ExportSpans(ctx context.Context, spans []trace.ReadOnlySpan) error {
	if e == nil || e.traceExporter == nil {
		return nil
	}
	return e.traceExporter.ExportSpans(ctx, spans)
}

// ExportProtoSpans exports raw proto spans to the cloud endpoint.
// This is used for forwarding OTLP data received from agents.
func (e *CloudExporter) ExportProtoSpans(ctx context.Context, resourceSpans []*tracepb.ResourceSpans) error {
	if e == nil {
		return nil
	}

	// Use gRPC client if available
	if e.grpcClient != nil {
		req := &coltracepb.ExportTraceServiceRequest{
			ResourceSpans: resourceSpans,
		}
		_, err := e.grpcClient.Export(ctx, req)
		return err
	}

	// Otherwise we can't forward raw proto data
	// This is acceptable for M1 - cloud export may not work without proper setup
	return nil
}

// Shutdown gracefully shuts down the exporter.
func (e *CloudExporter) Shutdown(ctx context.Context) error {
	if e == nil {
		return nil
	}

	var errs []error

	if e.traceExporter != nil {
		if err := e.traceExporter.Shutdown(ctx); err != nil {
			errs = append(errs, err)
		}
	}

	if e.grpcConn != nil {
		if err := e.grpcConn.Close(); err != nil {
			errs = append(errs, err)
		}
	}

	if len(errs) > 0 {
		return errs[0]
	}
	return nil
}

// SpanExporter returns the underlying trace.SpanExporter.
// This is useful for registering with a TracerProvider.
func (e *CloudExporter) SpanExporter() trace.SpanExporter {
	if e == nil {
		return nil
	}
	return e.traceExporter
}
