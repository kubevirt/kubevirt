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

package client

import (
	"github.com/machadovilaca/operator-observability/pkg/operatormetrics"
	"github.com/prometheus/client_golang/prometheus"
)

var (
	restMetrics = []operatormetrics.Metric{
		requestLatency,
		rateLimiterLatency,
		requestResult,
	}

	// requestLatency is a Prometheus Summary metric type partitioned by
	// "verb" and "url" labels. It is used for the rest client latency metrics.
	requestLatency = operatormetrics.NewHistogramVec(
		operatormetrics.MetricOpts{
			Name: "kubevirt_rest_client_request_latency_seconds",
			Help: "Request latency in seconds. Broken down by verb and URL.",
		},
		prometheus.HistogramOpts{
			// Buckets based on Kubernetes apiserver_request_duration_seconds.
			// See https://github.com/kubernetes/kubernetes/pull/73638 for the discussion.
			Buckets: []float64{
				0.005, 0.025, 0.05, 0.1, 0.2, 0.4, 0.6, 0.8, 1.0, 1.25, 1.5, 2, 3,
				4, 5, 6, 8, 10, 15, 20, 30, 45, 60,
			},
		},
		[]string{"verb", "url"},
	)

	rateLimiterLatency = operatormetrics.NewHistogramVec(
		operatormetrics.MetricOpts{
			Name: "kubevirt_rest_client_rate_limiter_duration_seconds",
			Help: "Client side rate limiter latency in seconds. Broken down by verb and URL.",
		},
		prometheus.HistogramOpts{
			Buckets: prometheus.ExponentialBuckets(0.001, 2, 10),
		},
		[]string{"verb", "url"},
	)

	requestResult = operatormetrics.NewCounterVec(
		operatormetrics.MetricOpts{
			Name: "kubevirt_rest_client_requests_total",
			Help: "Number of HTTP requests, partitioned by status code, method, and host.",
		},
		[]string{"code", "method", "host", "resource", "verb"},
	)
)
