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

package reflector

import (
	"github.com/machadovilaca/operator-observability/pkg/operatormetrics"
	"github.com/prometheus/client_golang/prometheus"
)

var (
	watchesMetrics = []operatormetrics.Metric{
		watchesTotal,
		shortWatchesTotal,
		watchDuration,
		itemsPerWatch,
	}

	watchesTotal = operatormetrics.NewCounterVec(
		operatormetrics.MetricOpts{
			Name: "reflector_watches_total",
			Help: "Total number of API watches done by the reflectors",
		},
		[]string{"name"},
	)

	shortWatchesTotal = operatormetrics.NewCounterVec(
		operatormetrics.MetricOpts{
			Name: "reflector_short_watches_total",
			Help: "Total number of short API watches done by the reflectors",
		},
		[]string{"name"},
	)

	watchDuration = operatormetrics.NewSummaryVec(
		operatormetrics.MetricOpts{
			Name: "reflector_watch_duration_seconds",
			Help: "How long an API watch takes to return and decode for the reflectors",
		},
		prometheus.SummaryOpts{},
		[]string{"name"},
	)

	itemsPerWatch = operatormetrics.NewSummaryVec(
		operatormetrics.MetricOpts{
			Name: "reflector_items_per_watch",
			Help: "How many items an API watch returns to the reflectors",
		},
		prometheus.SummaryOpts{},
		[]string{"name"},
	)
)
