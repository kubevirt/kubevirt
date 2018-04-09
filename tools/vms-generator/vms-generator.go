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
 * Copyright 2018 Red Hat, Inc.
 *
 */
 
package main

import (
	"encoding/base64"
	"flag"
	"fmt"
	"io/ioutil"
	"path/filepath"

	"github.com/ghodss/yaml"

	k8sv1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"kubevirt.io/kubevirt/pkg/api/v1"
	"k8s.io/apimachinery/pkg/types"
)

const (
	vmEphemeral   = "vm-ephemeral"
	vmFlavorSmall = "vm-flavor-small"
	vmSata        = "vm-sata"
	vmFedora      = "vm-fedora"
	vmNoCloud     = "vm-nocloud"
	vmPvc         = "vm-pvc"
	vmWindows     = "vm-windows"
)

const (
	busVirtio = "virtio"
	busSata   = "sata"
)

const (
	imageCirros = "cirros-registry-disk-demo"
	imageFedora = "fedora-cloud-registry-disk-demo"
)

const windowsFirmware   = "5d307ca9-b3ef-428c-8861-06e72d69f223"

var dockerPrefix = "kubevirt"
var dockerTag = "devel"

func getBaseVm(name string) *v1.VirtualMachine {
	gracePeriod := int64(0)
	return &v1.VirtualMachine{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "kubevirt.io/v1alpha1",
			Kind: "VirtualMachine",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: name,
		},
		Spec: v1.VirtualMachineSpec{
			TerminationGracePeriodSeconds: &gracePeriod,
			Domain: v1.DomainSpec{
				Resources: v1.ResourceRequirements{
					Requests: k8sv1.ResourceList{
						k8sv1.ResourceMemory: resource.MustParse("64M"),
					},
				},
			},
		},
	}
}

func addRegistryDisk(vm *v1.VirtualMachine, image string, bus string) *v1.VirtualMachine {
	vm.Spec.Domain.Devices = v1.Devices{
		Disks: []v1.Disk{
			{
				Name: "registrydisk",
				VolumeName: "registryvolume",
				DiskDevice: v1.DiskDevice{
					Disk: &v1.DiskTarget{
						Bus: bus,
					},
				},
			},
		},
	}
	vm.Spec.Volumes = []v1.Volume{
		{
			Name: "registryvolume",
			VolumeSource: v1.VolumeSource{
				RegistryDisk: &v1.RegistryDiskSource{
					Image: image,
				},
			},
		},
	}
	return vm
}

func addNoCloudDisk(vm *v1.VirtualMachine) *v1.VirtualMachine {
	vm.Spec.Domain.Devices.Disks = append(vm.Spec.Domain.Devices.Disks, v1.Disk{
		Name: "cloudinitdisk",
		VolumeName: "cloudinitvolume",
		DiskDevice: v1.DiskDevice{
			Disk: &v1.DiskTarget{
				Bus: busVirtio,
			},
		},
	})

	userData := fmt.Sprint("#!/bin/sh\n\necho 'printed from cloud-init userdata'\n")
	vm.Spec.Volumes = append(vm.Spec.Volumes, v1.Volume{
		Name: "cloudinitvolume",
		VolumeSource: v1.VolumeSource{
			CloudInitNoCloud: &v1.CloudInitNoCloudSource{
				UserDataBase64: base64.StdEncoding.EncodeToString([]byte(userData)),
			},
		},
	})
	return vm
}

func addEmptyDisk(vm *v1.VirtualMachine, size string) *v1.VirtualMachine {
	vm.Spec.Domain.Devices.Disks = append(vm.Spec.Domain.Devices.Disks, v1.Disk{
		Name: "emptydisk",
		VolumeName: "emptydiskvolume",
		DiskDevice: v1.DiskDevice{
			Disk: &v1.DiskTarget{
				Bus: busVirtio,
			},
		},
	})

	vm.Spec.Volumes = append(vm.Spec.Volumes, v1.Volume{
		Name: "emptydiskvolume",
		VolumeSource: v1.VolumeSource{
			EmptyDisk: &v1.EmptyDiskSource{
				Capacity: resource.MustParse(size),
			},
		},
	})
	return vm
}

func addPvcDisk(vm *v1.VirtualMachine, claimName string, bus string) *v1.VirtualMachine {
	vm.Spec.Domain.Devices.Disks = append(vm.Spec.Domain.Devices.Disks, v1.Disk{
		Name: "pvcdisk",
		VolumeName: "pvcvolume",
		DiskDevice: v1.DiskDevice{
			Disk: &v1.DiskTarget{
				Bus: bus,
			},
		},
	})

	vm.Spec.Volumes = append(vm.Spec.Volumes, v1.Volume{
		Name: "pvcvolume",
		VolumeSource: v1.VolumeSource{
			PersistentVolumeClaim: &k8sv1.PersistentVolumeClaimVolumeSource{
				ClaimName: claimName,
			},
		},
	})
	return vm
}

