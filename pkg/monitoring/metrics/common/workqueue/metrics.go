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

var (
	workqueueMetrics = []operatormetrics.Metric{
		depth,
		adds,
		latency,
		workDuration,
		retries,
		longestRunningProcessor,
		unfinishedWork,
	}

	depth = operatormetrics.NewGaugeVec(
		operatormetrics.MetricOpts{
			Name: "kubevirt_workqueue_depth",
			Help: "Current depth of workqueue",
		},
		[]string{"name"},
	)

	adds = operatormetrics.NewCounterVec(
		operatormetrics.MetricOpts{
			Name: "kubevirt_workqueue_adds_total",
			Help: "Total number of adds handled by workqueue",
		},
		[]string{"name"},
	)

	latency = operatormetrics.NewHistogramVec(
		operatormetrics.MetricOpts{
			Name: "kubevirt_workqueue_queue_duration_seconds",
			Help: "How long an item stays in workqueue before being requested.",
		},
		prometheus.HistogramOpts{
			Buckets: prometheus.ExponentialBuckets(10e-9, 10, 10),
		},
		[]string{"name"},
	)

	workDuration = operatormetrics.NewHistogramVec(
		operatormetrics.MetricOpts{
			Name: "kubevirt_workqueue_work_duration_seconds",
			Help: "How long in seconds processing an item from workqueue takes.",
		},
		prometheus.HistogramOpts{
			Buckets: prometheus.ExponentialBuckets(10e-9, 10, 10),
		},
		[]string{"name"},
	)

	retries = operatormetrics.NewCounterVec(
		operatormetrics.MetricOpts{
			Name: "kubevirt_workqueue_retries_total",
			Help: "Total number of retries handled by workqueue",
		},
		[]string{"name"},
	)

	longestRunningProcessor = operatormetrics.NewGaugeVec(
		operatormetrics.MetricOpts{
			Name: "kubevirt_workqueue_longest_running_processor_seconds",
			Help: "How many seconds has the longest running processor for workqueue been running.",
		},
		[]string{"name"},
	)

	unfinishedWork = operatormetrics.NewGaugeVec(
		operatormetrics.MetricOpts{
			Name: "kubevirt_workqueue_unfinished_work_seconds",
			Help: "How many seconds of work has done that is in progress and hasn't " +
				"been observed by work_duration. Large values indicate stuck " +
				"threads. One can deduce the number of stuck threads by observing " +
				"the rate at which this increases.",
		},
		[]string{"name"},
	)
)

type Provider struct{}

func init() {
	k8sworkqueue.SetProvider(Provider{})
}

func SetupMetrics() error {
	return operatormetrics.RegisterMetrics(workqueueMetrics)
}

func NewPrometheusMetricsProvider() Provider {
	return Provider{}
}

func (Provider) NewDepthMetric(name string) k8sworkqueue.GaugeMetric {
	return depth.WithLabelValues(name)
}

func (Provider) NewAddsMetric(name string) k8sworkqueue.CounterMetric {
	return adds.WithLabelValues(name)
}

func (Provider) NewLatencyMetric(name string) k8sworkqueue.HistogramMetric {
	return latency.WithLabelValues(name)
}

func (Provider) NewWorkDurationMetric(name string) k8sworkqueue.HistogramMetric {
	return workDuration.WithLabelValues(name)
}

func (Provider) NewRetriesMetric(name string) k8sworkqueue.CounterMetric {
	return retries.WithLabelValues(name)
}

func (Provider) NewLongestRunningProcessorSecondsMetric(name string) k8sworkqueue.SettableGaugeMetric {
	return longestRunningProcessor.WithLabelValues(name)
}

func (Provider) NewUnfinishedWorkSecondsMetric(name string) k8sworkqueue.SettableGaugeMetric {
	return unfinishedWork.WithLabelValues(name)
}
