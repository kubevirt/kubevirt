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

import "testing"

type fakeUtilizationReader struct {
	cpuPercent float64
	memPercent float64
	ok         bool
}

func (f fakeUtilizationReader) Utilization() (cpuPercent, memoryPercent float64, ok bool) {
	return f.cpuPercent, f.memPercent, f.ok
}

func TestOverSoftUtilizationLimit(t *testing.T) {
	t.Cleanup(ResetUtilizationReaderForTest)

	tests := []struct {
		name string
		cpu  float64
		mem  float64
		ok   bool
		over bool
	}{
		{name: "below both thresholds", cpu: 50, mem: 50, ok: true, over: false},
		{name: "cpu over threshold", cpu: 71, mem: 50, ok: true, over: true},
		{name: "memory over threshold", cpu: 50, mem: 71, ok: true, over: true},
		{name: "at threshold not over", cpu: 70, mem: 70, ok: true, over: false},
		{name: "unavailable metrics fail open", cpu: 100, mem: 100, ok: false, over: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			SetUtilizationReaderForTest(fakeUtilizationReader{
				cpuPercent: tt.cpu,
				memPercent: tt.mem,
				ok:         tt.ok,
			})
			if got := OverSoftUtilizationLimit(); got != tt.over {
				t.Fatalf("OverSoftUtilizationLimit() = %v, want %v", got, tt.over)
			}
		})
	}
}
