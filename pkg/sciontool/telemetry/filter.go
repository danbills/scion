/*
Copyright 2025 The Scion Authors.
*/

package telemetry

// Filter provides include/exclude filtering for event types.
type Filter struct {
	include map[string]bool // nil = include all
	exclude map[string]bool
}

// NewFilter creates a new filter from configuration.
func NewFilter(config FilterConfig) *Filter {
	f := &Filter{}

	// Build include set (nil means include all)
	if len(config.Include) > 0 {
		f.include = make(map[string]bool, len(config.Include))
		for _, t := range config.Include {
			f.include[t] = true
		}
	}

	// Build exclude set
	if len(config.Exclude) > 0 {
		f.exclude = make(map[string]bool, len(config.Exclude))
		for _, t := range config.Exclude {
			f.exclude[t] = true
		}
	}

	return f
}

// ShouldProcess returns true if the event type should be processed.
// An event is processed if:
// 1. It's in the include list (or include list is empty, meaning include all)
// 2. AND it's not in the exclude list
func (f *Filter) ShouldProcess(eventType string) bool {
	if f == nil {
		return true
	}

	// Check include list first (nil = include all)
	if f.include != nil && !f.include[eventType] {
		return false
	}

	// Check exclude list
	if f.exclude != nil && f.exclude[eventType] {
		return false
	}

	return true
}

// ShouldProcessSpan checks if a span should be processed based on its name.
// This is a convenience method that treats span name as the event type.
func (f *Filter) ShouldProcessSpan(spanName string) bool {
	return f.ShouldProcess(spanName)
}
