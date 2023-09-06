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
	"context"
	"fmt"
	"strings"
	"time"

	"kubevirt.io/kubevirt/tests/libmigration"

	"kubevirt.io/kubevirt/pkg/virt-operator/resource/generate/components"
	"kubevirt.io/kubevirt/tests/libnode"

	"kubevirt.io/kubevirt/tests/decorators"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	k8sv1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"kubevirt.io/kubevirt/tests/exec"
	"kubevirt.io/kubevirt/tests/framework/checks"
	"kubevirt.io/kubevirt/tests/framework/kubevirt"
	"kubevirt.io/kubevirt/tests/testsuite"
	"kubevirt.io/kubevirt/tests/util"

	v1 "kubevirt.io/api/core/v1"
	"kubevirt.io/client-go/kubecli"

	virtconfig "kubevirt.io/kubevirt/pkg/virt-config"
	"kubevirt.io/kubevirt/tests"
	"kubevirt.io/kubevirt/tests/console"
	cd "kubevirt.io/kubevirt/tests/containerdisk"
	"kubevirt.io/kubevirt/tests/libvmi"
	"kubevirt.io/kubevirt/tests/libwait"
)

const (
	capSysNice        k8sv1.Capability = "SYS_NICE"
	capNetBindService k8sv1.Capability = "NET_BIND_SERVICE"
)

