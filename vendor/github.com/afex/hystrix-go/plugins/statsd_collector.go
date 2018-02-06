package plugins

import (
	"log"
	"strings"
	"time"

	"github.com/afex/hystrix-go/hystrix/metric_collector"
	"github.com/cactus/go-statsd-client/statsd"
)

// StatsdCollector fulfills the metricCollector interface allowing users to ship circuit
// stats to a Statsd backend. To use users must call InitializeStatsdCollector before
// circuits are started. Then register NewStatsdCollector with metricCollector.Registry.Register(NewStatsdCollector).
//
// This Collector uses https://github.com/cactus/go-statsd-client/ for transport.
type StatsdCollector struct {
	client                  statsd.Statter
	circuitOpenPrefix       string
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
	sampleRate              float32
}

type StatsdCollectorClient struct {
	client     statsd.Statter
	sampleRate float32
}

// https://github.com/etsy/statsd/blob/master/docs/metric_types.md#multi-metric-packets
const (
	WANStatsdFlushBytes     = 512
	LANStatsdFlushBytes     = 1432
	GigabitStatsdFlushBytes = 8932
)

// StatsdCollectorConfig provides configuration that the Statsd client will need.
type StatsdCollectorConfig struct {
	// StatsdAddr is the tcp address of the Statsd server
	StatsdAddr string
	// Prefix is the prefix that will be prepended to all metrics sent from this collector.
	Prefix string
	// StatsdSampleRate sets statsd sampling. If 0, defaults to 1.0. (no sampling)
	SampleRate float32
	// FlushBytes sets message size for statsd packets. If 0, defaults to LANFlushSize.
	FlushBytes int
}

// InitializeStatsdCollector creates the connection to the Statsd server
// and should be called before any metrics are recorded.
//
// Users should ensure to call Close() on the client.
func InitializeStatsdCollector(config *StatsdCollectorConfig) (*StatsdCollectorClient, error) {
	flushBytes := config.FlushBytes
	if flushBytes == 0 {
		flushBytes = LANStatsdFlushBytes
	}

	sampleRate := config.SampleRate
	if sampleRate == 0 {
		sampleRate = 1
	}

	c, err := statsd.NewBufferedClient(config.StatsdAddr, config.Prefix, 1*time.Second, flushBytes)
	if err != nil {
		log.Printf("Could not initiale buffered client: %s. Falling back to a Noop Statsd client", err)
		c, _ = statsd.NewNoopClient()
	}
	return &StatsdCollectorClient{
		client:     c,
		sampleRate: sampleRate,
	}, err
}

// NewStatsdCollector creates a collector for a specific circuit. The
// prefix given to this circuit will be {config.Prefix}.{circuit_name}.{metric}.
// Circuits with "/" in their names will have them replaced with ".".
func (s *StatsdCollectorClient) NewStatsdCollector(name string) metricCollector.MetricCollector {
	if s.client == nil {
		log.Fatalf("Statsd client must be initialized before circuits are created.")
	}
	name = strings.Replace(name, "/", "-", -1)
	name = strings.Replace(name, ":", "-", -1)
	name = strings.Replace(name, ".", "-", -1)
	return &StatsdCollector{
		client:                  s.client,
		circuitOpenPrefix:       name + ".circuitOpen",
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
		sampleRate:              s.sampleRate,
	}
}

func (g *StatsdCollector) setGauge(prefix string, value int64) {
	err := g.client.Gauge(prefix, value, g.sampleRate)
	if err != nil {
		log.Printf("Error sending statsd metrics %s", prefix)
	}
}

func (g *StatsdCollector) incrementCounterMetric(prefix string) {
	err := g.client.Inc(prefix, 1, g.sampleRate)
	if err != nil {
		log.Printf("Error sending statsd metrics %s", prefix)
	}
}

func (g *StatsdCollector) updateTimerMetric(prefix string, dur time.Duration) {
	err := g.client.TimingDuration(prefix, dur, g.sampleRate)
	if err != nil {
		log.Printf("Error sending statsd metrics %s", prefix)
	}
}

// IncrementAttempts increments the number of calls to this circuit.
// This registers as a counter in the Statsd collector.
func (g *StatsdCollector) IncrementAttempts() {
	g.incrementCounterMetric(g.attemptsPrefix)
}

// IncrementErrors increments the number of unsuccessful attempts.
// Attempts minus Errors will equal successes within a time range.
// Errors are any result from an attempt that is not a success.
// This registers as a counter in the Statsd collector.
func (g *StatsdCollector) IncrementErrors() {
	g.incrementCounterMetric(g.errorsPrefix)

}

// IncrementSuccesses increments the number of requests that succeed.
// This registers as a counter in the Statsd collector.
func (g *StatsdCollector) IncrementSuccesses() {
	g.setGauge(g.circuitOpenPrefix, 0)
	g.incrementCounterMetric(g.successesPrefix)

}

// IncrementFailures increments the number of requests that fail.
// This registers as a counter in the Statsd collector.
func (g *StatsdCollector) IncrementFailures() {
	g.incrementCounterMetric(g.failuresPrefix)
}

// IncrementRejects increments the number of requests that are rejected.
// This registers as a counter in the Statsd collector.
func (g *StatsdCollector) IncrementRejects() {
	g.incrementCounterMetric(g.rejectsPrefix)
}

// IncrementShortCircuits increments the number of requests that short circuited due to the circuit being open.
// This registers as a counter in the Statsd collector.
func (g *StatsdCollector) IncrementShortCircuits() {
	g.setGauge(g.circuitOpenPrefix, 1)
	g.incrementCounterMetric(g.shortCircuitsPrefix)
}

// IncrementTimeouts increments the number of timeouts that occurred in the circuit breaker.
// This registers as a counter in the Statsd collector.
func (g *StatsdCollector) IncrementTimeouts() {
	g.incrementCounterMetric(g.timeoutsPrefix)
}

// IncrementFallbackSuccesses increments the number of successes that occurred during the execution of the fallback function.
// This registers as a counter in the Statsd collector.
func (g *StatsdCollector) IncrementFallbackSuccesses() {
	g.incrementCounterMetric(g.fallbackSuccessesPrefix)
}

// IncrementFallbackFailures increments the number of failures that occurred during the execution of the fallback function.
// This registers as a counter in the Statsd collector.
func (g *StatsdCollector) IncrementFallbackFailures() {
	g.incrementCounterMetric(g.fallbackFailuresPrefix)
}

// UpdateTotalDuration updates the internal counter of how long we've run for.
// This registers as a timer in the Statsd collector.
func (g *StatsdCollector) UpdateTotalDuration(timeSinceStart time.Duration) {
	g.updateTimerMetric(g.totalDurationPrefix, timeSinceStart)
}

// UpdateRunDuration updates the internal counter of how long the last run took.
// This registers as a timer in the Statsd collector.
func (g *StatsdCollector) UpdateRunDuration(runDuration time.Duration) {
	g.updateTimerMetric(g.runDurationPrefix, runDuration)
}

// Reset is a noop operation in this collector.
func (g *StatsdCollector) Reset() {}
