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
//nolint:dupl,lll,mnd
package utils

import (
	"fmt"
	"strings"

	"k8s.io/apimachinery/pkg/util/rand"
	"kubevirt.io/api/migrations/v1alpha1"
	"kubevirt.io/client-go/kubecli"

	k8sv1 "k8s.io/api/core/v1"
	schedulingv1 "k8s.io/api/scheduling/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	v1 "kubevirt.io/api/core/v1"
	poolv1 "kubevirt.io/api/pool/v1alpha1"

	"kubevirt.io/kubevirt/pkg/libdv"
	"kubevirt.io/kubevirt/pkg/libvmi"
	"kubevirt.io/kubevirt/pkg/libvmi/cloudinit"
	"kubevirt.io/kubevirt/pkg/pointer"
)

const (
	strFmt                     = "%s/%s:%s"
	kubevirtIoVM               = "kubevirt.io/vm"
	vmName                     = "vm-${NAME}"
	kubevirtVM                 = "kubevirt-vm"
	githubKubevirtIsVMTemplate = "miq.github.io/kubevirt-is-vm-template"
	rhel74                     = "rhel-7.4"
)

const (
	VMIEphemeral                = "vmi-ephemeral"
	VMIMigratable               = "vmi-migratable"
	VMISata                     = "vmi-sata"
	VMIFedora                   = "vmi-fedora"
	VMIFedoraIsolated           = "vmi-fedora-isolated"
	VMISecureBoot               = "vmi-secureboot"
	VMIAlpineEFI                = "vmi-alpine-efi"
	VMINoCloud                  = "vmi-nocloud"
	VMIPVC                      = "vmi-pvc"
	VMIWindows                  = "vmi-windows"
	VMIKernelBoot               = "vmi-kernel-boot"
	VMIMasquerade               = "vmi-masquerade"
	VMISRIOV                    = "vmi-sriov"
	VMIWithHookSidecar          = "vmi-with-sidecar-hook"
	VMIWithHookSidecarConfigMap = "vmi-with-sidecar-hook-configmap"
	VMIMultusPtp                = "vmi-multus-ptp"
	VMIMultusMultipleNet        = "vmi-multus-multiple-net"
	VMIHostDisk                 = "vmi-host-disk"
	VMIGPU                      = "vmi-gpu"
	VMIUSB                      = "vmi-usb"
	VMTemplateFedora            = "vm-template-fedora"
	VMTemplateRHEL7             = "vm-template-rhel7"
	VMTemplateWindows           = "vm-template-windows2012r2"
)

const (
	Preemtible    = "preemtible"
	NonPreemtible = "non-preemtible"
)

const (
	VMCirros                         = "vm-cirros"
	VMAlpineMultiPvc                 = "vm-alpine-multipvc"
	VMAlpineDataVolume               = "vm-alpine-datavolume"
	VMPriorityClass                  = "vm-priorityclass"
	VMCirrosSata                     = "vm-cirros-sata"
	VMCirrosWithHookSidecarConfigMap = "vm-cirros-with-sidecar-hook-configmap"
)

const VMIReplicaSetCirros = "vmi-replicaset-cirros"

const VMPoolCirros = "vm-pool-cirros"

const VMIPresetSmall = "vmi-preset-small"

const VMIMigration = "migration-job"

const MigrationPolicyName = "example-migration-policy"

const (
	imageAlpine     = "alpine-container-disk-demo"
	imageCirros     = "cirros-container-disk-demo"
	imageFedora     = "fedora-with-test-tooling-container-disk"
	imageKernelBoot = "alpine-ext-kernel-boot-demo"
)

const (
	windowsFirmware                        = "5d307ca9-b3ef-428c-8861-06e72d69f223"
	EthernetAdaptorModelToEnableMultiqueue = v1.VirtIO
)

const (
	cloudConfigHeader = "#cloud-config"

	cloudConfigInstallAndStartService = `packages:
  - nginx
runcmd:
  - [ "systemctl", "enable", "--now", "nginx" ]`

	cloudConfigUserPassword = `password: fedora
chpasswd: { expire: False }`

	secondaryIfaceDhcpNetworkData = `version: 2
ethernets:
  eth1:
    dhcp4: true
`
)

