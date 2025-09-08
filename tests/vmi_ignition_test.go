/*
 * This file is part of the kubevirt project
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

package tests_test

import (
	expect "github.com/google/goexpect"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"kubevirt.io/kubevirt/tests/decorators"
	"kubevirt.io/kubevirt/tests/libvmops"

	"kubevirt.io/kubevirt/tests/framework/checks"

	v1 "kubevirt.io/api/core/v1"

	"kubevirt.io/kubevirt/pkg/libvmi"

	"kubevirt.io/kubevirt/tests/console"
	"kubevirt.io/kubevirt/tests/libvmifact"
)

var _ = Describe("[rfe_id:151][crit:high][vendor:cnv-qe@redhat.com][level:component][sig-compute]IgnitionData", decorators.SigCompute, func() {

	BeforeEach(func() {
		if !checks.HasFeature("ExperimentalIgnitionSupport") {
			Fail("ExperimentalIgnitionSupport feature gate is not enabled")
		}
	})

	Describe("[rfe_id:151][crit:medium][vendor:cnv-qe@redhat.com][level:component]A new VirtualMachineInstance", func() {
		Context("with IgnitionData annotation", func() {
			Context("with injected data", func() {
				It("[test_id:1616]should have injected data under firmware directory", func() {
					ignitionData := "ignition injected"
					vmi := libvmops.RunVMIAndExpectLaunch(
						libvmifact.NewFedora(libvmi.WithAnnotation(v1.IgnitionAnnotation, ignitionData)),
						240)

					Expect(console.LoginToFedora(vmi)).To(Succeed())
					Expect(console.SafeExpectBatch(vmi, []expect.Batcher{
						&expect.BSnd{S: "cat /sys/firmware/qemu_fw_cfg/by_name/opt/com.coreos/config/raw\n"},
						&expect.BExp{R: ignitionData},
					}, 300)).To(Succeed())
				})
			})
		})

	})
})
