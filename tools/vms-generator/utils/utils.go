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

package utils

import (
	"encoding/json"
	"fmt"
	"os"

	k8sv1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"

	cdiv1 "kubevirt.io/containerized-data-importer/pkg/apis/core/v1alpha1"
	v1 "kubevirt.io/kubevirt/pkg/api/v1"
)

const (
	VmiEphemeral         = "vmi-ephemeral"
	VmiMigratable        = "vmi-migratable"
	VmiFlavorSmall       = "vmi-flavor-small"
	VmiSata              = "vmi-sata"
	VmiFedora            = "vmi-fedora"
	VmiAlpineEFI         = "vmi-alpine-efi"
	VmiNoCloud           = "vmi-nocloud"
	VmiPVC               = "vmi-pvc"
	VmiBlockPVC          = "vmi-block-pvc"
	VmiWindows           = "vmi-windows"
	VmiSlirp             = "vmi-slirp"
	VmiMasquerade        = "vmi-masquerade"
	VmiSRIOV             = "vmi-sriov"
	VmiWithHookSidecar   = "vmi-with-sidecar-hook"
	VmiMultusPtp         = "vmi-multus-ptp"
	VmiMultusMultipleNet = "vmi-multus-multiple-net"
	VmiGeniePtp          = "vmi-genie-ptp"
	VmiGenieMultipleNet  = "vmi-genie-multiple-net"
	VmiHostDisk          = "vmi-host-disk"
	VmTemplateFedora     = "vm-template-fedora"
	VmTemplateRHEL7      = "vm-template-rhel7"
	VmTemplateWindows    = "vm-template-windows2012r2"
)

const (
	VmCirros           = "vm-cirros"
	VmAlpineMultiPvc   = "vm-alpine-multipvc"
	VmAlpineDataVolume = "vm-alpine-datavolume"
)

const VmiReplicaSetCirros = "vmi-replicaset-cirros"

const VmiPresetSmall = "vmi-preset-small"

const VmiMigration = "migration-job"

const KubeVirt = "kubevirt-cr"

const (
	busVirtio = "virtio"
	busSata   = "sata"
)

const (
	imageAlpine = "alpine-container-disk-demo"
	imageCirros = "cirros-container-disk-demo"
	imageFedora = "fedora-cloud-container-disk-demo"
)

const windowsFirmware = "5d307ca9-b3ef-428c-8861-06e72d69f223"

const apiVersion = "kubevirt.io/v1alpha3"

var DockerPrefix = "registry:5000/kubevirt"
var DockerTag = "devel"

var gracePeriod = int64(0)

