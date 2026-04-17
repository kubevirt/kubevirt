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
	"context"
	"strconv"
	"time"

	"github.com/deckhouse/kube-api-rewriter/pkg/labels"
)

type ProxyMetrics struct {
	provider         MetricsProvider
	name             string
	resource         string
	method           string
	watch            string
	decision         string
	side             string
	toTargetAction   string
	fromTargetAction string
	status           string
}

func NewProxyMetrics(ctx context.Context, provider MetricsProvider) *ProxyMetrics {
	return &ProxyMetrics{
		provider:         provider,
		name:             labels.NameFromContext(ctx),
		resource:         labels.ResourceFromContext(ctx),
		method:           labels.MethodFromContext(ctx),
		watch:            labels.WatchFromContext(ctx),
		decision:         labels.DecisionFromContext(ctx),
		toTargetAction:   labels.ToTargetActionFromContext(ctx),
		fromTargetAction: labels.FromTargetActionFromContext(ctx),
		status:           labels.StatusFromContext(ctx),
	}
}

func WatchLabel(isWatch bool) string {
	if isWatch {
		return watchRequest
	}
	return regularRequest
}

func (p *ProxyMetrics) GotClientRequest() {
	p.provider.NewClientRequestsTotal(p.name, p.resource, p.method, p.watch, p.decision).Inc()
}

func (p *ProxyMetrics) TargetResponseSuccess(decision string) {
	p.provider.NewTargetResponsesTotal(p.name, p.resource, p.method, p.watch, decision, p.status, noError).Inc()
}

func (p *ProxyMetrics) TargetResponseError() {
	p.provider.NewTargetResponsesTotal(p.name, p.resource, p.method, p.watch, "", p.status, errorOccurred).Inc()
}

func (p *ProxyMetrics) TargetResponseInvalidJSON(status int) {
	p.provider.NewTargetResponseInvalidJSONTotal(p.name, p.resource, p.method, p.watch, strconv.Itoa(status))
}

func (p *ProxyMetrics) RequestHandleSuccess() {
	p.provider.NewRequestsHandledTotal(p.name, p.resource, p.method, p.watch, p.decision, p.status, noError).Inc()
}
func (p *ProxyMetrics) RequestHandleError() {
	p.provider.NewRequestsHandledTotal(p.name, p.resource, p.method, p.watch, p.decision, p.status, errorOccurred).Inc()
}

func (p *ProxyMetrics) RequestDuration(dur time.Duration) {
	p.provider.NewRequestsHandlingSeconds(p.name, p.resource, p.method, p.watch, p.decision, p.status).Observe(dur.Seconds())
}

func (p *ProxyMetrics) TargetResponseRewriteError() {
	p.provider.NewRewritesTotal(p.name, p.resource, p.method, p.watch, targetSide, p.fromTargetAction, errorOccurred).Inc()
}

func (p *ProxyMetrics) TargetResponseRewriteSuccess() {
	p.provider.NewRewritesTotal(p.name, p.resource, p.method, p.watch, targetSide, p.fromTargetAction, noError).Inc()
}

func (p *ProxyMetrics) ClientRequestRewriteError() {
	p.provider.NewRewritesTotal(p.name, p.resource, p.method, p.watch, clientSide, p.toTargetAction, errorOccurred).Inc()
}

func (p *ProxyMetrics) ClientRequestRewriteSuccess() {
	p.provider.NewRewritesTotal(p.name, p.resource, p.method, p.watch, clientSide, p.toTargetAction, noError).Inc()
}

func (p *ProxyMetrics) ClientRequestRewriteDuration(dur time.Duration) {
	p.provider.NewRewritesDurationSeconds(p.name, p.resource, p.method, p.watch, clientSide, p.toTargetAction).Observe(dur.Seconds())
}

func (p *ProxyMetrics) TargetResponseRewriteDuration(dur time.Duration) {
	p.provider.NewRewritesDurationSeconds(p.name, p.resource, p.method, p.watch, targetSide, p.fromTargetAction).Observe(dur.Seconds())
}

func (p *ProxyMetrics) FromClientBytesAdd(decision string, count int) {
	p.provider.NewFromClientBytesTotal(p.name, p.resource, p.method, p.watch, decision).Add(float64(count))
}

func (p *ProxyMetrics) ToTargetBytesAdd(decision string, count int) {
	p.provider.NewToTargetBytesTotal(p.name, p.resource, p.method, p.watch, decision).Add(float64(count))
}

func (p *ProxyMetrics) FromTargetBytesAdd(count int) {
	p.provider.NewFromTargetBytesTotal(p.name, p.resource, p.method, p.watch, p.decision).Add(float64(count))
}

func (p *ProxyMetrics) ToClientBytesAdd(count int) {
	p.provider.NewToClientBytesTotal(p.name, p.resource, p.method, p.watch, p.decision).Add(float64(count))
}
