/*
Copyright The KubeVirt Authors.
SPDX-License-Identifier: Apache-2.0
*/

package client

import (
	"context"
	"net/url"
	"time"

	"github.com/rhobs/operator-observability-toolkit/pkg/operatormetrics"
)

type latencyAdapter struct {
	m *operatormetrics.HistogramVec
}

func (l *latencyAdapter) Observe(_ context.Context, verb string, u url.URL, latency time.Duration) {
	l.m.WithLabelValues(getVerbFromHTTPVerb(u, verb), u.String()).Observe(latency.Seconds())
}
