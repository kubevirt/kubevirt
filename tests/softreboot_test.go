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
	"fmt"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	expect "github.com/google/goexpect"

	v1 "kubevirt.io/api/core/v1"
	"kubevirt.io/client-go/kubecli"
	virtctlpause "kubevirt.io/kubevirt/pkg/virtctl/pause"
	virtctlsoftreboot "kubevirt.io/kubevirt/pkg/virtctl/softreboot"
	"kubevirt.io/kubevirt/tests"
	"kubevirt.io/kubevirt/tests/console"
	cd "kubevirt.io/kubevirt/tests/containerdisk"
	"kubevirt.io/kubevirt/tests/util"
)

func WaitForVMIRebooted(vmi *v1.VirtualMachineInstance, login func(vmi *v1.VirtualMachineInstance) error) {
	By(fmt.Sprintf("Waiting for vmi %s rebooted", vmi.Name))
	if vmi.Namespace == "" {
		vmi.Namespace = util.NamespaceTestDefault
	}
	time.Sleep(30 * time.Second)
	Expect(login(vmi)).To(Succeed())
	Expect(console.SafeExpectBatch(vmi, []expect.Batcher{
		&expect.BSnd{S: "last reboot | grep reboot | wc -l\n"},
		&expect.BExp{R: "2"},
	}, 300)).To(Succeed(), "expected reboot record")
}

var _ = Describe("[crit:medium][vendor:cnv-qe@redhat.com][level:component][sig-compute]Soft reboot", func() {

	var err error
	var virtClient kubecli.KubevirtClient

	BeforeEach(func() {
		virtClient, err = kubecli.GetKubevirtClient()
		util.PanicOnError(err)

		tests.BeforeTestCleanup()
	})

	Context("Soft reboot VMI", func() {

		var vmi *v1.VirtualMachineInstance

		runVMI := func(withGuestAgent bool, ACPIEnabled bool) {
			if withGuestAgent {
				vmi = tests.NewRandomFedoraVMIWithGuestAgent()
			} else {
				vmi = tests.NewRandomVMIWithEphemeralDisk(cd.ContainerDiskFor(cd.ContainerDiskCirros))
				tests.AddUserData(vmi, "cloud-init", "#!/bin/bash\necho 'hello'\n")
			}
			if !ACPIEnabled {
				vmi.Spec.Domain.Features = &v1.Features{
					ACPI: v1.FeatureState{Enabled: &ACPIEnabled},
				}
			}
			tests.RunVMIAndExpectLaunch(vmi, 360)
		}

		When("soft reboot vmi with agent connected via API", func() {
			It("should succeed", func() {
				runVMI(true, false)

				tests.WaitAgentConnected(virtClient, vmi)

				err := virtClient.VirtualMachineInstance(util.NamespaceTestDefault).SoftReboot(vmi.Name)
				Expect(err).ToNot(HaveOccurred())

				WaitForVMIRebooted(vmi, console.LoginToFedora)
			})
		})

		When("soft reboot vmi with ACPI feature enabled via API", func() {
			It("should succeed", func() {
				runVMI(false, true)

				Expect(console.LoginToCirros(vmi)).To(Succeed())
				tests.WaitAgentDisconnected(virtClient, vmi)

				err := virtClient.VirtualMachineInstance(util.NamespaceTestDefault).SoftReboot(vmi.Name)
				Expect(err).ToNot(HaveOccurred())

				WaitForVMIRebooted(vmi, console.LoginToCirros)
			})
		})

		When("soft reboot vmi with agent connected via virtctl", func() {
			It("should succeed", func() {
				runVMI(true, false)

				tests.WaitAgentConnected(virtClient, vmi)

				command := tests.NewRepeatableVirtctlCommand(virtctlsoftreboot.COMMAND_SOFT_REBOOT, "--namespace", util.NamespaceTestDefault, vmi.Name)
				Expect(command()).To(Succeed())

				WaitForVMIRebooted(vmi, console.LoginToFedora)
			})
		})

		When("soft reboot vmi with ACPI feature enabled via virtctl", func() {
			It("should succeed", func() {
				runVMI(false, true)

				Expect(console.LoginToCirros(vmi)).To(Succeed())
				tests.WaitAgentDisconnected(virtClient, vmi)

				command := tests.NewRepeatableVirtctlCommand(virtctlsoftreboot.COMMAND_SOFT_REBOOT, "--namespace", util.NamespaceTestDefault, vmi.Name)
				Expect(command()).To(Succeed())

				WaitForVMIRebooted(vmi, console.LoginToCirros)
			})
		})

		When("soft reboot vmi neither have the agent connected nor the ACPI feature enabled via virtctl", func() {
			It("should failed", func() {
				runVMI(false, false)

				Expect(console.LoginToCirros(vmi)).To(Succeed())
				tests.WaitAgentDisconnected(virtClient, vmi)

				command := tests.NewRepeatableVirtctlCommand(virtctlsoftreboot.COMMAND_SOFT_REBOOT, "--namespace", util.NamespaceTestDefault, vmi.Name)
				err := command()
				Expect(err.Error()).To(ContainSubstring("VMI neither have the agent connected nor the ACPI feature enabled"))
			})
		})

		When("soft reboot vmi after paused and unpaused via virtctl", func() {
			It("should failed to soft reboot a paused vmi", func() {
				runVMI(true, true)
				tests.WaitAgentConnected(virtClient, vmi)

				command := tests.NewRepeatableVirtctlCommand(virtctlpause.COMMAND_PAUSE, "vmi", "--namespace", util.NamespaceTestDefault, vmi.Name)
				Expect(command()).To(Succeed())
				tests.WaitForVMICondition(virtClient, vmi, v1.VirtualMachineInstancePaused, 30)

				command = tests.NewRepeatableVirtctlCommand(virtctlsoftreboot.COMMAND_SOFT_REBOOT, "--namespace", util.NamespaceTestDefault, vmi.Name)
				err := command()
				Expect(err.Error()).To(ContainSubstring("VMI is paused"))

				command = tests.NewRepeatableVirtctlCommand(virtctlpause.COMMAND_UNPAUSE, "vmi", "--namespace", util.NamespaceTestDefault, vmi.Name)
				Expect(command()).To(Succeed())
				tests.WaitForVMIConditionRemovedOrFalse(virtClient, vmi, v1.VirtualMachineInstancePaused, 30)

				tests.WaitAgentConnected(virtClient, vmi)

				command = tests.NewRepeatableVirtctlCommand(virtctlsoftreboot.COMMAND_SOFT_REBOOT, "--namespace", util.NamespaceTestDefault, vmi.Name)
				Expect(command()).To(Succeed())

				WaitForVMIRebooted(vmi, console.LoginToFedora)
			})
		})
	})
})
