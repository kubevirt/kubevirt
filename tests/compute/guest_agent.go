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
 * Copyright The KubeVirt Authors
 *
 */

package compute

import (
	"context"
	"fmt"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	expect "github.com/google/goexpect"

	k8scorev1 "k8s.io/api/core/v1"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	v1 "kubevirt.io/api/core/v1"

	"kubevirt.io/kubevirt/pkg/libvmi"
	"kubevirt.io/kubevirt/tests/console"
	"kubevirt.io/kubevirt/tests/decorators"
	"kubevirt.io/kubevirt/tests/framework/kubevirt"
	"kubevirt.io/kubevirt/tests/framework/matcher"
	"kubevirt.io/kubevirt/tests/libnet"
	"kubevirt.io/kubevirt/tests/libvmifact"
	"kubevirt.io/kubevirt/tests/libvmops"
	"kubevirt.io/kubevirt/tests/testsuite"
)

var _ = Describe(SIG("GuestAgent", decorators.GuestAgentProbes, func() {
	Context("Readiness Probe", func() {
		const (
			period         = 5
			initialSeconds = 5
			timeoutSeconds = 1
		)

		It("should succeed", func() {
			readinessProbe := createExecProbe(period, initialSeconds, timeoutSeconds, "uname", "-a")
			vmi := libvmifact.NewFedora(
				libnet.WithMasqueradeNetworking(),
				withReadinessProbe(readinessProbe),
			)
			vmi = libvmops.RunVMIAndExpectLaunchIgnoreWarnings(vmi, 180)

			By("Waiting for agent to connect")
			Eventually(matcher.ThisVMI(vmi)).
				WithTimeout(12 * time.Minute).
				WithPolling(2 * time.Second).
				Should(matcher.HaveConditionTrue(v1.VirtualMachineInstanceAgentConnected))

			Eventually(matcher.ThisVMI(vmi)).
				WithTimeout(2 * time.Minute).
				WithPolling(2 * time.Second).
				Should(matcher.HaveConditionTrue(v1.VirtualMachineInstanceReady))
		})

		DescribeTable("Should fail", func(readinessProbe *v1.Probe) {
			vmi := libvmifact.NewFedora(
				libnet.WithMasqueradeNetworking(),
				withReadinessProbe(readinessProbe),
			)
			vmi = libvmops.RunVMIAndExpectLaunchIgnoreWarnings(vmi, 180)

			By("Checking that the VMI is consistently non-ready")
			Consistently(matcher.ThisVMI(vmi)).
				WithTimeout(30 * time.Second).
				WithPolling(100 * time.Millisecond).
				Should(matcher.HaveConditionMissingOrFalse(v1.VirtualMachineInstanceReady))
		},
			Entry("with working Exec probe and invalid command",
				createExecProbe(period, initialSeconds, timeoutSeconds, "exit", "1"),
			),
			Entry("with working Exec probe and infinitely running command",
				createExecProbe(period, initialSeconds, timeoutSeconds, "tail", "-f", "/dev/null"),
			),
		)
	})

	Context("Readiness probe with guest agent ping", func() {
		var vmi *v1.VirtualMachineInstance

		const (
			period         = 5
			initialSeconds = 5
		)

		BeforeEach(func() {
			vmi = libvmifact.NewFedora(libnet.WithMasqueradeNetworking(), withReadinessProbe(createGuestAgentPingProbe(period, initialSeconds)))
			vmi = libvmops.RunVMIAndExpectLaunchIgnoreWarnings(vmi, 180)
			By("Waiting for agent to connect")
			Eventually(matcher.ThisVMI(vmi), 12*time.Minute, 2*time.Second).Should(matcher.HaveConditionTrue(v1.VirtualMachineInstanceAgentConnected))

			Eventually(matcher.ThisVMI(vmi), 2*time.Minute, 2*time.Second).Should(matcher.HaveConditionTrue(v1.VirtualMachineInstanceReady))
			By("Disabling the guest-agent")
			Expect(console.LoginToFedora(vmi)).To(Succeed())
			Expect(stopGuestAgent(vmi)).To(Succeed())
			Eventually(matcher.ThisVMI(vmi)).
				WithTimeout(5 * time.Minute).
				WithPolling(2 * time.Second).
				Should(matcher.HaveConditionMissingOrFalse(v1.VirtualMachineInstanceReady))
		})

		When("the guest agent is enabled, after being disabled", func() {
			BeforeEach(func() {
				Expect(console.LoginToFedora(vmi)).To(Succeed())
				Expect(startGuestAgent(vmi)).To(Succeed())
			})

			It("[test_id:6741] the VMI enters `Ready` state once again", func() {
				Eventually(matcher.ThisVMI(vmi)).
					WithTimeout(2 * time.Minute).
					WithPolling(2 * time.Second).
					Should(matcher.HaveConditionTrue(v1.VirtualMachineInstanceReady))
			})
		})
	})

	Context("Liveness probe", func() {
		const (
			period         = 5
			initialSeconds = 90
			timeoutSeconds = 1
		)

		It("Should not fail the VMI", func() {
			livenessProbe := createExecProbe(period, initialSeconds, timeoutSeconds, "uname", "-a")
			vmi := libvmifact.NewFedora(libnet.WithMasqueradeNetworking(), withLivenessProbe(livenessProbe))
			vmi = libvmops.RunVMIAndExpectLaunchIgnoreWarnings(vmi, 180)

			By("Waiting for agent to connect")
			Eventually(matcher.ThisVMI(vmi)).
				WithTimeout(12 * time.Minute).
				WithPolling(2 * time.Second).
				Should(matcher.HaveConditionTrue(v1.VirtualMachineInstanceAgentConnected))

			By("Checking that the VMI is still running after a while")
			Consistently(func() bool {
				vmi, err := kubevirt.Client().VirtualMachineInstance(testsuite.GetTestNamespace(vmi)).Get(context.Background(), vmi.Name, metav1.GetOptions{})
				Expect(err).ToNot(HaveOccurred())
				return vmi.IsFinal()
			}).WithTimeout(2 * time.Minute).
				WithPolling(1 * time.Second).
				Should(Not(BeTrue()))
		})

		It("Should fail the VMI with working Exec probe and invalid command", func() {
			livenessProbe := createExecProbe(period, initialSeconds, timeoutSeconds, "exit", "1")
			vmi := libvmifact.NewFedora(withLivenessProbe(livenessProbe))
			vmi = libvmops.RunVMIAndExpectLaunchIgnoreWarnings(vmi, 180)

			By("Checking that the VMI is in a final state after a while")
			Eventually(func() bool {
				vmi, err := kubevirt.Client().VirtualMachineInstance(testsuite.GetTestNamespace(vmi)).Get(context.Background(), vmi.Name, metav1.GetOptions{})
				Expect(err).ToNot(HaveOccurred())
				return vmi.IsFinal()
			}).WithTimeout(2 * time.Minute).
				WithPolling(1 * time.Second).
				Should(BeTrue())
		})
	})

	Context("Liveness probe with guest agent ping", func() {
		var vmi *v1.VirtualMachineInstance

		const (
			period         = 5
			initialSeconds = 90
		)

		BeforeEach(func() {
			vmi = libvmifact.NewFedora(libnet.WithMasqueradeNetworking(), withLivenessProbe(createGuestAgentPingProbe(period, initialSeconds)))
			vmi = libvmops.RunVMIAndExpectLaunchIgnoreWarnings(vmi, 180)

			By("Waiting for agent to connect")
			Eventually(matcher.ThisVMI(vmi)).
				WithTimeout(12 * time.Minute).
				WithPolling(2 * time.Second).
				Should(matcher.HaveConditionTrue(v1.VirtualMachineInstanceAgentConnected))
			Expect(console.LoginToFedora(vmi)).To(Succeed())
		})

		It("[test_id:9299] VM stops when guest agent is disabled", func() {
			Expect(stopGuestAgent(vmi)).To(Succeed())

			Eventually(func() (*v1.VirtualMachineInstance, error) {
				return kubevirt.Client().VirtualMachineInstance(testsuite.GetTestNamespace(vmi)).Get(context.Background(), vmi.Name, metav1.GetOptions{})
			}).WithTimeout(2 * time.Minute).
				WithPolling(1 * time.Second).
				Should(Or(matcher.BeInPhase(v1.Failed), matcher.HaveSucceeded()))
		})
	})
}))

