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

package libvmi

import (
	kvirtv1 "kubevirt.io/api/core/v1"

	"kubevirt.io/kubevirt/pkg/pointer"
	cd "kubevirt.io/kubevirt/tests/containerdisk"
	"kubevirt.io/kubevirt/tests/framework/checks"
	"kubevirt.io/kubevirt/tests/testsuite"
)

const (
	windowsDiskName = "windows-disk"
	WindowsFirmware = "5d307ca9-b3ef-428c-8861-06e72d69f223"
	WindowsPVCName  = "disk-windows"
)

// NewFedora instantiates a new Fedora based VMI configuration,
// building its extra properties based on the specified With* options.
// This image has tooling for the guest agent, stress, SR-IOV and more.
func NewFedora(opts ...Option) *kvirtv1.VirtualMachineInstance {
	fedoraOptions := []Option{
		WithResourceMemory("512Mi"),
		WithRng(),
		WithContainerImage(cd.ContainerDiskFor(cd.ContainerDiskFedoraTestTooling)),
	}
	opts = append(fedoraOptions, opts...)
	return New(opts...)
}

// NewCirros instantiates a new CirrOS based VMI configuration
func NewCirros(opts ...Option) *kvirtv1.VirtualMachineInstance {
	// Supplied with no user data, Cirros image takes 230s to allow login
	withNonEmptyUserData := WithCloudInitNoCloudUserData("#!/bin/bash\necho hello\n", true)

	cirrosOpts := []Option{
		WithContainerImage(cd.ContainerDiskFor(cd.ContainerDiskCirros)),
		withNonEmptyUserData,
		WithResourceMemory(cirrosMemory()),
	}
	cirrosOpts = append(cirrosOpts, opts...)
	return New(cirrosOpts...)
}

// NewAlpine instantiates a new Alpine based VMI configuration
func NewAlpine(opts ...Option) *kvirtv1.VirtualMachineInstance {
	alpineMemory := cirrosMemory
	alpineOpts := []Option{
		WithContainerImage(cd.ContainerDiskFor(cd.ContainerDiskAlpine)),
		WithResourceMemory(alpineMemory()),
		WithRng(),
	}
	alpineOpts = append(alpineOpts, opts...)
	return New(alpineOpts...)
}

func NewAlpineWithTestTooling(opts ...Option) *kvirtv1.VirtualMachineInstance {
	// Supplied with no user data, AlpimeWithTestTooling image takes more than 200s to allow login
	withNonEmptyUserData := WithCloudInitNoCloudUserData("#!/bin/bash\necho hello\n", true)
	alpineMemory := cirrosMemory
	alpineOpts := []Option{
		WithContainerImage(cd.ContainerDiskFor(cd.ContainerDiskAlpineTestTooling)),
		withNonEmptyUserData,
		WithResourceMemory(alpineMemory()),
		WithRng(),
	}
	alpineOpts = append(alpineOpts, opts...)
	return New(alpineOpts...)
}

func cirrosMemory() string {
	if checks.IsARM64(testsuite.Arch) {
		return "256Mi"
	}
	return "128Mi"
}

func NewWindows(opts ...Option) *kvirtv1.VirtualMachineInstance {
	const cpuCount = 2
	const featureSpinlocks = 8191
	windowsOpts := []Option{
		WithTerminationGracePeriod(0),
		WithCPUCount(cpuCount, cpuCount, cpuCount),
		WithResourceMemory("2048Mi"),
		WithEphemeralPersistentVolumeClaim(windowsDiskName, WindowsPVCName),
	}

	windowsOpts = append(windowsOpts, opts...)
	vmi := New(windowsOpts...)

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