var (
	DockerPrefix = "registry:5000/kubevirt"
	DockerTag    = "devel"
)

var gracePeriod = int64(0)

func getBaseVMISpec() *v1.VirtualMachineInstanceSpec {
	return &v1.VirtualMachineInstanceSpec{
		TerminationGracePeriodSeconds: &gracePeriod,
		Domain: v1.DomainSpec{
			Resources: v1.ResourceRequirements{
				Requests: k8sv1.ResourceList{
					k8sv1.ResourceMemory: resource.MustParse("128Mi"),
				},
			},
		},
	}
}

func addContainerDisk(spec *v1.VirtualMachineInstanceSpec, image string, bus v1.DiskBus) *v1.VirtualMachineInstanceSpec {
	// Only add a reference to the disk if it isn't using the default v1.DiskBusSATA bus
	if bus != v1.DiskBusSATA {
		disk := &v1.Disk{
			Name: "containerdisk",
			DiskDevice: v1.DiskDevice{
				Disk: &v1.DiskTarget{
					Bus: bus,
				},
			},
		}
		spec.Domain.Devices.Disks = append(spec.Domain.Devices.Disks, *disk)
	}
	volume := &v1.Volume{
		Name: "containerdisk",
		VolumeSource: v1.VolumeSource{
			ContainerDisk: &v1.ContainerDiskSource{
				Image: image,
			},
		},
	}
	spec.Volumes = append(spec.Volumes, *volume)
	return spec
}

func addKernelBootContainer(spec *v1.VirtualMachineInstanceSpec, image, kernelArgs, kernelPath, initrdPath string) *v1.VirtualMachineInstanceSpec {
	if spec.Domain.Firmware == nil {
		spec.Domain.Firmware = &v1.Firmware{}
	}

	spec.Domain.Firmware.KernelBoot = &v1.KernelBoot{
		KernelArgs: kernelArgs,
		Container: &v1.KernelBootContainer{
			Image:      image,
			KernelPath: kernelPath,
			InitrdPath: initrdPath,
		},
	}

	return spec
}

func GetVMIMigratable() *v1.VirtualMachineInstance {
	return libvmi.New(
		libvmi.WithName(VMIMigratable),
		libvmi.WithResourceMemory("128Mi"),
		libvmi.WithContainerDisk("containerdisk", fmt.Sprintf(strFmt, DockerPrefix, imageAlpine, DockerTag)),
		libvmi.WithNetwork(v1.DefaultPodNetwork()),
		libvmi.WithInterface(*v1.DefaultMasqueradeNetworkInterface()),
	)
}

func GetVMIEphemeral() *v1.VirtualMachineInstance {
	return libvmi.New(
		libvmi.WithName(VMIEphemeral),
		libvmi.WithResourceMemory("128Mi"),
		libvmi.WithContainerDisk("containerdisk", fmt.Sprintf(strFmt, DockerPrefix, imageCirros, DockerTag)),
	)
}

func GetVMISata() *v1.VirtualMachineInstance {
	return libvmi.New(
		libvmi.WithName(VMISata),
		libvmi.WithResourceMemory("128Mi"),
		libvmi.WithContainerDisk("containerdisk", fmt.Sprintf(strFmt, DockerPrefix, imageCirros, DockerTag), libvmi.WithDiskBusSATA()),
	)
}

func GetVMIEphemeralFedora() *v1.VirtualMachineInstance {
	return libvmi.New(
		libvmi.WithName(VMIFedora),
		libvmi.WithResourceMemory("1024M"),
		libvmi.WithNetwork(v1.DefaultPodNetwork()),
		libvmi.WithInterface(*v1.DefaultMasqueradeNetworkInterface()),
		libvmi.WithContainerDisk("containerdisk", fmt.Sprintf(strFmt, DockerPrefix, imageFedora, DockerTag)),
		libvmi.WithRng(),
		libvmi.WithCloudInitNoCloud(
			cloudinit.WithNoCloudUserData(generateCloudConfigString(cloudConfigUserPassword)),
		),
	)
}

