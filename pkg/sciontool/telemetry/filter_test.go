/*
Copyright 2025 The Scion Authors.
*/

package telemetry

import "testing"

func TestFilter_ShouldProcess(t *testing.T) {
	tests := []struct {
		name      string
		config    FilterConfig
		eventType string
		expected  bool
	}{
		{
			name:      "nil filter allows all",
			config:    FilterConfig{},
			eventType: "any.event",
			expected:  true,
		},
		{
			name: "empty include allows all",
			config: FilterConfig{
				Include: []string{},
			},
			eventType: "any.event",
			expected:  true,
		},
		{
			name: "include list filters",
			config: FilterConfig{
				Include: []string{"event.a", "event.b"},
			},
			eventType: "event.a",
			expected:  true,
		},
		{
			name: "include list excludes non-matching",
			config: FilterConfig{
				Include: []string{"event.a", "event.b"},
			},
			eventType: "event.c",
			expected:  false,
		},
		{
			name: "exclude list filters",
			config: FilterConfig{
				Exclude: []string{"event.private"},
			},
			eventType: "event.private",
			expected:  false,
		},
		{
			name: "exclude list allows non-matching",
			config: FilterConfig{
				Exclude: []string{"event.private"},
			},
			eventType: "event.public",
			expected:  true,
		},
		{
			name: "exclude takes precedence over include",
			config: FilterConfig{
				Include: []string{"event.a", "event.b"},
				Exclude: []string{"event.b"},
			},
			eventType: "event.b",
			expected:  false,
		},
		{
			name: "include and exclude combined - allowed",
			config: FilterConfig{
				Include: []string{"event.a", "event.b", "event.c"},
				Exclude: []string{"event.b"},
			},
			eventType: "event.a",
			expected:  true,
		},
		{
			name: "default exclude list",
			config: FilterConfig{
				Exclude: DefaultFilterExclude,
			},
			eventType: "agent.user.prompt",
			expected:  false,
		},
		{
			name: "default exclude list allows other events",
			config: FilterConfig{
				Exclude: DefaultFilterExclude,
			},
			eventType: "agent.tool.invoke",
			expected:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := NewFilter(tt.config)
			if got := f.ShouldProcess(tt.eventType); got != tt.expected {
				t.Errorf("ShouldProcess(%q) = %v, want %v", tt.eventType, got, tt.expected)
			}
		})
	}
}

func TestFilter_NilFilter(t *testing.T) {
	var f *Filter
	if !f.ShouldProcess("any.event") {
		t.Error("nil filter should allow all events")
	}
}

func TestFilter_ShouldProcessSpan(t *testing.T) {
	f := NewFilter(FilterConfig{
		Exclude: []string{"private.span"},
	})

	if f.ShouldProcessSpan("private.span") {
		t.Error("ShouldProcessSpan should exclude private.span")
	}
	if !f.ShouldProcessSpan("public.span") {
		t.Error("ShouldProcessSpan should allow public.span")
	}
}
