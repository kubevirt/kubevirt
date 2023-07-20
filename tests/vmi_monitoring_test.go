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
 * Copyright 2017 Red Hat, Inc.
 *
 */

package tests_test

import (
	"context"
	"time"

	"kubevirt.io/kubevirt/tests/decorators"

	expect "github.com/google/goexpect"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	v1 "kubevirt.io/api/core/v1"
	"kubevirt.io/client-go/kubecli"

	"kubevirt.io/kubevirt/tests"
	"kubevirt.io/kubevirt/tests/console"
	"kubevirt.io/kubevirt/tests/framework/kubevirt"
	"kubevirt.io/kubevirt/tests/libvmi"
)

var _ = Describe("[sig-compute]Health Monitoring", decorators.SigCompute, func() {

	var virtClient kubecli.KubevirtClient

	BeforeEach(func() {
		virtClient = kubevirt.Client()
	})

	Describe("A VirtualMachineInstance with a watchdog device", func() {
		It("[test_id:4641]should be shut down when the watchdog expires", func() {
			vmi := tests.RunVMIAndExpectLaunch(
				libvmi.NewAlpine(libvmi.WithWatchdog(v1.WatchdogActionPoweroff)), 360)

			By("Expecting the VirtualMachineInstance console")
			Expect(console.LoginToAlpine(vmi)).To(Succeed())

			By("Killing the watchdog device")
			Expect(console.SafeExpectBatch(vmi, []expect.Batcher{
				&expect.BSnd{S: "watchdog -t 2000ms -T 4000ms /dev/watchdog && sleep 5 && killall -9 watchdog\n"},
				&expect.BExp{R: console.PromptExpression},
				&expect.BSnd{S: "echo $?\n"},
				&expect.BExp{R: console.RetValue("0")},
			}, 250)).To(Succeed())

			namespace := vmi.ObjectMeta.Namespace
			name := vmi.ObjectMeta.Name

			By("Checking that the VirtualMachineInstance has Failed status")
			Eventually(func() v1.VirtualMachineInstancePhase {
				startedVMI, err := virtClient.VirtualMachineInstance(namespace).Get(context.Background(), name, &metav1.GetOptions{})

				Expect(err).ToNot(HaveOccurred())
				return startedVMI.Status.Phase
			}, 40*time.Second).Should(Equal(v1.Failed))

		})
	})
})
