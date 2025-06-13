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
 * Copyright 2020 Red Hat, Inc.
 *
 */

package storage

import (
	"context"
	"fmt"
	"time"

	expect "github.com/google/goexpect"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"

	v1 "kubevirt.io/api/core/v1"
	"kubevirt.io/client-go/kubecli"
	cdiv1 "kubevirt.io/containerized-data-importer-api/pkg/apis/core/v1beta1"

	"kubevirt.io/kubevirt/pkg/apimachinery/patch"
	"kubevirt.io/kubevirt/pkg/libdv"
	"kubevirt.io/kubevirt/pkg/libvmi"
	"kubevirt.io/kubevirt/tests/console"
	cd "kubevirt.io/kubevirt/tests/containerdisk"
	"kubevirt.io/kubevirt/tests/framework/kubevirt"
	"kubevirt.io/kubevirt/tests/framework/matcher"
	"kubevirt.io/kubevirt/tests/libstorage"
	"kubevirt.io/kubevirt/tests/libvmifact"
	"kubevirt.io/kubevirt/tests/testsuite"
)

const (
	cdRomName       = "cdrom"
	hotplugDiskName = "hotplug-disk"
	volumeSize      = "1Gi"
)

var _ = Describe(SIG("Declarative Hotplug", func() {
	var virtClient kubecli.KubevirtClient

	BeforeEach(func() {
		virtClient = kubevirt.Client()
	})

	createVM := func(options ...libvmi.Option) *v1.VirtualMachine {
		vm := libvmi.NewVirtualMachine(
			libvmifact.NewCirros(options...),
			libvmi.WithRunStrategy(v1.RunStrategyAlways))
		vm, err := virtClient.VirtualMachine(testsuite.GetTestNamespace(nil)).Create(context.Background(), vm, metav1.CreateOptions{})
		Expect(err).ToNot(HaveOccurred())
		Eventually(matcher.ThisVM(vm)).WithTimeout(300 * time.Second).WithPolling(time.Second).Should(matcher.BeReady())
		return vm
	}

	createAndStartVMWithEmptyCDRom := func() *v1.VirtualMachine {
		return createVM(libvmi.WithEmptyCDRom(v1.DiskBusSATA, cdRomName))
	}

	createCDRomVolume := func(namespace string, cdisk cd.ContainerDisk) *cdiv1.DataVolume {
		url := "docker://" + cd.ContainerDiskFor(cdisk)
		dv := libdv.NewDataVolume(
			libdv.WithRegistryURLSourceAndPullMethod(url, cdiv1.RegistryPullNode),
			libdv.WithStorage(
				libdv.StorageWithVolumeSize(volumeSize),
			),
		)

		var err error
		dv, err = virtClient.CdiClient().CdiV1beta1().DataVolumes(namespace).Create(context.Background(), dv, metav1.CreateOptions{})
		Expect(err).ToNot(HaveOccurred())
		return dv
	}

	createBlankVolume := func(namespace string) *cdiv1.DataVolume {
		dv := libdv.NewDataVolume(
			libdv.WithBlankImageSource(),
			libdv.WithStorage(
				libdv.StorageWithVolumeSize(volumeSize),
			),
		)

		var err error
		dv, err = virtClient.CdiClient().CdiV1beta1().DataVolumes(namespace).Create(context.Background(), dv, metav1.CreateOptions{})
		Expect(err).ToNot(HaveOccurred())
		return dv
	}

	patchVM := func(vm *v1.VirtualMachine, disks []v1.Disk, volumes []v1.Volume) *v1.VirtualMachine {
		patchObj := patch.New()
		if disks != nil {
			patchObj.AddOption(patch.WithReplace("/spec/template/spec/domain/devices/disks", disks))
		}
		if volumes != nil {
			patchObj.AddOption(patch.WithReplace("/spec/template/spec/volumes", volumes))
		}
		patchBytes, err := patchObj.GeneratePayload()
		Expect(err).ToNot(HaveOccurred())
		vm, err = virtClient.VirtualMachine(vm.Namespace).Patch(context.Background(), vm.Name, types.JSONPatchType, patchBytes, metav1.PatchOptions{})
		Expect(err).ToNot(HaveOccurred())
		return vm
	}

	addHotplugVolume := func(vm *v1.VirtualMachine, volumeName, pvcName string) *v1.VirtualMachine {
		var err error
		vm, err = virtClient.VirtualMachine(vm.Namespace).Get(context.Background(), vm.Name, metav1.GetOptions{})
		Expect(err).ToNot(HaveOccurred())
		newVolumes := vm.DeepCopy().Spec.Template.Spec.Volumes
		newVolumes = append(newVolumes, v1.Volume{
			Name: volumeName,
			VolumeSource: v1.VolumeSource{
				DataVolume: &v1.DataVolumeSource{
					Name:         pvcName,
					Hotpluggable: true,
				},
			},
		})
		return patchVM(vm, nil, newVolumes)
	}

	hotplugDisk := func(vm *v1.VirtualMachine, volumeName, pvcName string) *v1.VirtualMachine {
		vmCpy := vm.DeepCopy()
		disks := vmCpy.Spec.Template.Spec.Domain.Devices.Disks
		volumes := vmCpy.Spec.Template.Spec.Volumes
		disks = append(disks, v1.Disk{
			Name: volumeName,
			DiskDevice: v1.DiskDevice{
				Disk: &v1.DiskTarget{
					Bus: "scsi",
				},
			},
		})
		volumes = append(volumes, v1.Volume{
			Name: volumeName,
			VolumeSource: v1.VolumeSource{
				DataVolume: &v1.DataVolumeSource{
					Name:         pvcName,
					Hotpluggable: true,
				},
			},
		})
		return patchVM(vm, disks, volumes)
	}

	removeHotplugVolume := func(vm *v1.VirtualMachine, volumeName string) *v1.VirtualMachine {
		var err error
		vm, err = virtClient.VirtualMachine(vm.Namespace).Get(context.Background(), vm.Name, metav1.GetOptions{})
		Expect(err).ToNot(HaveOccurred())
		var newVolumes []v1.Volume
		for _, volume := range vm.Spec.Template.Spec.Volumes {
			if volume.Name != volumeName {
				newVolumes = append(newVolumes, volume)
			}
		}
		return patchVM(vm, nil, newVolumes)
	}

	removeHotplugDiskAndVolume := func(vm *v1.VirtualMachine, volumeName string) *v1.VirtualMachine {
		var err error
		vm, err = virtClient.VirtualMachine(vm.Namespace).Get(context.Background(), vm.Name, metav1.GetOptions{})
		Expect(err).ToNot(HaveOccurred())
		var newDisks []v1.Disk
		for _, disk := range vm.Spec.Template.Spec.Domain.Devices.Disks {
			if disk.Name != volumeName {
				newDisks = append(newDisks, disk)
			}
		}

		var newVolumes []v1.Volume
		for _, volume := range vm.Spec.Template.Spec.Volumes {
			if volume.Name != volumeName {
				newVolumes = append(newVolumes, volume)
			}
		}
		return patchVM(vm, newDisks, newVolumes)
	}

	waitForHotplugToComplete := func(vm *v1.VirtualMachine, volumeName, claimName string, added bool) {
		Eventually(func() error {
			vmi, err := virtClient.VirtualMachineInstance(vm.Namespace).Get(context.Background(), vm.Name, metav1.GetOptions{})
			if err != nil {
				return err
			}
			found := false
			for _, volume := range vmi.Spec.Volumes {
				if volume.Name == volumeName && volume.DataVolume.Name == claimName {
					found = true
					break
				}
			}
			if found != added {
				return fmt.Errorf("volume %s in VM %s not in expected state %t, spec not updated", volumeName, vm.Name, added)
			}
			found = false
			for _, vs := range vmi.Status.VolumeStatus {
				if vs.Name == volumeName {
					if added && vs.Phase == v1.VolumeReady {
						return nil
					}
					if !added {
						found = true
						break
					}
				}
			}
			if !added && !found {
				return nil
			}
			return fmt.Errorf("volume %s in VM %s not in expected state %t, status not updated", volumeName, vm.Name, added)
		}, 240*time.Second, 2*time.Second).Should(Succeed())
	}

	swapClaim := func(vm *v1.VirtualMachine, volumeName, newClaimName string) *v1.VirtualMachine {
		vm, err := virtClient.VirtualMachine(vm.Namespace).Get(context.Background(), vm.Name, metav1.GetOptions{})
		Expect(err).ToNot(HaveOccurred())
		var newVolumes []v1.Volume
		for _, volume := range vm.Spec.Template.Spec.Volumes {
			if volume.Name == volumeName {
				volume.VolumeSource.DataVolume.Name = newClaimName
			}
			newVolumes = append(newVolumes, volume)
		}
		return patchVM(vm, nil, newVolumes)
	}

	loginToVM := func(vm *v1.VirtualMachine) {
		vmi, err := virtClient.VirtualMachineInstance(vm.Namespace).Get(context.Background(), vm.Name, metav1.GetOptions{})
		Expect(err).ToNot(HaveOccurred())
		err = console.LoginToCirros(vmi)
		Expect(err).ToNot(HaveOccurred())
	}

	validateVMHasCDRom := func(vm *v1.VirtualMachine, numItems string) {
		vmi, err := virtClient.VirtualMachineInstance(vm.Namespace).Get(context.Background(), vm.Name, metav1.GetOptions{})
		Expect(err).ToNot(HaveOccurred())
		Eventually(func() error {
			return console.SafeExpectBatch(vmi, []expect.Batcher{
				&expect.BSnd{S: "ls -A /mnt/ | wc -l\n"},
				&expect.BExp{R: console.RetValue("0")},
				&expect.BSnd{S: "sudo mount /dev/sr0 /mnt\n"},
				&expect.BExp{R: console.PromptExpression},
				&expect.BSnd{S: console.EchoLastReturnValue},
				&expect.BExp{R: console.RetValue("0")},
				&expect.BSnd{S: "ls -A /mnt/ | wc -l\n"},
				&expect.BExp{R: console.RetValue(numItems)},
				&expect.BSnd{S: "sudo umount /mnt\n"},
				&expect.BExp{R: console.PromptExpression},
				&expect.BSnd{S: console.EchoLastReturnValue},
				&expect.BExp{R: console.RetValue("0")},
				&expect.BSnd{S: "ls -A /mnt/ | wc -l\n"},
				&expect.BExp{R: console.RetValue("0")},
			}, 10)
		}, 40*time.Second, 2*time.Second).Should(Succeed())
	}

	validateVMHasNoCDRom := func(vm *v1.VirtualMachine) {
		vmi, err := virtClient.VirtualMachineInstance(vm.Namespace).Get(context.Background(), vm.Name, metav1.GetOptions{})
		Expect(err).ToNot(HaveOccurred())
		Eventually(func() error {
			return console.SafeExpectBatch(vmi, []expect.Batcher{
				&expect.BSnd{S: "ls -A /mnt/ | wc -l\n"},
				&expect.BExp{R: console.RetValue("0")},
				&expect.BSnd{S: "sudo mount /dev/sr0 /mnt\n"},
				&expect.BExp{R: console.RetValue("mount: mounting /dev/sr0 on /mnt failed: No medium found")},
			}, 10)
		}, 40*time.Second, 2*time.Second).Should(Succeed())
	}

	Context("Inject/Eject CD-ROM", func() {
		It("Should inject, swap, and eject a CD-ROM", func() {
			By("Creating a VM with an empty CD-ROM")
			vm := createAndStartVMWithEmptyCDRom()

			By("creating cd-rom volumes")
			dv1 := createCDRomVolume(vm.Namespace, cd.ContainerDiskVirtio)
			dv2 := createCDRomVolume(vm.Namespace, cd.ContainerDiskAlpine)

			By("Hotplugging a CD-ROM")
			vm = addHotplugVolume(vm, cdRomName, dv1.Name)
			waitForHotplugToComplete(vm, cdRomName, dv1.Name, true)
			libstorage.EventuallyDV(dv1, 240, matcher.HaveSucceeded())

			By("Validate the first CD-ROM is present in the VM")
			loginToVM(vm)
			validateVMHasCDRom(vm, "27")

			By("Swapping the CD-ROM")
			vm = swapClaim(vm, cdRomName, dv2.Name)
			waitForHotplugToComplete(vm, cdRomName, dv2.Name, true)
			libstorage.EventuallyDV(dv2, 240, matcher.HaveSucceeded())

			By("Validate the second CD-ROM is present in the VM")
			loginToVM(vm)
			validateVMHasCDRom(vm, "4")

			By("Ejecting the CD-ROM")
			vm = removeHotplugVolume(vm, cdRomName)
			waitForHotplugToComplete(vm, cdRomName, dv2.Name, false)

			By("Validate the CD-ROM is not present in the VM")
			validateVMHasNoCDRom(vm)
		})
	})

	Context("Hotplug disks", func() {
		It("Should add and remove a hotplug disk", func() {
			By("Creating a VM")
			vm := createVM()

			By("Hotplugging a disk")
			dv1 := createBlankVolume(vm.Namespace)
			vm = hotplugDisk(vm, hotplugDiskName, dv1.Name)
			waitForHotplugToComplete(vm, hotplugDiskName, dv1.Name, true)
			libstorage.EventuallyDV(dv1, 240, matcher.HaveSucceeded())

			By("Unplug the disk")
			vm = removeHotplugDiskAndVolume(vm, hotplugDiskName)
			waitForHotplugToComplete(vm, hotplugDiskName, dv1.Name, false)
		})
	})
}))