func GetVMIEphemeralFedoraIsolated() *v1.VirtualMachineInstance {
	return libvmi.New(
		libvmi.WithName(VMIFedora),
		libvmi.WithCPUCount(1, 0, 0),
		libvmi.WithIsolateEmulatorThread(),
		libvmi.WithDedicatedCPUPlacement(),
		libvmi.WithResourceMemory("1024M"),
		libvmi.WithContainerDisk("containerdisk", fmt.Sprintf(strFmt, DockerPrefix, imageFedora, DockerTag)),
		libvmi.WithRng(),
		libvmi.WithCloudInitNoCloud(
			cloudinit.WithNoCloudUserData(generateCloudConfigString(cloudConfigUserPassword)),
		),
	)
}

func GetVMISecureBoot() *v1.VirtualMachineInstance {
	return libvmi.New(
		libvmi.WithName(VMISecureBoot),
		libvmi.WithResourceMemory("1Gi"),
		libvmi.WithContainerDisk("containerdisk", fmt.Sprintf(strFmt, DockerPrefix, imageFedora, DockerTag)),
		libvmi.WithSMM(),
		libvmi.WithSecureBoot(true),
	)
}

func GetVMIAlpineEFI() *v1.VirtualMachineInstance {
	return libvmi.New(
		libvmi.WithName(VMIAlpineEFI),
		libvmi.WithResourceMemory("1Gi"),
		libvmi.WithContainerDisk("containerdisk", fmt.Sprintf(strFmt, DockerPrefix, imageAlpine, DockerTag)),
		libvmi.WithSecureBoot(false),
	)
}

func GetVMIMasquerade() *v1.VirtualMachineInstance {
	return libvmi.New(
		libvmi.WithName(VMIMasquerade),
		libvmi.WithResourceMemory("1024M"),
		libvmi.WithContainerDisk("containerdisk", fmt.Sprintf(strFmt, DockerPrefix, imageFedora, DockerTag)),
		libvmi.WithRng(),
		libvmi.WithNetwork(
			&v1.Network{
				Name:          "testmasquerade",
				NetworkSource: v1.NetworkSource{Pod: &v1.PodNetwork{}},
			},
		),
		libvmi.WithInterface(
			v1.Interface{
				Name:                   "testmasquerade",
				Ports:                  []v1.Port{{Name: "http", Protocol: "TCP", Port: 80}},
				InterfaceBindingMethod: v1.InterfaceBindingMethod{Masquerade: &v1.InterfaceMasquerade{}},
			},
		),
		libvmi.WithCloudInitNoCloud(
			cloudinit.WithNoCloudUserData(
				generateCloudConfigString(cloudConfigUserPassword, cloudConfigInstallAndStartService),
			),
			cloudinit.WithNoCloudNetworkData(
				"version: 2\nethernets:\n  eth0:\n    addresses: [ fd10:0:2::2/120 ]\n    dhcp4: true\n    gateway6: fd10:0:2::1\n",
			),
		),
	)
}

func GetVMISRIOV() *v1.VirtualMachineInstance {
	return libvmi.New(
		libvmi.WithName(VMISRIOV),
		libvmi.WithResourceMemory("1024M"),
		libvmi.WithContainerDisk("containerdisk", fmt.Sprintf(strFmt, DockerPrefix, imageFedora, DockerTag)),
		libvmi.WithRng(),
		libvmi.WithNetwork(v1.DefaultPodNetwork()),
		libvmi.WithNetwork(
			&v1.Network{
				Name:          "sriov-net",
				NetworkSource: v1.NetworkSource{Multus: &v1.MultusNetwork{NetworkName: "sriov/sriov-network"}},
			},
		),
		libvmi.WithInterface(
			v1.Interface{
				Name:                   "default",
				InterfaceBindingMethod: v1.InterfaceBindingMethod{Masquerade: &v1.InterfaceMasquerade{}},
			},
		),
		libvmi.WithInterface(
			v1.Interface{
				Name:                   "sriov-net",
				InterfaceBindingMethod: v1.InterfaceBindingMethod{SRIOV: &v1.InterfaceSRIOV{}},
			},
		),
		libvmi.WithCloudInitNoCloud(
			cloudinit.WithNoCloudUserData(generateCloudConfigString(cloudConfigUserPassword)),
			cloudinit.WithNoCloudNetworkData(secondaryIfaceDhcpNetworkData),
		),
	)
}

