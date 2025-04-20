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

package tests_test

import (
	"time"

	. "github.com/onsi/ginkgo/v2"
	"github.com/onsi/gomega/gmeasure"

	"kubevirt.io/kubevirt/tests/libvmifact"
	"kubevirt.io/kubevirt/tests/libvmops"
)

// Replace PDescribe with FDescribe in order to measure if your changes made
// VMI startup any worse
var _ = PDescribe("Ensure stable functionality", func() {
	It("by repeately starting vmis many times without issues", func() {
		experiment := gmeasure.NewExperiment("VMs creation")
		AddReportEntry(experiment.Name, experiment)

		experiment.Sample(func(idx int) {
			experiment.MeasureDuration("Create VM", func() {
				libvmops.RunVMIAndExpectLaunch(libvmifact.NewCirros(), 30)
			})
		}, gmeasure.SamplingConfig{N: 15, Duration: 10 * time.Minute})
	})
})