func getBaseVMISpec() *v1.VirtualMachineInstanceSpec {
	return &v1.VirtualMachineInstanceSpec{
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

func getBaseVMI(name string) *v1.VirtualMachineInstance {
	baseVMISpec := getBaseVMISpec()

	return &v1.VirtualMachineInstance{
		TypeMeta: metav1.TypeMeta{
			APIVersion: v1.GroupVersion.String(),
			Kind:       "VirtualMachineInstance",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:   name,
			Labels: map[string]string{"special": name},
		},
		Spec: *baseVMISpec,
	}
}

func addContainerDisk(spec *v1.VirtualMachineInstanceSpec, image string, bus string) *v1.VirtualMachineInstanceSpec {
	spec.Domain.Devices = v1.Devices{
		Disks: []v1.Disk{
			{
				Name: "containerdisk",
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
			Name: "containerdisk",
			VolumeSource: v1.VolumeSource{
				ContainerDisk: &v1.ContainerDiskSource{
					Image: image,
				},
			},
		},
	}
	return spec
}

func addNoCloudDisk(spec *v1.VirtualMachineInstanceSpec) *v1.VirtualMachineInstanceSpec {
	return addNoCloudDiskWitUserData(spec, "#!/bin/sh\n\necho 'printed from cloud-init userdata'\n")
}

func addNoCloudDiskWitUserData(spec *v1.VirtualMachineInstanceSpec, data string) *v1.VirtualMachineInstanceSpec {
	spec.Domain.Devices.Disks = append(spec.Domain.Devices.Disks, v1.Disk{
		Name: "cloudinitdisk",
		DiskDevice: v1.DiskDevice{
			Disk: &v1.DiskTarget{
				Bus: busVirtio,
			},
		},
	})

	spec.Volumes = append(spec.Volumes, v1.Volume{
		Name: "cloudinitdisk",
		VolumeSource: v1.VolumeSource{
			CloudInitNoCloud: &v1.CloudInitNoCloudSource{
				UserData: data,
			},
		},
	})
	return spec
}

func addEmptyDisk(spec *v1.VirtualMachineInstanceSpec, size string) *v1.VirtualMachineInstanceSpec {
	spec.Domain.Devices.Disks = append(spec.Domain.Devices.Disks, v1.Disk{
		Name: "emptydisk",
		DiskDevice: v1.DiskDevice{
			Disk: &v1.DiskTarget{
				Bus: busVirtio,
			},
		},
	})

	spec.Volumes = append(spec.Volumes, v1.Volume{
		Name: "emptydisk",
		VolumeSource: v1.VolumeSource{
			EmptyDisk: &v1.EmptyDiskSource{
				Capacity: resource.MustParse(size),
			},
		},
	})
	return spec
}

func addDataVolumeDisk(spec *v1.VirtualMachineInstanceSpec, dataVolumeName string, bus string, diskName string) *v1.VirtualMachineInstanceSpec {
	spec.Domain.Devices.Disks = append(spec.Domain.Devices.Disks, v1.Disk{
		Name: diskName,
		DiskDevice: v1.DiskDevice{
			Disk: &v1.DiskTarget{
				Bus: bus,
			},
		},
	})

	spec.Volumes = append(spec.Volumes, v1.Volume{
		Name: diskName,
		VolumeSource: v1.VolumeSource{
			DataVolume: &v1.DataVolumeSource{
				Name: dataVolumeName,
			},
		},
	})
	return spec
}

func addPVCDisk(spec *v1.VirtualMachineInstanceSpec, claimName string, bus string, diskName string) *v1.VirtualMachineInstanceSpec {
	spec.Domain.Devices.Disks = append(spec.Domain.Devices.Disks, v1.Disk{
		Name: diskName,
		DiskDevice: v1.DiskDevice{
			Disk: &v1.DiskTarget{
				Bus: bus,
			},
		},
	})

	spec.Volumes = append(spec.Volumes, v1.Volume{
		Name: diskName,
		VolumeSource: v1.VolumeSource{
			PersistentVolumeClaim: &k8sv1.PersistentVolumeClaimVolumeSource{
				ClaimName: claimName,
			},
		},
	})
	return spec
}

func addEphemeralPVCDisk(spec *v1.VirtualMachineInstanceSpec, claimName string, bus string, diskName string) *v1.VirtualMachineInstanceSpec {
	spec.Domain.Devices.Disks = append(spec.Domain.Devices.Disks, v1.Disk{
		Name: diskName,
		DiskDevice: v1.DiskDevice{
			Disk: &v1.DiskTarget{
				Bus: bus,
			},
		},
	})

	spec.Volumes = append(spec.Volumes, v1.Volume{
		Name: diskName,
		VolumeSource: v1.VolumeSource{
			Ephemeral: &v1.EphemeralVolumeSource{
				PersistentVolumeClaim: &k8sv1.PersistentVolumeClaimVolumeSource{
					ClaimName: claimName,
				},
			},
		},
	})
	return spec
}

func addHostDisk(spec *v1.VirtualMachineInstanceSpec, path string, hostDiskType v1.HostDiskType, size string) *v1.VirtualMachineInstanceSpec {
	spec.Domain.Devices.Disks = append(spec.Domain.Devices.Disks, v1.Disk{
		Name: "host-disk",
		DiskDevice: v1.DiskDevice{
			Disk: &v1.DiskTarget{
				Bus: busVirtio,
			},
		},
	})
	spec.Volumes = append(spec.Volumes, v1.Volume{
		Name: "host-disk",
		VolumeSource: v1.VolumeSource{
			HostDisk: &v1.HostDisk{
				Path:     path,
				Type:     hostDiskType,
				Capacity: resource.MustParse(size),
			},
		},
	})
	return spec
}

func GetVMIMigratable() *v1.VirtualMachineInstance {
	vmi := getBaseVMI(VmiMigratable)

	addContainerDisk(&vmi.Spec, fmt.Sprintf("%s/%s:%s", DockerPrefix, imageAlpine, DockerTag), busVirtio)
	return vmi
}

func GetVMIEphemeral() *v1.VirtualMachineInstance {
	vmi := getBaseVMI(VmiEphemeral)

	addContainerDisk(&vmi.Spec, fmt.Sprintf("%s/%s:%s", DockerPrefix, imageCirros, DockerTag), busVirtio)
	return vmi
}

func GetVMISata() *v1.VirtualMachineInstance {
	vmi := getBaseVMI(VmiSata)

	addContainerDisk(&vmi.Spec, fmt.Sprintf("%s/%s:%s", DockerPrefix, imageCirros, DockerTag), busSata)
	return vmi
}

func GetVMIEphemeralFedora() *v1.VirtualMachineInstance {
	vmi := getBaseVMI(VmiFedora)
	vmi.Spec.Domain.Resources.Requests[k8sv1.ResourceMemory] = resource.MustParse("1024M")

	addContainerDisk(&vmi.Spec, fmt.Sprintf("%s/%s:%s", DockerPrefix, imageFedora, DockerTag), busVirtio)
	addNoCloudDiskWitUserData(&vmi.Spec, "#cloud-config\npassword: fedora\nchpasswd: { expire: False }")
	return vmi
}

func GetVMIAlpineEFI() *v1.VirtualMachineInstance {
	vmi := getBaseVMI(VmiAlpineEFI)

	addContainerDisk(&vmi.Spec, fmt.Sprintf("%s/%s:%s", DockerPrefix, imageAlpine, DockerTag), busVirtio)
	vmi.Spec.Domain.Firmware = &v1.Firmware{
		Bootloader: &v1.Bootloader{
			EFI: &v1.EFI{},
		},
	}

	vmi.Spec.Domain.Resources.Requests[k8sv1.ResourceMemory] = resource.MustParse("1Gi")
	return vmi
}

func GetVMISlirp() *v1.VirtualMachineInstance {
	vm := getBaseVMI(VmiSlirp)
	vm.Spec.Domain.Resources.Requests[k8sv1.ResourceMemory] = resource.MustParse("1024M")
	vm.Spec.Networks = []v1.Network{v1.Network{Name: "testSlirp", NetworkSource: v1.NetworkSource{Pod: &v1.PodNetwork{}}}}

	addContainerDisk(&vm.Spec, fmt.Sprintf("%s/%s:%s", DockerPrefix, imageFedora, DockerTag), busVirtio)
	addNoCloudDiskWitUserData(&vm.Spec, "#!/bin/bash\necho \"fedora\" |passwd fedora --stdin\nyum install -y nginx\nsystemctl enable nginx\nsystemctl start nginx")

	slirp := &v1.InterfaceSlirp{}
	ports := []v1.Port{v1.Port{Name: "http", Protocol: "TCP", Port: 80}}
	vm.Spec.Domain.Devices.Interfaces = []v1.Interface{v1.Interface{Name: "testSlirp", Ports: ports, InterfaceBindingMethod: v1.InterfaceBindingMethod{Slirp: slirp}}}

	return vm
}

func GetVMIMasquerade() *v1.VirtualMachineInstance {
	vm := getBaseVMI(VmiMasquerade)
	vm.Spec.Domain.Resources.Requests[k8sv1.ResourceMemory] = resource.MustParse("1024M")
	vm.Spec.Networks = []v1.Network{v1.Network{Name: "testmasquerade", NetworkSource: v1.NetworkSource{Pod: &v1.PodNetwork{}}}}

	addContainerDisk(&vm.Spec, fmt.Sprintf("%s/%s:%s", DockerPrefix, imageFedora, DockerTag), busVirtio)
	addNoCloudDiskWitUserData(&vm.Spec, "#!/bin/bash\necho \"fedora\" |passwd fedora --stdin\nyum install -y nginx\nsystemctl enable nginx\nsystemctl start nginx")

	masquerade := &v1.InterfaceMasquerade{}
	ports := []v1.Port{v1.Port{Name: "http", Protocol: "TCP", Port: 80}}
	vm.Spec.Domain.Devices.Interfaces = []v1.Interface{v1.Interface{Name: "testmasquerade", Ports: ports, InterfaceBindingMethod: v1.InterfaceBindingMethod{Masquerade: masquerade}}}

	return vm
}

func GetVMISRIOV() *v1.VirtualMachineInstance {
	vm := getBaseVMI(VmiSRIOV)
	vm.Spec.Domain.Resources.Requests[k8sv1.ResourceMemory] = resource.MustParse("1024M")
	vm.Spec.Networks = []v1.Network{*v1.DefaultPodNetwork(), {Name: "sriov-net", NetworkSource: v1.NetworkSource{Multus: &v1.MultusNetwork{NetworkName: "sriov-net"}}}}
	addContainerDisk(&vm.Spec, fmt.Sprintf("%s/%s:%s", DockerPrefix, imageFedora, DockerTag), busVirtio)
	addNoCloudDiskWitUserData(&vm.Spec, "#!/bin/bash\necho \"fedora\" |passwd fedora --stdin\ndhclient eth1\n")

	vm.Spec.Domain.Devices.Interfaces = []v1.Interface{{Name: "default", InterfaceBindingMethod: v1.InterfaceBindingMethod{Bridge: &v1.InterfaceBridge{}}},
		{Name: "sriov-net", InterfaceBindingMethod: v1.InterfaceBindingMethod{SRIOV: &v1.InterfaceSRIOV{}}}}

	return vm
}

func GetVMIMultusPtp() *v1.VirtualMachineInstance {
	vm := getBaseVMI(VmiMultusPtp)
	vm.Spec.Domain.Resources.Requests[k8sv1.ResourceMemory] = resource.MustParse("1024M")
	vm.Spec.Networks = []v1.Network{{Name: "ptp", NetworkSource: v1.NetworkSource{Multus: &v1.MultusNetwork{NetworkName: "ptp-conf"}}}}
	addContainerDisk(&vm.Spec, fmt.Sprintf("%s/%s:%s", DockerPrefix, imageFedora, DockerTag), busVirtio)
	addNoCloudDiskWitUserData(&vm.Spec, "#!/bin/bash\necho \"fedora\" |passwd fedora --stdin\n")

	vm.Spec.Domain.Devices.Interfaces = []v1.Interface{{Name: "ptp", InterfaceBindingMethod: v1.InterfaceBindingMethod{Bridge: &v1.InterfaceBridge{}}}}

	return vm
}

func GetVMIMultusMultipleNet() *v1.VirtualMachineInstance {
	vm := getBaseVMI(VmiMultusMultipleNet)
	vm.Spec.Domain.Resources.Requests[k8sv1.ResourceMemory] = resource.MustParse("1024M")
	vm.Spec.Networks = []v1.Network{*v1.DefaultPodNetwork(), {Name: "ptp", NetworkSource: v1.NetworkSource{Multus: &v1.MultusNetwork{NetworkName: "ptp-conf"}}}}
	addContainerDisk(&vm.Spec, fmt.Sprintf("%s/%s:%s", DockerPrefix, imageFedora, DockerTag), busVirtio)
	addNoCloudDiskWitUserData(&vm.Spec, "#!/bin/bash\necho \"fedora\" |passwd fedora --stdin\ndhclient eth1\n")

	vm.Spec.Domain.Devices.Interfaces = []v1.Interface{{Name: "default", InterfaceBindingMethod: v1.InterfaceBindingMethod{Bridge: &v1.InterfaceBridge{}}},
		{Name: "ptp", InterfaceBindingMethod: v1.InterfaceBindingMethod{Bridge: &v1.InterfaceBridge{}}}}

	return vm
}

func GetVMIGeniePtp() *v1.VirtualMachineInstance {
	vm := getBaseVMI(VmiGeniePtp)
	vm.Spec.Domain.Resources.Requests[k8sv1.ResourceMemory] = resource.MustParse("1024M")
	vm.Spec.Networks = []v1.Network{
		{Name: "ptp", NetworkSource: v1.NetworkSource{Genie: &v1.GenieNetwork{NetworkName: "ptp"}}},
	}
	addContainerDisk(&vm.Spec, fmt.Sprintf("%s/%s:%s", DockerPrefix, imageFedora, DockerTag), busVirtio)
	addNoCloudDiskWitUserData(&vm.Spec, "#!/bin/bash\necho \"fedora\" | passwd fedora --stdin\n")

	vm.Spec.Domain.Devices.Interfaces = []v1.Interface{
		{Name: "ptp", InterfaceBindingMethod: v1.InterfaceBindingMethod{Bridge: &v1.InterfaceBridge{}}},
	}

	return vm
}

func GetVMIGenieMultipleNet() *v1.VirtualMachineInstance {
	vm := getBaseVMI(VmiGenieMultipleNet)
	vm.Spec.Domain.Resources.Requests[k8sv1.ResourceMemory] = resource.MustParse("1024M")
	vm.Spec.Networks = []v1.Network{
		{Name: "default", NetworkSource: v1.NetworkSource{Genie: &v1.GenieNetwork{NetworkName: "flannel"}}},
		{Name: "ptp", NetworkSource: v1.NetworkSource{Genie: &v1.GenieNetwork{NetworkName: "ptp"}}},
	}
	addContainerDisk(&vm.Spec, fmt.Sprintf("%s/%s:%s", DockerPrefix, imageFedora, DockerTag), busVirtio)
	addNoCloudDiskWitUserData(&vm.Spec, "#!/bin/bash\necho \"fedora\" | passwd fedora --stdin\ndhclient eth1\n")

	vm.Spec.Domain.Devices.Interfaces = []v1.Interface{
		{Name: "default", InterfaceBindingMethod: v1.InterfaceBindingMethod{Bridge: &v1.InterfaceBridge{}}},
		{Name: "ptp", InterfaceBindingMethod: v1.InterfaceBindingMethod{Bridge: &v1.InterfaceBridge{}}},
	}

	return vm
}
func GetVMINoCloud() *v1.VirtualMachineInstance {
	vmi := getBaseVMI(VmiNoCloud)

	addContainerDisk(&vmi.Spec, fmt.Sprintf("%s/%s:%s", DockerPrefix, imageCirros, DockerTag), busVirtio)
	addNoCloudDisk(&vmi.Spec)
	addEmptyDisk(&vmi.Spec, "2Gi")
	return vmi
}

func GetVMIFlavorSmall() *v1.VirtualMachineInstance {
	vmi := getBaseVMI(VmiFlavorSmall)
	vmi.ObjectMeta.Labels = map[string]string{
		"kubevirt.io/flavor": "small",
	}

	addContainerDisk(&vmi.Spec, fmt.Sprintf("%s/%s:%s", DockerPrefix, imageCirros, DockerTag), busVirtio)
	return vmi
}

func GetVMIPvc() *v1.VirtualMachineInstance {
	vmi := getBaseVMI(VmiPVC)

	addPVCDisk(&vmi.Spec, "disk-alpine", busVirtio, "pvcdisk")
	return vmi
}

func GetVMIBlockPvc() *v1.VirtualMachineInstance {
	vmi := getBaseVMI(VmiBlockPVC)

	addPVCDisk(&vmi.Spec, "local-block-storage-cirros", busVirtio, "blockpvcdisk")
	return vmi
}

func GetVMIHostDisk() *v1.VirtualMachineInstance {
	vmi := getBaseVMI(VmiHostDisk)
	addHostDisk(&vmi.Spec, "/data/disk.img", v1.HostDiskExistsOrCreate, "1Gi")
	return vmi
}

func GetVMIWindows() *v1.VirtualMachineInstance {
	vmi := getBaseVMI(VmiWindows)

	gracePeriod := int64(0)
	spinlocks := uint32(8191)
	firmware := types.UID(windowsFirmware)
	_false := false
	vmi.Spec = v1.VirtualMachineInstanceSpec{
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
			Machine: v1.Machine{Type: "q35"},
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
			Devices: v1.Devices{
				Interfaces: []v1.Interface{*v1.DefaultNetworkInterface()},
			},
		},
		Networks: []v1.Network{*v1.DefaultPodNetwork()},
	}

	// pick e1000 network model type for windows machines
	vmi.Spec.Domain.Devices.Interfaces[0].Model = "e1000"

	addPVCDisk(&vmi.Spec, "disk-windows", busSata, "pvcdisk")
	return vmi
}

func getBaseVM(name string, labels map[string]string) *v1.VirtualMachine {
	baseVMISpec := getBaseVMISpec()

	return &v1.VirtualMachine{
		TypeMeta: metav1.TypeMeta{
			APIVersion: v1.GroupVersion.String(),
			Kind:       "VirtualMachine",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:   name,
			Labels: labels,
		},
		Spec: v1.VirtualMachineSpec{
			Running: false,
			Template: &v1.VirtualMachineInstanceTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: labels,
				},
				Spec: *baseVMISpec,
			},
		},
	}
}

