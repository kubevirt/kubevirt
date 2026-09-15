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
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	v1 "kubevirt.io/api/core/v1"

	"kubevirt.io/kubevirt/pkg/libvmi"
	"kubevirt.io/kubevirt/tests/console"
	"kubevirt.io/kubevirt/tests/decorators"
	"kubevirt.io/kubevirt/tests/framework/kubevirt"
	"kubevirt.io/kubevirt/tests/framework/matcher"
	"kubevirt.io/kubevirt/tests/libmigration"
	"kubevirt.io/kubevirt/tests/libnet"
	"kubevirt.io/kubevirt/tests/libvmifact"
	"kubevirt.io/kubevirt/tests/libvmops"
)

var _ = Describe(SIG("GuestAgent probe pause during migration", decorators.GuestAgentProbes, decorators.SigComputeMigrations, decorators.RequiresTwoSchedulableNodes, func() {
	const shortProbeFailThreshold = 3

	// After live migration the target virt-launcher pod restarts the kubelet probe clock.
	postMigrationProbeKillWindow := time.Duration(
		guestAgentPingLivenessInitialDelaySeconds+shortProbeFailThreshold*guestAgentPingLivenessPeriodSeconds,
	) * time.Second

	shortGuestAgentPingLivenessProbe := func() *v1.Probe {
		return createGuestAgentPingLivenessProbeWithThreshold(shortProbeFailThreshold)
	}

	It("[test_id:CNV82132-004] should not restart VMI during or after migration", func() {
		vmi := libvmifact.NewFedora(
			libnet.WithMasqueradeNetworking(),
			withLivenessProbe(shortGuestAgentPingLivenessProbe()),
			libvmi.WithEvictionStrategy(v1.EvictionStrategyLiveMigrate),
		)
		vmi = libvmops.RunVMIAndExpectLaunchIgnoreWarnings(vmi, vmiStartTimeout)

		Eventually(matcher.ThisVMI(vmi)).
			WithTimeout(guestAgentConnectTimeout).
			WithPolling(2 * time.Second).
			Should(matcher.HaveConditionTrue(v1.VirtualMachineInstanceAgentConnected))

		patchPauseGuestAgentProbesAnnotationAndWait(vmi, "true")
		Expect(console.LoginToFedora(vmi)).To(Succeed())
		stopGuestAgentAndConfirmDisconnected(vmi)

		migration := libmigration.New(vmi.Name, vmi.Namespace)
		migrationUID := libmigration.RunMigrationAndExpectToCompleteWithDefaultTimeout(kubevirt.Client(), migration)
		vmi = libmigration.ConfirmVMIPostMigration(kubevirt.Client(), vmi, migrationUID)

		// Target virt-launcher starts with guestAgentProbePaused=false until the next SyncVMI.
		expectPauseGuestAgentProbesAnnotation(vmi, "true")
		waitForGuestAgentProbePausePropagation()

		expectVMIRemainsRunning(vmi, postMigrationProbeKillWindow+15*time.Second)
	})

	It("[test_id:CNV82132-005] should fail liveness probes after migration without pause annotation", func() {
		vmi := libvmifact.NewFedora(
			libnet.WithMasqueradeNetworking(),
			withLivenessProbe(shortGuestAgentPingLivenessProbe()),
			libvmi.WithEvictionStrategy(v1.EvictionStrategyLiveMigrate),
		)
		vmi = libvmops.RunVMIAndExpectLaunchIgnoreWarnings(vmi, vmiStartTimeout)

		Eventually(matcher.ThisVMI(vmi)).
			WithTimeout(guestAgentConnectTimeout).
			WithPolling(2 * time.Second).
			Should(matcher.HaveConditionTrue(v1.VirtualMachineInstanceAgentConnected))

		Expect(console.LoginToFedora(vmi)).To(Succeed())

		// Migrate while the guest agent is healthy and no pause annotation is set.
		// Stopping the agent before migration would let liveness kill the VMI on the
		// source pod before migration suppression engages; the during-migration case
		// is covered by unit tests in pkg/virt-launcher/virtwrap.
		migration := libmigration.New(vmi.Name, vmi.Namespace)
		migrationUID := libmigration.RunMigrationAndExpectToCompleteWithDefaultTimeout(kubevirt.Client(), migration)
		vmi = libmigration.ConfirmVMIPostMigration(kubevirt.Client(), vmi, migrationUID)

		By("Verifying migration suppression does not persist after migration completes")
		Expect(console.LoginToFedora(vmi)).To(Succeed())
		stopGuestAgentAndConfirmDisconnected(vmi)
		expectVMITerminatedFromLivenessProbe(vmi)
	})
}))
