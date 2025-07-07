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
	"context"
	"net/url"
	"time"

	"github.com/machadovilaca/operator-observability/pkg/operatormetrics"
)

type latencyAdapter struct {
	m *operatormetrics.HistogramVec
}

func (l *latencyAdapter) Observe(_ context.Context, verb string, u url.URL, latency time.Duration) {
	l.m.WithLabelValues(getVerbFromHTTPVerb(u, verb), u.String()).Observe(latency.Seconds())
}