func GetVMCirros() *v1.VirtualMachine {
	vm := getBaseVM(VmCirros, map[string]string{
		"kubevirt.io/vm": VmCirros,
	})

	addContainerDisk(&vm.Spec.Template.Spec, fmt.Sprintf("%s/%s:%s", DockerPrefix, imageCirros, DockerTag), busVirtio)
	addNoCloudDisk(&vm.Spec.Template.Spec)
	return vm
}

func GetTemplateFedora() *Template {
	vm := getBaseVM("", map[string]string{"kubevirt-vm": "vm-${NAME}", "kubevirt.io/os": "fedora27"})
	addContainerDisk(&vm.Spec.Template.Spec, fmt.Sprintf("%s/%s:%s", DockerPrefix, imageFedora, DockerTag), busVirtio)
	addNoCloudDiskWitUserData(&vm.Spec.Template.Spec, "#cloud-config\npassword: fedora\nchpasswd: { expire: False }")

	template := getBaseTemplate(vm, "4096Mi", "4")
	template.ObjectMeta = metav1.ObjectMeta{
		Name: VmTemplateFedora,
		Annotations: map[string]string{
			"description": "OCP KubeVirt Fedora 27 VM template",
			"tags":        "kubevirt,ocp,template,linux,virtualmachine",
			"iconClass":   "icon-fedora",
		},
		Labels: map[string]string{
			"kubevirt.io/os":                        "fedora27",
			"miq.github.io/kubevirt-is-vm-template": "true",
		},
	}
	return template
}

