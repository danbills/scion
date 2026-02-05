/*
Copyright 2025 The Scion Authors.
*/

package telemetry

import (
	"context"
	"os"
	"testing"
	"time"
)

func TestNew_Disabled(t *testing.T) {
	// Clear env and disable telemetry
	clearTelemetryEnv()
	os.Setenv(EnvEnabled, "false")
	defer clearTelemetryEnv()

	pipeline := New()
	if pipeline != nil {
		t.Error("Expected nil pipeline when telemetry is disabled")
	}
}

func TestNew_Enabled(t *testing.T) {
	clearTelemetryEnv()
	os.Setenv(EnvEnabled, "true")
	defer clearTelemetryEnv()

	pipeline := New()
	if pipeline == nil {
		t.Error("Expected non-nil pipeline when telemetry is enabled")
		return
	}

	if pipeline.Config() == nil {
		t.Error("Expected pipeline to have config")
	}
}

func TestPipeline_StartStop(t *testing.T) {
	clearTelemetryEnv()
	// Use non-standard ports to avoid conflicts
	os.Setenv(EnvEnabled, "true")
	os.Setenv(EnvCloudEnabled, "false") // Disable cloud to avoid GCP auth issues in tests
	os.Setenv(EnvGRPCPort, "54317")
	os.Setenv(EnvHTTPPort, "54318")
	defer clearTelemetryEnv()

	pipeline := New()
	if pipeline == nil {
		t.Fatal("Expected non-nil pipeline")
	}

	ctx := context.Background()

	// Start pipeline
	if err := pipeline.Start(ctx); err != nil {
		t.Fatalf("Failed to start pipeline: %v", err)
	}

	if !pipeline.IsRunning() {
		t.Error("Expected pipeline to be running after Start")
	}

	// Give servers time to start
	time.Sleep(50 * time.Millisecond)

	// Stop pipeline
	stopCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	if err := pipeline.Stop(stopCtx); err != nil {
		t.Fatalf("Failed to stop pipeline: %v", err)
	}

	if pipeline.IsRunning() {
		t.Error("Expected pipeline to not be running after Stop")
	}
}

func TestPipeline_DoubleStart(t *testing.T) {
	clearTelemetryEnv()
	os.Setenv(EnvEnabled, "true")
	os.Setenv(EnvCloudEnabled, "false")
	os.Setenv(EnvGRPCPort, "54319")
	os.Setenv(EnvHTTPPort, "54320")
	defer clearTelemetryEnv()

	pipeline := New()
	if pipeline == nil {
		t.Fatal("Expected non-nil pipeline")
	}

	ctx := context.Background()
	defer func() {
		stopCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
		pipeline.Stop(stopCtx)
		cancel()
	}()

	// First start should succeed
	if err := pipeline.Start(ctx); err != nil {
		t.Fatalf("First start failed: %v", err)
	}

	// Second start should fail
	if err := pipeline.Start(ctx); err == nil {
		t.Error("Expected error on double start")
	}
}

func TestPipeline_NilSafe(t *testing.T) {
	var pipeline *Pipeline

	// These should all be safe to call on nil
	if err := pipeline.Start(context.Background()); err != nil {
		t.Error("Start on nil should return nil")
	}
	if err := pipeline.Stop(context.Background()); err != nil {
		t.Error("Stop on nil should return nil")
	}
	if pipeline.IsRunning() {
		t.Error("IsRunning on nil should return false")
	}
	if pipeline.Config() != nil {
		t.Error("Config on nil should return nil")
	}
}

func TestNewWithConfig(t *testing.T) {
	// nil config
	if NewWithConfig(nil) != nil {
		t.Error("Expected nil pipeline for nil config")
	}

	// disabled config
	cfg := &Config{Enabled: false}
	if NewWithConfig(cfg) != nil {
		t.Error("Expected nil pipeline for disabled config")
	}

	// enabled config
	cfg = &Config{
		Enabled:  true,
		GRPCPort: 54321,
		HTTPPort: 54322,
	}
	pipeline := NewWithConfig(cfg)
	if pipeline == nil {
		t.Error("Expected non-nil pipeline for enabled config")
	}
}
