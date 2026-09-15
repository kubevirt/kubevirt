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

package admission

// UtilizationReader reports cgroup CPU and memory utilization as a percentage
// of the pod's configured limits.
type UtilizationReader interface {
	Utilization() (cpuPercent, memoryPercent float64, ok bool)
}

var utilizationReader UtilizationReader = newPlatformUtilizationReader()

// Utilization returns the current CPU and memory utilization percentages.
// ok is false when limits cannot be read (for example outside a cgroup).
func Utilization() (cpuPercent, memoryPercent float64, ok bool) {
	return utilizationReader.Utilization()
}

// OverSoftUtilizationLimit reports whether CPU or memory exceeds the soft
// utilization admission thresholds.
func OverSoftUtilizationLimit() bool {
	cpu, mem, ok := Utilization()
	if !ok {
		return false
	}
	return cpu > float64(SoftCPUUtilizationPercent) || mem > float64(SoftMemoryUtilizationPercent)
}

// SetUtilizationReaderForTest replaces the utilization reader for unit tests.
func SetUtilizationReaderForTest(reader UtilizationReader) {
	utilizationReader = reader
}

// ResetUtilizationReaderForTest restores the default platform utilization reader.
func ResetUtilizationReaderForTest() {
	utilizationReader = newPlatformUtilizationReader()
}
