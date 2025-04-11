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
	"time"

	"kubevirt.io/kubevirt/tests/decorators"
	"kubevirt.io/kubevirt/tests/libvmops"

	expect "github.com/google/goexpect"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	v1 "kubevirt.io/api/core/v1"

	"kubevirt.io/kubevirt/pkg/libvmi"

	"kubevirt.io/kubevirt/tests/console"
	"kubevirt.io/kubevirt/tests/framework/matcher"
	"kubevirt.io/kubevirt/tests/libnode"
	"kubevirt.io/kubevirt/tests/libvmifact"
)

var _ = Describe("[sig-compute]Watchdog", decorators.SigCompute, func() {

	Context("A VirtualMachineInstance with a watchdog device", func() {

		It("[test_id:4641]should be shut down when the watchdog expires", decorators.Conformance, decorators.WgS390x, func() {
			vmi := libvmops.RunVMIAndExpectLaunch(
				libvmifact.NewAlpine(libvmi.WithWatchdog(v1.WatchdogActionPoweroff, libnode.GetArch())), libvmops.StartupTimeoutSecondsXHuge)

			By("Expecting the VirtualMachineInstance console")
			Expect(console.LoginToAlpine(vmi)).To(Succeed())

			// Watchdog module need to be loaded as it will not be loaded by default for s390x
			if vmi.Spec.Architecture == "s390x" {
				By("Ensuring watchdog module is loaded")
				Expect(console.SafeExpectBatch(vmi, []expect.Batcher{
					&expect.BSnd{S: "modprobe diag288_wdt\n"},
					&expect.BExp{R: ""},
				}, 250)).To(Succeed())

				By("Checking watchdog device presence")
				Expect(console.SafeExpectBatch(vmi, []expect.Batcher{
					&expect.BSnd{S: "ls -l /dev/watchdog\n"},
					&expect.BExp{R: "/dev/watchdog"},
				}, 250)).To(Succeed())
			}

			By("Killing the watchdog device")
			Expect(console.SafeExpectBatch(vmi, []expect.Batcher{
				&expect.BSnd{S: "watchdog -t 2000ms -T 4000ms /dev/watchdog && sleep 5 && killall -9 watchdog\n"},
				&expect.BExp{R: ""},
				&expect.BSnd{S: "echo $?\n"},
				&expect.BExp{R: console.RetValue("0")},
			}, 250)).To(Succeed())

			By("Checking that the VirtualMachineInstance has Failed status")
			Eventually(matcher.ThisVMI(vmi)).WithTimeout(40 * time.Second).WithPolling(time.Second).
				Should(matcher.BeInPhase(v1.Failed))
		})

	})

})