func GetVMIMultusPtp() *v1.VirtualMachineInstance {
	return libvmi.New(
		libvmi.WithName(VMIMultusPtp),
		libvmi.WithResourceMemory("1024M"),
		libvmi.WithContainerDisk("containerdisk", fmt.Sprintf(strFmt, DockerPrefix, imageFedora, DockerTag)),
		libvmi.WithRng(),
		libvmi.WithNetwork(
			&v1.Network{
				Name:          "ptp",
				NetworkSource: v1.NetworkSource{Multus: &v1.MultusNetwork{NetworkName: "ptp-conf"}},
			},
		),
		libvmi.WithInterface(
			v1.Interface{
				Name:                   "ptp",
				InterfaceBindingMethod: v1.InterfaceBindingMethod{Bridge: &v1.InterfaceBridge{}},
			},
		),
		libvmi.WithCloudInitNoCloud(
			cloudinit.WithNoCloudUserData(generateCloudConfigString(cloudConfigUserPassword)),
		),
	)
}

func GetVMIMultusMultipleNet() *v1.VirtualMachineInstance {
	return libvmi.New(
		libvmi.WithName(VMIMultusMultipleNet),
		libvmi.WithResourceMemory("1024M"),
		libvmi.WithContainerDisk("containerdisk", fmt.Sprintf(strFmt, DockerPrefix, imageFedora, DockerTag)),
		libvmi.WithRng(),
		libvmi.WithNetwork(v1.DefaultPodNetwork()),
		libvmi.WithNetwork(
			&v1.Network{
				Name:          "ptp",
				NetworkSource: v1.NetworkSource{Multus: &v1.MultusNetwork{NetworkName: "ptp-conf"}},
			},
		),
		libvmi.WithInterface(
			v1.Interface{
				Name:                   "default",
				InterfaceBindingMethod: v1.InterfaceBindingMethod{Masquerade: &v1.InterfaceMasquerade{}},
			},
		),
		libvmi.WithInterface(
			v1.Interface{
				Name:                   "ptp",
				InterfaceBindingMethod: v1.InterfaceBindingMethod{Bridge: &v1.InterfaceBridge{}},
			},
		),
		libvmi.WithCloudInitNoCloud(
			cloudinit.WithNoCloudUserData(generateCloudConfigString(cloudConfigUserPassword)),
			cloudinit.WithNoCloudNetworkData(secondaryIfaceDhcpNetworkData),
		),
	)
}

func GetVMINoCloud() *v1.VirtualMachineInstance {
	return libvmi.New(
		libvmi.WithName(VMINoCloud),
		libvmi.WithResourceMemory("128Mi"),
		libvmi.WithContainerDisk("containerdisk", fmt.Sprintf(strFmt, DockerPrefix, imageCirros, DockerTag)),
		libvmi.WithCloudInitNoCloud(
			cloudinit.WithNoCloudUserData("#!/bin/sh\n\necho 'printed from cloud-init userdata'\n"),
		),
		libvmi.WithEmptyDisk("emptydisk", v1.DiskBusVirtio, resource.MustParse("2Gi")),
	)
}

func GetVMIPvc() *v1.VirtualMachineInstance {
	return libvmi.New(
		libvmi.WithName(VMIPVC),
		libvmi.WithResourceMemory("128Mi"),
		libvmi.WithPersistentVolumeClaim("pvcdisk", "disk-alpine"),
	)
}

func GetVMIHostDisk() *v1.VirtualMachineInstance {
	return libvmi.New(
		libvmi.WithName(VMIHostDisk),
		libvmi.WithResourceMemory("128Mi"),
		libvmi.WithHostDiskAndCapacity("host-disk", "/var/data/disk.img", v1.HostDiskExistsOrCreate, "1Gi"),
	)
}

