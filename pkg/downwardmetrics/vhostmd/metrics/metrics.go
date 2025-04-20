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
 */

package metrics

import (
	"fmt"

	"kubevirt.io/kubevirt/pkg/downwardmetrics/vhostmd/api"
)

func MustToUnitlessHostMetric(value interface{}, name string) api.Metric {
	return MustToMetric(value, name, "", api.MetricContextHost)
}

func MustToHostMetric(value interface{}, name string, unit string) api.Metric {
	return MustToMetric(value, name, unit, api.MetricContextHost)
}

func MustToVMMetric(value interface{}, name string, unit string) api.Metric {
	return MustToMetric(value, name, unit, api.MetricContextVM)
}

func MustToMetric(value interface{}, name string, unit string, context api.MetricContext) api.Metric {
	m, err := ToMetric(value, name, unit, context)
	if err != nil {
		panic(fmt.Errorf("MustToMetric faild which is a hint for a programming error: %v", err))
	}
	return m
}

func ToMetric(value interface{}, name string, unit string, context api.MetricContext) (api.Metric, error) {

	metric := api.Metric{
		Name:    name,
		Context: context,
	}
	switch value.(type) {
	case int, int64:
		metric.Type = api.MetricTypeInt64
	case int8, int16, int32:
		metric.Type = api.MetricTypeInt32
	case uint, uint64:
		metric.Type = api.MetricTypeUInt64
	case uint8, uint16, uint32:
		metric.Type = api.MetricTypeUInt32
	case float64:
		metric.Type = api.MetricTypeReal64
	case float32:
		metric.Type = api.MetricTypeReal32
	case string:
		metric.Type = api.MetricTypeString
	default:
		return api.Metric{}, fmt.Errorf("unknown type for metric %v: %T", name, value)
	}

	switch value.(type) {
	case float64, float32:
		metric.Value = fmt.Sprintf("%.6f", value)
	default:
		metric.Value = fmt.Sprintf("%v", value)
	}

	if unit != "" {
		metric.Unit = unit
	}

	return metric, nil
}