func GetTemplateRHEL7() *Template {
	vm := getBaseVM("", map[string]string{"kubevirt-vm": "vm-${NAME}", "kubevirt.io/os": "rhel-7.4"})
	addPVCDisk(&vm.Spec.Template.Spec, "linux-vm-pvc-${NAME}", busVirtio, "disk0")

	pvc := getPVCForTemplate("linux-vm-pvc-${NAME}")
	template := newTemplateForRHEL7VM(vm)
	template.Objects = append(template.Objects, pvc)

	return template
}

func GetTestTemplateRHEL7() *Template {
	vm := getBaseVM("", map[string]string{"kubevirt-vm": "vm-${NAME}", "kubevirt.io/os": "rhel-7.4"})
	addEphemeralPVCDisk(&vm.Spec.Template.Spec, "disk-rhel", busSata, "pvcdisk")

	return newTemplateForRHEL7VM(vm)
}

func newTemplateForRHEL7VM(vm *v1.VirtualMachine) *Template {
	template := getBaseTemplate(vm, "4096Mi", "4")
	template.ObjectMeta = metav1.ObjectMeta{
		Name: VmTemplateRHEL7,
		Annotations: map[string]string{
			"iconClass":   "icon-rhel",
			"description": "OCP KubeVirt Red Hat Enterprise Linux 7.4 VM template",
			"tags":        "kubevirt,ocp,template,linux,virtualmachine",
		},
		Labels: map[string]string{
			"kubevirt.io/os":                        "rhel-7.4",
			"miq.github.io/kubevirt-is-vm-template": "true",
		},
	}
	return template
}