func GetVMIWindows() *v1.VirtualMachineInstance {
	spinlocks := uint32(8191)
	firmware := types.UID(windowsFirmware)
	return libvmi.New(
		libvmi.WithName(VMIWindows),
		libvmi.WithTerminationGracePeriod(0),
		libvmi.WithCPUCount(2, 0, 0),
		libvmi.WithResourceMemory("2048Mi"),
		libvmi.WithACPI(),
		libvmi.WithAPIC(),
		libvmi.WithSMM(),
		libvmi.WithHyperv(
			&v1.FeatureHyperv{
				Relaxed:   &v1.FeatureState{},
				VAPIC:     &v1.FeatureState{},
				Spinlocks: &v1.FeatureSpinlocks{Retries: &spinlocks},
			},
		),
		libvmi.WithClock(
			&v1.Clock{
				ClockOffset: v1.ClockOffset{UTC: &v1.ClockOffsetUTC{}},
				Timer: &v1.Timer{
					HPET:   &v1.HPETTimer{Enabled: pointer.P(false)},
					PIT:    &v1.PITTimer{TickPolicy: v1.PITTickPolicyDelay},
					RTC:    &v1.RTCTimer{TickPolicy: v1.RTCTickPolicyCatchup},
					Hyperv: &v1.HypervTimer{},
				},
			},
		),
		libvmi.WithFirmwareUUID(firmware),
		libvmi.WithSecureBoot(true),
		libvmi.WithTPM(false),
		libvmi.WithNetwork(v1.DefaultPodNetwork()),
		libvmi.WithInterface(
			v1.Interface{
				Name:                   "default",
				InterfaceBindingMethod: v1.InterfaceBindingMethod{Masquerade: &v1.InterfaceMasquerade{}},
				Model:                  "e1000",
			},
		),
		libvmi.WithPersistentVolumeClaim("pvcdisk", "disk-windows"),
	)
}

func GetVMIKernelBoot() *v1.VirtualMachineInstance {
	const kernelArgs = "console=ttyS0"
	const kernelPath = "/boot/vmlinuz-virt"
	const initrdPath = "/boot/initramfs-virt"
	return libvmi.New(
		libvmi.WithName(VMIKernelBoot),
		libvmi.WithResourceMemory("1Gi"),
		libvmi.WithKernelBoot(
			&v1.KernelBoot{
				KernelArgs: kernelArgs,
				Container: &v1.KernelBootContainer{
					Image:      fmt.Sprintf(strFmt, DockerPrefix, imageKernelBoot, DockerTag),
					KernelPath: kernelPath,
					InitrdPath: initrdPath,
				},
			},
		),
	)
}

func GetVMIKernelBootWithRandName() *v1.VirtualMachineInstance {
	vmi := GetVMIKernelBoot()
	vmi.Name += "-" + rand.String(5)

	return vmi
}

func AddKernelBootToVMI(vmi *v1.VirtualMachineInstance) {
	image := fmt.Sprintf(strFmt, DockerPrefix, imageKernelBoot, DockerTag)
	const KernelArgs = "console=ttyS0"
	const kernelPath = "/boot/vmlinuz-virt"
	const initrdPath = "/boot/initramfs-virt"

	addKernelBootContainer(&vmi.Spec, image, KernelArgs, kernelPath, initrdPath)
}

func GetPreemtible() *schedulingv1.PriorityClass {
	preemtionPolicy := k8sv1.PreemptLowerPriority
	pc := schedulingv1.PriorityClass{
		TypeMeta: metav1.TypeMeta{
			APIVersion: schedulingv1.SchemeGroupVersion.String(),
			Kind:       "PriorityClass",
		},
		GlobalDefault:    false,
		Description:      "Priority class for VMs which are allowed to be preemtited.",
		PreemptionPolicy: &preemtionPolicy,
		Value:            1000000,
	}
	pc.ObjectMeta.Name = "preemtible"
	return &pc
}

func GetNonPreemtible() *schedulingv1.PriorityClass {
	preemtionPolicy := k8sv1.PreemptNever
	pc := schedulingv1.PriorityClass{
		TypeMeta: metav1.TypeMeta{
			APIVersion: schedulingv1.SchemeGroupVersion.String(),
			Kind:       "PriorityClass",
		},
		GlobalDefault:    false,
		Description:      "Priority class for VMs which should not be preemtited.",
		PreemptionPolicy: &preemtionPolicy,
		Value:            999999999,
	}
	pc.ObjectMeta.Name = NonPreemtible
	return &pc
}

