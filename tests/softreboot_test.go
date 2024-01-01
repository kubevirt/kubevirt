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
 * Copyright 2019 Red Hat, Inc.
 *
 */

package tests_test

import (
	"context"
	"fmt"
	"time"

	"kubevirt.io/kubevirt/tests/decorators"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	expect "github.com/google/goexpect"

	v1 "kubevirt.io/api/core/v1"
	"kubevirt.io/client-go/kubecli"

	virtctlpause "kubevirt.io/kubevirt/pkg/virtctl/pause"
	virtctlsoftreboot "kubevirt.io/kubevirt/pkg/virtctl/softreboot"
	"kubevirt.io/kubevirt/tests"
	"kubevirt.io/kubevirt/tests/clientcmd"
	"kubevirt.io/kubevirt/tests/console"
	"kubevirt.io/kubevirt/tests/framework/kubevirt"
	"kubevirt.io/kubevirt/tests/framework/matcher"
	"kubevirt.io/kubevirt/tests/libvmi"
	"kubevirt.io/kubevirt/tests/testsuite"
)

func waitForVMIRebooted(vmi *v1.VirtualMachineInstance, login func(vmi *v1.VirtualMachineInstance) error) {
	By(fmt.Sprintf("Waiting for vmi %s rebooted", vmi.Name))
	if vmi.Namespace == "" {
		vmi.Namespace = testsuite.GetTestNamespace(vmi)
	}
	time.Sleep(30 * time.Second)
	Expect(login(vmi)).To(Succeed())
	Expect(console.SafeExpectBatch(vmi, []expect.Batcher{
		&expect.BSnd{S: "last reboot | grep reboot | wc -l\n"},
		&expect.BExp{R: "2"},
	}, 300)).To(Succeed(), "expected reboot record")
}

func withoutACPI() libvmi.Option {
	return func(vmi *v1.VirtualMachineInstance) {
		acpiEnabled := false
		vmi.Spec.Domain.Features = &v1.Features{
			ACPI: v1.FeatureState{Enabled: &acpiEnabled},
		}
	}
}

const vmiLaunchTimeout = 360

