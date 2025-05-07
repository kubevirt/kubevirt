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
	"strings"
	"time"

	expect "github.com/google/goexpect"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	k8sv1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	v1 "kubevirt.io/api/core/v1"
	"kubevirt.io/client-go/kubecli"

	"kubevirt.io/kubevirt/pkg/libvmi"

	"kubevirt.io/kubevirt/tests/console"
	cd "kubevirt.io/kubevirt/tests/containerdisk"
	"kubevirt.io/kubevirt/tests/decorators"
	"kubevirt.io/kubevirt/tests/framework/kubevirt"
	"kubevirt.io/kubevirt/tests/framework/matcher"
	"kubevirt.io/kubevirt/tests/libkubevirt"
	"kubevirt.io/kubevirt/tests/libkubevirt/config"
	"kubevirt.io/kubevirt/tests/libvmifact"
	"kubevirt.io/kubevirt/tests/libvmops"
	"kubevirt.io/kubevirt/tests/libwait"
	"kubevirt.io/kubevirt/tests/testsuite"
)

var _ = Describe("[rfe_id:588][crit:medium][vendor:cnv-qe@redhat.com][level:component][sig-compute]ContainerDisk", decorators.SigCompute, func() {

	var virtClient kubecli.KubevirtClient

	BeforeEach(func() {
		virtClient = kubevirt.Client()
	})

	Describe("[rfe_id:273][crit:medium][vendor:cnv-qe@redhat.com][level:component]Starting and stopping the same VirtualMachine", func() {
		Context("with ephemeral registry disk", func() {
			It("[test_id:1463] should success multiple times", decorators.Conformance, func() {
				By("Creating the VirtualMachine")
				vm := libvmi.NewVirtualMachine(libvmifact.NewCirros())
				vm, err := virtClient.VirtualMachine(testsuite.GetTestNamespace(vm)).Create(context.TODO(), vm, metav1.CreateOptions{})
				Expect(err).ToNot(HaveOccurred())
				for range 5 {
					By("Starting the VirtualMachine")
					err := virtClient.VirtualMachine(vm.Namespace).Start(context.TODO(), vm.Name, &v1.StartOptions{})
					Expect(err).ToNot(HaveOccurred())

					By("Waiting for VMI to be running")
					Eventually(matcher.ThisVMIWith(vm.Namespace, vm.Name), 2*time.Minute, 1*time.Second).Should(matcher.BeRunning())

					By("Expecting to be able to login")
					vmi, err := virtClient.VirtualMachineInstance(vm.Namespace).Get(context.TODO(), vm.Name, metav1.GetOptions{})
					Expect(err).ToNot(HaveOccurred())
					Expect(console.LoginToCirros(vmi)).To(Succeed())

					By("Stopping the VirtualMachine")
					err = virtClient.VirtualMachine(vm.Namespace).Stop(context.TODO(), vm.Name, &v1.StopOptions{})
					Expect(err).ToNot(HaveOccurred())

					By("Waiting until the VirtualMachineInstance is gone")
					libwait.WaitForVirtualMachineToDisappearWithTimeout(vmi, 120)
				}
			})
		})
	})

	Describe("[rfe_id:273][crit:medium][vendor:cnv-qe@redhat.com][level:component]Starting a VirtualMachineInstance", func() {
		Context("should obey the disk verification limits in the KubeVirt CR", Serial, func() {
			It("[test_id:7182]disk verification should fail when the memory limit is too low", func() {
				By("Reducing the diskVerificaton memory usage limit")
				kv := libkubevirt.GetCurrentKv(virtClient)
				kv.Spec.Configuration.DeveloperConfiguration.DiskVerification = &v1.DiskVerification{
					MemoryLimit: resource.NewScaledQuantity(42, resource.Kilo),
				}
				config.UpdateKubeVirtConfigValueAndWait(kv.Spec.Configuration)

				By("Starting the VirtualMachineInstance")
				vmi := libvmifact.NewCirros()
				_, err := virtClient.VirtualMachineInstance(testsuite.GetTestNamespace(vmi)).Create(context.Background(), vmi, metav1.CreateOptions{})
				Expect(err).ToNot(HaveOccurred())
				By("Checking that the VMI failed")
				Eventually(func() bool {
					vmi, err := virtClient.VirtualMachineInstance(testsuite.GetTestNamespace(vmi)).Get(context.Background(), vmi.Name, metav1.GetOptions{})
					Expect(err).ToNot(HaveOccurred())
					for _, condition := range vmi.Status.Conditions {
						if condition.Type == v1.VirtualMachineInstanceSynchronized && condition.Status == k8sv1.ConditionFalse {
							return strings.Contains(condition.Message, "failed to invoke qemu-img")
						}
					}
					return false
				}, 3*time.Minute, 2*time.Second).Should(BeTrue())
			})
		})
	})

	Describe("[rfe_id:273][crit:medium][vendor:cnv-qe@redhat.com][level:component]Starting from custom image location", func() {
		Context("with disk at /custom-disk/downloaded", func() {

			It("[test_id:1466]should boot normally", func() {
				overrideCustomLocation := func(vmi *v1.VirtualMachineInstance) {
					vmi.Spec.Volumes[0].ContainerDisk.Image = cd.ContainerDiskFor(cd.ContainerDiskCirrosCustomLocation)
					vmi.Spec.Volumes[0].ContainerDisk.Path = "/custom-disk/downloaded"
				}

				By("Starting the VirtualMachineInstance")
				vmi := libvmops.RunVMIAndExpectLaunch(libvmifact.NewCirros(overrideCustomLocation), 60)

				By("Verify VMI is booted")
				Expect(console.LoginToCirros(vmi)).To(Succeed())
			})
		})

	})

	Describe("[rfe_id:273][crit:medium][vendor:cnv-qe@redhat.com][level:component]Starting with virtio-win", func() {
		Context("with virtio-win as secondary disk", func() {
			It("[test_id:1467]should boot and have the virtio as sata CDROM", func() {
				vmi := libvmifact.NewAlpine(
					libvmi.WithEphemeralCDRom("disk4", v1.DiskBusSATA, cd.ContainerDiskFor(cd.ContainerDiskVirtio)),
				)
				vmi = libvmops.RunVMIAndExpectLaunch(vmi, 60)

				By("Checking whether the second disk really contains virtio drivers")
				Expect(console.LoginToAlpine(vmi)).To(Succeed(), "expected alpine to login properly")

				Expect(console.SafeExpectBatch(vmi, []expect.Batcher{
					// mount virtio cdrom and check files are there
					&expect.BSnd{S: "mount -t iso9600 /dev/cdrom\n"},
					&expect.BExp{R: console.PromptExpression},
					&expect.BSnd{S: "echo $?\n"},
					&expect.BExp{R: console.RetValue("0")},
					&expect.BSnd{S: "cd /media/cdrom\n"},
					&expect.BExp{R: console.PromptExpression},
					&expect.BSnd{S: "ls virtio-win_license.txt guest-agent\n"},
					&expect.BExp{R: console.PromptExpression},
					&expect.BSnd{S: "echo $?\n"},
					&expect.BExp{R: console.RetValue("0")},
				}, 200)).To(Succeed(), "expected virtio files to be mounted properly")
			})
		})
	})

	Describe("Bogus container disk path", func() {
		Context("that points to outside of the volume", func() {
			//TODO this could be unit test
			It("should be rejected on VMI creation", func() {
				vmi := libvmifact.NewAlpine()
				vmi.Spec.Volumes[0].ContainerDisk.Path = "../test"
				By("Starting the VirtualMachineInstance")
				_, err := virtClient.RestClient().Post().Resource("virtualmachineinstances").Namespace(testsuite.GetTestNamespace(vmi)).Body(vmi).Do(context.Background()).Get()
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("admission webhook"))
				Expect(err.Error()).To(ContainSubstring("denied the request"))
				Expect(err.Error()).To(ContainSubstring("must be an absolute path to a file without relative components"))
			})
		})
	})
})
