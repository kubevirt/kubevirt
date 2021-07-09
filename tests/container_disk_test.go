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

	expect "github.com/google/goexpect"
	. "github.com/onsi/ginkgo"
	"github.com/onsi/ginkgo/extensions/table"
	. "github.com/onsi/gomega"
	k8sv1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"

	v1 "kubevirt.io/client-go/api/v1"
	"kubevirt.io/client-go/kubecli"
	"kubevirt.io/kubevirt/tests"
	"kubevirt.io/kubevirt/tests/console"
	cd "kubevirt.io/kubevirt/tests/containerdisk"
	"kubevirt.io/kubevirt/tests/libvmi"
)

var _ = Describe("[rfe_id:588][crit:medium][vendor:cnv-qe@redhat.com][level:component][sig-compute]ContainerDisk", func() {

	var virtClient kubecli.KubevirtClient
	var err error

	BeforeEach(func() {
		virtClient, err = kubecli.GetKubevirtClient()
		tests.PanicOnError(err)

		tests.BeforeTestCleanup()
	})

	LaunchVMI := func(vmi *v1.VirtualMachineInstance) runtime.Object {
		By("Starting a VirtualMachineInstance")
		obj, err := virtClient.RestClient().Post().Resource("virtualmachineinstances").Namespace(tests.NamespaceTestDefault).Body(vmi).Do(context.Background()).Get()
		Expect(err).To(BeNil())
		return obj
	}

	VerifyContainerDiskVMI := func(vmi *v1.VirtualMachineInstance, obj runtime.Object, ignoreWarnings bool) {
		_, ok := obj.(*v1.VirtualMachineInstance)
		Expect(ok).To(BeTrue(), "Object is not of type *v1.VirtualMachineInstance")
		if ignoreWarnings == true {
			tests.WaitForSuccessfulVMIStartIgnoreWarnings(obj)
		} else {
			tests.WaitForSuccessfulVMIStart(obj)
		}

		// Verify Registry Disks are Online
		pods, err := virtClient.CoreV1().Pods(tests.NamespaceTestDefault).List(context.Background(), tests.UnfinishedVMIPodSelector(vmi))
		Expect(err).To(BeNil())

		By("Checking the number of VirtualMachineInstance disks")
		disksFound := 0
		for _, pod := range pods.Items {
			if pod.ObjectMeta.DeletionTimestamp != nil {
				continue
			}
			for _, containerStatus := range pod.Status.ContainerStatuses {
				if strings.HasPrefix(containerStatus.Name, "volume") == false {
					// only check readiness of disk containers
					continue
				}
				disksFound++
			}
			break
		}
		Expect(disksFound).To(Equal(1))
	}

	table.DescribeTable("should", func(image string, policy k8sv1.PullPolicy, expectedPolicy k8sv1.PullPolicy) {
		vmi := tests.NewRandomVMIWithEphemeralDisk(image)
		vmi.Spec.Volumes[0].ContainerDisk.ImagePullPolicy = policy
		vmi = tests.RunVMIAndExpectScheduling(vmi, 60)
		Expect(vmi.Spec.Volumes[0].ContainerDisk.ImagePullPolicy).To(Equal(expectedPolicy))
		pod := libvmi.GetPodByVirtualMachineInstance(vmi, vmi.Namespace)
		container := tests.GetContainerDiskContainerOfPod(pod, vmi.Spec.Volumes[0].Name)
		Expect(container.ImagePullPolicy).To(Equal(expectedPolicy))
	},
		table.Entry("[test_id:3246]generate and set Always pull policy", "test", k8sv1.PullPolicy(""), k8sv1.PullAlways),
		table.Entry("[test_id:3247]generate and set Always pull policy", "test:latest", k8sv1.PullPolicy(""), k8sv1.PullAlways),
		table.Entry("[test_id:3248]generate and set IfNotPresent pull policy", "test@sha256:9c2b78e11c25b3fd0b24b0ed684a112052dff03eee4ca4bdcc4f3168f9a14396", k8sv1.PullPolicy(""), k8sv1.PullIfNotPresent),
		table.Entry("[test_id:3249]pass through Never pull policy to the pod", "test@sha256:9c2b78e11c25b3fd0b24b0ed684a112052dff03eee4ca4bdcc4f3168f9a14396", k8sv1.PullNever, k8sv1.PullNever),
		table.Entry("[test_id:3250]pass through IfNotPresent pull policy to the pod", "test:latest", k8sv1.PullIfNotPresent, k8sv1.PullIfNotPresent),
	)

	Describe("[rfe_id:273][crit:medium][vendor:cnv-qe@redhat.com][level:component]Starting and stopping the same VirtualMachineInstance", func() {
		Context("with ephemeral registry disk", func() {
			It("[test_id:1463][Conformance] should success multiple times", func() {
				vmi := tests.NewRandomVMIWithEphemeralDiskAndUserdata(cd.ContainerDiskFor(cd.ContainerDiskCirros), "#!/bin/bash\necho 'hello'\n")
				num := 2
				for i := 0; i < num; i++ {
					By("Starting the VirtualMachineInstance")
					obj, err := virtClient.RestClient().Post().Resource("virtualmachineinstances").Namespace(tests.NamespaceTestDefault).Body(vmi).Do(context.Background()).Get()
					Expect(err).To(BeNil())
					tests.WaitForSuccessfulVMIStart(obj)

					By("Stopping the VirtualMachineInstance")
					_, err = virtClient.RestClient().Delete().Resource("virtualmachineinstances").Namespace(vmi.GetObjectMeta().GetNamespace()).Name(vmi.GetObjectMeta().GetName()).Do(context.Background()).Get()
					Expect(err).To(BeNil())
					By("Waiting until the VirtualMachineInstance is gone")
					tests.WaitForVirtualMachineToDisappearWithTimeout(vmi, 120)
				}
			})
		})
	})

	Describe("[rfe_id:273][crit:medium][vendor:cnv-qe@redhat.com][level:component]Starting a VirtualMachineInstance", func() {
		Context("with ephemeral registry disk", func() {
			It("[test_id:1464]should not modify the spec on status update", func() {
				vmi := tests.NewRandomVMIWithEphemeralDiskAndUserdata(cd.ContainerDiskFor(cd.ContainerDiskCirros), "#!/bin/bash\necho 'hello'\n")
				v1.SetObjectDefaults_VirtualMachineInstance(vmi)

				By("Starting the VirtualMachineInstance")
				vmi, err := virtClient.VirtualMachineInstance(tests.NamespaceTestDefault).Create(vmi)
				Expect(err).To(BeNil())
				tests.WaitForSuccessfulVMIStart(vmi)
				startedVMI, err := virtClient.VirtualMachineInstance(tests.NamespaceTestDefault).Get(vmi.ObjectMeta.Name, &metav1.GetOptions{})
				Expect(err).To(BeNil())
				By("Checking that the VirtualMachineInstance spec did not change")
				Expect(startedVMI.Spec).To(Equal(vmi.Spec))
			})
		})
	})

	Describe("[rfe_id:273][crit:medium][vendor:cnv-qe@redhat.com][level:component]Starting multiple VMIs", func() {
		Context("with ephemeral registry disk", func() {
			It("[test_id:1465]should success", func() {
				num := 5
				vmis := make([]*v1.VirtualMachineInstance, 0, num)
				objs := make([]runtime.Object, 0, num)
				for i := 0; i < num; i++ {
					vmi := tests.NewRandomVMIWithEphemeralDiskAndUserdata(cd.ContainerDiskFor(cd.ContainerDiskCirros), "#!/bin/bash\necho 'hello'\n")
					// FIXME if we give too much ram, the vmis really boot and eat all our memory (cache?)
					vmi.Spec.Domain.Resources.Requests[k8sv1.ResourceMemory] = resource.MustParse("1M")
					obj := LaunchVMI(vmi)
					vmis = append(vmis, vmi)
					objs = append(objs, obj)
				}

				for idx, vmi := range vmis {
					// TODO once networking is implemented properly set ignoreWarnings == false here.
					// We have to ignore warnings because VMIs started in parallel
					// may cause libvirt to fail to create the macvtap device in
					// the host network.
					// The new network implementation we're working on should resolve this.
					// NOTE the VirtualMachineInstance still starts successfully regardless of this warning.
					// It just requires virt-handler to retry the Start command at the moment.
					VerifyContainerDiskVMI(vmi, objs[idx], true)
				}
			}) // Timeout is long because this test involves multiple parallel VirtualMachineInstance launches.
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
				obj, err := virtClient.RestClient().Post().Resource("virtualmachineinstances").Namespace(tests.NamespaceTestDefault).Body(vmi).Do(context.Background()).Get()
				Expect(err).To(BeNil())
				tests.WaitForSuccessfulVMIStart(obj)
			})
		})

	})

	Describe("[rfe_id:273][crit:medium][vendor:cnv-qe@redhat.com][level:component]Starting with virtio-win", func() {
		Context("with virtio-win as secondary disk", func() {
			It("[test_id:1467]should boot and have the virtio as sata CDROM", func() {
				vmi := tests.NewRandomVMIWithEphemeralDisk(cd.ContainerDiskFor(cd.ContainerDiskAlpine))
				tests.AddEphemeralCdrom(vmi, "disk4", "sata", cd.ContainerDiskFor(cd.ContainerDiskVirtio))

				By("Starting the VirtualMachineInstance")
				obj, err := virtClient.RestClient().Post().Resource("virtualmachineinstances").Namespace(tests.NamespaceTestDefault).Body(vmi).Do(context.Background()).Get()
				Expect(err).To(BeNil(), "expected vmi to start with no problem")
				tests.WaitForSuccessfulVMIStart(obj)

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

	Describe("[rfe_id:4052][crit:high][vendor:cnv-qe@redhat.com][level:component]VMI disk permissions", func() {
		Context("with ephemeral registry disk", func() {
			It("[test_id:4299]should not have world write permissions", func() {
				vmi := tests.NewRandomVMIWithEphemeralDisk(cd.ContainerDiskFor(cd.ContainerDiskAlpine))

				By("Starting a New VMI")
				vmi, err = virtClient.VirtualMachineInstance(tests.NamespaceTestDefault).Create(vmi)
				Expect(err).ToNot(HaveOccurred())
				tests.WaitForSuccessfulVMIStart(vmi)

				By("Ensuring VMI is running by logging in")
				tests.WaitUntilVMIReady(vmi, console.LoginToAlpine)

				By("Fetching virt-launcher Pod")
				pod := libvmi.GetPodByVirtualMachineInstance(vmi, tests.NamespaceTestDefault)

				writableImagePath := fmt.Sprintf("/var/run/kubevirt-ephemeral-disks/disk-data/%v/disk.qcow2", vmi.Spec.Domain.Devices.Disks[0].Name)

				writableImageOctalMode, err := tests.ExecuteCommandOnPod(
					virtClient,
					pod,
					"compute",
					[]string{"/usr/bin/bash", "-c", fmt.Sprintf("stat -c %%a %s", writableImagePath)},
				)
				Expect(err).ToNot(HaveOccurred())

				By("Checking the writable Image Octal mode")
				Expect(strings.Trim(writableImageOctalMode, "\n")).To(Equal("640"), "Octal Mode of writable Image should be 640")

				readonlyImageOctalMode, err := tests.ExecuteCommandOnPod(
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
})
