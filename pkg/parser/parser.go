package parser

import (
	"bufio"
	"fmt"
	"os"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
)

// FieldExtraction defines how to extract a metric value.
type FieldExtraction struct {
	Duration      *DurationExtraction      `yaml:"duration,omitempty"`
	TimestampRange *TimestampRangeExtraction `yaml:"timestamp_range,omitempty"`
}

type DurationExtraction struct {
	Pattern      string `yaml:"pattern"`
	Unit         string `yaml:"unit"`
	CaptureGroup int    `yaml:"capture_group,omitempty"`
	UseMin       bool   `yaml:"use_min,omitempty"`
	UseMax       bool   `yaml:"use_max,omitempty"`
	Aggregate    string `yaml:"aggregate,omitempty"`
}

type TimestampRangeExtraction struct {
	StartPattern     string `yaml:"start_pattern"`
	EndPattern       string `yaml:"end_pattern"`
	TimestampPattern string `yaml:"timestamp_pattern"`
	Layout           string `yaml:"layout,omitempty"`
}

// FieldExtractionList supports YAML unmarshaling of either a single extraction or a list.
type FieldExtractionList []FieldExtraction

// UnmarshalYAML allows fields to be defined as a single extraction or a list of extractions.
func (f *FieldExtractionList) UnmarshalYAML(value *yaml.Node) error {
	if value.Kind == yaml.SequenceNode {
		var list []FieldExtraction
		if err := value.Decode(&list); err != nil {
			return err
		}
		*f = list
		return nil
	}
	var single FieldExtraction
	if err := value.Decode(&single); err != nil {
		return err
	}
	*f = []FieldExtraction{single}
	return nil
}

// Template defines extraction rules for a log format.
type Template struct {
	Name   string                       `yaml:"name"`
	Fields map[string]FieldExtractionList `yaml:"fields"`
}

// ExtractedMetric holds a parsed metric value.
type ExtractedMetric struct {
	Name     string  `json:"name"`
	ValueMs  float64 `json:"value_ms"`
	Source   string  `json:"source"`
	Unit     string  `json:"unit,omitempty"`
	RawValue string  `json:"raw_value,omitempty"`
}

// compiledRule holds pre-compiled regexes for one extraction rule.
type compiledRule struct {
	name         string
	duration     *DurationExtraction
	durationRe   *regexp.Regexp
	timestamp    *TimestampRangeExtraction
	startRe      *regexp.Regexp
	endRe        *regexp.Regexp
	tsRe         *regexp.Regexp
	layout       string
}

// Parser applies a template to a log stream.
type Parser struct {
	template Template
	rules    []compiledRule
}

// LoadTemplate loads a template from a YAML file.
func LoadTemplate(path string) (*Template, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var t Template
	if err := yaml.Unmarshal(data, &t); err != nil {
		return nil, err
	}
	return &t, nil
}

// NewParser creates a parser for the given template.
func NewParser(t *Template) (*Parser, error) {
	p := &Parser{template: *t}

	names := make([]string, 0, len(t.Fields))
	for name := range t.Fields {
		names = append(names, name)
	}
	sort.Strings(names)

	for _, name := range names {
		list := t.Fields[name]
		for i, fe := range list {
			var r compiledRule
			r.name = name

			if fe.Duration != nil {
				re, err := regexp.Compile(fe.Duration.Pattern)
				if err != nil {
					return nil, fmt.Errorf("field %s rule %d duration pattern: %w", name, i, err)
				}
				r.duration = fe.Duration
				r.durationRe = re
			}
			if fe.TimestampRange != nil {
				startRe, err := regexp.Compile(fe.TimestampRange.StartPattern)
				if err != nil {
					return nil, fmt.Errorf("field %s rule %d start pattern: %w", name, i, err)
				}
				endRe, err := regexp.Compile(fe.TimestampRange.EndPattern)
				if err != nil {
					return nil, fmt.Errorf("field %s rule %d end pattern: %w", name, i, err)
				}
				tsRe, err := regexp.Compile(fe.TimestampRange.TimestampPattern)
				if err != nil {
					return nil, fmt.Errorf("field %s rule %d timestamp pattern: %w", name, i, err)
				}
				r.timestamp = fe.TimestampRange
				r.startRe = startRe
				r.endRe = endRe
				r.tsRe = tsRe
				r.layout = fe.TimestampRange.Layout
				if r.layout == "" {
					r.layout = "2006-01-02 15:04:05.999999999"
				}
			}
			p.rules = append(p.rules, r)
		}
	}
	return p, nil
}

