package plugins

import (
	"time"

	// Developed on https://github.com/DataDog/datadog-go/tree/a27810dd518c69be741a7fd5d0e39f674f615be8
	"github.com/DataDog/datadog-go/statsd"
	"github.com/afex/hystrix-go/hystrix/metric_collector"
)

// These metrics are constants because we're leveraging the Datadog tagging
// extension to statsd.
//
// They only apply to the DatadogCollector and are only useful if providing your
// own implemenation of DatadogClient
const (
	// DM = Datadog Metric
	DM_CircuitOpen       = "hystrix.circuitOpen"
	DM_Attempts          = "hystrix.attempts"
	DM_Errors            = "hystrix.errors"
	DM_Successes         = "hystrix.successes"
	DM_Failures          = "hystrix.failures"
	DM_Rejects           = "hystrix.rejects"
	DM_ShortCircuits     = "hystrix.shortCircuits"
	DM_Timeouts          = "hystrix.timeouts"
	DM_FallbackSuccesses = "hystrix.fallbackSuccesses"
	DM_FallbackFailures  = "hystrix.fallbackFailures"
	DM_TotalDuration     = "hystrix.totalDuration"
	DM_RunDuration       = "hystrix.runDuration"
)

type (
	// DatadogClient is the minimum interface needed by
	// NewDatadogCollectorWithClient
	DatadogClient interface {
		Count(name string, value int64, tags []string, rate float64) error
		Gauge(name string, value float64, tags []string, rate float64) error
		TimeInMilliseconds(name string, value float64, tags []string, rate float64) error
	}

	// DatadogCollector fulfills the metricCollector interface allowing users to
	// ship circuit stats to Datadog.
	//
	// This Collector, by default, uses github.com/DataDog/datadog-go/statsd for
	// transport. The main advantage of this over statsd is building graphs and
	// multi-alert monitors around single metrics (constantized above) and
	// adding tag dimensions. You can set up a single monitor to rule them all
	// across services and geographies. Graphs become much simpler to setup by
	// allowing you to create queries like the following
	//
	//   {
	//     "viz": "timeseries",
	//     "requests": [
	//       {
	//         "q": "max:hystrix.runDuration.95percentile{$region} by {hystrixcircuit}",
	//         "type": "line"
	//       }
	//     ]
	//   }
	//
	// As new circuits come online you get graphing and monitoring "for free".
	DatadogCollector struct {
		client DatadogClient
		tags   []string
	}
)

// NewDatadogCollector creates a collector for a specific circuit with a
// "github.com/DataDog/datadog-go/statsd".(*Client).
//
// addr is in the format "<host>:<port>" (e.g. "localhost:8125")
//
// prefix may be an empty string
//
// Example use
//  package main
//
//  import (
//  	"github.com/afex/hystrix-go/plugins"
//  	"github.com/afex/hystrix-go/hystrix/metric_collector"
//  )
//
//  func main() {
//  	collector, err := plugins.NewDatadogCollector("localhost:8125", "")
//  	if err != nil {
//  		panic(err)
//  	}
//  	metricCollector.Registry.Register(collector)
//  }
func NewDatadogCollector(addr, prefix string) (func(string) metricCollector.MetricCollector, error) {

	c, err := statsd.NewBuffered(addr, 100)
	if err != nil {
		return nil, err
	}

	// Prefix every metric with the app name
	c.Namespace = prefix

	return NewDatadogCollectorWithClient(c), nil
}

// NewDatadogCollectorWithClient accepts an interface which allows you to
// provide your own implementation of a statsd client, alter configuration on
// "github.com/DataDog/datadog-go/statsd".(*Client), provide additional tags per
// circuit-metric tuple, and add logging if you need it.
func NewDatadogCollectorWithClient(client DatadogClient) func(string) metricCollector.MetricCollector {

	return func(name string) metricCollector.MetricCollector {

		return &DatadogCollector{
			client: client,
			tags:   []string{"hystrixcircuit:" + name},
		}
	}
}

// IncrementAttempts increments the number of calls to this circuit.
func (dc *DatadogCollector) IncrementAttempts() {
	dc.client.Count(DM_Attempts, 1, dc.tags, 1.0)
}

// IncrementErrors increments the number of unsuccessful attempts.
// Attempts minus Errors will equal successes within a time range.
// Errors are any result from an attempt that is not a success.
func (dc *DatadogCollector) IncrementErrors() {
	dc.client.Count(DM_Errors, 1, dc.tags, 1.0)
}

// IncrementSuccesses increments the number of requests that succeed.
func (dc *DatadogCollector) IncrementSuccesses() {
	dc.client.Gauge(DM_CircuitOpen, 0, dc.tags, 1.0)
	dc.client.Count(DM_Successes, 1, dc.tags, 1.0)
}

// IncrementFailures increments the number of requests that fail.
func (dc *DatadogCollector) IncrementFailures() {
	dc.client.Count(DM_Failures, 1, dc.tags, 1.0)
}

// IncrementRejects increments the number of requests that are rejected.
func (dc *DatadogCollector) IncrementRejects() {
	dc.client.Count(DM_Rejects, 1, dc.tags, 1.0)
}

// IncrementShortCircuits increments the number of requests that short circuited
// due to the circuit being open.
func (dc *DatadogCollector) IncrementShortCircuits() {
	dc.client.Gauge(DM_CircuitOpen, 1, dc.tags, 1.0)
	dc.client.Count(DM_ShortCircuits, 1, dc.tags, 1.0)
}

// IncrementTimeouts increments the number of timeouts that occurred in the
// circuit breaker.
func (dc *DatadogCollector) IncrementTimeouts() {
	dc.client.Count(DM_Timeouts, 1, dc.tags, 1.0)
}

// IncrementFallbackSuccesses increments the number of successes that occurred
// during the execution of the fallback function.
func (dc *DatadogCollector) IncrementFallbackSuccesses() {
	dc.client.Count(DM_FallbackSuccesses, 1, dc.tags, 1.0)
}

// IncrementFallbackFailures increments the number of failures that occurred
// during the execution of the fallback function.
func (dc *DatadogCollector) IncrementFallbackFailures() {
	dc.client.Count(DM_FallbackFailures, 1, dc.tags, 1.0)
}

// UpdateTotalDuration updates the internal counter of how long we've run for.
func (dc *DatadogCollector) UpdateTotalDuration(timeSinceStart time.Duration) {
	ms := float64(timeSinceStart.Nanoseconds() / 1000000)
	dc.client.TimeInMilliseconds(DM_TotalDuration, ms, dc.tags, 1.0)
}

// UpdateRunDuration updates the internal counter of how long the last run took.
func (dc *DatadogCollector) UpdateRunDuration(runDuration time.Duration) {
	ms := float64(runDuration.Nanoseconds() / 1000000)
	dc.client.TimeInMilliseconds(DM_RunDuration, ms, dc.tags, 1.0)
}

// Reset is a noop operation in this collector.
func (dc *DatadogCollector) Reset() {}
