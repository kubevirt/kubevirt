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
	"k8s.io/client-go/tools/cache"
)

func SetupMetrics() error {
	if err := operatormetrics.RegisterMetrics(
		listsMetrics,
		resourceMetrics,
		watchesMetrics,
	); err != nil {
		return err
	}

	cache.SetReflectorMetricsProvider(prometheusMetricsProvider{})

	return nil
}

type prometheusMetricsProvider struct{}

func (prometheusMetricsProvider) NewListsMetric(name string) cache.CounterMetric {
	return listsTotal.WithLabelValues(name)
}

// NewListDurationMetric uses summary to get averages and percentiles
func (prometheusMetricsProvider) NewListDurationMetric(name string) cache.SummaryMetric {
	return listsDuration.WithLabelValues(name)
}

// NewItemsInListMetric uses summary to get averages and percentiles
func (prometheusMetricsProvider) NewItemsInListMetric(name string) cache.SummaryMetric {
	return itemsPerList.WithLabelValues(name)
}

func (prometheusMetricsProvider) NewWatchesMetric(name string) cache.CounterMetric {
	return watchesTotal.WithLabelValues(name)
}

func (prometheusMetricsProvider) NewShortWatchesMetric(name string) cache.CounterMetric {
	return shortWatchesTotal.WithLabelValues(name)
}

// NewWatchDurationMetric uses summary to get averages and percentiles
func (prometheusMetricsProvider) NewWatchDurationMetric(name string) cache.SummaryMetric {
	return watchDuration.WithLabelValues(name)
}

// NewItemsInWatchMetric uses summary to get averages and percentiles
func (prometheusMetricsProvider) NewItemsInWatchMetric(name string) cache.SummaryMetric {
	return itemsPerWatch.WithLabelValues(name)
}

func (prometheusMetricsProvider) NewLastResourceVersionMetric(name string) cache.GaugeMetric {
	return lastResourceVersion.WithLabelValues(name)
}
