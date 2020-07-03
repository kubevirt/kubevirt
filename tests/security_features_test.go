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
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	k8sv1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	v1 "kubevirt.io/client-go/api/v1"
	"kubevirt.io/client-go/kubecli"
	virtconfig "kubevirt.io/kubevirt/pkg/virt-config"
	"kubevirt.io/kubevirt/tests"
)

var _ = Describe("SecurityFeatures", func() {
	tests.FlagParse()

	virtClient, err := kubecli.GetKubevirtClient()
	tests.PanicOnError(err)

	BeforeEach(func() {
		tests.BeforeTestCleanup()
	})

	Context("Check virt-launcher securityContext", func() {

		var container k8sv1.Container
		var vmi *v1.VirtualMachineInstance

		Context("With selinuxLauncherType undefined", func() {
			BeforeEach(func() {
				tests.UpdateClusterConfigValueAndWait(virtconfig.SELinuxLauncherTypeKey, "")
				vmi = tests.NewRandomVMIWithEphemeralDiskAndUserdata(tests.ContainerDiskFor(tests.ContainerDiskCirros), "#!/bin/bash\necho 'hello'\n")
			})

			It("[test_id:2953]Ensure virt-launcher pod securityContext type is not forced", func() {

				By("Starting a VirtualMachineInstance")
				vmi, err = virtClient.VirtualMachineInstance(tests.NamespaceTestDefault).Create(vmi)
				Expect(err).ToNot(HaveOccurred())
				tests.WaitForSuccessfulVMIStart(vmi)

				By("Check virt-launcher pod SecurityContext values")
				vmiPod := tests.GetRunningPodByVirtualMachineInstance(vmi, tests.NamespaceTestDefault)
				Expect(vmiPod.Spec.SecurityContext.SELinuxOptions).To(BeNil())
			})

			It("[test_id:2895]Make sure the virt-launcher pod is not priviledged", func() {

				By("Starting a VirtualMachineInstance")
				vmi, err = virtClient.VirtualMachineInstance(tests.NamespaceTestDefault).Create(vmi)
				Expect(err).ToNot(HaveOccurred())
				tests.WaitForSuccessfulVMIStart(vmi)

				By("Check virt-launcher pod SecurityContext values")
				vmiPod := tests.GetRunningPodByVirtualMachineInstance(vmi, tests.NamespaceTestDefault)
				for _, containerSpec := range vmiPod.Spec.Containers {
					if containerSpec.Name == "compute" {
						container = containerSpec
						break
					}
				}
				Expect(*container.SecurityContext.Privileged).To(BeFalse())
			})
		})

		Context("With selinuxLauncherType defined", func() {
			var superPrivilegedType string

			BeforeEach(func() {
				superPrivilegedType = "spc_t"
				tests.UpdateClusterConfigValueAndWait(virtconfig.SELinuxLauncherTypeKey, superPrivilegedType)
				vmi = tests.NewRandomVMIWithEphemeralDisk(tests.ContainerDiskFor(tests.ContainerDiskAlpine))
			})

			It("[test_id:3787]Should honor custom SELinux type for virt-launcher", func() {
				By("Starting a New VMI")
				vmi, err = virtClient.VirtualMachineInstance(tests.NamespaceTestDefault).Create(vmi)
				Expect(err).ToNot(HaveOccurred())
				tests.WaitForSuccessfulVMIStart(vmi)

				By("Ensuring VMI is running by logging in")
				tests.WaitUntilVMIReady(vmi, tests.LoggedInAlpineExpecter)

				By("Fetching virt-launcher Pod")
				pod := tests.GetPodByVirtualMachineInstance(vmi, tests.NamespaceTestDefault)

				By("Verifying SELinux context contains custom type")
				Expect(pod.Spec.SecurityContext.SELinuxOptions.Type).To(Equal(superPrivilegedType))

				By("Deleting the VMI")
				err = virtClient.VirtualMachineInstance(tests.NamespaceTestDefault).Delete(vmi.Name, &metav1.DeleteOptions{})
				Expect(err).ToNot(HaveOccurred())
			})
		})
	})

	Context("Check virt-launcher capabilities", func() {
		var container k8sv1.Container
		var vmi *v1.VirtualMachineInstance

		It("[test_id:4300]has precisely the documented extra capabilities relative to a regular user pod", func() {
			vmi = tests.NewRandomVMIWithEphemeralDisk(tests.ContainerDiskFor(tests.ContainerDiskAlpine))

			By("Starting a New VMI")
			vmi, err = virtClient.VirtualMachineInstance(tests.NamespaceTestDefault).Create(vmi)
			Expect(err).ToNot(HaveOccurred())
			tests.WaitForSuccessfulVMIStart(vmi)

			By("Ensuring VMI is running by logging in")
			tests.WaitUntilVMIReady(vmi, tests.LoggedInAlpineExpecter)

			By("Fetching virt-launcher Pod")
			pod := tests.GetPodByVirtualMachineInstance(vmi, tests.NamespaceTestDefault)

			for _, containerSpec := range pod.Spec.Containers {
				if containerSpec.Name == "compute" {
					container = containerSpec
					break
				}
			}
			caps := *container.SecurityContext.Capabilities
			Expect(len(caps.Add)).To(Equal(3))

			By("Checking virt-launcher Pod's compute container has precisely the documented extra capabilities")
			for _, cap := range caps.Add {
				Expect(tests.IsLauncherCapabilityValid(cap)).To(BeTrue(), "Expected compute container of virt_launcher to be granted only specific capabilities")
			}
		})
	})
})