func GetVMPriorityClass() *v1.VirtualMachine {
	return libvmi.NewVirtualMachine(
		libvmi.New(
			libvmi.WithName("vm-non-preemtible"),
			libvmi.WithLabel(kubevirtIoVM, VMCirros),
			libvmi.WithResourceMemory("128Mi"),
			libvmi.WithContainerDisk("containerdisk", fmt.Sprintf(strFmt, DockerPrefix, imageCirros, DockerTag)),
			libvmi.WithCloudInitNoCloud(
				cloudinit.WithNoCloudUserData("#!/bin/sh\n\necho 'printed from cloud-init userdata'\n"),
			),
			libvmi.WithPriorityClass(NonPreemtible),
		),
		libvmi.WithLabels(map[string]string{kubevirtIoVM: VMCirros}),
	)
}

func GetVMCirros() *v1.VirtualMachine {
	return libvmi.NewVirtualMachine(
		libvmi.New(
			libvmi.WithName(VMCirros),
			libvmi.WithLabel(kubevirtIoVM, VMCirros),
			libvmi.WithResourceMemory("128Mi"),
			libvmi.WithContainerDisk("containerdisk", fmt.Sprintf(strFmt, DockerPrefix, imageCirros, DockerTag)),
			libvmi.WithCloudInitNoCloud(
				cloudinit.WithNoCloudUserData("#!/bin/sh\n\necho 'printed from cloud-init userdata'\n"),
			),
		),
		libvmi.WithLabels(map[string]string{kubevirtIoVM: VMCirros}),
	)
}

func GetVMCirrosWithHookSidecarConfigMap() *v1.VirtualMachine {
	return libvmi.NewVirtualMachine(
		libvmi.New(
			libvmi.WithName(VMCirrosWithHookSidecarConfigMap),
			libvmi.WithLabel(kubevirtIoVM, VMCirrosWithHookSidecarConfigMap),
			libvmi.WithResourceMemory("128Mi"),
			libvmi.WithContainerDisk("containerdisk", fmt.Sprintf(strFmt, DockerPrefix, imageCirros, DockerTag)),
			libvmi.WithCloudInitNoCloud(
				cloudinit.WithNoCloudUserData("#!/bin/sh\n\necho 'printed from cloud-init userdata'\n"),
			),
			libvmi.WithAnnotation(
				"hooks.kubevirt.io/hookSidecars",
				`[{"args": ["--version", "v1alpha2"], "configMap": {"name": "my-config-map","key": "my_script.sh", "hookPath": "/usr/bin/onDefineDomain"}}]`,
			),
		),
		libvmi.WithLabels(map[string]string{kubevirtIoVM: VMCirrosWithHookSidecarConfigMap}),
	)
}

func GetVMCirrosSata() *v1.VirtualMachine {
	vm := libvmi.NewVirtualMachine(
		libvmi.New(
			libvmi.WithName(VMCirrosSata),
			libvmi.WithLabel(kubevirtIoVM, VMCirrosSata),
			libvmi.WithResourceMemory("128Mi"),
			libvmi.WithContainerDisk("containerdisk", fmt.Sprintf(strFmt, DockerPrefix, imageCirros, DockerTag)),
			libvmi.WithCloudInitNoCloud(
				cloudinit.WithNoCloudUserData("#!/bin/sh\n\necho 'printed from cloud-init userdata'\n"),
			),
		),
		libvmi.WithLabels(map[string]string{kubevirtIoVM: VMCirrosSata}),
	)
	vm.Spec.Template.Spec.Domain.Devices = v1.Devices{}
	return vm
}