func GetTemplateWindows() *Template {
	vm := getBaseVM("", map[string]string{"kubevirt-vm": "vm-${NAME}", "kubevirt.io/os": "win2k12r2"})
	windows := GetVMIWindows()
	vm.Spec.Template.Spec = windows.Spec
	vm.Spec.Template.ObjectMeta.Annotations = windows.ObjectMeta.Annotations
	addPVCDisk(&vm.Spec.Template.Spec, "windows-vm-pvc-${NAME}", busVirtio, "disk0")

	pvc := getPVCForTemplate("windows-vm-pvc-${NAME}")

	template := getBaseTemplate(vm, "4096Mi", "4")
	template.ObjectMeta = metav1.ObjectMeta{
		Name: VmTemplateWindows,
		Annotations: map[string]string{
			"iconClass":   "icon-windows",
			"description": "OCP KubeVirt Microsoft Windows Server 2012 R2 VM template",
			"tags":        "kubevirt,ocp,template,windows,virtualmachine",
		},
		Labels: map[string]string{
			"kubevirt.io/os":                        "win2k12r2",
			"miq.github.io/kubevirt-is-vm-template": "true",
		},
	}
	template.Objects = append(template.Objects, pvc)
	return template
}

func getPVCForTemplate(name string) *k8sv1.PersistentVolumeClaim {

	return &k8sv1.PersistentVolumeClaim{
		TypeMeta: metav1.TypeMeta{
			Kind:       "PersistentVolumeClaim",
			APIVersion: "v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: name,
		},
		Spec: k8sv1.PersistentVolumeClaimSpec{
			AccessModes: []k8sv1.PersistentVolumeAccessMode{k8sv1.ReadWriteOnce},
			Resources: k8sv1.ResourceRequirements{
				Requests: k8sv1.ResourceList{
					k8sv1.ResourceStorage: resource.MustParse("10Gi"),
				},
			},
		},
	}
}

