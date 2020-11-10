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
	"strings"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	k8sv1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	v1 "kubevirt.io/client-go/api/v1"
	"kubevirt.io/client-go/kubecli"
	"kubevirt.io/kubevirt/tests"
	cd "kubevirt.io/kubevirt/tests/containerdisk"
)

var _ = Describe("[Serial]SecurityFeatures", func() {
	var err error
	var virtClient kubecli.KubevirtClient

	BeforeEach(func() {
		virtClient, err = kubecli.GetKubevirtClient()
		tests.PanicOnError(err)

		tests.BeforeTestCleanup()
	})

	Context("Check virt-launcher securityContext", func() {
		var kubevirtConfiguration *v1.KubeVirtConfiguration

		tests.BeforeAll(func() {
			kv := tests.GetCurrentKv(virtClient)
			kubevirtConfiguration = &kv.Spec.Configuration

			tests.SkipSELinuxTestIfRunnigOnKindInfra()
		})

		var container k8sv1.Container
		var vmi *v1.VirtualMachineInstance

		Context("With selinuxLauncherType as container_t", func() {
			BeforeEach(func() {
				config := kubevirtConfiguration.DeepCopy()
				config.SELinuxLauncherType = "container_t"
				tests.UpdateKubeVirtConfigValueAndWait(*config)

				vmi = tests.NewRandomVMIWithEphemeralDiskAndUserdata(cd.ContainerDiskFor(cd.ContainerDiskCirros), "#!/bin/bash\necho 'hello'\n")

				// VMIs with selinuxLauncherType container_t cannot have network interfaces, since that requires
				// the `virt_launcher.process` selinux context
				autoattachPodInterface := false
				vmi.Spec.Domain.Devices.AutoattachPodInterface = &autoattachPodInterface
				vmi.Spec.Domain.Devices.Interfaces = []v1.Interface{}
				vmi.Spec.Networks = []v1.Network{}
			})

			It("[test_id:2953]Ensure virt-launcher pod securityContext type is correctly set", func() {

				By("Starting a VirtualMachineInstance")
				vmi, err = virtClient.VirtualMachineInstance(tests.NamespaceTestDefault).Create(vmi)
				Expect(err).ToNot(HaveOccurred())
				tests.WaitForSuccessfulVMIStart(vmi)

				By("Check virt-launcher pod SecurityContext values")
				vmiPod := tests.GetRunningPodByVirtualMachineInstance(vmi, tests.NamespaceTestDefault)
				Expect(vmiPod.Spec.SecurityContext.SELinuxOptions).To(Equal(&k8sv1.SELinuxOptions{Type: "container_t"}))
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

			It("[test_id:4297]Make sure qemu processes are MCS constrained", func() {

				By("Starting a VirtualMachineInstance")
				vmi, err = virtClient.VirtualMachineInstance(tests.NamespaceTestDefault).Create(vmi)
				Expect(err).ToNot(HaveOccurred())
				tests.WaitForSuccessfulVMIStart(vmi)

				pod := tests.GetRunningPodByVirtualMachineInstance(vmi, tests.NamespaceTestDefault)

				qemuProcessSelinuxContext, err := tests.ExecuteCommandOnPod(
					virtClient,
					pod,
					"compute",
					[]string{"/usr/bin/bash", "-c", "ps -efZ | grep [/]usr/libexec/qemu-kvm | awk '{print $1}'"},
				)
				Expect(err).ToNot(HaveOccurred())

				By("Checking that qemu-kvm process is of the SELinux type container_t")
				Expect(strings.Split(qemuProcessSelinuxContext, ":")[2]).To(Equal("container_t"))

				By("Checking that qemu-kvm process has SELinux category_set")
				Expect(len(strings.Split(qemuProcessSelinuxContext, ":"))).To(Equal(5))

				err = virtClient.VirtualMachineInstance(tests.NamespaceTestDefault).Delete(vmi.Name, &metav1.DeleteOptions{})
				Expect(err).ToNot(HaveOccurred())
			})
		})

		Context("With selinuxLauncherType defined as spc_t", func() {

			It("[test_id:3787]Should honor custom SELinux type for virt-launcher", func() {
				config := kubevirtConfiguration.DeepCopy()
				superPrivilegedType := "spc_t"
				config.SELinuxLauncherType = superPrivilegedType
				tests.UpdateKubeVirtConfigValueAndWait(*config)

				vmi = tests.NewRandomVMIWithEphemeralDisk(cd.ContainerDiskFor(cd.ContainerDiskAlpine))

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

		Context("With selinuxLauncherType defined as virt_launcher.process", func() {

			It("[test_id:4298]qemu process type is virt_launcher.process, when selinuxLauncherType is virt_launcher.process", func() {
				config := kubevirtConfiguration.DeepCopy()
				launcherType := "virt_launcher.process"
				config.SELinuxLauncherType = launcherType
				tests.UpdateKubeVirtConfigValueAndWait(*config)

				vmi = tests.NewRandomVMIWithEphemeralDisk(cd.ContainerDiskFor(cd.ContainerDiskAlpine))

				By("Starting a New VMI")
				vmi, err = virtClient.VirtualMachineInstance(tests.NamespaceTestDefault).Create(vmi)
				Expect(err).ToNot(HaveOccurred())
				tests.WaitForSuccessfulVMIStart(vmi)

				By("Ensuring VMI is running by logging in")
				tests.WaitUntilVMIReady(vmi, tests.LoggedInAlpineExpecter)

				By("Fetching virt-launcher Pod")
				pod := tests.GetPodByVirtualMachineInstance(vmi, tests.NamespaceTestDefault)

				qemuProcessSelinuxContext, err := tests.ExecuteCommandOnPod(
					virtClient,
					pod,
					"compute",
					[]string{"/usr/bin/bash", "-c", "ps -efZ | grep [/]usr/libexec/qemu-kvm | awk '{print $1}'"},
				)
				Expect(err).ToNot(HaveOccurred())

				By("Checking that qemu-kvm process is of the SELinux type virt_launcher.process")
				Expect(strings.Split(qemuProcessSelinuxContext, ":")[2]).To(Equal(launcherType))

				By("Verifying SELinux context contains custom type in pod")
				Expect(pod.Spec.SecurityContext.SELinuxOptions.Type).To(Equal(launcherType))

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
			vmi = tests.NewRandomVMIWithEphemeralDisk(cd.ContainerDiskFor(cd.ContainerDiskAlpine))

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
			Expect(len(caps.Add)).To(Equal(2))

			By("Checking virt-launcher Pod's compute container has precisely the documented extra capabilities")
			for _, cap := range caps.Add {
				Expect(tests.IsLauncherCapabilityValid(cap)).To(BeTrue(), "Expected compute container of virt_launcher to be granted only specific capabilities")
			}
			By("Checking virt-launcher Pod's compute container has precisely the documented dropped capabilities")
			Expect(len(caps.Drop)).To(Equal(1))
			for _, cap := range caps.Drop {
				Expect(tests.IsLauncherCapabilityDropped(cap)).To(BeTrue(), "Expected compute container of virt_launcher to drop only specific capabilities")
			}
		})
	})
})
