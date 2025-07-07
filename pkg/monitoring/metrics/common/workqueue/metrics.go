/*
 * This file is part of the KubeVirt project
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 *
 * Copyright the KubeVirt Authors.
 */

package workqueue

import (
	"github.com/machadovilaca/operator-observability/pkg/operatormetrics"
	"github.com/prometheus/client_golang/prometheus"
	k8sworkqueue "k8s.io/client-go/util/workqueue"
)

func SetupMetrics() error {
	k8sworkqueue.SetProvider(prometheusMetricsProvider{})
	return nil
}

type prometheusMetricsProvider struct{}

func (_ prometheusMetricsProvider) NewDepthMetric(name string) k8sworkqueue.GaugeMetric {
	depth := operatormetrics.NewGauge(operatormetrics.MetricOpts{
		Name:        "kubevirt_workqueue_depth",
		Help:        "Current depth of workqueue",
		ConstLabels: prometheus.Labels{"name": name},
	})
	_ = operatormetrics.RegisterMetrics([]operatormetrics.Metric{depth})

	return depth
}

func (_ prometheusMetricsProvider) NewAddsMetric(name string) k8sworkqueue.CounterMetric {
	adds := operatormetrics.NewCounter(operatormetrics.MetricOpts{
		Name:        "kubevirt_workqueue_adds_total",
		Help:        "Total number of adds handled by workqueue",
		ConstLabels: prometheus.Labels{"name": name},
	})
	_ = operatormetrics.RegisterMetrics([]operatormetrics.Metric{adds})

	return adds
}

func (_ prometheusMetricsProvider) NewLatencyMetric(name string) k8sworkqueue.HistogramMetric {
	latency := operatormetrics.NewHistogram(
		operatormetrics.MetricOpts{
			Name:        "kubevirt_workqueue_queue_duration_seconds",
			Help:        "How long an item stays in workqueue before being requested.",
			ConstLabels: prometheus.Labels{"name": name},
		},
		prometheus.HistogramOpts{
			Buckets: prometheus.ExponentialBuckets(10e-9, 10, 10),
		},
	)
	_ = operatormetrics.RegisterMetrics([]operatormetrics.Metric{latency})

	return latency
}

func (_ prometheusMetricsProvider) NewWorkDurationMetric(name string) k8sworkqueue.HistogramMetric {
	workDuration := operatormetrics.NewHistogram(
		operatormetrics.MetricOpts{
			Name:        "kubevirt_workqueue_work_duration_seconds",
			Help:        "How long in seconds processing an item from workqueue takes.",
			ConstLabels: prometheus.Labels{"name": name},
		},
		prometheus.HistogramOpts{
			Buckets: prometheus.ExponentialBuckets(10e-9, 10, 10),
		},
	)
	_ = operatormetrics.RegisterMetrics([]operatormetrics.Metric{workDuration})

	return workDuration
}

func (_ prometheusMetricsProvider) NewRetriesMetric(name string) k8sworkqueue.CounterMetric {
	retries := operatormetrics.NewCounter(operatormetrics.MetricOpts{
		Name:        "kubevirt_workqueue_retries_total",
		Help:        "Total number of retries handled by workqueue",
		ConstLabels: prometheus.Labels{"name": name},
	})
	_ = operatormetrics.RegisterMetrics([]operatormetrics.Metric{retries})

	return retries
}

func (_ prometheusMetricsProvider) NewLongestRunningProcessorSecondsMetric(name string) k8sworkqueue.SettableGaugeMetric {
	longestRunningProcessor := operatormetrics.NewGauge(operatormetrics.MetricOpts{
		Name:        "kubevirt_workqueue_longest_running_processor_seconds",
		Help:        "How many seconds has the longest running processor for workqueue been running.",
		ConstLabels: prometheus.Labels{"name": name},
	})
	_ = operatormetrics.RegisterMetrics([]operatormetrics.Metric{longestRunningProcessor})

	return longestRunningProcessor
}

func (_ prometheusMetricsProvider) NewUnfinishedWorkSecondsMetric(name string) k8sworkqueue.SettableGaugeMetric {
	unfinishedWork := operatormetrics.NewGauge(operatormetrics.MetricOpts{
		Name: "kubevirt_workqueue_unfinished_work_seconds",
		Help: "How many seconds of work has done that is in progress and hasn't " +
			"been observed by work_duration. Large values indicate stuck " +
			"threads. One can deduce the number of stuck threads by observing " +
			"the rate at which this increases.",
		ConstLabels: prometheus.Labels{"name": name},
	})
	_ = operatormetrics.RegisterMetrics([]operatormetrics.Metric{unfinishedWork})

	return unfinishedWork
}