func getBaseTemplate(vm *v1.VirtualMachine, memory string, cores string) *Template {

	obj := toUnstructured(vm)
	unstructured.SetNestedField(obj.Object, "${{CPU_CORES}}", "spec", "template", "spec", "domain", "cpu", "cores")
	unstructured.SetNestedField(obj.Object, "${MEMORY}", "spec", "template", "spec", "domain", "resources", "requests", "memory")
	obj.SetName("${NAME}")

	return &Template{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Template",
			APIVersion: "v1",
		},
		Objects: []runtime.Object{
			obj,
		},
		Parameters: templateParameters(memory, cores),
	}
}

func toUnstructured(object runtime.Object) *unstructured.Unstructured {
	raw, err := json.Marshal(object)
	if err != nil {
		panic(err)
	}
	var objmap map[string]interface{}
	err = json.Unmarshal(raw, &objmap)

	return &unstructured.Unstructured{Object: objmap}
}

func templateParameters(memory string, cores string) []Parameter {
	return []Parameter{
		{
			Name:        "NAME",
			Description: "Name for the new VM",
		},
		{
			Name:        "MEMORY",
			Description: "Amount of memory",
			Value:       memory,
		},
		{
			Name:        "CPU_CORES",
			Description: "Amount of cores",
			Value:       cores,
		},
	}
}