func createExecProbe(period, initialSeconds, timeoutSeconds int32, command ...string) *v1.Probe {
	execHandler := v1.Handler{Exec: &k8scorev1.ExecAction{Command: command}}
	return createProbeSpecification(period, initialSeconds, timeoutSeconds, execHandler)
}

func createGuestAgentPingProbe(period, initialSeconds int32) *v1.Probe {
	handler := v1.Handler{GuestAgentPing: &v1.GuestAgentPing{}}
	return createProbeSpecification(period, initialSeconds, 1, handler)
}

func createProbeSpecification(period, initialSeconds, timeoutSeconds int32, handler v1.Handler) *v1.Probe {
	return &v1.Probe{
		PeriodSeconds:       period,
		InitialDelaySeconds: initialSeconds,
		Handler:             handler,
		TimeoutSeconds:      timeoutSeconds,
	}
}

func withReadinessProbe(probe *v1.Probe) libvmi.Option {
	return func(vmi *v1.VirtualMachineInstance) {
		vmi.Spec.ReadinessProbe = probe
	}
}

func withLivenessProbe(probe *v1.Probe) libvmi.Option {
	return func(vmi *v1.VirtualMachineInstance) {
		vmi.Spec.LivenessProbe = probe
	}
}

const (
	startAgent = "start"
	stopAgent  = "stop"
)

func startGuestAgent(vmi *v1.VirtualMachineInstance) error {
	return guestAgentOperation(vmi, startAgent)
}

func stopGuestAgent(vmi *v1.VirtualMachineInstance) error {
	return guestAgentOperation(vmi, stopAgent)
}

func guestAgentOperation(vmi *v1.VirtualMachineInstance, startStopOperation string) error {
	if startStopOperation != startAgent && startStopOperation != stopAgent {
		return fmt.Errorf("invalid qemu-guest-agent request: %s. Allowed values are: '%s' *or* '%s'", startStopOperation, startAgent, stopAgent)
	}
	guestAgentSysctlString := fmt.Sprintf("sudo systemctl %s qemu-guest-agent\n", startStopOperation)
	return console.SafeExpectBatch(vmi, []expect.Batcher{
		&expect.BSnd{S: guestAgentSysctlString},
		&expect.BExp{R: ""},
	}, 120)
}
