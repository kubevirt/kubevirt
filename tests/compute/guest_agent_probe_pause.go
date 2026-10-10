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

	k8scorev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"

	v1 "kubevirt.io/api/core/v1"

	"kubevirt.io/kubevirt/pkg/apimachinery/patch"
	"kubevirt.io/kubevirt/pkg/libvmi"
	"kubevirt.io/kubevirt/tests/console"
	"kubevirt.io/kubevirt/tests/decorators"
	"kubevirt.io/kubevirt/tests/events"
	"kubevirt.io/kubevirt/tests/framework/kubevirt"
	"kubevirt.io/kubevirt/tests/framework/matcher"
	"kubevirt.io/kubevirt/tests/libmigration"
	"kubevirt.io/kubevirt/tests/libnet"
	"kubevirt.io/kubevirt/tests/libpod"
	"kubevirt.io/kubevirt/tests/libvmifact"
	"kubevirt.io/kubevirt/tests/libvmops"
	"kubevirt.io/kubevirt/tests/testsuite"
)

const (
	guestAgentPingLivenessInitialDelaySeconds int32 = 30
	guestAgentPingLivenessPeriodSeconds       int32 = 5

	guestAgentProbePausePropagationTimeout = 30 * time.Second

	// Logged by syncGuestAgentProbePaused in pkg/virt-launcher/virtwrap/manager.go.
	guestAgentProbePauseStateChangedTrueLog = "Guest agent probe pause state changed: paused=true"

	guestAgentPingFailedReason = "GuestAgentPingFailed"
)

