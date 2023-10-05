package storage_util

import (
	"context"
	"fmt"
	"time"

	. "github.com/onsi/gomega"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/rand"
	v1 "kubevirt.io/api/core/v1"
	"kubevirt.io/client-go/kubecli"
	"kubevirt.io/client-go/log"

	cd "kubevirt.io/kubevirt/tests/containerdisk"
	"kubevirt.io/kubevirt/tests/framework/matcher"
	"kubevirt.io/kubevirt/tests/libdv"
	"kubevirt.io/kubevirt/tests/libstorage"
)

const (
	WaitDiskTemplateError         = "waiting on new disk to appear in template"
	WaitVolumeTemplateError       = "waiting on new volume to appear in template"
	WaitVolumeRequestProcessError = "waiting on all VolumeRequests to be processed"
)

func AddVolumeAndVerify(virtClient kubecli.KubevirtClient, storageClass string, vm *v1.VirtualMachine, addVMIOnly bool) string {
	dv := libdv.NewDataVolume(
		libdv.WithBlankImageSource(),
		libdv.WithPVC(libdv.PVCWithStorageClass(storageClass), libdv.PVCWithVolumeSize(cd.BlankVolumeSize)),
	)

	var err error
	dv, err = virtClient.CdiClient().CdiV1beta1().DataVolumes(vm.Namespace).Create(context.Background(), dv, metav1.CreateOptions{})
	Expect(err).ToNot(HaveOccurred())

	libstorage.EventuallyDV(dv, 240, matcher.HaveSucceeded())
	volumeSource := &v1.HotplugVolumeSource{
		DataVolume: &v1.DataVolumeSource{
			Name: dv.Name,
		},
	}
	addVolumeName := "test-volume-" + rand.String(12)
	addVolumeOptions := &v1.AddVolumeOptions{
		Name: addVolumeName,
		Disk: &v1.Disk{
			DiskDevice: v1.DiskDevice{
				Disk: &v1.DiskTarget{
					Bus: v1.DiskBusSCSI,
				},
			},
			Serial: addVolumeName,
		},
		VolumeSource: volumeSource,
	}

	if addVMIOnly {
		Eventually(func() error {
			return virtClient.VirtualMachineInstance(vm.Namespace).AddVolume(context.Background(), vm.Name, addVolumeOptions)
		}, 3*time.Second, 1*time.Second).ShouldNot(HaveOccurred())
	} else {
		Eventually(func() error {
			return virtClient.VirtualMachine(vm.Namespace).AddVolume(context.Background(), vm.Name, addVolumeOptions)
		}, 3*time.Second, 1*time.Second).ShouldNot(HaveOccurred())
		VerifyVolumeAndDiskVMAdded(virtClient, vm, addVolumeName)
	}

	vmi, err := virtClient.VirtualMachineInstance(vm.Namespace).Get(context.Background(), vm.Name, &metav1.GetOptions{})
	Expect(err).ToNot(HaveOccurred())
	VerifyVolumeAndDiskVMIAdded(virtClient, vmi, addVolumeName)

	return addVolumeName
}

func VerifyVolumeAndDiskVMAdded(virtClient kubecli.KubevirtClient, vm *v1.VirtualMachine, volumeNames ...string) {
	nameMap := make(map[string]bool)
	for _, volumeName := range volumeNames {
		nameMap[volumeName] = true
	}
	log.Log.Infof("Checking %d volumes", len(volumeNames))
	Eventually(func() error {
		updatedVM, err := virtClient.VirtualMachine(vm.Namespace).Get(context.Background(), vm.Name, &metav1.GetOptions{})
		if err != nil {
			return err
		}

		if len(updatedVM.Status.VolumeRequests) > 0 {
			return fmt.Errorf(WaitVolumeRequestProcessError)
		}

		foundVolume := 0
		foundDisk := 0

		for _, volume := range updatedVM.Spec.Template.Spec.Volumes {
			if _, ok := nameMap[volume.Name]; ok {
				foundVolume++
			}
		}
		for _, disk := range updatedVM.Spec.Template.Spec.Domain.Devices.Disks {
			if _, ok := nameMap[disk.Name]; ok {
				foundDisk++
			}
		}

		if foundDisk != len(volumeNames) {
			return fmt.Errorf(WaitDiskTemplateError)
		}
		if foundVolume != len(volumeNames) {
			return fmt.Errorf(WaitVolumeTemplateError)
		}

		return nil
	}, 90*time.Second, 1*time.Second).ShouldNot(HaveOccurred())
}

func VerifyVolumeAndDiskVMIAdded(virtClient kubecli.KubevirtClient, vmi *v1.VirtualMachineInstance, volumeNames ...string) {
	nameMap := make(map[string]bool)
	for _, volumeName := range volumeNames {
		nameMap[volumeName] = true
	}
	Eventually(func() error {
		updatedVMI, err := virtClient.VirtualMachineInstance(vmi.Namespace).Get(context.Background(), vmi.Name, &metav1.GetOptions{})
		if err != nil {
			return err
		}

		foundVolume := 0
		foundDisk := 0

		for _, volume := range updatedVMI.Spec.Volumes {
			if _, ok := nameMap[volume.Name]; ok {
				foundVolume++
			}
		}
		for _, disk := range updatedVMI.Spec.Domain.Devices.Disks {
			if _, ok := nameMap[disk.Name]; ok {
				foundDisk++
			}
		}

		if foundDisk != len(volumeNames) {
			return fmt.Errorf(WaitDiskTemplateError)
		}
		if foundVolume != len(volumeNames) {
			return fmt.Errorf(WaitVolumeTemplateError)
		}

		return nil
	}, 90*time.Second, 1*time.Second).ShouldNot(HaveOccurred())
}