func GetVMDataVolume() *v1.VirtualMachine {
	return libvmi.NewVirtualMachine(
		libvmi.New(
			libvmi.WithName(VMAlpineDataVolume),
			libvmi.WithLabel(kubevirtIoVM, VMAlpineDataVolume),
			libvmi.WithResourceMemory("128Mi"),
			libvmi.WithDataVolume("datavolumedisk1", "alpine-dv"),
		),
		libvmi.WithDataVolumeTemplate(
			libdv.NewDataVolume(
				libdv.WithName("alpine-dv"),
				libdv.WithRegistryURLSource(fmt.Sprintf("docker://%s/%s:%s", DockerPrefix, imageAlpine, DockerTag)),
				libdv.WithPVC(
					libdv.PVCWithAccessModes([]k8sv1.PersistentVolumeAccessMode{k8sv1.ReadWriteOnce}),
					libdv.PVCWithStorageResource("2Gi"),
					libdv.PVCWithStorageClass("local"),
				),
			),
		),
		libvmi.WithLabels(map[string]string{kubevirtIoVM: VMAlpineDataVolume}),
	)
}

func GetVMMultiPvc() *v1.VirtualMachine {
	return libvmi.NewVirtualMachine(
		libvmi.New(
			libvmi.WithName(VMAlpineMultiPvc),
			libvmi.WithLabel(kubevirtIoVM, VMAlpineMultiPvc),
			libvmi.WithResourceMemory("128Mi"),
			libvmi.WithPersistentVolumeClaim("pvcdisk1", "disk-alpine"),
			libvmi.WithPersistentVolumeClaim("pvcdisk2", "disk-custom"),
		),
		libvmi.WithLabels(map[string]string{kubevirtIoVM: VMAlpineMultiPvc}),
	)
}

func getBaseVMPool(name string, replicas int, selectorLabels map[string]string) *poolv1.VirtualMachinePool {
	baseVMISpec := getBaseVMISpec()
	replicasInt32 := int32(replicas)

	return &poolv1.VirtualMachinePool{
		TypeMeta: metav1.TypeMeta{
			APIVersion: poolv1.SchemeGroupVersion.String(),
			Kind:       "VirtualMachinePool",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: name,
		},
		Spec: poolv1.VirtualMachinePoolSpec{
			Replicas: &replicasInt32,
			Selector: &metav1.LabelSelector{
				MatchLabels: selectorLabels,
			},
			VirtualMachineTemplate: &poolv1.VirtualMachineTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: selectorLabels,
				},
				Spec: v1.VirtualMachineSpec{
					RunStrategy: pointer.P(v1.RunStrategyAlways),
					Template: &v1.VirtualMachineInstanceTemplateSpec{
						ObjectMeta: metav1.ObjectMeta{
							Labels: selectorLabels,
						},
						Spec: *baseVMISpec,
					},
				},
			},
		},
	}
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

func GetVMPoolCirros() *poolv1.VirtualMachinePool {
	vmPool := getBaseVMPool(VMPoolCirros, 3, map[string]string{
		"kubevirt.io/vmpool": VMPoolCirros,
	})

	addContainerDisk(&vmPool.Spec.VirtualMachineTemplate.Spec.Template.Spec, fmt.Sprintf("%s/%s:%s", DockerPrefix, imageCirros, DockerTag), v1.DiskBusVirtio)
	return vmPool
}

func GetVMIReplicaSetCirros() *v1.VirtualMachineInstanceReplicaSet {
	vmReplicaSet := getBaseVMIReplicaSet(VMIReplicaSetCirros, 3, map[string]string{
		"kubevirt.io/vmReplicaSet": VMIReplicaSetCirros,
	})

	addContainerDisk(&vmReplicaSet.Spec.Template.Spec, fmt.Sprintf(strFmt, DockerPrefix, imageCirros, DockerTag), v1.DiskBusVirtio)
	return vmReplicaSet
}

func GetVMIMigration() *v1.VirtualMachineInstanceMigration {
	return &v1.VirtualMachineInstanceMigration{
		TypeMeta: metav1.TypeMeta{
			APIVersion: v1.GroupVersion.String(),
			Kind:       "VirtualMachineInstanceMigration",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: VMIMigration,
		},
		Spec: v1.VirtualMachineInstanceMigrationSpec{
			VMIName: VMIMigratable,
		},
	}
}