// E2e tests for the pause-guest-agent-probes annotation ([test_id:CNV82132-003] is in guest_agent.go).
var _ = Describe(SIG("GuestAgent probe pause", decorators.GuestAgentProbes, decorators.WgS390x, func() {
	const (
		pausedWindow            = 45 * time.Second
		shortProbeFailThreshold = 3
	)

	// After live migration the target virt-launcher pod restarts the kubelet probe clock.
	postMigrationProbeKillWindow := time.Duration(
		guestAgentPingLivenessInitialDelaySeconds+shortProbeFailThreshold*guestAgentPingLivenessPeriodSeconds,
	) * time.Second

	It("[test_id:CNV82132-001] should keep the VMI alive and silent while paused, and resume probes after removal", func() {
		vmi := runFedoraWithGuestAgentPingLiveness(shortProbeFailThreshold)

		By("Pausing guest-agent probes")
		pauseGuestAgentProbes(vmi)
		// Pings can fail while the agent is still connecting at boot; start from a clean slate.
		events.DeleteEvents(vmi, k8scorev1.EventTypeWarning, guestAgentPingFailedReason)

		By("Stopping the guest agent to simulate maintenance")
		Expect(console.LoginToFedora(vmi)).To(Succeed())
		stopGuestAgentAndConfirmDisconnected(vmi)

		By("Verifying the VMI stays Running with no GuestAgentPingFailed events while paused")
		expectVMIRemainsRunning(vmi, pausedWindow)
		events.ExpectNoEvent(vmi, k8scorev1.EventTypeWarning, guestAgentPingFailedReason)

		By("Removing the pause annotation to resume probes (agent still stopped)")
		resumeGuestAgentProbes(vmi)
		Eventually(matcher.ThisVMI(vmi)).
			WithTimeout(2 * time.Minute).
			WithPolling(1 * time.Second).
			Should(Or(matcher.BeInPhase(v1.Failed), matcher.HaveSucceeded()))

		// Checked after termination: events outlive the VMI's Running phase, so this avoids racing the kill.
		By("Verifying GuestAgentPingFailed events resumed once the annotation was removed")
		events.ExpectEvent(vmi, k8scorev1.EventTypeWarning, guestAgentPingFailedReason)
	})

	It("[test_id:CNV82132-002] should remain Ready with paused readiness probes", func() {
		const (
			readinessPeriod         = 5
			readinessInitialSeconds = 5
		)
		vmi := libvmifact.NewFedora(
			libnet.WithMasqueradeNetworking(),
			withReadinessProbe(createGuestAgentPingProbe(readinessPeriod, readinessInitialSeconds)),
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

		pauseGuestAgentProbes(vmi)
		Expect(console.LoginToFedora(vmi)).To(Succeed())
		stopGuestAgentAndConfirmDisconnected(vmi)

		Consistently(matcher.ThisVMI(vmi)).
			WithTimeout(pausedWindow).
			WithPolling(2 * time.Second).
			Should(matcher.HaveConditionTrue(v1.VirtualMachineInstanceReady))
	})

	It("[test_id:CNV82132-004] should not restart VMI during or after migration", decorators.SigComputeMigrations, decorators.RequiresTwoSchedulableNodes, func() {
		vmi := runFedoraWithGuestAgentPingLiveness(shortProbeFailThreshold,
			libvmi.WithEvictionStrategy(v1.EvictionStrategyLiveMigrate))

		pauseGuestAgentProbes(vmi)
		Expect(console.LoginToFedora(vmi)).To(Succeed())
		stopGuestAgentAndConfirmDisconnected(vmi)

		migration := libmigration.New(vmi.Name, vmi.Namespace)
		migrationUID := libmigration.RunMigrationAndExpectToCompleteWithDefaultTimeout(kubevirt.Client(), migration)
		vmi = libmigration.ConfirmVMIPostMigration(kubevirt.Client(), vmi, migrationUID)

		// Target virt-launcher starts with guestAgentProbePaused=false until the next SyncVMI.
		Expect(vmi.Status.MigrationState.TargetPod).ToNot(BeEmpty())
		waitForGuestAgentProbePausePropagation(vmi, vmi.Status.MigrationState.TargetPod)

		// One extra failure cycle, so a restart slightly after the kill window is still observed.
		expectVMIRemainsRunning(vmi, postMigrationProbeKillWindow+15*time.Second)

		// Without the pause, the dead agent must now fail the probe on the target. This also shows
		// migration-time suppression has ended, so the survival above came from the pause.
		By("Resuming probes on the target")
		resumeGuestAgentProbes(vmi)
		Eventually(matcher.ThisVMI(vmi)).
			WithTimeout(2 * time.Minute).
			WithPolling(1 * time.Second).
			Should(Or(matcher.BeInPhase(v1.Failed), matcher.HaveSucceeded()))
	})
}))

func createGuestAgentPingLivenessProbeWithThreshold(failureThreshold int32) *v1.Probe {
	probe := createGuestAgentPingProbe(guestAgentPingLivenessPeriodSeconds, guestAgentPingLivenessInitialDelaySeconds)
	probe.FailureThreshold = failureThreshold
	return probe
}

func runFedoraWithGuestAgentPingLiveness(failureThreshold int32, opts ...libvmi.Option) *v1.VirtualMachineInstance {
	options := []libvmi.Option{
		libnet.WithMasqueradeNetworking(),
		withLivenessProbe(createGuestAgentPingLivenessProbeWithThreshold(failureThreshold)),
	}
	options = append(options, opts...)
	vmi := libvmifact.NewFedora(options...)
	vmi = libvmops.RunVMIAndExpectLaunchIgnoreWarnings(vmi, vmiStartTimeout)
	Eventually(matcher.ThisVMI(vmi)).
		WithTimeout(guestAgentConnectTimeout).
		WithPolling(2 * time.Second).
		Should(matcher.HaveConditionTrue(v1.VirtualMachineInstanceAgentConnected))
	return vmi
}

func stopGuestAgentAndConfirmDisconnected(vmi *v1.VirtualMachineInstance) {
	ExpectWithOffset(1, stopGuestAgent(vmi)).To(Succeed())
	EventuallyWithOffset(1, matcher.ThisVMI(vmi)).
		WithTimeout(2 * time.Minute).
		WithPolling(2 * time.Second).
		Should(matcher.HaveConditionMissingOrFalse(v1.VirtualMachineInstanceAgentConnected))
}

func pauseGuestAgentProbes(vmi *v1.VirtualMachineInstance) {
	patchBytes, err := patch.New(
		patch.WithAdd(
			fmt.Sprintf("/metadata/annotations/%s",
				patch.EscapeJSONPointer(v1.PauseGuestAgentProbesAnnotation)),
			"true",
		),
	).GeneratePayload()
	ExpectWithOffset(1, err).ToNot(HaveOccurred())

	_, err = kubevirt.Client().VirtualMachineInstance(testsuite.GetTestNamespace(vmi)).
		Patch(context.Background(), vmi.Name, types.JSONPatchType, patchBytes, metav1.PatchOptions{})
	ExpectWithOffset(1, err).ToNot(HaveOccurred())

	waitForGuestAgentProbePausePropagation(vmi, "")
}

func resumeGuestAgentProbes(vmi *v1.VirtualMachineInstance) {
	patchBytes, err := patch.New(
		patch.WithRemove(
			fmt.Sprintf("/metadata/annotations/%s",
				patch.EscapeJSONPointer(v1.PauseGuestAgentProbesAnnotation)),
		),
	).GeneratePayload()
	ExpectWithOffset(1, err).ToNot(HaveOccurred())

	_, err = kubevirt.Client().VirtualMachineInstance(testsuite.GetTestNamespace(vmi)).
		Patch(context.Background(), vmi.Name, types.JSONPatchType, patchBytes, metav1.PatchOptions{})
	ExpectWithOffset(1, err).ToNot(HaveOccurred())
}

// waitForGuestAgentProbePausePropagation waits until virtLauncherPodName has logged
// the pause. An empty name reads the current launcher pod.
func waitForGuestAgentProbePausePropagation(vmi *v1.VirtualMachineInstance, virtLauncherPodName string) {
	namespace := testsuite.GetTestNamespace(vmi)
	podName := virtLauncherPodName
	if podName == "" {
		pod, err := libpod.GetPodByVirtualMachineInstance(vmi, namespace)
		ExpectWithOffset(1, err).ToNot(HaveOccurred(),
			"failed to find virt-launcher pod for VMI %s/%s", namespace, vmi.Name)
		podName = pod.Name
	}
	EventuallyWithOffset(1, func() (string, error) {
		logsRaw, err := kubevirt.Client().CoreV1().Pods(namespace).GetLogs(podName, &k8scorev1.PodLogOptions{
			Container: "compute",
		}).DoRaw(context.Background())
		if err != nil {
			return "", err
		}
		return string(logsRaw), nil
	}).WithTimeout(guestAgentProbePausePropagationTimeout).
		WithPolling(2*time.Second).
		Should(ContainSubstring(guestAgentProbePauseStateChangedTrueLog),
			"virt-launcher pod %s/%s did not log %q; if the wording changed, update this test to match "+
				"syncGuestAgentProbePaused in pkg/virt-launcher/virtwrap/manager.go",
			namespace, podName, guestAgentProbePauseStateChangedTrueLog)
}

func expectVMIRemainsRunning(vmi *v1.VirtualMachineInstance, duration time.Duration) {
	Consistently(matcher.ThisVMI(vmi)).
		WithTimeout(duration).
		WithPolling(2 * time.Second).
		Should(matcher.BeInPhase(v1.Running))
}
