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

	"kubevirt.io/kubevirt/pkg/libvmi"

	"kubevirt.io/kubevirt/tests/decorators"
	"kubevirt.io/kubevirt/tests/framework/kubevirt"

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
	"kubevirt.io/kubevirt/tests/libpod"
	"kubevirt.io/kubevirt/tests/libvmifact"
	"kubevirt.io/kubevirt/tests/libwait"
)

var _ = Describe("[rfe_id:588][crit:medium][vendor:cnv-qe@redhat.com][level:component][sig-compute]ContainerDisk", decorators.SigCompute, func() {

	var virtClient kubecli.KubevirtClient
	var err error

	BeforeEach(func() {
		virtClient = kubevirt.Client()
	})

	DescribeTable("should", func(image string, policy k8sv1.PullPolicy, expectedPolicy k8sv1.PullPolicy) {
		vmi := libvmifact.NewGuestless(libvmi.WithContainerDisk("disk0", image))

		vmi.Spec.Volumes[0].ContainerDisk.ImagePullPolicy = policy

		vmi = tests.RunVMIAndExpectScheduling(vmi, 60)
		Expect(vmi.Spec.Volumes[0].ContainerDisk.ImagePullPolicy).To(Equal(expectedPolicy))
		pod, err := libpod.GetPodByVirtualMachineInstance(vmi, vmi.Namespace)
		Expect(err).ToNot(HaveOccurred())
		container := getContainerDiskContainerOfPod(pod, vmi.Spec.Volumes[0].Name)
		Expect(container.ImagePullPolicy).To(Equal(expectedPolicy))
	},
		Entry("[test_id:3246]generate and set Always pull policy", "test", k8sv1.PullPolicy(""), k8sv1.PullAlways),
		Entry("[test_id:3247]generate and set Always pull policy", "test:latest", k8sv1.PullPolicy(""), k8sv1.PullAlways),
		Entry("[test_id:3248]generate and set IfNotPresent pull policy", "test@sha256:9c2b78e11c25b3fd0b24b0ed684a112052dff03eee4ca4bdcc4f3168f9a14396", k8sv1.PullPolicy(""), k8sv1.PullIfNotPresent),
		Entry("[test_id:3249]pass through Never pull policy to the pod", "test@sha256:9c2b78e11c25b3fd0b24b0ed684a112052dff03eee4ca4bdcc4f3168f9a14396", k8sv1.PullNever, k8sv1.PullNever),
		Entry("[test_id:3250]pass through IfNotPresent pull policy to the pod", "test:latest", k8sv1.PullIfNotPresent, k8sv1.PullIfNotPresent),
	)

	Describe("[rfe_id:273][crit:medium][vendor:cnv-qe@redhat.com][level:component]Starting and stopping the same VirtualMachineInstance", func() {
		Context("with ephemeral registry disk", func() {
			It("[test_id:1463] should success multiple times", decorators.Conformance, func() {
				vmi := tests.NewRandomVMIWithEphemeralDiskAndUserdata(cd.ContainerDiskFor(cd.ContainerDiskCirros), "#!/bin/bash\necho 'hello'\n")
				num := 2
				for i := 0; i < num; i++ {
					By("Starting the VirtualMachineInstance")
					obj, err := virtClient.RestClient().Post().Resource("virtualmachineinstances").Namespace(testsuite.GetTestNamespace(vmi)).Body(vmi).Do(context.Background()).Get()
					Expect(err).ToNot(HaveOccurred())
					vmiObj, ok := obj.(*v1.VirtualMachineInstance)
					Expect(ok).To(BeTrue(), "Object is not of type *v1.VirtualMachineInstance")
					libwait.WaitForSuccessfulVMIStart(vmiObj)

					By("Stopping the VirtualMachineInstance")
					_, err = virtClient.RestClient().Delete().Resource("virtualmachineinstances").Namespace(vmi.GetObjectMeta().GetNamespace()).Name(vmi.GetObjectMeta().GetName()).Do(context.Background()).Get()
					Expect(err).ToNot(HaveOccurred())
					By("Waiting until the VirtualMachineInstance is gone")
					libwait.WaitForVirtualMachineToDisappearWithTimeout(vmi, 120)
				}
			})
		})
	})

	Describe("[rfe_id:273][crit:medium][vendor:cnv-qe@redhat.com][level:component]Starting a VirtualMachineInstance", func() {
		Context("with ephemeral registry disk", func() {
			It("[test_id:1464]should not modify the spec on status update", func() {
				vmi := libvmifact.NewCirros()
				v1.SetObjectDefaults_VirtualMachineInstance(vmi)

				By("Starting the VirtualMachineInstance")
				vmi, err := virtClient.VirtualMachineInstance(testsuite.GetTestNamespace(vmi)).Create(context.Background(), vmi, metav1.CreateOptions{})
				Expect(err).ToNot(HaveOccurred())
				libwait.WaitForSuccessfulVMIStart(vmi)
				startedVMI, err := virtClient.VirtualMachineInstance(testsuite.GetTestNamespace(vmi)).Get(context.Background(), vmi.ObjectMeta.Name, metav1.GetOptions{})
				Expect(err).ToNot(HaveOccurred())
				By("Checking that the VirtualMachineInstance spec did not change")
				Expect(startedVMI.Spec).To(Equal(vmi.Spec))
			})
		})
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
				vmi := libvmifact.NewCirros()
				_, err = virtClient.VirtualMachineInstance(testsuite.GetTestNamespace(vmi)).Create(context.Background(), vmi, metav1.CreateOptions{})
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

	Describe("[rfe_id:273][crit:medium][vendor:cnv-qe@redhat.com][level:component]Starting multiple VMIs", func() {
		Context("with ephemeral registry disk", func() {
			It("[test_id:1465]should success", func() {
				const count = 5
				var vmis []*v1.VirtualMachineInstance
				for i := 0; i < count; i++ {
					// Provide 1Mi of memory to prevent VMIs from actually booting.
					// We only care about the volume containers inside the virt-launcher Pod.
					vmi := libvmifact.NewCirros(libvmi.WithResourceMemory("1Mi"))
					vmi, err := virtClient.VirtualMachineInstance(testsuite.GetTestNamespace(vmi)).Create(context.Background(), vmi, metav1.CreateOptions{})
					Expect(err).ToNot(HaveOccurred())
					vmis = append(vmis, vmi)
				}

				By("Verifying the containerdisks are online")
				for _, vmi := range vmis {
					libwait.WaitForSuccessfulVMIStart(vmi)
					pods, err := virtClient.CoreV1().Pods(testsuite.GetTestNamespace(vmi)).List(context.Background(), tests.UnfinishedVMIPodSelector(vmi))
					Expect(err).ToNot(HaveOccurred())
					Expect(hasContainerDisk(pods.Items)).To(BeTrue())
				}
			})
		})
	})

	Describe("[rfe_id:273][crit:medium][vendor:cnv-qe@redhat.com][level:component]Starting from custom image location", func() {
		Context("with disk at /custom-disk/downloaded", func() {
			It("[test_id:1466]should boot normally", func() {
				vmi := tests.NewRandomVMIWithEphemeralDiskAndUserdata(cd.ContainerDiskFor(cd.ContainerDiskCirrosCustomLocation), "#!/bin/bash\necho 'hello'\n")
				for ind, volume := range vmi.Spec.Volumes {
					if volume.ContainerDisk != nil {
						vmi.Spec.Volumes[ind].ContainerDisk.Path = "/custom-disk/downloaded"
					}
				}
				By("Starting the VirtualMachineInstance")
				vmi = tests.RunVMIAndExpectLaunch(vmi, 60)
			})
		})

	})

	Describe("[rfe_id:273][crit:medium][vendor:cnv-qe@redhat.com][level:component]Starting with virtio-win", func() {
		Context("with virtio-win as secondary disk", func() {
			It("[test_id:1467]should boot and have the virtio as sata CDROM", func() {
				vmi := libvmifact.NewAlpine(
					libvmi.WithEphemeralCDRom("disk4", v1.DiskBusSATA, cd.ContainerDiskFor(cd.ContainerDiskVirtio)),
				)
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

	Describe("[rfe_id:4052][crit:high][arm64][vendor:cnv-qe@redhat.com][level:component]VMI disk permissions", func() {
		Context("with ephemeral registry disk", func() {
			It("[test_id:4299]should not have world write permissions", func() {
				vmi := libvmifact.NewAlpine()
				vmi = tests.RunVMIAndExpectLaunch(vmi, 60)

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

func getContainerDiskContainerOfPod(pod *k8sv1.Pod, volumeName string) *k8sv1.Container {
	diskContainerName := fmt.Sprintf("volume%s", volumeName)
	return libpod.LookupContainer(pod, diskContainerName)
}

func hasContainerDisk(pods []k8sv1.Pod) bool {
	for _, pod := range pods {
		if pod.ObjectMeta.DeletionTimestamp != nil {
			continue
		}
		for _, containerStatus := range pod.Status.ContainerStatuses {
			// only check readiness of containerdisk container
			if strings.HasPrefix(containerStatus.Name, "volume") &&
				containerStatus.Ready && containerStatus.State.Running != nil {
				return true
			}
		}
	}

	return false
}