// ParseFile parses a log file and returns extracted metrics.
func (p *Parser) ParseFile(path string) ([]ExtractedMetric, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	return p.ParseStream(bufio.NewScanner(f))
}

// ParseStream parses a log stream (scanner) and returns extracted metrics.
func (p *Parser) ParseStream(scanner *bufio.Scanner) ([]ExtractedMetric, error) {
	var lines []string
	for scanner.Scan() {
		lines = append(lines, scanner.Text())
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}
	return p.ParseLines(lines)
}

// ParseLines parses a slice of log lines.
func (p *Parser) ParseLines(lines []string) ([]ExtractedMetric, error) {
	var results []ExtractedMetric
	seen := make(map[string]bool)

	for _, r := range p.rules {
		if seen[r.name] {
			continue
		}
		if r.duration != nil {
			metrics := p.extractDuration(&r, lines)
			if len(metrics) > 0 {
				results = append(results, metrics...)
				seen[r.name] = true
			}
		}
		if r.timestamp != nil {
			if m, err := p.extractTimestampRange(&r, lines); err == nil {
				results = append(results, *m)
				seen[r.name] = true
			}
		}
	}
	return results, nil
}

func (p *Parser) extractDuration(r *compiledRule, lines []string) []ExtractedMetric {
	if r.durationRe == nil {
		return nil
	}
	d := r.duration

	var values []float64
	captureGroup := 1
	if d.CaptureGroup > 0 {
		captureGroup = d.CaptureGroup
	}

	for _, line := range lines {
		matches := r.durationRe.FindStringSubmatch(line)
		if len(matches) <= captureGroup {
			continue
		}
		valStr := matches[captureGroup]
		if d.UseMax && len(matches) > 2 {
			valStr = matches[2]
		}
		val, err := strconv.ParseFloat(strings.TrimSpace(valStr), 64)
		if err != nil {
			continue
		}
		switch d.Unit {
		case "s", "sec":
			val *= 1000
		case "us", "µs":
			val /= 1000
		case "raw":
			// no conversion (e.g. TFLOPS)
		}
		values = append(values, val)
	}

	if len(values) == 0 {
		return nil
	}

	var finalVal float64
	switch d.Aggregate {
	case "last":
		finalVal = values[len(values)-1]
	case "first":
		finalVal = values[0]
	case "min":
		finalVal = values[0]
		for _, v := range values[1:] {
			if v < finalVal {
				finalVal = v
			}
		}
	case "max":
		finalVal = values[0]
		for _, v := range values[1:] {
			if v > finalVal {
				finalVal = v
			}
		}
	case "avg":
		sum := 0.0
		for _, v := range values {
			sum += v
		}
		finalVal = sum / float64(len(values))
	default:
		finalVal = values[len(values)-1]
	}

	return []ExtractedMetric{{
		Name:    r.name,
		ValueMs: finalVal,
		Source:  "duration",
		Unit:    d.Unit,
	}}
}

func (p *Parser) extractTimestampRange(r *compiledRule, lines []string) (*ExtractedMetric, error) {
	if r.startRe == nil || r.endRe == nil || r.tsRe == nil {
		return nil, fmt.Errorf("incomplete timestamp rule")
	}
	layout := r.layout
	if layout == "" {
		layout = "2006-01-02 15:04:05.999999999"
	}

	var startTime, endTime time.Time
	var startFound, endFound bool

	for _, line := range lines {
		if !startFound && r.startRe.MatchString(line) {
			ts := r.tsRe.FindStringSubmatch(line)
			if len(ts) > 1 {
				t, err := time.Parse(layout, strings.TrimSpace(ts[1]))
				if err == nil {
					startTime = t
					startFound = true
				}
			}
		}
		if startFound && !endFound && r.endRe.MatchString(line) {
			ts := r.tsRe.FindStringSubmatch(line)
			if len(ts) > 1 {
				t, err := time.Parse(layout, strings.TrimSpace(ts[1]))
				if err == nil {
					endTime = t
					endFound = true
					break
				}
			}
		}
	}

	if !startFound || !endFound {
		return nil, fmt.Errorf("could not find start/end timestamps for %s", r.name)
	}

	diff := endTime.Sub(startTime)
	return &ExtractedMetric{
		Name:    r.name,
		ValueMs: float64(diff.Milliseconds()),
		Source:  "timestamp_range",
	}, nil
}

