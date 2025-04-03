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
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	k8scorev1 "k8s.io/api/core/v1"

	v1 "kubevirt.io/api/core/v1"

	"kubevirt.io/kubevirt/pkg/libvmi"
	"kubevirt.io/kubevirt/tests/framework/matcher"
	"kubevirt.io/kubevirt/tests/libnet"
	"kubevirt.io/kubevirt/tests/libvmifact"
	"kubevirt.io/kubevirt/tests/libvmops"
)

var _ = Describe(SIG("GuestAgent", func() {
	Context("Readiness Probe", func() {
		It("should succeed", func() {
			const (
				period         = 5
				initialSeconds = 5
				timeoutSeconds = 1
			)

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
	})
}))

func createExecProbe(period, initialSeconds, timeoutSeconds int32, command ...string) *v1.Probe {
	execHandler := v1.Handler{Exec: &k8scorev1.ExecAction{Command: command}}
	return createProbeSpecification(period, initialSeconds, timeoutSeconds, execHandler)
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
