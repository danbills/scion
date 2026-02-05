/*
Copyright 2025 The Scion Authors.
*/

package telemetry

import (
	"context"
	"fmt"
	"io"
	"net"
	"net/http"
	"sync"

	coltracepb "go.opentelemetry.io/proto/otlp/collector/trace/v1"
	tracepb "go.opentelemetry.io/proto/otlp/trace/v1"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/proto"
)

// SpanHandler is called when spans are received.
type SpanHandler func(ctx context.Context, spans []*tracepb.ResourceSpans) error

// Receiver accepts OTLP trace data via gRPC and HTTP.
type Receiver struct {
	config     *Config
	grpcServer *grpc.Server
	httpServer *http.Server
	handler    SpanHandler
	mu         sync.Mutex
	running    bool
}

// NewReceiver creates a new OTLP receiver.
func NewReceiver(config *Config, handler SpanHandler) *Receiver {
	return &Receiver{
		config:  config,
		handler: handler,
	}
}

// Start starts the OTLP gRPC and HTTP receivers.
func (r *Receiver) Start(ctx context.Context) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if r.running {
		return fmt.Errorf("receiver already running")
	}

	// Start gRPC server
	grpcAddr := fmt.Sprintf(":%d", r.config.GRPCPort)
	grpcLis, err := net.Listen("tcp", grpcAddr)
	if err != nil {
		return fmt.Errorf("failed to listen on gRPC port %d: %w", r.config.GRPCPort, err)
	}

	r.grpcServer = grpc.NewServer()
	coltracepb.RegisterTraceServiceServer(r.grpcServer, &traceServiceServer{handler: r.handler})

	go func() {
		if err := r.grpcServer.Serve(grpcLis); err != nil && err != grpc.ErrServerStopped {
			// Log error but don't fail - receiver may be stopping
		}
	}()

	// Start HTTP server
	httpAddr := fmt.Sprintf(":%d", r.config.HTTPPort)
	httpLis, err := net.Listen("tcp", httpAddr)
	if err != nil {
		r.grpcServer.Stop()
		return fmt.Errorf("failed to listen on HTTP port %d: %w", r.config.HTTPPort, err)
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/v1/traces", r.handleHTTPTraces)

	r.httpServer = &http.Server{
		Handler: mux,
	}

	go func() {
		if err := r.httpServer.Serve(httpLis); err != nil && err != http.ErrServerClosed {
			// Log error but don't fail
		}
	}()

	r.running = true
	return nil
}

// Stop stops the OTLP receivers.
func (r *Receiver) Stop(ctx context.Context) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if !r.running {
		return nil
	}

	var errs []error

	// Stop gRPC server
	if r.grpcServer != nil {
		r.grpcServer.GracefulStop()
	}

	// Stop HTTP server
	if r.httpServer != nil {
		if err := r.httpServer.Shutdown(ctx); err != nil {
			errs = append(errs, fmt.Errorf("HTTP shutdown error: %w", err))
		}
	}

	r.running = false

	if len(errs) > 0 {
		return errs[0]
	}
	return nil
}

// IsRunning returns true if the receiver is running.
func (r *Receiver) IsRunning() bool {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.running
}

// handleHTTPTraces handles OTLP HTTP trace requests.
func (r *Receiver) handleHTTPTraces(w http.ResponseWriter, req *http.Request) {
	if req.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Read the request body
	body, err := io.ReadAll(req.Body)
	if err != nil {
		http.Error(w, "Failed to read body", http.StatusBadRequest)
		return
	}

	var exportReq coltracepb.ExportTraceServiceRequest
	if err := proto.Unmarshal(body, &exportReq); err != nil {
		http.Error(w, "Failed to parse OTLP request", http.StatusBadRequest)
		return
	}

	// Process spans
	if r.handler != nil {
		if err := r.handler(req.Context(), exportReq.ResourceSpans); err != nil {
			http.Error(w, "Failed to process spans", http.StatusInternalServerError)
			return
		}
	}

	// Return success response
	resp := &coltracepb.ExportTraceServiceResponse{}
	respBytes, _ := proto.Marshal(resp)
	w.Header().Set("Content-Type", "application/x-protobuf")
	w.WriteHeader(http.StatusOK)
	w.Write(respBytes)
}

// traceServiceServer implements the OTLP gRPC trace service.
type traceServiceServer struct {
	coltracepb.UnimplementedTraceServiceServer
	handler SpanHandler
}

// Export implements the OTLP trace export RPC.
func (s *traceServiceServer) Export(ctx context.Context, req *coltracepb.ExportTraceServiceRequest) (*coltracepb.ExportTraceServiceResponse, error) {
	if s.handler != nil {
		if err := s.handler(ctx, req.ResourceSpans); err != nil {
			return nil, err
		}
	}
	return &coltracepb.ExportTraceServiceResponse{}, nil
}
