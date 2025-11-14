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

package tests_test

import (
	"context"
	"fmt"
	"strings"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	k8sv1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	v1 "kubevirt.io/api/core/v1"
	"kubevirt.io/client-go/kubecli"

	"kubevirt.io/kubevirt/pkg/virt-config/featuregate"
	"kubevirt.io/kubevirt/tests/console"
	"kubevirt.io/kubevirt/tests/decorators"
	"kubevirt.io/kubevirt/tests/exec"
	"kubevirt.io/kubevirt/tests/framework/checks"
	"kubevirt.io/kubevirt/tests/framework/k8s"
	"kubevirt.io/kubevirt/tests/framework/kubevirt"
	"kubevirt.io/kubevirt/tests/libdomain"
	"kubevirt.io/kubevirt/tests/libkubevirt"
	kvconfig "kubevirt.io/kubevirt/tests/libkubevirt/config"
	"kubevirt.io/kubevirt/tests/libmigration"
	"kubevirt.io/kubevirt/tests/libnet"
	"kubevirt.io/kubevirt/tests/libpod"
	"kubevirt.io/kubevirt/tests/libvmifact"
	"kubevirt.io/kubevirt/tests/libwait"
	"kubevirt.io/kubevirt/tests/testsuite"
)

var _ = Describe("[sig-compute]SecurityFeatures", decorators.SigCompute, func() {
	var err error
	var virtClient kubecli.KubevirtClient

	BeforeEach(func() {
		virtClient = kubevirt.Client()
	})

	Context("Check virt-launcher securityContext", Serial, func() {
		var kubevirtConfiguration *v1.KubeVirtConfiguration

		BeforeEach(func() {
			kv := libkubevirt.GetCurrentKv(virtClient)
			kubevirtConfiguration = &kv.Spec.Configuration
		})

		var vmi *v1.VirtualMachineInstance

		Context("With selinuxLauncherType as container_t", func() {
			BeforeEach(func() {
				config := kubevirtConfiguration.DeepCopy()
				config.SELinuxLauncherType = "container_t"
				kvconfig.UpdateKubeVirtConfigValueAndWait(*config)

				vmi = libvmifact.NewCirros()
			})

			It("[test_id:2953][test_id:2895]Ensure virt-launcher pod securityContext type is correctly set and not privileged", func() {

				By("Starting a VirtualMachineInstance")
				vmi, err = virtClient.VirtualMachineInstance(testsuite.GetTestNamespace(nil)).Create(context.Background(), vmi, metav1.CreateOptions{})
				Expect(err).ToNot(HaveOccurred())
				libwait.WaitForSuccessfulVMIStart(vmi)

				By("Check virt-launcher pod SecurityContext values")
				vmiPod, err := libpod.GetPodByVirtualMachineInstance(vmi, vmi.Namespace)
				Expect(err).NotTo(HaveOccurred())

				Expect(vmiPod.Spec.SecurityContext.SELinuxOptions).To(Equal(&k8sv1.SELinuxOptions{Type: "container_t"}))
			})

			It("[test_id:4297]Make sure qemu processes are MCS constrained", func() {

				By("Starting a VirtualMachineInstance")
				vmi, err = virtClient.VirtualMachineInstance(testsuite.GetTestNamespace(nil)).Create(context.Background(), vmi, metav1.CreateOptions{})
				Expect(err).ToNot(HaveOccurred())
				libwait.WaitForSuccessfulVMIStart(vmi)

				domSpec, err := libdomain.GetRunningVMIDomainSpec(vmi)
				Expect(err).ToNot(HaveOccurred())

				emulator := "[/]" + strings.TrimPrefix(domSpec.Devices.Emulator, "/")

				pod, err := libpod.GetPodByVirtualMachineInstance(vmi, vmi.Namespace)
				Expect(err).NotTo(HaveOccurred())

				qemuProcessSelinuxContext, err := exec.ExecuteCommandOnPod(
					pod,
					"compute",
					[]string{"/usr/bin/bash", "-c", fmt.Sprintf("ps -efZ | grep %s | awk '{print $1}'", emulator)},
				)
				Expect(err).ToNot(HaveOccurred())

				By("Checking that qemu process is of the SELinux type container_t")
				Expect(strings.Split(qemuProcessSelinuxContext, ":")[2]).To(Equal("container_t"))

				By("Checking that qemu process has SELinux category_set")
				Expect(strings.Split(qemuProcessSelinuxContext, ":")).To(HaveLen(5))

				err = virtClient.VirtualMachineInstance(testsuite.GetTestNamespace(vmi)).Delete(context.Background(), vmi.Name, metav1.DeleteOptions{})
				Expect(err).ToNot(HaveOccurred())
			})
		})

		Context("With selinuxLauncherType defined as spc_t", func() {

			It("[test_id:3787]Should honor custom SELinux type for virt-launcher", func() {
				config := kubevirtConfiguration.DeepCopy()
				superPrivilegedType := "spc_t"
				config.SELinuxLauncherType = superPrivilegedType
				kvconfig.UpdateKubeVirtConfigValueAndWait(*config)

				vmi = libvmifact.NewAlpine()

				By("Starting a New VMI")
				vmi, err = virtClient.VirtualMachineInstance(testsuite.NamespacePrivileged).Create(context.Background(), vmi, metav1.CreateOptions{})
				Expect(err).ToNot(HaveOccurred())
				libwait.WaitForSuccessfulVMIStart(vmi)

				By("Ensuring VMI is running by logging in")
				libwait.WaitUntilVMIReady(vmi, console.LoginToAlpine)

				By("Fetching virt-launcher Pod")
				pod, err := libpod.GetPodByVirtualMachineInstance(vmi, testsuite.NamespacePrivileged)
				Expect(err).ToNot(HaveOccurred())

				By("Verifying SELinux context contains custom type")
				Expect(pod.Spec.SecurityContext.SELinuxOptions.Type).To(Equal(superPrivilegedType))

				By("Deleting the VMI")
				err = virtClient.VirtualMachineInstance(testsuite.NamespacePrivileged).Delete(context.Background(), vmi.Name, metav1.DeleteOptions{})
				Expect(err).ToNot(HaveOccurred())
			})
		})
	})

	Context("Check virt-launcher capabilities", func() {
		var container k8sv1.Container

		It("[test_id:4300]has precisely the documented extra capabilities relative to a regular user pod", decorators.Conformance, func() {
			vmi := libvmifact.NewAlpine()

			By("Starting a New VMI")
			vmi, err = virtClient.VirtualMachineInstance(testsuite.GetTestNamespace(nil)).Create(context.Background(), vmi, metav1.CreateOptions{})
			Expect(err).ToNot(HaveOccurred())
			libwait.WaitForSuccessfulVMIStart(vmi)

			By("Ensuring VMI is running by logging in")
			libwait.WaitUntilVMIReady(vmi, console.LoginToAlpine)

			By("Fetching virt-launcher Pod")
			pod, err := libpod.GetPodByVirtualMachineInstance(vmi, testsuite.GetTestNamespace(vmi))
			Expect(err).ToNot(HaveOccurred())

			for _, containerSpec := range pod.Spec.Containers {
				if containerSpec.Name == "compute" {
					container = containerSpec
					break
				}
			}
			caps := *container.SecurityContext.Capabilities
			if !checks.HasFeature(featuregate.Root) {
				Expect(caps.Add).To(ConsistOf(k8sv1.Capability("NET_BIND_SERVICE")))
				By("Checking virt-launcher Pod's compute container has precisely the documented dropped capabilities")
				Expect(caps.Drop).To(ConsistOf(k8sv1.Capability("ALL")), "Expected compute container of virt_launcher to drop all caps")
			} else {
				Expect(caps.Add).To(ConsistOf(k8sv1.Capability("NET_BIND_SERVICE"), k8sv1.Capability("SYS_NICE")))
				Expect(caps.Drop).To(BeEmpty())
			}
		})
	})

	Context("The VMI SELinux context status", func() {
		It("Should get set and stay the the same after a migration", decorators.RequiresTwoSchedulableNodes, func() {
			vmi := libvmifact.NewAlpine(libnet.WithMasqueradeNetworking())

			By("Starting a New VMI")
			vmi, err = virtClient.VirtualMachineInstance(testsuite.GetTestNamespace(nil)).Create(context.Background(), vmi, metav1.CreateOptions{})
			Expect(err).NotTo(HaveOccurred())
			libwait.WaitForSuccessfulVMIStart(vmi)

			By("Ensuring VMI is running by logging in")
			libwait.WaitUntilVMIReady(vmi, console.LoginToAlpine)

			By("Ensuring the VMI SELinux context status gets set")
			seContext := ""
			Eventually(func() string {
				vmi, err = virtClient.VirtualMachineInstance(testsuite.GetTestNamespace(nil)).Get(context.Background(), vmi.Name, metav1.GetOptions{})
				Expect(err).NotTo(HaveOccurred())
				seContext = vmi.Status.SelinuxContext
				return seContext
			}, 30*time.Second, 10*time.Second).ShouldNot(BeEmpty(), "VMI SELinux context status never got set")

			By("Ensuring the VMI SELinux context status matches the virt-launcher pod files")
			stdout := libpod.RunCommandOnVmiPod(vmi, []string{"ls", "-lZd", "/"})
			Expect(stdout).To(ContainSubstring(seContext))

			By("Migrating the VMI")
			migration := libmigration.New(vmi.Name, vmi.Namespace)
			libmigration.RunMigrationAndExpectToCompleteWithDefaultTimeout(virtClient, migration)

			By("Ensuring the VMI SELinux context status didn't change")
			vmi, err = virtClient.VirtualMachineInstance(testsuite.GetTestNamespace(nil)).Get(context.Background(), vmi.Name, metav1.GetOptions{})
			Expect(err).NotTo(HaveOccurred())
			Expect(vmi.Status.SelinuxContext).To(Equal(seContext))

			By("Fetching virt-launcher Pod")
			pod, err := k8s.Client().CoreV1().Pods(vmi.Namespace).Get(context.Background(), vmi.Status.MigrationState.TargetPod, metav1.GetOptions{})
			Expect(err).ToNot(HaveOccurred())

			By("Ensuring the right SELinux context is set on the target pod")
			Expect(pod.Spec.SecurityContext).NotTo(BeNil())
			Expect(pod.Spec.SecurityContext.SELinuxOptions).NotTo(BeNil(), fmt.Sprintf("%#v", pod.Spec.SecurityContext))
			ctx := strings.Split(seContext, ":")
			Expect(ctx).To(HaveLen(5))
			Expect(pod.Spec.SecurityContext.SELinuxOptions.Level).To(Equal(strings.Join(ctx[3:], ":")))

			By("Ensuring the target virt-launcher has the same SELinux context as the source")
			stdout = libpod.RunCommandOnVmiPod(vmi, []string{"ls", "-lZd", "/"})
			Expect(stdout).To(ContainSubstring(seContext))
		})
	})
})