func GetVMDataVolume() *v1.VirtualMachine {
	vm := getBaseVM(VmAlpineDataVolume, map[string]string{
		"kubevirt.io/vm": VmAlpineDataVolume,
	})

	quantity, err := resource.ParseQuantity("2Gi")
	if err != nil {
		fmt.Fprintf(os.Stderr, "%s\n", err)
		panic(err)
	}
	storageClassName := "local"
	dataVolume := cdiv1.DataVolume{
		ObjectMeta: metav1.ObjectMeta{
			Name: "alpine-dv",
		},
		Spec: cdiv1.DataVolumeSpec{
			Source: cdiv1.DataVolumeSource{
				HTTP: &cdiv1.DataVolumeSourceHTTP{
					URL: "http://cdi-http-import-server.kubevirt/images/alpine.iso",
				},
			},
			PVC: &k8sv1.PersistentVolumeClaimSpec{
				AccessModes: []k8sv1.PersistentVolumeAccessMode{k8sv1.ReadWriteOnce},
				Resources: k8sv1.ResourceRequirements{
					Requests: k8sv1.ResourceList{
						"storage": quantity,
					},
				},
				StorageClassName: &storageClassName,
			},
		},
	}

	vm.Spec.DataVolumeTemplates = append(vm.Spec.DataVolumeTemplates, dataVolume)
	addDataVolumeDisk(&vm.Spec.Template.Spec, "alpine-dv", busVirtio, "datavolumedisk1")

	return vm
}