func GetMigrationPolicy() *v1alpha1.MigrationPolicy {
	policy := kubecli.NewMinimalMigrationPolicy(MigrationPolicyName)
	policy.Spec = v1alpha1.MigrationPolicySpec{
		AllowAutoConverge:       pointer.P(false),
		BandwidthPerMigration:   pointer.P(resource.MustParse("2000Mi")),
		CompletionTimeoutPerGiB: pointer.P(int64(123456789)),
		AllowPostCopy:           pointer.P(false),
		Selectors: &v1alpha1.Selectors{
			NamespaceSelector:              map[string]string{"namespace-key": "namespace-value"},
			VirtualMachineInstanceSelector: map[string]string{"vmi-key": "vmi-value"},
		},
	}

	return policy
}

func GetVMIWithHookSidecar() *v1.VirtualMachineInstance {
	return libvmi.New(
		libvmi.WithName(VMIWithHookSidecar),
		libvmi.WithResourceMemory("1024M"),
		libvmi.WithContainerDisk("containerdisk", fmt.Sprintf(strFmt, DockerPrefix, imageFedora, DockerTag)),
		libvmi.WithRng(),
		libvmi.WithCloudInitNoCloud(
			cloudinit.WithNoCloudUserData(generateCloudConfigString(cloudConfigUserPassword)),
		),
		libvmi.WithAnnotation(
			"hooks.kubevirt.io/hookSidecars",
			fmt.Sprintf("[{\"args\": [\"--version\", \"v1alpha2\"], \"image\": \"%s/example-hook-sidecar:%s\"}]", DockerPrefix, DockerTag),
		),
		libvmi.WithAnnotation(
			"smbios.vm.kubevirt.io/baseBoardManufacturer",
			"Radical Edward",
		),
	)
}

func GetVMIWithHookSidecarConfigMap() *v1.VirtualMachineInstance {
	return libvmi.New(
		libvmi.WithName(VMIWithHookSidecarConfigMap),
		libvmi.WithResourceMemory("1024M"),
		libvmi.WithContainerDisk("containerdisk", fmt.Sprintf(strFmt, DockerPrefix, imageFedora, DockerTag)),
		libvmi.WithRng(),
		libvmi.WithCloudInitNoCloud(
			cloudinit.WithNoCloudUserData(generateCloudConfigString(cloudConfigUserPassword)),
		),
		libvmi.WithAnnotation(
			"hooks.kubevirt.io/hookSidecars",
			`[{"args": ["--version", "v1alpha2"], "configMap": {"name": "my-config-map",`+
				`"key": "my_script.sh", "hookPath": "/usr/bin/onDefineDomain"}}]`,
		),
	)
}

func GetVMIGPU() *v1.VirtualMachineInstance {
	return libvmi.New(
		libvmi.WithName(VMIGPU),
		libvmi.WithResourceMemory("1024M"),
		libvmi.WithContainerDisk("containerdisk", fmt.Sprintf(strFmt, DockerPrefix, imageFedora, DockerTag)),
		libvmi.WithRng(),
		libvmi.WithCloudInitNoCloud(
			cloudinit.WithNoCloudUserData(generateCloudConfigString(cloudConfigUserPassword)),
		),
		libvmi.WithGPU(
			v1.GPU{
				Name:       "gpu1",
				DeviceName: "nvidia.com/GP102GL_Tesla_P40",
			},
		),
	)
}

func GetVMIUSB() *v1.VirtualMachineInstance {
	return libvmi.New(
		libvmi.WithName(VMIUSB),
		libvmi.WithResourceMemory("1024M"),
		libvmi.WithContainerDisk("containerdisk", fmt.Sprintf(strFmt, DockerPrefix, imageFedora, DockerTag)),
		libvmi.WithRng(),
		libvmi.WithCloudInitNoCloud(
			cloudinit.WithNoCloudUserData(generateCloudConfigString(cloudConfigUserPassword, cloudConfigInstallAndStartService)),
		),
		libvmi.WithHostDevice(
			v1.HostDevice{
				Name:       "node-usb-to-vmi-storage",
				DeviceName: "kubevirt.io/storage",
			},
		),
	)
}

func generateCloudConfigString(cloudConfigElement ...string) string {
	return strings.Join(
		append([]string{cloudConfigHeader}, cloudConfigElement...), "\n")
}
