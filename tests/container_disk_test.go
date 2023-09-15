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
 * Copyright 2017 Red Hat, Inc.
 *
 */

package tests_test

import (
	"context"
	"fmt"
	"strings"
	"time"

	"kubevirt.io/kubevirt/tests/decorators"
	"kubevirt.io/kubevirt/tests/framework/kubevirt"
	"kubevirt.io/kubevirt/tests/framework/matcher"

	expect "github.com/google/goexpect"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	k8sv1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"kubevirt.io/kubevirt/tests/exec"
	"kubevirt.io/kubevirt/tests/testsuite"
	"kubevirt.io/kubevirt/tests/util"

	v1 "kubevirt.io/api/core/v1"
	"kubevirt.io/client-go/kubecli"

	"kubevirt.io/kubevirt/tests"
	"kubevirt.io/kubevirt/tests/console"
	cd "kubevirt.io/kubevirt/tests/containerdisk"
	"kubevirt.io/kubevirt/tests/libvmi"
	"kubevirt.io/kubevirt/tests/libwait"
)

var _ = Describe("[rfe_id:588][crit:medium][vendor:cnv-qe@redhat.com][level:component][sig-compute]ContainerDisk", decorators.SigCompute, func() {

	var virtClient kubecli.KubevirtClient
	var err error

	BeforeEach(func() {
		virtClient = kubevirt.Client()
	})

	Describe("[rfe_id:273][crit:medium][vendor:cnv-qe@redhat.com][level:component]Starting and stopping the same VirtualMachine", func() {
		Context("with ephemeral registry disk", func() {
			It("[test_id:1463][Conformance] should success multiple times", func() {
				// TODO We might want to pin the VM into particular node
				By("Creating the VirtualMachine")
				vm := tests.NewRandomVirtualMachine(libvmi.NewCirros(), false)
				var err error
				vm, err = virtClient.VirtualMachine(testsuite.GetTestNamespace(vm)).Create(context.TODO(), vm)
				Expect(err).ToNot(HaveOccurred())
				for i := 0; i < 5; i++ {
					By("Starting the VirtualMachine")
					err := virtClient.VirtualMachine(vm.Namespace).Start(context.TODO(), vm.Name, &v1.StartOptions{})
					Expect(err).ToNot(HaveOccurred())

					By("Waiting for VMI to be running")
					Eventually(matcher.ThisVMIWith(vm.Namespace, vm.Name), 2*time.Minute, 1*time.Second).Should(matcher.BeRunning())

					By("Expecting to be able to login")
					vmi, err := virtClient.VirtualMachineInstance(vm.Namespace).Get(context.TODO(), vm.Name, &metav1.GetOptions{})
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
		Context("[Serial]should obey the disk verification limits in the KubeVirt CR", Serial, func() {
			It("[test_id:7182]disk verification should fail when the memory limit is too low", func() {
				By("Reducing the diskVerificaton memory usage limit")
				kv := util.GetCurrentKv(virtClient)
				kv.Spec.Configuration.DeveloperConfiguration.DiskVerification = &v1.DiskVerification{
					MemoryLimit: resource.NewScaledQuantity(42, resource.Kilo),
				}
				tests.UpdateKubeVirtConfigValueAndWait(kv.Spec.Configuration)
				Expect(err).ToNot(HaveOccurred())

				By("Starting the VirtualMachineInstance")
				vmi := libvmi.NewCirros()
				_, err = virtClient.VirtualMachineInstance(testsuite.GetTestNamespace(vmi)).Create(context.Background(), vmi)
				Expect(err).ToNot(HaveOccurred())
				By("Checking that the VMI failed")
				Eventually(func() bool {
					vmi, err := virtClient.VirtualMachineInstance(testsuite.GetTestNamespace(vmi)).Get(context.Background(), vmi.Name, &metav1.GetOptions{})
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
				vmi := libvmi.NewCirros(overrideCustomLocation)

				By("Starting the VirtualMachineInstance")
				vmi = tests.RunVMIAndExpectLaunch(vmi, 60)

				By("Verify VMI is booted")
				Expect(console.LoginToCirros(vmi)).To(Succeed())
			})
		})

	})

	Describe("[rfe_id:273][crit:medium][vendor:cnv-qe@redhat.com][level:component]Starting with virtio-win", func() {
		Context("with virtio-win as secondary disk", func() {
			It("[test_id:1467]should boot and have the virtio as sata CDROM", func() {
				withCDRom := func(cdRomName string, bus v1.DiskBus, image string) libvmi.Option {
					return func(vmi *v1.VirtualMachineInstance) {
						vmi.Spec.Domain.Devices.Disks = append(vmi.Spec.Domain.Devices.Disks,
							v1.Disk{
								Name: cdRomName,
								DiskDevice: v1.DiskDevice{
									CDRom: &v1.CDRomTarget{
										Bus: bus,
									},
								},
							},
						)
						vmi.Spec.Volumes = append(vmi.Spec.Volumes,
							v1.Volume{
								Name: cdRomName,
								VolumeSource: v1.VolumeSource{
									ContainerDisk: &v1.ContainerDiskSource{
										Image: image,
									},
								},
							},
						)
					}
				}
				vmi := libvmi.NewAlpine(withCDRom("disk4", v1.DiskBusSATA, cd.ContainerDiskFor(cd.ContainerDiskVirtio)))
				vmi = tests.RunVMIAndExpectLaunch(vmi, 60)

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

	// TODO: Could we make this unit test?
	// The main point is - we should not modify container disk - cow
	// What is left is to check qcow - cow
	Describe("[rfe_id:4052][crit:high][arm64][vendor:cnv-qe@redhat.com][level:component]VMI disk permissions", func() {
		Context("with ephemeral registry disk", func() {
			It("[test_id:4299]should not have world write permissions", func() {
				vmi := libvmi.NewAlpine()
				vmi = tests.RunVMIAndExpectLaunch(vmi, 60)

				By("Ensuring VMI is running by logging in")
				libwait.WaitUntilVMIReady(vmi, console.LoginToAlpine)

				By("Fetching virt-launcher Pod")
				pod, err := libvmi.GetPodByVirtualMachineInstance(vmi, testsuite.GetTestNamespace(vmi))
				Expect(err).ToNot(HaveOccurred())

				writableImagePath := fmt.Sprintf("/var/run/kubevirt-ephemeral-disks/disk-data/%v/disk.qcow2", vmi.Spec.Domain.Devices.Disks[0].Name)

				writableImageOctalMode, err := exec.ExecuteCommandOnPod(
					virtClient,
					pod,
					"compute",
					[]string{"/usr/bin/bash", "-c", fmt.Sprintf("stat -c %%a %s", writableImagePath)},
				)
				Expect(err).ToNot(HaveOccurred())

				By("Checking the writable Image Octal mode")
				Expect(strings.Trim(writableImageOctalMode, "\n")).To(Equal("640"), "Octal Mode of writable Image should be 640")

				readonlyImageOctalMode, err := exec.ExecuteCommandOnPod(
					virtClient,
					pod,
					"compute",
					[]string{"/usr/bin/bash", "-c", "stat -c %a /var/run/kubevirt/container-disks/disk_0.img"},
				)
				Expect(err).ToNot(HaveOccurred())

				By("Checking the read-only Image Octal mode")
				Expect(strings.Trim(readonlyImageOctalMode, "\n")).To(Equal("440"), "Octal Mode of read-only Image should be 440")

			})
		})
	})
	Describe("Bogus container disk path", func() {
		Context("that points to outside of the volume", func() {
			It("should be rejected on VMI creation", func() {
				vmi := libvmi.NewAlpine()
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