var _ = Describe("[crit:medium][vendor:cnv-qe@redhat.com][level:component][sig-compute]Soft reboot", decorators.SigCompute, func() {

	var virtClient kubecli.KubevirtClient

	BeforeEach(func() {
		virtClient = kubevirt.Client()
	})

	Context("Soft reboot VMI", func() {

		var vmi *v1.VirtualMachineInstance

		When("soft reboot vmi with agent connected via API", func() {
			It("[test_cid:15690]should succeed", func() {
				vmi = tests.RunVMIAndExpectLaunch(libvmi.NewFedora(withoutACPI()), vmiLaunchTimeout)

				Eventually(matcher.ThisVMI(vmi), 12*time.Minute, 2*time.Second).Should(matcher.HaveConditionTrue(v1.VirtualMachineInstanceAgentConnected))

				err := virtClient.VirtualMachineInstance(testsuite.GetTestNamespace(vmi)).SoftReboot(context.Background(), vmi.Name)
				Expect(err).ToNot(HaveOccurred())

				waitForVMIRebooted(vmi, console.LoginToFedora)
			})
		})

		When("soft reboot vmi with ACPI feature enabled via API", func() {
			It("[test_cid:26682]should succeed", func() {
				vmi = tests.RunVMIAndExpectLaunch(libvmi.NewCirros(), vmiLaunchTimeout)

				Expect(console.LoginToCirros(vmi)).To(Succeed())
				Eventually(matcher.ThisVMI(vmi), 30*time.Second, 2*time.Second).Should(matcher.HaveConditionMissingOrFalse(v1.VirtualMachineInstanceAgentConnected))

				err := virtClient.VirtualMachineInstance(testsuite.GetTestNamespace(vmi)).SoftReboot(context.Background(), vmi.Name)
				Expect(err).ToNot(HaveOccurred())

				waitForVMIRebooted(vmi, console.LoginToCirros)
			})
		})

		When("soft reboot vmi with agent connected via virtctl", func() {
			It("[test_cid:12838]should succeed", func() {
				vmi = tests.RunVMIAndExpectLaunch(libvmi.NewFedora(withoutACPI()), vmiLaunchTimeout)

				Eventually(matcher.ThisVMI(vmi), 12*time.Minute, 2*time.Second).Should(matcher.HaveConditionTrue(v1.VirtualMachineInstanceAgentConnected))

				command := clientcmd.NewRepeatableVirtctlCommand(virtctlsoftreboot.COMMAND_SOFT_REBOOT, "--namespace", testsuite.GetTestNamespace(vmi), vmi.Name)
				Expect(command()).To(Succeed())

				waitForVMIRebooted(vmi, console.LoginToFedora)
			})
		})

		When("soft reboot vmi with ACPI feature enabled via virtctl", func() {
			It("[test_cid:35539]should succeed", func() {
				vmi = tests.RunVMIAndExpectLaunch(libvmi.NewCirros(), vmiLaunchTimeout)

				Expect(console.LoginToCirros(vmi)).To(Succeed())
				Eventually(matcher.ThisVMI(vmi), 30*time.Second, 2*time.Second).Should(matcher.HaveConditionMissingOrFalse(v1.VirtualMachineInstanceAgentConnected))

				command := clientcmd.NewRepeatableVirtctlCommand(virtctlsoftreboot.COMMAND_SOFT_REBOOT, "--namespace", testsuite.GetTestNamespace(vmi), vmi.Name)
				Expect(command()).To(Succeed())

				waitForVMIRebooted(vmi, console.LoginToCirros)
			})
		})

		When("soft reboot vmi neither have the agent connected nor the ACPI feature enabled via virtctl", func() {
			It("[test_cid:13968]should failed", func() {
				vmi = tests.RunVMIAndExpectLaunch(libvmi.NewCirros(withoutACPI()), vmiLaunchTimeout)

				Expect(console.LoginToCirros(vmi)).To(Succeed())
				Eventually(matcher.ThisVMI(vmi), 30*time.Second, 2*time.Second).Should(matcher.HaveConditionMissingOrFalse(v1.VirtualMachineInstanceAgentConnected))

				command := clientcmd.NewRepeatableVirtctlCommand(virtctlsoftreboot.COMMAND_SOFT_REBOOT, "--namespace", testsuite.GetTestNamespace(vmi), vmi.Name)
				err := command()
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("VMI neither have the agent connected nor the ACPI feature enabled"))
			})
		})

		When("soft reboot vmi after paused and unpaused via virtctl", func() {
			It("[test_cid:18418]should failed to soft reboot a paused vmi", func() {
				vmi = tests.RunVMIAndExpectLaunch(libvmi.NewFedora(), vmiLaunchTimeout)
				Eventually(matcher.ThisVMI(vmi), 12*time.Minute, 2*time.Second).Should(matcher.HaveConditionTrue(v1.VirtualMachineInstanceAgentConnected))

				command := clientcmd.NewRepeatableVirtctlCommand(virtctlpause.COMMAND_PAUSE, "vmi", "--namespace", testsuite.GetTestNamespace(vmi), vmi.Name)
				Expect(command()).To(Succeed())
				Eventually(matcher.ThisVMI(vmi), 30*time.Second, 2*time.Second).Should(matcher.HaveConditionTrue(v1.VirtualMachineInstancePaused))

				command = clientcmd.NewRepeatableVirtctlCommand(virtctlsoftreboot.COMMAND_SOFT_REBOOT, "--namespace", testsuite.GetTestNamespace(vmi), vmi.Name)
				err := command()
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("VMI is paused"))

				command = clientcmd.NewRepeatableVirtctlCommand(virtctlpause.COMMAND_UNPAUSE, "vmi", "--namespace", testsuite.GetTestNamespace(vmi), vmi.Name)
				Expect(command()).To(Succeed())
				Eventually(matcher.ThisVMI(vmi), 30*time.Second, 2*time.Second).Should(matcher.HaveConditionMissingOrFalse(v1.VirtualMachineInstancePaused))

				Eventually(matcher.ThisVMI(vmi), 12*time.Minute, 2*time.Second).Should(matcher.HaveConditionTrue(v1.VirtualMachineInstanceAgentConnected))

				command = clientcmd.NewRepeatableVirtctlCommand(virtctlsoftreboot.COMMAND_SOFT_REBOOT, "--namespace", testsuite.GetTestNamespace(vmi), vmi.Name)
				Expect(command()).To(Succeed())

				waitForVMIRebooted(vmi, console.LoginToFedora)
			})
		})
	})
})
