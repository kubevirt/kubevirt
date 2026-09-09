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

package compute

import (
	"context"
	"fmt"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"

	v1 "kubevirt.io/api/core/v1"

	"kubevirt.io/kubevirt/tests/console"
	"kubevirt.io/kubevirt/tests/decorators"
	"kubevirt.io/kubevirt/tests/framework/kubevirt"
	"kubevirt.io/kubevirt/tests/framework/matcher"
	"kubevirt.io/kubevirt/tests/libnet"
	"kubevirt.io/kubevirt/tests/libvmifact"
	"kubevirt.io/kubevirt/tests/libvmops"
	"kubevirt.io/kubevirt/tests/testsuite"
)

var _ = Describe(SIG("GuestAgent probe pause", decorators.GuestAgentProbes, Serial, func() {
	It("[test_id:CNV82132-001] should resume probes and restart the VMI after annotation removal", func() {
		vmi := runFedoraWithGuestAgentPingLiveness(3)

		Expect(console.LoginToFedora(vmi)).To(Succeed())

		By("Adding the pause-guest-agent-probes annotation")
		patchPauseGuestAgentProbesAnnotationAndWait(vmi, "true")

		By("Stopping guest agent to simulate maintenance")
		stopGuestAgentAndConfirmDisconnected(vmi)

		By("Verifying the VMI stays alive while annotation is set (probes paused)")
		expectVMIRemainsRunning(vmi, 45*time.Second)

		By("Removing the pause annotation to resume probes")
		removePauseGuestAgentProbesAnnotation(vmi)

		By("Verifying the VMI eventually terminates after annotation is removed (agent still stopped)")
		expectVMITerminatedFromLivenessProbe(vmi)
	})

	Context("readiness probe with annotation-paused probes", func() {
		const (
			period         = 5
			initialSeconds = 5
		)

		It("[test_id:CNV82132-002] should remain Ready with paused readiness probes", func() {
			vmi := libvmifact.NewFedora(
				libnet.WithMasqueradeNetworking(),
				withReadinessProbe(createGuestAgentPingProbe(period, initialSeconds)),
			)
			vmi = libvmops.RunVMIAndExpectLaunchIgnoreWarnings(vmi, vmiStartTimeout)

			Eventually(matcher.ThisVMI(vmi)).
				WithTimeout(guestAgentConnectTimeout).
				WithPolling(2 * time.Second).
				Should(matcher.HaveConditionTrue(v1.VirtualMachineInstanceAgentConnected))

			Eventually(matcher.ThisVMI(vmi)).
				WithTimeout(2 * time.Minute).
				WithPolling(2 * time.Second).
				Should(matcher.HaveConditionTrue(v1.VirtualMachineInstanceReady))

			patchPauseGuestAgentProbesAnnotationAndWait(vmi, "true")
			Expect(console.LoginToFedora(vmi)).To(Succeed())
			stopGuestAgentAndConfirmDisconnected(vmi)

			Consistently(matcher.ThisVMI(vmi)).
				WithTimeout(45 * time.Second).
				WithPolling(2 * time.Second).
				Should(matcher.HaveConditionTrue(v1.VirtualMachineInstanceReady))
		})

		It("[test_id:CNV82132-002b] should lose Ready when agent stops without pause annotation", func() {
			vmi := libvmifact.NewFedora(
				libnet.WithMasqueradeNetworking(),
				withReadinessProbe(createGuestAgentPingProbe(period, initialSeconds)),
			)
			vmi = libvmops.RunVMIAndExpectLaunchIgnoreWarnings(vmi, vmiStartTimeout)

			Eventually(matcher.ThisVMI(vmi)).
				WithTimeout(guestAgentConnectTimeout).
				WithPolling(2 * time.Second).
				Should(matcher.HaveConditionTrue(v1.VirtualMachineInstanceAgentConnected))

			Eventually(matcher.ThisVMI(vmi)).
				WithTimeout(2 * time.Minute).
				WithPolling(2 * time.Second).
				Should(matcher.HaveConditionTrue(v1.VirtualMachineInstanceReady))

			Expect(console.LoginToFedora(vmi)).To(Succeed())
			stopGuestAgentAndConfirmDisconnected(vmi)

			Eventually(matcher.ThisVMI(vmi)).
				WithTimeout(5 * time.Minute).
				WithPolling(2 * time.Second).
				Should(matcher.HaveConditionMissingOrFalse(v1.VirtualMachineInstanceReady))
		})
	})

	It("[test_id:CNV82132-009] should remain paused across reconciliation cycles", func() {
		vmi := runFedoraWithGuestAgentPingLiveness(3)

		patchPauseGuestAgentProbesAnnotationAndWait(vmi, "true")
		Expect(console.LoginToFedora(vmi)).To(Succeed())
		stopGuestAgentAndConfirmDisconnected(vmi)

		for i := 0; i < 3; i++ {
			mergePatch := []byte(fmt.Sprintf(`{"metadata":{"labels":{"reconcile-trigger-%d":"true"}}}`, i))
			_, err := kubevirt.Client().VirtualMachineInstance(testsuite.GetTestNamespace(vmi)).
				Patch(context.Background(), vmi.Name, types.MergePatchType, mergePatch, metav1.PatchOptions{})
			Expect(err).ToNot(HaveOccurred())
		}

		expectVMIRemainsRunning(vmi, 45*time.Second)
	})
}))
