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
 * Copyright 2023 Red Hat, Inc.
 *
 */

package vmi_configuration

import (
	"context"

	k8sv1 "k8s.io/api/core/v1"
	nodev1 "k8s.io/api/node/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/pointer"

	cdiv1 "kubevirt.io/containerized-data-importer-api/pkg/apis/core/v1beta1"

	v1 "kubevirt.io/api/core/v1"
	"kubevirt.io/client-go/kubecli"

	"kubevirt.io/kubevirt/tests/framework/kubevirt"

	"kubevirt.io/kubevirt/tests/compute"
	cd "kubevirt.io/kubevirt/tests/containerdisk"
	"kubevirt.io/kubevirt/tests/libdv"
	"kubevirt.io/kubevirt/tests/libstorage"
	"kubevirt.io/kubevirt/tests/libvmi"
	"kubevirt.io/kubevirt/tests/testsuite"
)

func ConfigDescribe(text string, args ...interface{}) bool {
	return compute.SIGDescribe("Configurations"+text, args)
}

func FConfigDescribe(text string, args ...interface{}) bool {
	return compute.SIGDescribe("Configurations"+text, args)
}

func createRuntimeClass(name, handler string) error {
	virtCli := kubevirt.Client()

	_, err := virtCli.NodeV1().RuntimeClasses().Create(
		context.Background(),
		&nodev1.RuntimeClass{
			ObjectMeta: metav1.ObjectMeta{Name: name},
			Handler:    handler,
		},
		metav1.CreateOptions{},
	)
	return err
}

func deleteRuntimeClass(name string) error {
	virtCli := kubevirt.Client()

	return virtCli.NodeV1().RuntimeClasses().Delete(context.Background(), name, metav1.DeleteOptions{})
}

func withNoRng() libvmi.Option {
	return func(vmi *v1.VirtualMachineInstance) {
		vmi.Spec.Domain.Devices.Rng = nil
	}
}

func overcommitGuestOverhead() libvmi.Option {
	return func(vmi *v1.VirtualMachineInstance) {
		vmi.Spec.Domain.Resources.OvercommitGuestOverhead = true
	}
}

func withMachineType(machineType string) libvmi.Option {
	return func(vmi *v1.VirtualMachineInstance) {
		vmi.Spec.Domain.Machine = &v1.Machine{Type: machineType}
	}
}

func WithSchedulerName(schedulerName string) libvmi.Option {
	return func(vmi *v1.VirtualMachineInstance) {
		vmi.Spec.SchedulerName = schedulerName
	}
}

func withSerialBIOS() libvmi.Option {
	return func(vmi *v1.VirtualMachineInstance) {
		if vmi.Spec.Domain.Firmware == nil {
			vmi.Spec.Domain.Firmware = &v1.Firmware{}
		}
		if vmi.Spec.Domain.Firmware.Bootloader == nil {
			vmi.Spec.Domain.Firmware.Bootloader = &v1.Bootloader{}
		}
		if vmi.Spec.Domain.Firmware.Bootloader.BIOS == nil {
			vmi.Spec.Domain.Firmware.Bootloader.BIOS = &v1.BIOS{}
		}
		vmi.Spec.Domain.Firmware.Bootloader.BIOS.UseSerial = pointer.Bool(true)
	}
}

func createBlockDataVolume(virtClient kubecli.KubevirtClient) (*cdiv1.DataVolume, error) {
	sc, foundSC := libstorage.GetBlockStorageClass(k8sv1.ReadWriteOnce)
	if !foundSC {
		return nil, nil
	}

	dataVolume := libdv.NewDataVolume(
		libdv.WithRegistryURLSource(cd.DataVolumeImportUrlForContainerDisk(cd.ContainerDiskCirros)),
		libdv.WithPVC(libdv.PVCWithStorageClass(sc), libdv.PVCWithBlockVolumeMode()),
	)

	return virtClient.CdiClient().CdiV1beta1().DataVolumes(testsuite.GetTestNamespace(nil)).Create(context.Background(), dataVolume, metav1.CreateOptions{})
}
