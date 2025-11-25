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

package libvmifact

import (
	"encoding/base64"

	kvirtv1 "kubevirt.io/api/core/v1"

	"kubevirt.io/kubevirt/pkg/libvmi"
	libvmici "kubevirt.io/kubevirt/pkg/libvmi/cloudinit"
	"kubevirt.io/kubevirt/pkg/pointer"

	cd "kubevirt.io/kubevirt/tests/containerdisk"
)

const (
	windowsDiskName = "windows-disk"
	WindowsFirmware = "5d307ca9-b3ef-428c-8861-06e72d69f223"
	WindowsPVCName  = "disk-windows"
)

// NewFedora instantiates a new Fedora based VMI configuration,
// building its extra properties based on the specified With* options.
// This image has tooling for the guest agent, stress, SR-IOV and more.
func NewFedora(opts ...libvmi.Option) *kvirtv1.VirtualMachineInstance {
	fedoraOptions := []libvmi.Option{
		libvmi.WithMemoryRequest("1Gi"),
		libvmi.WithRng(),
		libvmi.WithContainerDisk("disk0", cd.ContainerDiskFor(cd.ContainerDiskFedoraTestTooling)),
	}
	opts = append(fedoraOptions, opts...)
	return libvmi.New(opts...)
}

// NewCirros instantiates a new CirrOS based VMI configuration
func NewCirros(opts ...libvmi.Option) *kvirtv1.VirtualMachineInstance {
	cirrosOpts := []libvmi.Option{
		libvmi.WithContainerDisk("disk0", cd.ContainerDiskFor(cd.ContainerDiskCirros)),
		libvmi.WithMemoryRequest(cirrosMemory()),
	}
	cirrosOpts = append(cirrosOpts, opts...)
	vmi := libvmi.New(cirrosOpts...)

	// Supplied with no user data, Cirros image takes 230s to allow login
	if libvmi.GetCloudInitVolume(vmi) == nil {
		withNonEmptyUserData := libvmi.WithCloudInitNoCloud(WithDummyCloudForFastBoot())
		withNonEmptyUserData(vmi)
	}
	return vmi
}

// NewAlpine instantiates a new Alpine based VMI configuration
func NewAlpine(opts ...libvmi.Option) *kvirtv1.VirtualMachineInstance {
	alpineMemory := cirrosMemory
	alpineOpts := []libvmi.Option{
		libvmi.WithContainerDisk("disk0", cd.ContainerDiskFor(cd.ContainerDiskAlpine)),
		libvmi.WithMemoryRequest(alpineMemory()),
		libvmi.WithRng(),
	}
	alpineOpts = append(alpineOpts, opts...)
	return libvmi.New(alpineOpts...)
}

func NewAlpineWithTestTooling(opts ...libvmi.Option) *kvirtv1.VirtualMachineInstance {
	alpineMemory := cirrosMemory
	alpineOpts := []libvmi.Option{
		libvmi.WithContainerDisk("disk0", cd.ContainerDiskFor(cd.ContainerDiskAlpineTestTooling)),
		libvmi.WithMemoryRequest(alpineMemory()),
		libvmi.WithRng(),
	}
	alpineOpts = append(alpineOpts, opts...)
	vmi := libvmi.New(alpineOpts...)

	// Supplied with no user data, AlpimeWithTestTooling image takes more than 200s to allow login
	if libvmi.GetCloudInitVolume(vmi) == nil {
		withNonEmptyUserData := libvmi.WithCloudInitNoCloud(WithDummyCloudForFastBoot())
		withNonEmptyUserData(vmi)
	}
	return vmi
}

func NewGuestless(opts ...libvmi.Option) *kvirtv1.VirtualMachineInstance {
	opts = append(
		[]libvmi.Option{libvmi.WithMemoryRequest(qemuMinimumMemory())},
		opts...)
	return libvmi.New(opts...)
}

func qemuMinimumMemory() string {
	if isARM64() {
		// required to start qemu on ARM with UEFI firmware
		// https://github.com/kubevirt/kubevirt/pull/11366#issuecomment-1970247448
		const armMinimalBootableMemory = "128Mi"
		return armMinimalBootableMemory
	}
	return "1Mi"
}

func cirrosMemory() string {
	if isARM64() {
		return "256Mi"
	}
	return "128Mi"
}

func NewWindows(opts ...libvmi.Option) *kvirtv1.VirtualMachineInstance {
	const cpuCount = 2
	const featureSpinlocks = 8191
	windowsOpts := []libvmi.Option{
		libvmi.WithTerminationGracePeriod(0),
		libvmi.WithCPUCount(cpuCount, cpuCount, cpuCount),
		libvmi.WithMemoryRequest("2048Mi"),
		libvmi.WithEphemeralPersistentVolumeClaim(windowsDiskName, WindowsPVCName),
	}

	windowsOpts = append(windowsOpts, opts...)
	vmi := libvmi.New(windowsOpts...)

	vmi.Spec.Domain.Features = &kvirtv1.Features{
		ACPI: kvirtv1.FeatureState{},
		APIC: &kvirtv1.FeatureAPIC{},
		Hyperv: &kvirtv1.FeatureHyperv{
			Relaxed:    &kvirtv1.FeatureState{},
			SyNICTimer: &kvirtv1.SyNICTimer{Direct: &kvirtv1.FeatureState{}},
			VAPIC:      &kvirtv1.FeatureState{},
			Spinlocks:  &kvirtv1.FeatureSpinlocks{Retries: pointer.P(uint32(featureSpinlocks))},
		},
	}
	vmi.Spec.Domain.Clock = &kvirtv1.Clock{
		ClockOffset: kvirtv1.ClockOffset{UTC: &kvirtv1.ClockOffsetUTC{}},
		Timer: &kvirtv1.Timer{
			HPET:   &kvirtv1.HPETTimer{Enabled: pointer.P(false)},
			PIT:    &kvirtv1.PITTimer{TickPolicy: kvirtv1.PITTickPolicyDelay},
			RTC:    &kvirtv1.RTCTimer{TickPolicy: kvirtv1.RTCTickPolicyCatchup},
			Hyperv: &kvirtv1.HypervTimer{},
		},
	}
	vmi.Spec.Domain.Firmware = &kvirtv1.Firmware{UUID: WindowsFirmware}
	return vmi
}

func WithDummyCloudForFastBoot() libvmici.NoCloudOption {
	return func(source *kvirtv1.CloudInitNoCloudSource) {
		source.UserDataBase64 = base64.StdEncoding.EncodeToString([]byte("#!/bin/bash\necho 'hello'\n"))
	}
}
