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
	"k8s.io/apimachinery/pkg/types"

	"kubevirt.io/kubevirt/pkg/api/v1"
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
	ovmCirros         = "ovm-cirros"
	ovmAlpineMultiPvc = "ovm-alpine-multipvc"
)

const vmReplicaSetCirros = "vm-replicaset-cirros"

const vmPresetSmall = "vm-preset-small"

const (
	busVirtio = "virtio"
	busSata   = "sata"
)

const (
	imageCirros = "cirros-registry-disk-demo"
	imageFedora = "fedora-cloud-registry-disk-demo"
)

const windowsFirmware = "5d307ca9-b3ef-428c-8861-06e72d69f223"

const apiVersion = "kubevirt.io/v1alpha1"

var dockerPrefix = "kubevirt"
var dockerTag = "devel"

var gracePeriod = int64(0)

func getBaseVmSpec() *v1.VirtualMachineSpec {
	return &v1.VirtualMachineSpec{
		TerminationGracePeriodSeconds: &gracePeriod,
		Domain: v1.DomainSpec{
			Resources: v1.ResourceRequirements{
				Requests: k8sv1.ResourceList{
					k8sv1.ResourceMemory: resource.MustParse("64M"),
				},
			},
		},
	}
}

func getBaseVm(name string) *v1.VirtualMachine {
	baseVmSpec := getBaseVmSpec()

	return &v1.VirtualMachine{
		TypeMeta: metav1.TypeMeta{
			APIVersion: apiVersion,
			Kind:       "VirtualMachine",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: name,
		},
		Spec: *baseVmSpec,
	}
}

func addRegistryDisk(spec *v1.VirtualMachineSpec, image string, bus string) *v1.VirtualMachineSpec {
	spec.Domain.Devices = v1.Devices{
		Disks: []v1.Disk{
			{
				Name:       "registrydisk",
				VolumeName: "registryvolume",
				DiskDevice: v1.DiskDevice{
					Disk: &v1.DiskTarget{
						Bus: bus,
					},
				},
			},
		},
	}
	spec.Volumes = []v1.Volume{
		{
			Name: "registryvolume",
			VolumeSource: v1.VolumeSource{
				RegistryDisk: &v1.RegistryDiskSource{
					Image: image,
				},
			},
		},
	}
	return spec
}

func addNoCloudDisk(spec *v1.VirtualMachineSpec) *v1.VirtualMachineSpec {
	spec.Domain.Devices.Disks = append(spec.Domain.Devices.Disks, v1.Disk{
		Name:       "cloudinitdisk",
		VolumeName: "cloudinitvolume",
		DiskDevice: v1.DiskDevice{
			Disk: &v1.DiskTarget{
				Bus: busVirtio,
			},
		},
	})

	userData := fmt.Sprint("#!/bin/sh\n\necho 'printed from cloud-init userdata'\n")
	spec.Volumes = append(spec.Volumes, v1.Volume{
		Name: "cloudinitvolume",
		VolumeSource: v1.VolumeSource{
			CloudInitNoCloud: &v1.CloudInitNoCloudSource{
				UserDataBase64: base64.StdEncoding.EncodeToString([]byte(userData)),
			},
		},
	})
	return spec
}

func addEmptyDisk(spec *v1.VirtualMachineSpec, size string) *v1.VirtualMachineSpec {
	spec.Domain.Devices.Disks = append(spec.Domain.Devices.Disks, v1.Disk{
		Name:       "emptydisk",
		VolumeName: "emptydiskvolume",
		DiskDevice: v1.DiskDevice{
			Disk: &v1.DiskTarget{
				Bus: busVirtio,
			},
		},
	})

	spec.Volumes = append(spec.Volumes, v1.Volume{
		Name: "emptydiskvolume",
		VolumeSource: v1.VolumeSource{
			EmptyDisk: &v1.EmptyDiskSource{
				Capacity: resource.MustParse(size),
			},
		},
	})
	return spec
}

