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
	"encoding/json"
	"fmt"
	"strings"
	"time"

	k8sversion "k8s.io/apimachinery/pkg/version"

	expect "github.com/google/goexpect"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	k8sv1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	v1 "kubevirt.io/api/core/v1"
	"kubevirt.io/client-go/kubecli"

	"kubevirt.io/kubevirt/pkg/libvmi"
	"kubevirt.io/kubevirt/pkg/virt-config/featuregate"

	"kubevirt.io/kubevirt/tests/console"
	cd "kubevirt.io/kubevirt/tests/containerdisk"
	"kubevirt.io/kubevirt/tests/decorators"
	"kubevirt.io/kubevirt/tests/exec"
	"kubevirt.io/kubevirt/tests/framework/kubevirt"
	"kubevirt.io/kubevirt/tests/framework/matcher"
	"kubevirt.io/kubevirt/tests/libkubevirt"
	"kubevirt.io/kubevirt/tests/libkubevirt/config"
	"kubevirt.io/kubevirt/tests/libmigration"
	"kubevirt.io/kubevirt/tests/libpod"
	"kubevirt.io/kubevirt/tests/libvmifact"
	"kubevirt.io/kubevirt/tests/libvmops"
	"kubevirt.io/kubevirt/tests/libwait"
	"kubevirt.io/kubevirt/tests/testsuite"
	"kubevirt.io/kubevirt/tools/vms-generator/utils"
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

	Describe("[rfe_id:4052][crit:high][vendor:cnv-qe@redhat.com][level:component]VMI disk permissions", decorators.WgS390x, decorators.WgArm64, func() {
		Context("with ephemeral registry disk", func() {
			It("[test_id:4299]should not have world write permissions", func() {
				vmi := libvmops.RunVMIAndExpectLaunch(libvmifact.NewAlpine(), 60)

				By("Ensuring VMI is running by logging in")
				libwait.WaitUntilVMIReady(vmi, console.LoginToAlpine)

				By("Fetching virt-launcher Pod")
				pod, err := libpod.GetPodByVirtualMachineInstance(vmi, testsuite.GetTestNamespace(vmi))
				Expect(err).ToNot(HaveOccurred())

				writableImagePath := fmt.Sprintf("/var/run/kubevirt-ephemeral-disks/disk-data/%v/disk.qcow2", vmi.Spec.Domain.Devices.Disks[0].Name)

				writableImageOctalMode, err := exec.ExecuteCommandOnPod(
					pod,
					"compute",
					[]string{"/usr/bin/bash", "-c", fmt.Sprintf("stat -c %%a %s", writableImagePath)},
				)
				Expect(err).ToNot(HaveOccurred())

				By("Checking the writable Image Octal mode")
				Expect(strings.Trim(writableImageOctalMode, "\n")).To(Equal("640"), "Octal Mode of writable Image should be 640")

				readonlyImageOctalMode, err := exec.ExecuteCommandOnPod(
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

	Describe("Simulate an upgrade from a version where ImageVolume was disabled to a version where it is enabled", Serial, func() {
		BeforeEach(func() {
			v, err := getKubernetesVersion()
			Expect(err).ToNot(HaveOccurred())
			if v < "1.33" {
				// Skip the test if the Kubernetes version is lower than 1.33
				// ImageVolume won't work for versions < 1.33 because of this bug:
				// https://github.com/kubernetes/kubernetes/pull/130394
				// Additionally there is currently no way to enable k8s ImageVolume FG
				// through kubevirtci. It is enabled by default since 1.33.
				Skip("this test requires Kubernetes version >= 1.33")
			}
		})

		// TODO: Remove the PDescribeTable in favor of DescribeTable once ImageVolume k8s FG is enabled by default
		PDescribeTable("Migration from a source launcher with the bind mount workaround to a target launcher without the bind mount workaround should succeed when ", func(vmi *v1.VirtualMachineInstance) {
			config.DisableFeatureGate(featuregate.ImageVolume)
			vmi = libvmops.RunVMIAndExpectLaunch(vmi, 60)
			By("Fetching virt-launcher pod without ImageVolume")
			sourcePod, err := libpod.GetPodByVirtualMachineInstance(vmi, vmi.Namespace)
			Expect(sourcePod.Spec.InitContainers).ToNot(BeEmpty(), "without ImageVolume should include container-disk-binary init container to copy the container-disk binary")
			config.EnableFeatureGate(featuregate.ImageVolume)
			By("Starting new migration and waiting for it to succeed")
			migration := libmigration.New(vmi.Name, vmi.Namespace)
			migration = libmigration.RunMigrationAndExpectToCompleteWithDefaultTimeout(virtClient, migration)

			By("Verifying Migration Succeeeds")
			libmigration.ConfirmVMIPostMigration(virtClient, vmi, migration)

			By("Fetching virt-launcher pod with ImageVolume")
			vmi, err = virtClient.VirtualMachineInstance(testsuite.GetTestNamespace(vmi)).Get(context.Background(), vmi.Name, metav1.GetOptions{})
			Expect(err).ToNot(HaveOccurred())
			targetPod, err := libpod.GetPodByVirtualMachineInstance(vmi, vmi.Namespace)
			Expect(err).ToNot(HaveOccurred())
			Expect(targetPod.Spec.InitContainers).To(BeEmpty(), "with ImageVolume should not include container-disk-binary init container")
		},
			Entry("using simple Cirros vmi",
				libvmifact.NewCirros(
					libvmi.WithInterface(libvmi.InterfaceDeviceWithMasqueradeBinding()),
					libvmi.WithNetwork(v1.DefaultPodNetwork()),
				),
			),
			Entry("using  Cirros vmi with custom location", decorators.Periodic,
				libvmifact.NewCirros(
					libvmi.WithInterface(libvmi.InterfaceDeviceWithMasqueradeBinding()),
					libvmi.WithNetwork(v1.DefaultPodNetwork()),
					overrideCustomLocation,
				),
			),
			Entry("using  Cirros vmi with kernel boot", decorators.Periodic,
				newCirrosWithKernelBoot(),
			),
		)
	})
})

func overrideCustomLocation(vmi *v1.VirtualMachineInstance) {
	vmi.Spec.Volumes[0].ContainerDisk.Image = cd.ContainerDiskFor(cd.ContainerDiskCirrosCustomLocation)
	vmi.Spec.Volumes[0].ContainerDisk.Path = "/custom-disk/downloaded"
}

func newCirrosWithKernelBoot() *v1.VirtualMachineInstance {
	vmi := libvmifact.NewCirros(
		libvmi.WithInterface(libvmi.InterfaceDeviceWithMasqueradeBinding()),
		libvmi.WithNetwork(v1.DefaultPodNetwork()),
	)
	utils.AddKernelBootToVMI(vmi)
	return vmi
}

func getKubernetesVersion() (string, error) {
	var info k8sversion.Info
	virtClient, err := kubecli.GetKubevirtClient()
	if err != nil {
		return "", err
	}
	response, err := virtClient.RestClient().Get().AbsPath("/version").DoRaw(context.Background())
	if err != nil {
		return "", err
	}
	if err := json.Unmarshal(response, &info); err != nil {
		return "", err
	}
	curVersion := strings.Split(info.GitVersion, "+")[0]
	curVersion = strings.Trim(curVersion, "v")
	return curVersion, nil
}