func GetVMMultiPvc() *v1.VirtualMachine {
	vm := getBaseVM(VmAlpineMultiPvc, map[string]string{
		"kubevirt.io/vm": VmAlpineMultiPvc,
	})

	addPVCDisk(&vm.Spec.Template.Spec, "disk-alpine", busVirtio, "pvcdisk1")
	addPVCDisk(&vm.Spec.Template.Spec, "disk-custom", busVirtio, "pvcdisk2")

	return vm
}

func getBaseVMIReplicaSet(name string, replicas int, selectorLabels map[string]string) *v1.VirtualMachineInstanceReplicaSet {
	baseVMISpec := getBaseVMISpec()
	replicasInt32 := int32(replicas)

	return &v1.VirtualMachineInstanceReplicaSet{
		TypeMeta: metav1.TypeMeta{
			APIVersion: v1.GroupVersion.String(),
			Kind:       "VirtualMachineInstanceReplicaSet",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: name,
		},
		Spec: v1.VirtualMachineInstanceReplicaSetSpec{
			Replicas: &replicasInt32,
			Selector: &metav1.LabelSelector{
				MatchLabels: selectorLabels,
			},
			Template: &v1.VirtualMachineInstanceTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: selectorLabels,
				},
				Spec: *baseVMISpec,
			},
		},
	}
}

func GetVMIReplicaSetCirros() *v1.VirtualMachineInstanceReplicaSet {
	vmReplicaSet := getBaseVMIReplicaSet(VmiReplicaSetCirros, 3, map[string]string{
		"kubevirt.io/vmReplicaSet": VmiReplicaSetCirros,
	})

	addContainerDisk(&vmReplicaSet.Spec.Template.Spec, fmt.Sprintf("%s/%s:%s", DockerPrefix, imageCirros, DockerTag), busVirtio)
	return vmReplicaSet
}

func getBaseVMIPreset(name string, selectorLabels map[string]string) *v1.VirtualMachineInstancePreset {
	return &v1.VirtualMachineInstancePreset{
		TypeMeta: metav1.TypeMeta{
			APIVersion: v1.GroupVersion.String(),
			Kind:       "VirtualMachineInstancePreset",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: name,
		},
		Spec: v1.VirtualMachineInstancePresetSpec{
			Selector: metav1.LabelSelector{
				MatchLabels: selectorLabels,
			},
		},
	}
}

func GetVMIMigration() *v1.VirtualMachineInstanceMigration {
	return &v1.VirtualMachineInstanceMigration{
		TypeMeta: metav1.TypeMeta{
			APIVersion: v1.GroupVersion.String(),
			Kind:       "VirtualMachineInstanceMigration",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: VmiMigration,
		},
		Spec: v1.VirtualMachineInstanceMigrationSpec{
			VMIName: VmiMigratable,
		},
	}
}

func GetVMIPresetSmall() *v1.VirtualMachineInstancePreset {
	vmPreset := getBaseVMIPreset(VmiPresetSmall, map[string]string{
		"kubevirt.io/vmPreset": VmiPresetSmall,
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

func GetVMIWithHookSidecar() *v1.VirtualMachineInstance {
	vmi := getBaseVMI(VmiWithHookSidecar)
	vmi.Spec.Domain.Resources.Requests[k8sv1.ResourceMemory] = resource.MustParse("1024M")

	addContainerDisk(&vmi.Spec, fmt.Sprintf("%s/%s:%s", DockerPrefix, imageFedora, DockerTag), busVirtio)
	addNoCloudDiskWitUserData(&vmi.Spec, "#cloud-config\npassword: fedora\nchpasswd: { expire: False }")

	vmi.ObjectMeta.Annotations = map[string]string{
		"hooks.kubevirt.io/hookSidecars":              fmt.Sprintf("[{\"image\": \"%s/example-hook-sidecar:%s\"}]", DockerPrefix, DockerTag),
		"smbios.vm.kubevirt.io/baseBoardManufacturer": "Radical Edward",
	}
	return vmi
}