func getVmEphemeral() *v1.VirtualMachine {
	vm := getBaseVm(vmEphemeral)

	addRegistryDisk(vm, fmt.Sprintf("%s/%s:%s", dockerPrefix, imageCirros, dockerTag), busVirtio)
	return vm
}

func getVmSata() *v1.VirtualMachine {
	vm := getBaseVm(vmSata)

	addRegistryDisk(vm, fmt.Sprintf("%s/%s:%s", dockerPrefix, imageCirros, dockerTag), busSata)
	return vm
}

func getVmEphemeralFedora() *v1.VirtualMachine {
	vm := getBaseVm(vmFedora)
	vm.Spec.Domain.Resources.Requests[k8sv1.ResourceMemory] = resource.MustParse("1024M")

	addRegistryDisk(vm, fmt.Sprintf("%s/%s:%s", dockerPrefix, imageFedora, dockerTag), busVirtio)
	addNoCloudDisk(vm)
	return vm
}

func getVmNoCloud() *v1.VirtualMachine {
	vm := getBaseVm(vmNoCloud)

	addRegistryDisk(vm, fmt.Sprintf("%s/%s:%s", dockerPrefix, imageCirros, dockerTag), busVirtio)
	addNoCloudDisk(vm)
	addEmptyDisk(vm, "2Gi")
	return vm
}

func getVmFlavorSmall() *v1.VirtualMachine {
	vm := getBaseVm(vmFlavorSmall)
	vm.ObjectMeta.Labels = map[string]string {
		"kubevirt.io/flavor": "small",
	}

	addRegistryDisk(vm, fmt.Sprintf("%s/%s:%s", dockerPrefix, imageCirros, dockerTag), busVirtio)
	return vm
}

func getVmPvc() *v1.VirtualMachine {
	vm := getBaseVm(vmPvc)

	addPvcDisk(vm,"disk-alpine", busVirtio)
	return vm
}

func getVmWindows() *v1.VirtualMachine {
	vm := getBaseVm(vmWindows)

	gracePeriod := int64(0)
	spinlocks := uint32(8191)
	firmware := types.UID(windowsFirmware)
	_false := false
	vm.Spec = v1.VirtualMachineSpec{
		TerminationGracePeriodSeconds: &gracePeriod,
		Domain: v1.DomainSpec{
			CPU: &v1.CPU{Cores: 2},
			Features: &v1.Features{
				ACPI: v1.FeatureState{},
				APIC: &v1.FeatureAPIC{},
				Hyperv: &v1.FeatureHyperv{
					Relaxed:   &v1.FeatureState{},
					VAPIC:     &v1.FeatureState{},
					Spinlocks: &v1.FeatureSpinlocks{Retries: &spinlocks},
				},
			},
			Clock: &v1.Clock{
				ClockOffset: v1.ClockOffset{UTC: &v1.ClockOffsetUTC{}},
				Timer: &v1.Timer{
					HPET:   &v1.HPETTimer{Enabled: &_false},
					PIT:    &v1.PITTimer{TickPolicy: v1.PITTickPolicyDelay},
					RTC:    &v1.RTCTimer{TickPolicy: v1.RTCTickPolicyCatchup},
					Hyperv: &v1.HypervTimer{},
				},
			},
			Firmware: &v1.Firmware{UUID: firmware},
			Resources: v1.ResourceRequirements{
				Requests: k8sv1.ResourceList{
					k8sv1.ResourceMemory: resource.MustParse("2048Mi"),
				},
			},
		},
	}

	addPvcDisk(vm,"disk-windows", busSata)
	return vm
}

func main() {
	flag.StringVar(&dockerPrefix, "docker-prefix", dockerPrefix, "")
	flag.StringVar(&dockerTag,"docker-tag", dockerTag, "")
	genDir := flag.String("generated-vms-dir", "", "")
	flag.Parse()

	var vms = map[string]*v1.VirtualMachine {
		vmEphemeral: getVmEphemeral(),
		vmFlavorSmall: getVmFlavorSmall(),
		vmSata: getVmSata(),
		vmFedora: getVmEphemeralFedora(),
		vmNoCloud: getVmNoCloud(),
		vmPvc: getVmPvc(),
		vmWindows: getVmWindows(),
	}
	for vmName, vm := range vms {
		data, err := yaml.Marshal(vm)
		if err != nil {
			fmt.Errorf("failed to generate yaml for vm %s", vmName)
		}

		err = ioutil.WriteFile(filepath.Join(*genDir, fmt.Sprintf("%s.yaml", vmName)), data, 0644)
		if err != nil {
			fmt.Errorf("failed to write yaml file")
		}
	}
}
