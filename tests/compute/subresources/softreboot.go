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

package compute

import (
	"context"
	"fmt"
	"time"

	expect "github.com/google/goexpect"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	v1 "kubevirt.io/api/core/v1"
	"kubevirt.io/client-go/kubecli"

	"kubevirt.io/kubevirt/pkg/libvmi"
	"kubevirt.io/kubevirt/tests/compute"
	"kubevirt.io/kubevirt/tests/console"
	"kubevirt.io/kubevirt/tests/framework/kubevirt"
	"kubevirt.io/kubevirt/tests/framework/matcher"
	"kubevirt.io/kubevirt/tests/libvmifact"
	"kubevirt.io/kubevirt/tests/libvmops"
	"kubevirt.io/kubevirt/tests/testsuite"
)

func waitForVMIRebooted(vmi *v1.VirtualMachineInstance, login console.LoginToFunction) {
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

var _ = compute.SIGDescribe("[crit:medium][vendor:cnv-qe@redhat.com][level:component]Soft reboot", func() {
	const vmiLaunchTimeout = 360

	var virtClient kubecli.KubevirtClient

	BeforeEach(func() {
		virtClient = kubevirt.Client()
	})

	It("soft reboot vmi with agent connected should succeed", func() {
		vmi := libvmops.RunVMIAndExpectLaunch(libvmifact.NewFedora(withoutACPI()), vmiLaunchTimeout)

		Eventually(matcher.ThisVMI(vmi), 12*time.Minute, 2*time.Second).Should(matcher.HaveConditionTrue(v1.VirtualMachineInstanceAgentConnected))

		err := virtClient.VirtualMachineInstance(testsuite.GetTestNamespace(vmi)).SoftReboot(context.Background(), vmi.Name)
		Expect(err).ToNot(HaveOccurred())

		waitForVMIRebooted(vmi, console.LoginToFedora)
	})

	It("soft reboot vmi with ACPI feature enabled should succeed", func() {
		vmi := libvmops.RunVMIAndExpectLaunch(libvmifact.NewCirros(), vmiLaunchTimeout)

		Expect(console.LoginToCirros(vmi)).To(Succeed())
		Eventually(matcher.ThisVMI(vmi), 30*time.Second, 2*time.Second).Should(matcher.HaveConditionMissingOrFalse(v1.VirtualMachineInstanceAgentConnected))

		err := virtClient.VirtualMachineInstance(testsuite.GetTestNamespace(vmi)).SoftReboot(context.Background(), vmi.Name)
		Expect(err).ToNot(HaveOccurred())

		waitForVMIRebooted(vmi, console.LoginToCirros)
	})

	It("soft reboot vmi neither have the agent connected nor the ACPI feature enabled should fail", func() {
		vmi := libvmops.RunVMIAndExpectLaunch(libvmifact.NewCirros(withoutACPI()), vmiLaunchTimeout)

		Expect(console.LoginToCirros(vmi)).To(Succeed())
		Eventually(matcher.ThisVMI(vmi), 30*time.Second, 2*time.Second).Should(matcher.HaveConditionMissingOrFalse(v1.VirtualMachineInstanceAgentConnected))

		err := virtClient.VirtualMachineInstance(testsuite.GetTestNamespace(vmi)).SoftReboot(context.Background(), vmi.Name)
		Expect(err).To(MatchError(ContainSubstring("VMI neither have the agent connected nor the ACPI feature enabled")))
	})

	It("soft reboot vmi should fail to soft reboot a paused vmi", func() {
		vmi := libvmops.RunVMIAndExpectLaunch(libvmifact.NewFedora(), vmiLaunchTimeout)
		Eventually(matcher.ThisVMI(vmi), 12*time.Minute, 2*time.Second).Should(matcher.HaveConditionTrue(v1.VirtualMachineInstanceAgentConnected))

		err := virtClient.VirtualMachineInstance(testsuite.GetTestNamespace(vmi)).Pause(context.Background(), vmi.Name, &v1.PauseOptions{})
		Expect(err).ToNot(HaveOccurred())
		Eventually(matcher.ThisVMI(vmi), 30*time.Second, 2*time.Second).Should(matcher.HaveConditionTrue(v1.VirtualMachineInstancePaused))

		err = virtClient.VirtualMachineInstance(testsuite.GetTestNamespace(vmi)).SoftReboot(context.Background(), vmi.Name)
		Expect(err).To(MatchError(ContainSubstring("VMI is paused")))

		err = virtClient.VirtualMachineInstance(testsuite.GetTestNamespace(vmi)).Unpause(context.Background(), vmi.Name, &v1.UnpauseOptions{})
		Expect(err).ToNot(HaveOccurred())
		Eventually(matcher.ThisVMI(vmi), 30*time.Second, 2*time.Second).Should(matcher.HaveConditionMissingOrFalse(v1.VirtualMachineInstancePaused))

		Eventually(matcher.ThisVMI(vmi), 12*time.Minute, 2*time.Second).Should(matcher.HaveConditionTrue(v1.VirtualMachineInstanceAgentConnected))

		err = virtClient.VirtualMachineInstance(testsuite.GetTestNamespace(vmi)).SoftReboot(context.Background(), vmi.Name)
		Expect(err).ToNot(HaveOccurred())

		waitForVMIRebooted(vmi, console.LoginToFedora)
	})
})