var _ = Describe("[Serial][sig-compute]SecurityFeatures", Serial, decorators.SigCompute, func() {
	var err error
	var virtClient kubecli.KubevirtClient

	BeforeEach(func() {
		virtClient = kubevirt.Client()
	})

	Context("Check virt-launcher securityContext", func() {
		var kubevirtConfiguration *v1.KubeVirtConfiguration

		BeforeEach(func() {
			kv := util.GetCurrentKv(virtClient)
			kubevirtConfiguration = &kv.Spec.Configuration
		})

		var container k8sv1.Container
		var vmi *v1.VirtualMachineInstance

		Context("With selinuxLauncherType as container_t", func() {
			BeforeEach(func() {
				config := kubevirtConfiguration.DeepCopy()
				config.SELinuxLauncherType = "container_t"
				tests.UpdateKubeVirtConfigValueAndWait(*config)

				vmi = libvmi.NewCirros()

				// VMIs with selinuxLauncherType container_t cannot have network interfaces, since that requires
				// the `virt_launcher.process` selinux context
				autoattachPodInterface := false
				vmi.Spec.Domain.Devices.AutoattachPodInterface = &autoattachPodInterface
				vmi.Spec.Domain.Devices.Interfaces = []v1.Interface{}
				vmi.Spec.Networks = []v1.Network{}
			})

			It("[test_id:2953]Ensure virt-launcher pod securityContext type is correctly set", func() {

				By("Starting a VirtualMachineInstance")
				vmi, err = virtClient.VirtualMachineInstance(testsuite.GetTestNamespace(nil)).Create(context.Background(), vmi)
				Expect(err).ToNot(HaveOccurred())
				libwait.WaitForSuccessfulVMIStart(vmi)

				By("Check virt-launcher pod SecurityContext values")
				vmiPod := tests.GetRunningPodByVirtualMachineInstance(vmi, testsuite.GetTestNamespace(vmi))
				Expect(vmiPod.Spec.SecurityContext.SELinuxOptions).To(Equal(&k8sv1.SELinuxOptions{Type: "container_t"}))
			})

			It("[test_id:2895]Make sure the virt-launcher pod is not priviledged", func() {

				By("Starting a VirtualMachineInstance")
				vmi, err = virtClient.VirtualMachineInstance(testsuite.GetTestNamespace(nil)).Create(context.Background(), vmi)
				Expect(err).ToNot(HaveOccurred())
				libwait.WaitForSuccessfulVMIStart(vmi)

				By("Check virt-launcher pod SecurityContext values")
				vmiPod := tests.GetRunningPodByVirtualMachineInstance(vmi, testsuite.GetTestNamespace(vmi))
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
				vmi, err = virtClient.VirtualMachineInstance(testsuite.GetTestNamespace(nil)).Create(context.Background(), vmi)
				Expect(err).ToNot(HaveOccurred())
				libwait.WaitForSuccessfulVMIStart(vmi)

				emulator, err := tests.GetRunningVMIEmulator(vmi)
				Expect(err).ToNot(HaveOccurred())
				emulator = "[/]" + strings.TrimPrefix(emulator, "/")

				pod := tests.GetRunningPodByVirtualMachineInstance(vmi, testsuite.GetTestNamespace(vmi))
				qemuProcessSelinuxContext, err := exec.ExecuteCommandOnPod(
					virtClient,
					pod,
					"compute",
					[]string{"/usr/bin/bash", "-c", fmt.Sprintf("ps -efZ | grep %s | awk '{print $1}'", emulator)},
				)
				Expect(err).ToNot(HaveOccurred())

				By("Checking that qemu process is of the SELinux type container_t")
				Expect(strings.Split(qemuProcessSelinuxContext, ":")[2]).To(Equal("container_t"))

				By("Checking that qemu process has SELinux category_set")
				Expect(strings.Split(qemuProcessSelinuxContext, ":")).To(HaveLen(5))

				err = virtClient.VirtualMachineInstance(testsuite.GetTestNamespace(vmi)).Delete(context.Background(), vmi.Name, &metav1.DeleteOptions{})
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
				vmi.Namespace = testsuite.NamespacePrivileged

				By("Starting a New VMI")
				vmi, err = virtClient.VirtualMachineInstance(testsuite.NamespacePrivileged).Create(context.Background(), vmi)
				Expect(err).ToNot(HaveOccurred())
				libwait.WaitForSuccessfulVMIStart(vmi)

				By("Ensuring VMI is running by logging in")
				libwait.WaitUntilVMIReady(vmi, console.LoginToAlpine)

				By("Fetching virt-launcher Pod")
				pod, err := libvmi.GetPodByVirtualMachineInstance(vmi, testsuite.NamespacePrivileged)
				Expect(err).ToNot(HaveOccurred())

				By("Verifying SELinux context contains custom type")
				Expect(pod.Spec.SecurityContext.SELinuxOptions.Type).To(Equal(superPrivilegedType))

				By("Deleting the VMI")
				err = virtClient.VirtualMachineInstance(testsuite.NamespacePrivileged).Delete(context.Background(), vmi.Name, &metav1.DeleteOptions{})
				Expect(err).ToNot(HaveOccurred())
			})
		})

		Context("With selinuxLauncherType defined as virt_launcher.process", func() {

			It("[test_id:4298]qemu process type is virt_launcher.process, when selinuxLauncherType is virt_launcher.process", decorators.CustomSELinux, func() {
				config := kubevirtConfiguration.DeepCopy()
				launcherType := "virt_launcher.process"
				config.SELinuxLauncherType = launcherType
				tests.UpdateKubeVirtConfigValueAndWait(*config)

				vmi = tests.NewRandomVMIWithEphemeralDisk(cd.ContainerDiskFor(cd.ContainerDiskAlpine))
				vmi.Namespace = testsuite.NamespacePrivileged

				By("Starting a New VMI")
				vmi, err = virtClient.VirtualMachineInstance(testsuite.NamespacePrivileged).Create(context.Background(), vmi)
				Expect(err).ToNot(HaveOccurred())
				libwait.WaitForSuccessfulVMIStart(vmi)

				By("Ensuring VMI is running by logging in")
				libwait.WaitUntilVMIReady(vmi, console.LoginToAlpine)

				By("Fetching virt-launcher Pod")
				emulator, err := tests.GetRunningVMIEmulator(vmi)
				Expect(err).ToNot(HaveOccurred())
				emulator = "[/]" + strings.TrimPrefix(emulator, "/")

				pod, err := libvmi.GetPodByVirtualMachineInstance(vmi, testsuite.NamespacePrivileged)
				Expect(err).ToNot(HaveOccurred())
				qemuProcessSelinuxContext, err := exec.ExecuteCommandOnPod(
					virtClient,
					pod,
					"compute",
					[]string{"/usr/bin/bash", "-c", fmt.Sprintf("ps -efZ | grep %s | awk '{print $1}'", emulator)},
				)
				Expect(err).ToNot(HaveOccurred())

				By("Checking that qemu process is of the SELinux type virt_launcher.process")
				Expect(strings.Split(qemuProcessSelinuxContext, ":")[2]).To(Equal(launcherType))

				By("Verifying SELinux context contains custom type in pod")
				Expect(pod.Spec.SecurityContext.SELinuxOptions.Type).To(Equal(launcherType))

				By("Deleting the VMI")
				err = virtClient.VirtualMachineInstance(testsuite.NamespacePrivileged).Delete(context.Background(), vmi.Name, &metav1.DeleteOptions{})
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
			vmi, err = virtClient.VirtualMachineInstance(testsuite.GetTestNamespace(nil)).Create(context.Background(), vmi)
			Expect(err).ToNot(HaveOccurred())
			libwait.WaitForSuccessfulVMIStart(vmi)

			By("Ensuring VMI is running by logging in")
			libwait.WaitUntilVMIReady(vmi, console.LoginToAlpine)

			By("Fetching virt-launcher Pod")
			pod, err := libvmi.GetPodByVirtualMachineInstance(vmi, testsuite.GetTestNamespace(vmi))
			Expect(err).ToNot(HaveOccurred())

			for _, containerSpec := range pod.Spec.Containers {
				if containerSpec.Name == "compute" {
					container = containerSpec
					break
				}
			}
			caps := *container.SecurityContext.Capabilities
			if !checks.HasFeature(virtconfig.Root) {
				Expect(caps.Add).To(HaveLen(1), fmt.Sprintf("Found capabilities %s, expected NET_BIND_SERVICE", caps.Add))
				Expect(caps.Add).To(ContainElement(k8sv1.Capability("NET_BIND_SERVICE")))
			} else {
				Expect(caps.Add).To(HaveLen(2), fmt.Sprintf("Found capabilities %s, expected NET_BIND_SERVICE and SYS_NICE", caps.Add))
				Expect(caps.Add).To(ContainElements(k8sv1.Capability("NET_BIND_SERVICE"), k8sv1.Capability("SYS_NICE")))
			}

			By("Checking virt-launcher Pod's compute container has precisely the documented extra capabilities")
			for _, capa := range caps.Add {
				Expect(isLauncherCapabilityValid(capa)).To(BeTrue(), "Expected compute container of virt_launcher to be granted only specific capabilities")
			}

			if !checks.HasFeature(virtconfig.Root) {
				By("Checking virt-launcher Pod's compute container has precisely the documented dropped capabilities")
				Expect(caps.Drop).To(HaveLen(1))
				Expect(caps.Drop[0]).To(Equal(k8sv1.Capability("ALL")), "Expected compute container of virt_launcher to drop all caps")
			}
		})
	})
	Context("Disabling the custom SELinux policy", func() {
		var policyRemovedByTest = false
		AfterEach(func() {
			if policyRemovedByTest {
				By("Re-installing custom SELinux policy on all nodes")
				err = runOnAllSchedulableNodes(virtClient, []string{"cp", "/var/run/kubevirt/virt_launcher.cil", "/proc/1/root/tmp/"}, "")
				// That file may not be deployed on clusters that don't need the policy anymore
				if err == nil {
					err = runOnAllSchedulableNodes(virtClient, []string{"chroot", "/proc/1/root", "semodule", "-i", "/tmp/virt_launcher.cil"}, "")
					Expect(err).ToNot(HaveOccurred())
					err = runOnAllSchedulableNodes(virtClient, []string{"rm", "-f", "/proc/1/root/tmp/virt_launcher.cil"}, "")
					Expect(err).ToNot(HaveOccurred())
				}
			}
		})

		It("Should prevent virt-handler from installing the custom policy", func() {
			By("Removing custom SELinux policy from all nodes")
			// The policy may or may not be installed on the node, regardless of the feature gate value,
			// since the feature gate could have been enabled after deployment. Use error as indication of removal.
			err = runOnAllSchedulableNodes(virtClient, []string{"chroot", "/proc/1/root", "semodule", "-r", "virt_launcher"}, "")
			policyRemovedByTest = err == nil

			By("Disabling the custom policy by adding the corresponding feature gate")
			tests.EnableFeatureGate(virtconfig.DisableCustomSELinuxPolicy)

			By("Ensuring the custom SELinux policy is absent from all nodes")
			Consistently(func() error {
				return runOnAllSchedulableNodes(virtClient, []string{"chroot", "/proc/1/root", "semodule", "-l"}, "virt_launcher")
			}, 30*time.Second, 10*time.Second).Should(BeNil())
		})
	})
	Context("The VMI SELinux context status", func() {
		It("Should get set and stay the the same after a migration", decorators.RequiresTwoSchedulableNodes, func() {
			vmi := libvmi.NewAlpine(libvmi.WithMasqueradeNetworking()...)

			By("Starting a New VMI")
			vmi, err = virtClient.VirtualMachineInstance(testsuite.GetTestNamespace(nil)).Create(context.Background(), vmi)
			Expect(err).NotTo(HaveOccurred())
			libwait.WaitForSuccessfulVMIStart(vmi)

			By("Ensuring VMI is running by logging in")
			libwait.WaitUntilVMIReady(vmi, console.LoginToAlpine)

			By("Ensuring the VMI SELinux context status gets set")
			seContext := ""
			Eventually(func() string {
				vmi, err = virtClient.VirtualMachineInstance(testsuite.GetTestNamespace(nil)).Get(context.Background(), vmi.Name, &metav1.GetOptions{})
				Expect(err).NotTo(HaveOccurred())
				seContext = vmi.Status.SelinuxContext
				return seContext
			}, 30*time.Second, 10*time.Second).ShouldNot(BeEmpty(), "VMI SELinux context status never got set")

			By("Ensuring the VMI SELinux context status matches the virt-launcher pod files")
			stdout := tests.RunCommandOnVmiPod(vmi, []string{"ls", "-lZd", "/"})
			Expect(stdout).To(ContainSubstring(seContext))

			By("Migrating the VMI")
			migration := tests.NewRandomMigration(vmi.Name, vmi.Namespace)
			libmigration.RunMigrationAndExpectToCompleteWithDefaultTimeout(virtClient, migration)

			By("Ensuring the VMI SELinux context status didn't change")
			vmi, err = virtClient.VirtualMachineInstance(testsuite.GetTestNamespace(nil)).Get(context.Background(), vmi.Name, &metav1.GetOptions{})
			Expect(err).NotTo(HaveOccurred())
			Expect(vmi.Status.SelinuxContext).To(Equal(seContext))

			By("Fetching virt-launcher Pod")
			pod, err := virtClient.CoreV1().Pods(vmi.Namespace).Get(context.Background(), vmi.Status.MigrationState.TargetPod, metav1.GetOptions{})
			Expect(err).ToNot(HaveOccurred())

			By("Ensuring the right SELinux context is set on the target pod")
			Expect(pod.Spec.SecurityContext).NotTo(BeNil())
			Expect(pod.Spec.SecurityContext.SELinuxOptions).NotTo(BeNil(), fmt.Sprintf("%#v", pod.Spec.SecurityContext))
			ctx := strings.Split(seContext, ":")
			Expect(ctx).To(HaveLen(5))
			Expect(pod.Spec.SecurityContext.SELinuxOptions.Level).To(Equal(strings.Join(ctx[3:], ":")))

			By("Ensuring the target virt-launcher has the same SELinux context as the source")
			stdout = tests.RunCommandOnVmiPod(vmi, []string{"ls", "-lZd", "/"})
			Expect(stdout).To(ContainSubstring(seContext))
		})
	})
})

func runOnAllSchedulableNodes(virtClient kubecli.KubevirtClient, command []string, forbiddenString string) error {
	nodes := libnode.GetAllSchedulableNodes(virtClient)
	for _, node := range nodes.Items {
		pod, err := libnode.GetVirtHandlerPod(virtClient, node.Name)
		if err != nil {
			return err
		}
		stdout, stderr, err := exec.ExecuteCommandOnPodWithResults(virtClient, pod, components.VirtHandlerName, command)
		if err != nil {
			_, _ = GinkgoWriter.Write([]byte(stderr))
			return err
		}
		if forbiddenString != "" {
			if strings.Contains(stdout, forbiddenString) {
				return fmt.Errorf("found unexpected %s on node %s", forbiddenString, node.Name)
			}
		}
	}

	return nil
}

func isLauncherCapabilityValid(capability k8sv1.Capability) bool {
	switch capability {
	case
		capNetBindService, capSysNice:
		return true
	}
	return false
}
