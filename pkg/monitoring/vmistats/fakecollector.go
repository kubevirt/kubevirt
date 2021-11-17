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
 * Copyright 2018 Red Hat, Inc.
 *
 */

package vmistats

import (
	"github.com/prometheus/client_golang/prometheus"

	k6tv1 "kubevirt.io/api/core/v1"
)

type fakeCollector struct {
}

func (fc fakeCollector) Describe(_ chan<- *prometheus.Desc) {
}

//Collect needs to report all metrics to see it in docs
func (fc fakeCollector) Collect(ch chan<- prometheus.Metric) {
	vmi := k6tv1.VirtualMachineInstance{
		Status: k6tv1.VirtualMachineInstanceStatus{
			Phase:    k6tv1.Running,
			NodeName: "test",
		},
	}
	updateVMIsPhase([]*k6tv1.VirtualMachineInstance{&vmi}, ch)
}

type fakeIdentifier struct {
}

func (*fakeIdentifier) GetName() (string, error) {
	return "test", nil
}

func (*fakeIdentifier) GetUUIDString() (string, error) {
	return "uuid", nil
}

func RegisterFakeCollector() {
	prometheus.MustRegister(fakeCollector{})
}
