// Plugins allows users to operate on statistics recorded for each circuit operation.
// Plugins should be careful to be lightweight as they will be called frequently.
package plugins

import (
	"net"
	"strings"
	"time"

	"github.com/afex/hystrix-go/hystrix/metric_collector"
	"github.com/rcrowley/go-metrics"
)

var makeTimerFunc = func() interface{} { return metrics.NewTimer() }
var makeCounterFunc = func() interface{} { return metrics.NewCounter() }

// GraphiteCollector fulfills the metricCollector interface allowing users to ship circuit
// stats to a graphite backend. To use users must call InitializeGraphiteCollector before
// circuits are started. Then register NewGraphiteCollector with metricCollector.Registry.Register(NewGraphiteCollector).
//
// This Collector uses github.com/rcrowley/go-metrics for aggregation. See that repo for more details
// on how metrics are aggregated and expressed in graphite.
type GraphiteCollector struct {
	attemptsPrefix          string
	errorsPrefix            string
	successesPrefix         string
	failuresPrefix          string
	rejectsPrefix           string
	shortCircuitsPrefix     string
	timeoutsPrefix          string
	fallbackSuccessesPrefix string
	fallbackFailuresPrefix  string
	totalDurationPrefix     string
	runDurationPrefix       string
}

// GraphiteCollectorConfig provides configuration that the graphite client will need.
type GraphiteCollectorConfig struct {
	// GraphiteAddr is the tcp address of the graphite server
	GraphiteAddr *net.TCPAddr
	// Prefix is the prefix that will be prepended to all metrics sent from this collector.
	Prefix string
	// TickInterval spcifies the period that this collector will send metrics to the server.
	TickInterval time.Duration
}

// InitializeGraphiteCollector creates the connection to the graphite server
// and should be called before any metrics are recorded.
func InitializeGraphiteCollector(config *GraphiteCollectorConfig) {
	go metrics.Graphite(metrics.DefaultRegistry, config.TickInterval, config.Prefix, config.GraphiteAddr)
}

// NewGraphiteCollector creates a collector for a specific circuit. The
// prefix given to this circuit will be {config.Prefix}.{circuit_name}.{metric}.
// Circuits with "/" in their names will have them replaced with ".".
func NewGraphiteCollector(name string) metricCollector.MetricCollector {
	name = strings.Replace(name, "/", "-", -1)
	name = strings.Replace(name, ":", "-", -1)
	name = strings.Replace(name, ".", "-", -1)
	return &GraphiteCollector{
		attemptsPrefix:          name + ".attempts",
		errorsPrefix:            name + ".errors",
		successesPrefix:         name + ".successes",
		failuresPrefix:          name + ".failures",
		rejectsPrefix:           name + ".rejects",
		shortCircuitsPrefix:     name + ".shortCircuits",
		timeoutsPrefix:          name + ".timeouts",
		fallbackSuccessesPrefix: name + ".fallbackSuccesses",
		fallbackFailuresPrefix:  name + ".fallbackFailures",
		totalDurationPrefix:     name + ".totalDuration",
		runDurationPrefix:       name + ".runDuration",
	}
}

func (g *GraphiteCollector) incrementCounterMetric(prefix string) {
	c, ok := metrics.GetOrRegister(prefix, makeCounterFunc).(metrics.Counter)
	if !ok {
		return
	}
	c.Inc(1)
}

func (g *GraphiteCollector) updateTimerMetric(prefix string, dur time.Duration) {
	c, ok := metrics.GetOrRegister(prefix, makeTimerFunc).(metrics.Timer)
	if !ok {
		return
	}
	c.Update(dur)
}

// IncrementAttempts increments the number of calls to this circuit.
// This registers as a counter in the graphite collector.
func (g *GraphiteCollector) IncrementAttempts() {
	g.incrementCounterMetric(g.attemptsPrefix)
}

// IncrementErrors increments the number of unsuccessful attempts.
// Attempts minus Errors will equal successes within a time range.
// Errors are any result from an attempt that is not a success.
// This registers as a counter in the graphite collector.
func (g *GraphiteCollector) IncrementErrors() {
	g.incrementCounterMetric(g.errorsPrefix)

}

// IncrementSuccesses increments the number of requests that succeed.
// This registers as a counter in the graphite collector.
func (g *GraphiteCollector) IncrementSuccesses() {
	g.incrementCounterMetric(g.successesPrefix)

}

// IncrementFailures increments the number of requests that fail.
// This registers as a counter in the graphite collector.
func (g *GraphiteCollector) IncrementFailures() {
	g.incrementCounterMetric(g.failuresPrefix)
}

// IncrementRejects increments the number of requests that are rejected.
// This registers as a counter in the graphite collector.
func (g *GraphiteCollector) IncrementRejects() {
	g.incrementCounterMetric(g.rejectsPrefix)
}

// IncrementShortCircuits increments the number of requests that short circuited due to the circuit being open.
// This registers as a counter in the graphite collector.
func (g *GraphiteCollector) IncrementShortCircuits() {
	g.incrementCounterMetric(g.shortCircuitsPrefix)
}

// IncrementTimeouts increments the number of timeouts that occurred in the circuit breaker.
// This registers as a counter in the graphite collector.
func (g *GraphiteCollector) IncrementTimeouts() {
	g.incrementCounterMetric(g.timeoutsPrefix)
}

// IncrementFallbackSuccesses increments the number of successes that occurred during the execution of the fallback function.
// This registers as a counter in the graphite collector.
func (g *GraphiteCollector) IncrementFallbackSuccesses() {
	g.incrementCounterMetric(g.fallbackSuccessesPrefix)
}

// IncrementFallbackFailures increments the number of failures that occurred during the execution of the fallback function.
// This registers as a counter in the graphite collector.
func (g *GraphiteCollector) IncrementFallbackFailures() {
	g.incrementCounterMetric(g.fallbackFailuresPrefix)
}

// UpdateTotalDuration updates the internal counter of how long we've run for.
// This registers as a timer in the graphite collector.
func (g *GraphiteCollector) UpdateTotalDuration(timeSinceStart time.Duration) {
	g.updateTimerMetric(g.totalDurationPrefix, timeSinceStart)
}

// UpdateRunDuration updates the internal counter of how long the last run took.
// This registers as a timer in the graphite collector.
func (g *GraphiteCollector) UpdateRunDuration(runDuration time.Duration) {
	g.updateTimerMetric(g.runDurationPrefix, runDuration)
}

// Reset is a noop operation in this collector.
func (g *GraphiteCollector) Reset() {}
