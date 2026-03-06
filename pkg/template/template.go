package template

import (
	"regexp"
	"time"
)

// FieldExtraction defines how to extract a metric value.
// Exactly one of Duration or TimestampRange should be set.
type FieldExtraction struct {
	// Duration extracts value directly from a duration field in the log.
	// Use when log contains "X ms" or "X seconds" etc.
	Duration *DurationExtraction `yaml:"duration,omitempty"`

	// TimestampRange computes duration from start and end timestamps.
	// Use when log has marker lines with timestamps.
	TimestampRange *TimestampRangeExtraction `yaml:"timestamp_range,omitempty"`
}

// DurationExtraction extracts a numeric duration from a log line.
type DurationExtraction struct {
	// Pattern is a regex with a capture group for the numeric value.
	// Example: `train_step_timing in s: (\d+\.?\d*)`
	// Example: `load-checkpoint .*: \((\d+\.?\d*), \d+\.?\d*\)` (for min value)
	Pattern string `yaml:"pattern"`

	// Unit of the extracted value: "ms", "s", "us"
	Unit string `yaml:"unit"`

	// CaptureGroup selects which capture group to use (1-indexed). Default 1.
	CaptureGroup int `yaml:"capture_group,omitempty"`

	// UseMin when true, for patterns like (min, max), use the first number. Default true for (min,max) patterns.
	UseMin bool `yaml:"use_min,omitempty"`

	// UseMax when true, for patterns like (min, max), use the second number.
	UseMax bool `yaml:"use_max,omitempty"`

	// Aggregate for repeated matches: "first", "last", "avg", "min", "max"
	Aggregate string `yaml:"aggregate,omitempty"`
}

// TimestampRangeExtraction computes duration from two timestamp markers.
type TimestampRangeExtraction struct {
	// StartPattern is a regex that matches the line containing the start timestamp.
	// Must contain a capture group for the timestamp, or use TimestampLayout to parse the whole line.
	StartPattern string `yaml:"start_pattern"`

	// EndPattern is a regex that matches the line containing the end timestamp.
	EndPattern string `yaml:"end_pattern"`

	// TimestampPattern extracts the timestamp from matched lines.
	// Common formats:
	//   - `(\d{4}-\d{2}-\d{2} \d{2}:\d{2}:\d{2}\.\d+)` for "2026-02-21 01:32:44.979469"
	//   - `(\d{4}-\d{2}-\d{2} \d{2}:\d{2}:\d{2})` for "2026-01-12 12:23:28"
	TimestampPattern string `yaml:"timestamp_pattern"`

	// Layout for time.Parse. Default: "2006-01-02 15:04:05.999999999"
	Layout string `yaml:"layout,omitempty"`
}

// Template defines extraction rules for a log format.
type Template struct {
	Name   string                    `yaml:"name"`
	Fields map[string]FieldExtraction `yaml:"fields"`
}

// ExtractedMetric holds a parsed metric value.
type ExtractedMetric struct {
	Name     string        `json:"name"`
	Duration time.Duration `json:"duration_ms"`
	ValueMs  float64       `json:"value_ms"`
	Source   string        `json:"source"` // "duration" or "timestamp_range"
	Line     string        `json:"line,omitempty"`
}

// CompiledTemplate is a template with pre-compiled regexes.
type CompiledTemplate struct {
	Template
	startRegexes map[string]*regexp.Regexp
	endRegexes   map[string]*regexp.Regexp
	durationRe   map[string]*regexp.Regexp
}