func addPvcDisk(spec *v1.VirtualMachineSpec, claimName string, bus string) *v1.VirtualMachineSpec {
	spec.Domain.Devices.Disks = append(spec.Domain.Devices.Disks, v1.Disk{
		Name:       "pvcdisk",
		VolumeName: "pvcvolume",
		DiskDevice: v1.DiskDevice{
			Disk: &v1.DiskTarget{
				Bus: bus,
			},
		},
	})

	spec.Volumes = append(spec.Volumes, v1.Volume{
		Name: "pvcvolume",
		VolumeSource: v1.VolumeSource{
			PersistentVolumeClaim: &k8sv1.PersistentVolumeClaimVolumeSource{
				ClaimName: claimName,
			},
		},
	})
	return spec
}

func getVmEphemeral() *v1.VirtualMachine {
	vm := getBaseVm(vmEphemeral)

	addRegistryDisk(&vm.Spec, fmt.Sprintf("%s/%s:%s", dockerPrefix, imageCirros, dockerTag), busVirtio)
	return vm
}

func getVmSata() *v1.VirtualMachine {
	vm := getBaseVm(vmSata)

	addRegistryDisk(&vm.Spec, fmt.Sprintf("%s/%s:%s", dockerPrefix, imageCirros, dockerTag), busSata)
	return vm
}

func getVmEphemeralFedora() *v1.VirtualMachine {
	vm := getBaseVm(vmFedora)
	vm.Spec.Domain.Resources.Requests[k8sv1.ResourceMemory] = resource.MustParse("1024M")

	addRegistryDisk(&vm.Spec, fmt.Sprintf("%s/%s:%s", dockerPrefix, imageFedora, dockerTag), busVirtio)
	addNoCloudDisk(&vm.Spec)
	return vm
}

func getVmNoCloud() *v1.VirtualMachine {
	vm := getBaseVm(vmNoCloud)

	addRegistryDisk(&vm.Spec, fmt.Sprintf("%s/%s:%s", dockerPrefix, imageCirros, dockerTag), busVirtio)
	addNoCloudDisk(&vm.Spec)
	addEmptyDisk(&vm.Spec, "2Gi")
	return vm
}

func getVmFlavorSmall() *v1.VirtualMachine {
	vm := getBaseVm(vmFlavorSmall)
	vm.ObjectMeta.Labels = map[string]string{
		"kubevirt.io/flavor": "small",
	}

	addRegistryDisk(&vm.Spec, fmt.Sprintf("%s/%s:%s", dockerPrefix, imageCirros, dockerTag), busVirtio)
	return vm
}

func getVmPvc() *v1.VirtualMachine {
	vm := getBaseVm(vmPvc)

	addPvcDisk(&vm.Spec, "disk-alpine", busVirtio)
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

	addPvcDisk(&vm.Spec, "disk-windows", busSata)
	return vm
}

func getBaseOvm(name string, labels map[string]string) *v1.OfflineVirtualMachine {
	baseVmSpec := getBaseVmSpec()

	return &v1.OfflineVirtualMachine{
		TypeMeta: metav1.TypeMeta{
			APIVersion: apiVersion,
			Kind:       "OfflineVirtualMachine",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:   name,
			Labels: labels,
		},
		Spec: v1.OfflineVirtualMachineSpec{
			Running: false,
			Template: &v1.VMTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: labels,
				},
				Spec: *baseVmSpec,
			},
		},
	}
}

func getOvmCirros() *v1.OfflineVirtualMachine {
	ovm := getBaseOvm(ovmCirros, map[string]string{
		"kubevirt.io/ovm": ovmCirros,
	})

	addRegistryDisk(&ovm.Spec.Template.Spec, fmt.Sprintf("%s/%s:%s", dockerPrefix, imageCirros, dockerTag), busVirtio)
	addNoCloudDisk(&ovm.Spec.Template.Spec)
	return ovm
}

