/*
Copyright 2025 The Scion Authors.
*/

// Package telemetry provides OTLP telemetry collection and forwarding for sciontool.
// It enables agents to collect and forward traces to Google Cloud backend.
package telemetry

import (
	"os"
	"strconv"
	"strings"
)

// Environment variable names for telemetry configuration.
const (
	// EnvEnabled controls whether telemetry collection is enabled.
	EnvEnabled = "SCION_TELEMETRY_ENABLED"
	// EnvCloudEnabled controls whether telemetry is forwarded to cloud backend.
	EnvCloudEnabled = "SCION_TELEMETRY_CLOUD_ENABLED"
	// EnvEndpoint is the cloud OTLP endpoint (required for cloud forwarding).
	EnvEndpoint = "SCION_OTEL_ENDPOINT"
	// EnvProtocol is the OTLP protocol to use (grpc or http).
	EnvProtocol = "SCION_OTEL_PROTOCOL"
	// EnvInsecure controls whether TLS verification is skipped.
	EnvInsecure = "SCION_OTEL_INSECURE"
	// EnvGRPCPort is the local gRPC receiver port.
	EnvGRPCPort = "SCION_OTEL_GRPC_PORT"
	// EnvHTTPPort is the local HTTP receiver port.
	EnvHTTPPort = "SCION_OTEL_HTTP_PORT"
	// EnvFilterExclude is a comma-separated list of event types to exclude.
	EnvFilterExclude = "SCION_TELEMETRY_FILTER_EXCLUDE"
	// EnvFilterInclude is a comma-separated list of event types to include.
	EnvFilterInclude = "SCION_TELEMETRY_FILTER_INCLUDE"
	// EnvProjectID is the GCP project ID for the exporter.
	EnvProjectID = "SCION_GCP_PROJECT_ID"
)

// Default configuration values.
const (
	DefaultGRPCPort = 4317
	DefaultHTTPPort = 4318
	DefaultProtocol = "grpc"
)

// Default event types to exclude for privacy.
var DefaultFilterExclude = []string{"agent.user.prompt"}

// Config holds the telemetry configuration.
type Config struct {
	// Enabled controls whether telemetry collection is active.
	Enabled bool
	// CloudEnabled controls whether data is forwarded to cloud backend.
	CloudEnabled bool
	// Endpoint is the cloud OTLP endpoint.
	Endpoint string
	// Protocol is the OTLP protocol ("grpc" or "http").
	Protocol string
	// Insecure skips TLS verification if true.
	Insecure bool
	// GRPCPort is the local gRPC receiver port.
	GRPCPort int
	// HTTPPort is the local HTTP receiver port.
	HTTPPort int
	// ProjectID is the GCP project ID for the exporter.
	ProjectID string
	// Filter contains the filtering configuration.
	Filter FilterConfig
}

// FilterConfig holds include/exclude patterns for event filtering.
type FilterConfig struct {
	// Include is a list of event types to include (if empty, include all).
	Include []string
	// Exclude is a list of event types to exclude (always applied after include).
	Exclude []string
}

// LoadConfig loads telemetry configuration from environment variables.
func LoadConfig() *Config {
	cfg := &Config{
		Enabled:      parseBoolEnv(EnvEnabled, true),
		CloudEnabled: parseBoolEnv(EnvCloudEnabled, true),
		Endpoint:     os.Getenv(EnvEndpoint),
		Protocol:     getEnvOrDefault(EnvProtocol, DefaultProtocol),
		Insecure:     parseBoolEnv(EnvInsecure, false),
		GRPCPort:     parseIntEnv(EnvGRPCPort, DefaultGRPCPort),
		HTTPPort:     parseIntEnv(EnvHTTPPort, DefaultHTTPPort),
		ProjectID:    os.Getenv(EnvProjectID),
		Filter: FilterConfig{
			Include: parseCSVEnv(EnvFilterInclude),
			Exclude: parseCSVEnv(EnvFilterExclude),
		},
	}

	// Apply default exclude list if not explicitly set
	if len(cfg.Filter.Exclude) == 0 && os.Getenv(EnvFilterExclude) == "" {
		cfg.Filter.Exclude = DefaultFilterExclude
	}

	return cfg
}

// IsCloudConfigured returns true if cloud forwarding is properly configured.
func (c *Config) IsCloudConfigured() bool {
	if c == nil {
		return false
	}
	return c.CloudEnabled && c.Endpoint != ""
}

// parseBoolEnv parses a boolean environment variable with a default value.
func parseBoolEnv(key string, defaultVal bool) bool {
	val := os.Getenv(key)
	if val == "" {
		return defaultVal
	}
	val = strings.ToLower(val)
	switch val {
	case "true", "1", "yes", "on":
		return true
	case "false", "0", "no", "off":
		return false
	default:
		return defaultVal
	}
}

// parseIntEnv parses an integer environment variable with a default value.
func parseIntEnv(key string, defaultVal int) int {
	val := os.Getenv(key)
	if val == "" {
		return defaultVal
	}
	intVal, err := strconv.Atoi(val)
	if err != nil {
		return defaultVal
	}
	return intVal
}

// getEnvOrDefault returns the environment variable value or a default.
func getEnvOrDefault(key, defaultVal string) string {
	val := os.Getenv(key)
	if val == "" {
		return defaultVal
	}
	return val
}

// parseCSVEnv parses a comma-separated environment variable into a slice.
func parseCSVEnv(key string) []string {
	val := os.Getenv(key)
	if val == "" {
		return nil
	}
	parts := strings.Split(val, ",")
	result := make([]string, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			result = append(result, p)
		}
	}
	return result
}
