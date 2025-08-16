package testutil

import (
	"bufio"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"
)

// MetricResult represents a single metric.
type MetricResult struct {
	Name      string            `json:"name"`
	Labels    map[string]string `json:"labels"`
	Value     float64           `json:"value"`
	Timestamp *time.Time        `json:"timestamp,omitempty"`
}

// MetricsFetcher defines the interface for fetching and loading metrics.
type MetricsFetcher interface {
	AddNameFilter(name string)
	AddLabelFilter(labelsKeyValue ...string)
	AddTimestampAfterFilter(ts time.Time)
	AddTimestampBeforeFilter(ts time.Time)
	Run() (map[string][]MetricResult, error)
	LoadMetrics(payload string) (map[string][]MetricResult, error)
}

// DefaultMetricsGetter is the default implementation of MetricsFetcher.
type DefaultMetricsGetter struct {
	URL              string
	metricNameFilter string
	labelFilters     map[string]string
	afterTimestamp   *time.Time
	beforeTimestamp  *time.Time
	metrics          map[string][]MetricResult
}

// NewMetricsFetcher creates a new MetricsFetcher instance.
func NewMetricsFetcher(URL string) MetricsFetcher {
	return &DefaultMetricsGetter{
		URL:          URL,
		labelFilters: make(map[string]string),
		metrics:      make(map[string][]MetricResult),
	}
}

func (dmg *DefaultMetricsGetter) AddNameFilter(name string) {
	dmg.metricNameFilter = name
}

func (dmg *DefaultMetricsGetter) AddLabelFilter(labelsKeyValue ...string) {
	for i := 0; i < len(labelsKeyValue); i += 2 {
		if i+1 < len(labelsKeyValue) {
			dmg.labelFilters[labelsKeyValue[i]] = labelsKeyValue[i+1]
		}
	}
}

func (dmg *DefaultMetricsGetter) AddTimestampAfterFilter(ts time.Time) {
	dmg.afterTimestamp = &ts
}

func (dmg *DefaultMetricsGetter) AddTimestampBeforeFilter(ts time.Time) {
	dmg.beforeTimestamp = &ts
}

// Run fetches metrics via HTTP, parses them, and applies filters.
func (dmg *DefaultMetricsGetter) Run() (map[string][]MetricResult, error) {
	resp, err := http.Get(dmg.URL)
	if err != nil {
		return nil, fmt.Errorf("failed to query service endpoint: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected response status: %s", resp.Status)
	}

	dmg.metrics = make(map[string][]MetricResult)

	if err := dmg.processReader(resp.Body); err != nil {
		return nil, err
	}

	return dmg.metrics, nil
}

// LoadMetrics parses a provided metrics payload string and applies filters.
func (dmg *DefaultMetricsGetter) LoadMetrics(payload string) (map[string][]MetricResult, error) {
	dmg.metrics = make(map[string][]MetricResult)

	reader := strings.NewReader(payload)
	if err := dmg.processReader(reader); err != nil {
		return nil, err
	}

	return dmg.metrics, nil
}

// processReader reads from the provided io.Reader line by line,
// parses the metric lines, applies the filters, and stores the results.
func (dmg *DefaultMetricsGetter) processReader(r io.Reader) error {
	scanner := bufio.NewScanner(r)
	for scanner.Scan() {
		line := scanner.Text()
		mr, found, err := dmg.parseLine(line)
		if err != nil {
			return err
		}
		if !found {
			continue
		}
		if dmg.applyFilters(mr) {
			dmg.metrics[mr.Name] = append(dmg.metrics[mr.Name], *mr)
		}
	}

	if err := scanner.Err(); err != nil {
		return fmt.Errorf("failed to read input: %w", err)
	}
	return nil
}

// applyFilters checks whether the given MetricResult passes all filters.
func (dmg *DefaultMetricsGetter) applyFilters(mr *MetricResult) bool {
	if dmg.metricNameFilter != "" && !strings.HasPrefix(mr.Name, dmg.metricNameFilter) {
		return false
	}
	for k, v := range dmg.labelFilters {
		if mr.Labels[k] != v {
			return false
		}
	}
	if (dmg.afterTimestamp != nil || dmg.beforeTimestamp != nil) && mr.Timestamp == nil {
		return false
	}
	if dmg.afterTimestamp != nil && mr.Timestamp.Before(*dmg.afterTimestamp) {
		return false
	}
	if dmg.beforeTimestamp != nil && mr.Timestamp.After(*dmg.beforeTimestamp) {
		return false
	}
	return true
}

// parseLine parses a single line of the metrics payload.
func (dmg *DefaultMetricsGetter) parseLine(line string) (*MetricResult, bool, error) {
	// Ignore comments and empty lines.
	if strings.HasPrefix(line, "#") || strings.TrimSpace(line) == "" {
		return nil, false, nil
	}

	// Use a custom splitter that respects quotes and curly braces.
	metaPart, valuePart, err := splitMetricLine(line)
	if err != nil {
		return nil, false, err
	}

	// Parse the metric name and labels from metaPart.
	name, labels := dmg.parseMetricNameAndLabels(metaPart)

	// Now parse the value (and optional timestamp) from the valuePart.
	parts := strings.Fields(valuePart)
	if len(parts) < 1 {
		err = fmt.Errorf("invalid metric line, no value: %s", line)
		return nil, false, err
	}

	value, err := strconv.ParseFloat(parts[0], 64)
	if err != nil {
		return nil, false, fmt.Errorf("failed to parse metric value: %w", err)
	}

	var timestamp *time.Time
	if len(parts) >= 2 {
		ts, err := strconv.ParseInt(parts[1], 10, 64)
		if err == nil {
			t := time.Unix(ts, 0)
			timestamp = &t
		}
	}

	mr := &MetricResult{
		Name:      name,
		Labels:    labels,
		Value:     value,
		Timestamp: timestamp,
	}
	return mr, true, nil
}

// splitMetricLine splits a metric line into its meta part (name and labels)
// and its value part. It takes into account quoted strings and curly braces so that
// spaces inside them are not treated as delimiters.
func splitMetricLine(line string) (metaPart string, valuePart string, err error) {
	inQuotes := false
	inBraces := false
	for i, r := range line {
		switch r {
		case '"':
			inQuotes = !inQuotes
		case '{':
			if !inQuotes {
				inBraces = true
			}
		case '}':
			if !inQuotes {
				inBraces = false
			}
		case ' ':
			if !inQuotes && !inBraces {
				metaPart = strings.TrimSpace(line[:i])
				valuePart = strings.TrimSpace(line[i:])
				return metaPart, valuePart, nil
			}
		}
	}
	return "", "", fmt.Errorf("invalid metric line, no value part found: %s", line)
}

// parseMetricNameAndLabels splits the metric name from its labels (if any).
func (dmg *DefaultMetricsGetter) parseMetricNameAndLabels(input string) (string, map[string]string) {
	labels := make(map[string]string)
	nameEnd := strings.Index(input, "{")
	if nameEnd == -1 {
		return input, labels
	}

	name := input[:nameEnd]
	labelStr := strings.Trim(input[nameEnd:], "{}")
	labelPairs := strings.Split(labelStr, ",")
	for _, pair := range labelPairs {
		kv := strings.SplitN(pair, "=", 2)
		if len(kv) == 2 {
			labels[kv[0]] = strings.Trim(kv[1], "\"")
		}
	}
	return name, labels
}
