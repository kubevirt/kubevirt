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
 * Copyright The KubeVirt Authors.
 *
 */

package virtcontroller

import (
	"strings"

	"github.com/rhobs/operator-observability-toolkit/pkg/operatormetrics"
	k8sv1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/tools/cache"
)

const inventoryPhaseUnset = "unset"

func listStoreObjects[T any](store cache.Store) []*T {
	if store == nil {
		return nil
	}
	cachedObjs := store.List()
	items := make([]*T, len(cachedObjs))
	for i, obj := range cachedObjs {
		items[i] = obj.(*T)
	}
	return items
}

func resourcePhaseLabel(phase string) string {
	if phase == "" {
		return inventoryPhaseUnset
	}
	return strings.ToLower(phase)
}

func typedLocalObjectName(ref *k8sv1.TypedLocalObjectReference) string {
	if ref == nil {
		return none
	}
	return ref.Name
}

func typedLocalObjectKind(ref *k8sv1.TypedLocalObjectReference) string {
	if ref == nil {
		return none
	}
	return ref.Kind
}

func optionalStringLabel(value *string) string {
	if value == nil || *value == "" {
		return none
	}
	return *value
}

func boolGaugeValue(enabled bool) float64 {
	if enabled {
		return 1
	}
	return 0
}

func collectUnixTimestamp(
	metric operatormetrics.Metric,
	timestamp metav1.Time,
	labels []string,
) []operatormetrics.CollectorResult {
	if timestamp.IsZero() {
		return nil
	}
	return []operatormetrics.CollectorResult{{
		Metric: metric,
		Value:  float64(timestamp.Unix()),
		Labels: labels,
	}}
}

func collectOptionalUnixTimestamp(
	metric operatormetrics.Metric,
	timestamp *metav1.Time,
	labels []string,
) []operatormetrics.CollectorResult {
	if timestamp == nil {
		return nil
	}
	return collectUnixTimestamp(metric, *timestamp, labels)
}