func getOvmMultiPvc() *v1.OfflineVirtualMachine {
	ovm := getBaseOvm(ovmAlpineMultiPvc, map[string]string{
		"kubevirt.io/ovm": ovmAlpineMultiPvc,
	})

	addPvcDisk(&ovm.Spec.Template.Spec, "disk-alpine", busVirtio)
	addPvcDisk(&ovm.Spec.Template.Spec, "disk-custom", busVirtio)

	return ovm
}

func getBaseVmReplicaSet(name string, replicas int, selectorLabels map[string]string) *v1.VirtualMachineReplicaSet {
	baseVmSpec := getBaseVmSpec()
	replicasInt32 := int32(replicas)

	return &v1.VirtualMachineReplicaSet{
		TypeMeta: metav1.TypeMeta{
			APIVersion: apiVersion,
			Kind:       "VirtualMachineReplicaSet",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: name,
		},
		Spec: v1.VMReplicaSetSpec{
			Replicas: &replicasInt32,
			Selector: &metav1.LabelSelector{
				MatchLabels: selectorLabels,
			},
			Template: &v1.VMTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: selectorLabels,
				},
				Spec: *baseVmSpec,
			},
		},
	}
}

func getVmReplicaSetCirros() *v1.VirtualMachineReplicaSet {
	vmReplicaSet := getBaseVmReplicaSet(vmReplicaSetCirros, 3, map[string]string{
		"kubevirt.io/vmReplicaSet": vmReplicaSetCirros,
	})

	addRegistryDisk(&vmReplicaSet.Spec.Template.Spec, fmt.Sprintf("%s/%s:%s", dockerPrefix, imageCirros, dockerTag), busVirtio)
	return vmReplicaSet
}

func getBaseVmPreset(name string, selectorLabels map[string]string) *v1.VirtualMachinePreset {
	return &v1.VirtualMachinePreset{
		TypeMeta: metav1.TypeMeta{
			APIVersion: apiVersion,
			Kind:       "VirtualMachinePreset",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: name,
		},
		Spec: v1.VirtualMachinePresetSpec{
			Selector: metav1.LabelSelector{
				MatchLabels: selectorLabels,
			},
		},
	}
}

func getVmPresetSmall() *v1.VirtualMachinePreset {
	vmPreset := getBaseVmPreset(vmPresetSmall, map[string]string{
		"kubevirt.io/vmPreset": vmPresetSmall,
	})

	vmPreset.Spec.Domain = &v1.DomainSpec{
		Resources: v1.ResourceRequirements{
			Requests: k8sv1.ResourceList{
				k8sv1.ResourceMemory: resource.MustParse("64M"),
			},
		},
	}
	return vmPreset
}

func main() {
	flag.StringVar(&dockerPrefix, "docker-prefix", dockerPrefix, "")
	flag.StringVar(&dockerTag, "docker-tag", dockerTag, "")
	genDir := flag.String("generated-vms-dir", "", "")
	flag.Parse()

	var vms = map[string]interface{}{
		vmEphemeral:        getVmEphemeral(),
		vmFlavorSmall:      getVmFlavorSmall(),
		vmSata:             getVmSata(),
		vmFedora:           getVmEphemeralFedora(),
		vmNoCloud:          getVmNoCloud(),
		vmPvc:              getVmPvc(),
		vmWindows:          getVmWindows(),
		ovmCirros:          getOvmCirros(),
		ovmAlpineMultiPvc:  getOvmMultiPvc(),
		vmReplicaSetCirros: getVmReplicaSetCirros(),
		vmPresetSmall:      getVmPresetSmall(),
	}
	for name, obj := range vms {
		data, err := yaml.Marshal(obj)
		if err != nil {
			fmt.Printf("Cannot marshal json: %s", fmt.Errorf("failed to generate yaml for vm %s", name))
		}

		err = ioutil.WriteFile(filepath.Join(*genDir, fmt.Sprintf("%s.yaml", name)), data, 0644)
		if err != nil {
			fmt.Printf("Cannot write file: %s", fmt.Errorf("failed to write yaml file"))
		}
	}
}
