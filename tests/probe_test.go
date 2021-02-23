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
 * Copyright 2021 Red Hat, Inc.
 *
 */

package tests_test

import (
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	kubev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	v1 "kubevirt.io/client-go/api/v1"
	"kubevirt.io/client-go/kubecli"
	"kubevirt.io/kubevirt/tests"
)

var _ = Describe("Exec Probe", func() {

	var err error
	var virtClient kubecli.KubevirtClient

	var (
		LaunchVMI func(*v1.VirtualMachineInstance) *v1.VirtualMachineInstance
	)

	tests.BeforeAll(func() {
		virtClient, err = kubecli.GetKubevirtClient()
		tests.PanicOnError(err)

		LaunchVMI = vmiLauncher(virtClient)
	})

	BeforeEach(func() {
		tests.BeforeTestCleanup()
	})

	Context("with qemu guest agent", func() {
		It("should result in a ready VM", func() {
			vmi := tests.NewRandomFedoraVMIWitGuestAgent()
			vmi.Namespace = tests.NamespaceTestDefault
			vmi.Spec.ReadinessProbe = &v1.Probe{
				Handler:             v1.Handler{Exec: &kubev1.ExecAction{Command: []string{"uname", "-a"}}},
				InitialDelaySeconds: 10,
				TimeoutSeconds:      5,
				PeriodSeconds:       10,
				FailureThreshold:    10,
			}

			LaunchVMI(vmi)

			By("Waiting for agent to connect")
			tests.WaitAgentConnected(virtClient, vmi)

			By("Waiting for the VM to be ready")
			Eventually(func() bool {
				vmi, err := virtClient.VirtualMachineInstance(tests.NamespaceTestDefault).Get(vmi.Name, &metav1.GetOptions{})
				Expect(err).ToNot(HaveOccurred())
				for _, cond := range vmi.Status.Conditions {
					if cond.Type == v1.VirtualMachineInstanceReady && cond.Status == kubev1.ConditionTrue {
						return true
					}
				}
				return false
			}, 120*time.Second, time.Second).Should(BeTrue())
		})
		It("should stop the VM on a failed livenessProbe", func() {
			vmi := tests.NewRandomFedoraVMIWitGuestAgent()
			vmi.Namespace = tests.NamespaceTestDefault
			vmi.Spec.LivenessProbe = &v1.Probe{
				Handler:             v1.Handler{Exec: &kubev1.ExecAction{Command: []string{"exit", "1"}}},
				InitialDelaySeconds: 60,
				TimeoutSeconds:      5,
				PeriodSeconds:       5,
				FailureThreshold:    3,
			}

			LaunchVMI(vmi)

			By("Waiting for agent to connect")
			tests.WaitAgentConnected(virtClient, vmi)

			By("Waiting for the VM to be failed")
			Eventually(func() bool {
				vmi, err := virtClient.VirtualMachineInstance(tests.NamespaceTestDefault).Get(vmi.Name, &metav1.GetOptions{})
				Expect(err).ToNot(HaveOccurred())
				return vmi.Status.Phase == v1.Failed
			}, 180*time.Second, time.Second).Should(BeTrue())
		})
	})
})
