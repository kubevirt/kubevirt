/*
Copyright 2016 The Kubernetes Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package prometheus

import (
	"github.com/prometheus/client_golang/prometheus"
	"k8s.io/client-go/util/workqueue"
)

// Package prometheus sets the workqueue DefaultMetricsFactory to produce
// prometheus metrics. To use this package, you just have to import it.

func init() {
	workqueue.SetProvider(prometheusMetricsProvider{})
}

// Metrics namespace, subsystem and keys used by the workqueue.
const (
	WorkQueueNamespace         = "kubevirt"
	WorkQueueSubsystem         = "workqueue"
	DepthKey                   = "depth"
	AddsKey                    = "adds_total"
	QueueLatencyKey            = "queue_duration_seconds"
	WorkDurationKey            = "work_duration_seconds"
	UnfinishedWorkKey          = "unfinished_work_seconds"
	LongestRunningProcessorKey = "longest_running_processor_seconds"
	RetriesKey                 = "retries_total"
)

type prometheusMetricsProvider struct{}

func (_ prometheusMetricsProvider) NewDepthMetric(name string) workqueue.GaugeMetric {
	depth := prometheus.NewGauge(prometheus.GaugeOpts{
		Namespace:   WorkQueueNamespace,
		Subsystem:   WorkQueueSubsystem,
		Name:        DepthKey,
		Help:        "Current depth of workqueue",
		ConstLabels: prometheus.Labels{"name": name},
	})
	prometheus.Register(depth)
	return depth
}

func (_ prometheusMetricsProvider) NewAddsMetric(name string) workqueue.CounterMetric {
	adds := prometheus.NewCounter(prometheus.CounterOpts{
		Namespace:   WorkQueueNamespace,
		Subsystem:   WorkQueueSubsystem,
		Name:        AddsKey,
		Help:        "Total number of adds handled by workqueue",
		ConstLabels: prometheus.Labels{"name": name},
	})
	prometheus.Register(adds)
	return adds
}

func (_ prometheusMetricsProvider) NewLatencyMetric(name string) workqueue.HistogramMetric {
	latency := prometheus.NewHistogram(prometheus.HistogramOpts{
		Namespace:   WorkQueueNamespace,
		Subsystem:   WorkQueueSubsystem,
		Name:        QueueLatencyKey,
		Help:        "How long an item stays in workqueue before being requested.",
		ConstLabels: prometheus.Labels{"name": name},
		Buckets:     prometheus.ExponentialBuckets(10e-9, 10, 10),
	})
	prometheus.Register(latency)
	return latency
}

func (_ prometheusMetricsProvider) NewWorkDurationMetric(name string) workqueue.HistogramMetric {
	workDuration := prometheus.NewHistogram(prometheus.HistogramOpts{
		Namespace:   WorkQueueNamespace,
		Subsystem:   WorkQueueSubsystem,
		Name:        WorkDurationKey,
		Help:        "How long in seconds processing an item from workqueue takes.",
		ConstLabels: prometheus.Labels{"name": name},
		Buckets:     prometheus.ExponentialBuckets(10e-9, 10, 10),
	})
	prometheus.Register(workDuration)
	return workDuration
}

func (_ prometheusMetricsProvider) NewRetriesMetric(name string) workqueue.CounterMetric {
	retries := prometheus.NewCounter(prometheus.CounterOpts{
		Namespace:   WorkQueueNamespace,
		Subsystem:   WorkQueueSubsystem,
		Name:        RetriesKey,
		Help:        "Total number of retries handled by workqueue",
		ConstLabels: prometheus.Labels{"name": name},
	})
	prometheus.Register(retries)
	return retries
}

func (_ prometheusMetricsProvider) NewLongestRunningProcessorSecondsMetric(name string) workqueue.SettableGaugeMetric {
	longestRunningProcessor := prometheus.NewGauge(prometheus.GaugeOpts{
		Namespace: WorkQueueNamespace,
		Subsystem: WorkQueueSubsystem,
		Name:      LongestRunningProcessorKey,
		Help: "How many seconds has the longest running " +
			"processor for workqueue been running.",
		ConstLabels: prometheus.Labels{"name": name},
	})
	prometheus.Register(longestRunningProcessor)
	return longestRunningProcessor
}

func (_ prometheusMetricsProvider) NewUnfinishedWorkSecondsMetric(name string) workqueue.SettableGaugeMetric {
	unfinishedWork := prometheus.NewGauge(prometheus.GaugeOpts{
		Namespace: WorkQueueNamespace,
		Subsystem: WorkQueueSubsystem,
		Name:      UnfinishedWorkKey,
		Help: "How many seconds of work has done that " +
			"is in progress and hasn't been observed by work_duration. Large " +
			"values indicate stuck threads. One can deduce the number of stuck " +
			"threads by observing the rate at which this increases.",
		ConstLabels: prometheus.Labels{"name": name},
	})
	prometheus.Register(unfinishedWork)
	return unfinishedWork
}
