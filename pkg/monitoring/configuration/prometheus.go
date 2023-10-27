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
 *
 */

package configuration

import (
	"github.com/prometheus/client_golang/prometheus"
)

const emulationEnabledGauge = "emulationEnabledGauge"

var (
	metrics = map[string]prometheus.Opts{
		emulationEnabledGauge: {
			Name: "kubevirt_configuration_emulation_enabled",
			Help: "Indicates whether the Software Emulation is enabled in the configuration.",
		},
	}

	emulationEnabled = prometheus.NewGauge(
		prometheus.GaugeOpts{
			Name: metrics[emulationEnabledGauge].Name,
			Help: metrics[emulationEnabledGauge].Help,
		},
	)
)

func init() {
	prometheus.MustRegister(emulationEnabled)
}

func SetEmulationEnabledMetric(isEmulationEnabled bool) {
	emulationEnabled.Set(boolToFloat64(isEmulationEnabled))
}

func boolToFloat64(b bool) float64 {
	if b {
		return 1
	}
	return 0
}
