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
	"encoding/base64"
	"time"

	expect "github.com/google/goexpect"
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
	"kubevirt.io/kubevirt/tests/libnet"
	"kubevirt.io/kubevirt/tests/libsecret"
	"kubevirt.io/kubevirt/tests/libstorage"
	"kubevirt.io/kubevirt/tests/libvmifact"
	"kubevirt.io/kubevirt/tests/libvmops"
	"kubevirt.io/kubevirt/tests/testsuite"
)

var _ = Describe(SIG("GuestAgent probe pause scope", decorators.GuestAgentProbes, Serial, func() {
	const (
		probePeriod         = 5
		probeInitialSeconds = 90
		probeTimeoutSeconds = 1
	)

	It("[test_id:CNV82132-003] should not affect exec probe with pause annotation set", func() {
		livenessProbe := createExecProbe(probePeriod, probeInitialSeconds, probeTimeoutSeconds, "exit", "1")
		vmi := libvmifact.NewFedora(
			libnet.WithMasqueradeNetworking(),
			withLivenessProbe(livenessProbe),
		)
		vmi = libvmops.RunVMIAndExpectLaunchIgnoreWarnings(vmi, vmiStartTimeout)

		patchPauseGuestAgentProbesAnnotationAndWait(vmi, "true")

		expectVMITerminatedFromLivenessProbe(vmi)
	})

	It("[test_id:CNV82132-006] should suppress GuestAgentPingFailed events during pause", func() {
		vmi := libvmifact.NewFedora(
			libnet.WithMasqueradeNetworking(),
			withLivenessProbe(createGuestAgentPingLivenessProbeWithThreshold(20)),
		)
		vmi = libvmops.RunVMIAndExpectLaunchIgnoreWarnings(vmi, vmiStartTimeout)

		Eventually(matcher.ThisVMI(vmi)).
			WithTimeout(guestAgentConnectTimeout).
			WithPolling(2 * time.Second).
			Should(matcher.HaveConditionTrue(v1.VirtualMachineInstanceAgentConnected))

		patchPauseGuestAgentProbesAnnotationAndWait(vmi, "true")
		events.DeleteEvents(vmi, k8scorev1.EventTypeWarning, "GuestAgentPingFailed")
		Expect(console.LoginToFedora(vmi)).To(Succeed())
		stopGuestAgentAndConfirmDisconnected(vmi)

		events.ConsistentlyExpectNoEvent(vmi, k8scorev1.EventTypeWarning, "GuestAgentPingFailed", 45*time.Second, 2*time.Second)
	})

	It("[test_id:CNV82132-006b] should emit GuestAgentPingFailed events when pause annotation is not set", func() {
		vmi := libvmifact.NewFedora(
			libnet.WithMasqueradeNetworking(),
			withLivenessProbe(createGuestAgentPingLivenessProbeWithThreshold(20)),
		)
		vmi = libvmops.RunVMIAndExpectLaunchIgnoreWarnings(vmi, vmiStartTimeout)

		Eventually(matcher.ThisVMI(vmi)).
			WithTimeout(guestAgentConnectTimeout).
			WithPolling(2 * time.Second).
			Should(matcher.HaveConditionTrue(v1.VirtualMachineInstanceAgentConnected))

		Expect(console.LoginToFedora(vmi)).To(Succeed())
		stopGuestAgentAndConfirmDisconnected(vmi)
		events.ExpectEvent(vmi, k8scorev1.EventTypeWarning, "GuestAgentPingFailed")
	})

	It("[test_id:CNV82132-007] should allow SSH key injection with pause annotation set", func() {
		const secretName = "guest-agent-probe-pause-ssh"
		Expect(createNewSecret(testsuite.GetTestNamespace(nil), secretName, libsecret.DataBytes{
			"my-key": []byte("ssh-rsa AAAAB3NzaC1yc2EAAAABIwAAAQEA6NF8iallvQVp22WDkT test-ssh-key"),
		})).To(Succeed())

		vmi := libvmifact.NewFedora(
			libnet.WithMasqueradeNetworking(),
			withLivenessProbe(createGuestAgentPingLivenessProbeWithThreshold(3)),
			libvmi.WithAccessCredentialSSHPublicKey(secretName, "fedora"),
		)
		vmi = libvmops.RunVMIAndExpectLaunchIgnoreWarnings(vmi, vmiStartTimeout)

		Eventually(matcher.ThisVMI(vmi)).
			WithTimeout(guestAgentConnectTimeout).
			WithPolling(2 * time.Second).
			Should(matcher.HaveConditionTrue(v1.VirtualMachineInstanceAgentConnected))

		Eventually(matcher.ThisVMI(vmi)).
			WithTimeout(2 * time.Minute).
			WithPolling(2 * time.Second).
			Should(matcher.HaveConditionTrue(v1.VirtualMachineInstanceAccessCredentialsSynchronized))

		patchPauseGuestAgentProbesAnnotationAndWait(vmi, "true")

		// rotate key while probes are paused — forces propagation under the annotation
		patchBytes, err := patch.New(
			patch.WithReplace("/data/my-key", base64.StdEncoding.EncodeToString(
				[]byte("ssh-rsa AAAAB3NzaC1yc2EAAAABIwAAAQEA6NF8iallvQVp22WDkT rotated-key"))),
		).GeneratePayload()
		Expect(err).ToNot(HaveOccurred())
		_, err = kubevirt.Client().CoreV1().Secrets(testsuite.GetTestNamespace(vmi)).Patch(
			context.Background(), secretName, types.JSONPatchType, patchBytes, metav1.PatchOptions{})
		Expect(err).ToNot(HaveOccurred())

		Expect(console.LoginToFedora(vmi)).To(Succeed())
		Eventually(func() error {
			return console.SafeExpectBatch(vmi, []expect.Batcher{
				&expect.BSnd{S: "cat /home/fedora/.ssh/authorized_keys\n"},
				&expect.BExp{R: "rotated-key"},
			}, 60)
		}).WithTimeout(2 * time.Minute).WithPolling(5 * time.Second).Should(Succeed())
	})

	It("[test_id:CNV82132-010] should complete snapshot independently of probe pause", func() {
		namespace := testsuite.GetTestNamespace(nil)
		vmi := libvmifact.NewFedora(
			libvmi.WithNamespace(namespace),
			libnet.WithMasqueradeNetworking(),
			withLivenessProbe(createGuestAgentPingProbe(probePeriod, probeInitialSeconds)),
		)
		vm := libvmi.NewVirtualMachine(vmi)
		vm, err := kubevirt.Client().VirtualMachine(namespace).Create(
			context.Background(), vm, metav1.CreateOptions{})
		Expect(err).ToNot(HaveOccurred())
		vm = libvmops.StartVirtualMachine(vm)

		vmi, err = kubevirt.Client().VirtualMachineInstance(vm.Namespace).Get(
			context.Background(), vm.Name, metav1.GetOptions{})
		Expect(err).ToNot(HaveOccurred())

		Eventually(matcher.ThisVMI(vmi)).
			WithTimeout(guestAgentConnectTimeout).
			WithPolling(2 * time.Second).
			Should(matcher.HaveConditionTrue(v1.VirtualMachineInstanceAgentConnected))

		patchPauseGuestAgentProbesAnnotationAndWait(vmi, "true")

		snapshot := libstorage.NewSnapshot(vm.Name, vm.Namespace)
		_, err = kubevirt.Client().VirtualMachineSnapshot(snapshot.Namespace).Create(
			context.Background(), snapshot, metav1.CreateOptions{})
		Expect(err).ToNot(HaveOccurred())
		libstorage.WaitSnapshotSucceeded(kubevirt.Client(), vm.Namespace, snapshot.Name)

		Eventually(matcher.ThisVMI(vmi)).
			WithTimeout(30 * time.Second).
			WithPolling(2 * time.Second).
			Should(matcher.BeInPhase(v1.Running))
	})

	It("[test_id:CNV82132-011] should persist through VM pause and unpause cycle", func() {
		vmi := libvmifact.NewFedora(
			libnet.WithMasqueradeNetworking(),
			withLivenessProbe(createGuestAgentPingProbe(probePeriod, probeInitialSeconds)),
		)
		vmi = libvmops.RunVMIAndExpectLaunchIgnoreWarnings(vmi, vmiStartTimeout)

		Eventually(matcher.ThisVMI(vmi)).
			WithTimeout(guestAgentConnectTimeout).
			WithPolling(2 * time.Second).
			Should(matcher.HaveConditionTrue(v1.VirtualMachineInstanceAgentConnected))

		patchPauseGuestAgentProbesAnnotationAndWait(vmi, "true")
		Expect(console.LoginToFedora(vmi)).To(Succeed())
		stopGuestAgentAndConfirmDisconnected(vmi)

		Expect(kubevirt.Client().VirtualMachineInstance(testsuite.GetTestNamespace(vmi)).
			Pause(context.Background(), vmi.Name, &v1.PauseOptions{})).To(Succeed())
		Eventually(matcher.ThisVMI(vmi)).
			WithTimeout(30 * time.Second).
			WithPolling(2 * time.Second).
			Should(matcher.HaveConditionTrue(v1.VirtualMachineInstancePaused))

		Expect(kubevirt.Client().VirtualMachineInstance(testsuite.GetTestNamespace(vmi)).
			Unpause(context.Background(), vmi.Name, &v1.UnpauseOptions{})).To(Succeed())
		Eventually(matcher.ThisVMI(vmi)).
			WithTimeout(30 * time.Second).
			WithPolling(2 * time.Second).
			Should(matcher.HaveConditionMissingOrFalse(v1.VirtualMachineInstancePaused))

		expectVMIRemainsRunning(vmi, 30*time.Second)

		expectPauseGuestAgentProbesAnnotation(vmi, "true")
	})
}))
