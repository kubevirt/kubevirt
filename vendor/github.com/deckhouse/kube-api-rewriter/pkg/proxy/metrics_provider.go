/*
Copyright 2024 Flant JSC

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

package proxy

import (
	"github.com/prometheus/client_golang/prometheus"

	"github.com/deckhouse/kube-api-rewriter/pkg/monitoring/metrics"
)

var Subsystem = defaultSubsystem

const (
	defaultSubsystem = "kube_api_rewriter"

	clientRequestsTotalName            = "client_requests_total"
	targetResponsesTotalName           = "target_responses_total"
	targetResponseInvalidJSONTotalName = "target_response_invalid_json_total"

	requestsHandledTotalName           = "requests_handled_total"
	requestHandlingDurationSecondsName = "request_handling_duration_seconds"

	rewritesTotalName          = "rewrites_total"
	rewriteDurationSecondsName = "rewrite_duration_seconds"

	fromClientBytesName = "from_client_bytes_total"
	toTargetBytesName   = "to_target_bytes_total"
	fromTargetBytesName = "from_target_bytes_total"
	toClientBytesName   = "to_client_bytes_total"

	nameLabel      = "name"
	resourceLabel  = "resource"
	methodLabel    = "method"
	watchLabel     = "watch"
	decisionLabel  = "decision"
	sideLabel      = "side"
	operationLabel = "operation"
	statusLabel    = "status"
	errorLabel     = "error"

	watchRequest   = "1"
	regularRequest = "0"

	decisionRewrite = "rewrite"
	decisionPass    = "pass"

	targetSide = "target"
	clientSide = "client"

	operationRename  = "rename"
	operationRestore = "restore"

	errorOccurred = "1"
	noError       = "0"
)

var (
	clientRequestsTotal = prometheus.NewCounterVec(prometheus.CounterOpts{
		Subsystem: Subsystem,
		Name:      clientRequestsTotalName,
		Help:      "Total number of received client requests",
	}, []string{nameLabel, resourceLabel, methodLabel, watchLabel, decisionLabel})

	targetResponsesTotal = prometheus.NewCounterVec(prometheus.CounterOpts{
		Subsystem: Subsystem,
		Name:      targetResponsesTotalName,
		Help:      "Total number of responses from the target",
	}, []string{nameLabel, resourceLabel, methodLabel, watchLabel, decisionLabel, statusLabel, errorLabel})

	targetResponseInvalidJSONTotal = prometheus.NewCounterVec(prometheus.CounterOpts{
		Subsystem: Subsystem,
		Name:      targetResponseInvalidJSONTotalName,
		Help:      "Total target responses with invalid JSON",
	}, []string{nameLabel, resourceLabel, methodLabel, watchLabel, statusLabel})

	requestsHandledTotal = prometheus.NewCounterVec(prometheus.CounterOpts{
		Subsystem: Subsystem,
		Name:      requestsHandledTotalName,
		Help:      "Total number of requests handled by the proxy instance",
	}, []string{nameLabel, resourceLabel, methodLabel, watchLabel, decisionLabel, statusLabel, errorLabel})

	requestHandlingDurationSeconds = prometheus.NewHistogramVec(prometheus.HistogramOpts{
		Subsystem: Subsystem,
		Name:      requestHandlingDurationSecondsName,
		Help:      "Duration of request handling for non-watching and watch event handling for watch requests",
		Buckets: []float64{
			0.0,
			0.001, 0.002, 0.005, // 1, 2, 5 milliseconds
			0.01, 0.02, 0.05, // 10, 20, 50 milliseconds
			0.1, 0.2, 0.5, // 100, 200, 500 milliseconds
			1, 2, 5, // 1, 2, 5 seconds
		},
	}, []string{nameLabel, resourceLabel, methodLabel, watchLabel, decisionLabel, statusLabel})

	rewritesTotal = prometheus.NewCounterVec(prometheus.CounterOpts{
		Subsystem: Subsystem,
		Name:      rewritesTotalName,
		Help:      "Total rewrites executed by the proxy instance",
	}, []string{nameLabel, resourceLabel, methodLabel, watchLabel, sideLabel, operationLabel, errorLabel})

	rewritesDurationSeconds = prometheus.NewHistogramVec(prometheus.HistogramOpts{
		Subsystem: Subsystem,
		Name:      rewriteDurationSecondsName,
		Help:      "Duration of rewrite operations",
		Buckets: []float64{
			0.0,
			0.001, 0.002, 0.005, // 1, 2, 5 milliseconds
			0.01, 0.02, 0.05, // 10, 20, 50 milliseconds
			0.1, 0.2, 0.5, // 100, 200, 500 milliseconds
			1, 2, 5, // 1, 2, 5 seconds
		},
	}, []string{nameLabel, resourceLabel, methodLabel, watchLabel, sideLabel, operationLabel})

	fromClientBytes = prometheus.NewCounterVec(prometheus.CounterOpts{
		Subsystem: Subsystem,
		Name:      fromClientBytesName,
		Help:      "Total bytes received from the client",
	}, []string{nameLabel, resourceLabel, methodLabel, watchLabel, decisionLabel})

	toTargetBytes = prometheus.NewCounterVec(prometheus.CounterOpts{
		Subsystem: Subsystem,
		Name:      toTargetBytesName,
		Help:      "Total bytes transferred to the target",
	}, []string{nameLabel, resourceLabel, methodLabel, watchLabel, decisionLabel})

	fromTargetBytes = prometheus.NewCounterVec(prometheus.CounterOpts{
		Subsystem: Subsystem,
		Name:      fromTargetBytesName,
		Help:      "Total bytes received from the target",
	}, []string{nameLabel, resourceLabel, methodLabel, watchLabel, decisionLabel})

	toClientBytes = prometheus.NewCounterVec(prometheus.CounterOpts{
		Subsystem: Subsystem,
		Name:      toClientBytesName,
		Help:      "Total bytes transferred back to the client",
	}, []string{nameLabel, resourceLabel, methodLabel, watchLabel, decisionLabel})
)

func RegisterMetrics() {
	metrics.Registry.MustRegister(
		clientRequestsTotal,
		targetResponsesTotal,
		targetResponseInvalidJSONTotal,
		requestsHandledTotal,
		requestHandlingDurationSeconds,
		fromClientBytes,
		toTargetBytes,
		fromTargetBytes,
		toClientBytes,
		rewritesTotal,
		rewritesDurationSeconds,
	)
}

type MetricsProvider interface {
	NewClientRequestsTotal(name, resource, method, watch, decision string) prometheus.Counter
	NewTargetResponsesTotal(name, resource, method, watch, decision, status, error string) prometheus.Counter
	NewTargetResponseInvalidJSONTotal(name, resource, method, watch, status string) prometheus.Counter
	NewRequestsHandledTotal(name, resource, method, watch, decision, status, error string) prometheus.Counter
	NewRequestsHandlingSeconds(name, resource, method, watch, decision, status string) prometheus.Observer
	NewRewritesTotal(name, resource, method, watch, side, operation, error string) prometheus.Counter
	NewRewritesDurationSeconds(name, resource, method, watch, side, operation string) prometheus.Observer
	NewFromClientBytesTotal(name, resource, method, watch, decision string) prometheus.Counter
	NewToTargetBytesTotal(name, resource, method, watch, decision string) prometheus.Counter
	NewFromTargetBytesTotal(name, resource, method, watch, decision string) prometheus.Counter
	NewToClientBytesTotal(name, resource, method, watch, decision string) prometheus.Counter
}

func NewMetricsProvider() MetricsProvider {
	return &proxyMetricsProvider{}
}

type proxyMetricsProvider struct{}

func (p *proxyMetricsProvider) NewClientRequestsTotal(name, resource, method, watch, decision string) prometheus.Counter {
	return clientRequestsTotal.WithLabelValues(name, resource, method, watch, decision)
}

func (p *proxyMetricsProvider) NewTargetResponsesTotal(name, resource, method, watch, decision, status, error string) prometheus.Counter {
	return targetResponsesTotal.WithLabelValues(name, resource, method, watch, decision, status, error)
}

func (p *proxyMetricsProvider) NewTargetResponseInvalidJSONTotal(name, resource, method, watch, status string) prometheus.Counter {
	return targetResponseInvalidJSONTotal.WithLabelValues(name, resource, method, watch, status)
}

func (p *proxyMetricsProvider) NewRequestsHandledTotal(name, resource, method, watch, decision, status, error string) prometheus.Counter {
	return requestsHandledTotal.WithLabelValues(name, resource, method, watch, decision, status, error)
}

func (p *proxyMetricsProvider) NewRequestsHandlingSeconds(name, resource, method, watch, decision, status string) prometheus.Observer {
	return requestHandlingDurationSeconds.WithLabelValues(name, resource, method, watch, decision, status)
}

func (p *proxyMetricsProvider) NewRewritesTotal(name, resource, method, watch, side, operation, error string) prometheus.Counter {
	return rewritesTotal.WithLabelValues(name, resource, method, watch, side, operation, error)
}

func (p *proxyMetricsProvider) NewRewritesDurationSeconds(name, resource, method, watch, side, operation string) prometheus.Observer {
	return rewritesDurationSeconds.WithLabelValues(name, resource, method, watch, side, operation)
}

func (p *proxyMetricsProvider) NewFromClientBytesTotal(name, resource, method, watch, decision string) prometheus.Counter {
	return fromClientBytes.WithLabelValues(name, resource, method, watch, decision)
}

func (p *proxyMetricsProvider) NewToTargetBytesTotal(name, resource, method, watch, decision string) prometheus.Counter {
	return toTargetBytes.WithLabelValues(name, resource, method, watch, decision)
}

func (p *proxyMetricsProvider) NewFromTargetBytesTotal(name, resource, method, watch, decision string) prometheus.Counter {
	return fromTargetBytes.WithLabelValues(name, resource, method, watch, decision)
}

func (p *proxyMetricsProvider) NewToClientBytesTotal(name, resource, method, watch, decision string) prometheus.Counter {
	return toClientBytes.WithLabelValues(name, resource, method, watch, decision)
}

func NoopMetricsProvider() MetricsProvider {
	return noopMetricsProvider{}
}

type noopMetric struct {
	prometheus.Counter
	prometheus.Observer
}

type noopMetricsProvider struct{}

func (_ noopMetricsProvider) NewClientRequestsTotal(name, resource, method, watch, decision string) prometheus.Counter {
	return noopMetric{}
}
func (_ noopMetricsProvider) NewTargetResponsesTotal(name, resource, method, watch, decision, status, error string) prometheus.Counter {
	return noopMetric{}
}
func (_ noopMetricsProvider) NewTargetResponseInvalidJSONTotal(name, resource, method, watch, status string) prometheus.Counter {
	return noopMetric{}
}
func (_ noopMetricsProvider) NewRequestsHandledTotal(name, resource, method, watch, decision, status, error string) prometheus.Counter {
	return noopMetric{}
}
func (_ noopMetricsProvider) NewRequestsHandlingSeconds(name, resource, method, watch, decision, status string) prometheus.Observer {
	return noopMetric{}
}
func (_ noopMetricsProvider) NewRewritesTotal(name, resource, method, watch, side, operation, error string) prometheus.Counter {
	return noopMetric{}
}
func (_ noopMetricsProvider) NewRewritesDurationSeconds(name, resource, method, watch, side, operation string) prometheus.Observer {
	return noopMetric{}
}
func (_ noopMetricsProvider) NewFromClientBytesTotal(name, resource, method, watch, decision string) prometheus.Counter {
	return noopMetric{}
}
func (_ noopMetricsProvider) NewToTargetBytesTotal(name, resource, method, watch, decision string) prometheus.Counter {
	return noopMetric{}
}
func (_ noopMetricsProvider) NewFromTargetBytesTotal(name, resource, method, watch, decision string) prometheus.Counter {
	return noopMetric{}
}
func (_ noopMetricsProvider) NewToClientBytesTotal(name, resource, method, watch, decision string) prometheus.Counter {
	return noopMetric{}
}
