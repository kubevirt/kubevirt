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
	k8sv1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	k8smetav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/rand"

	v1 "kubevirt.io/api/core/v1"

	"kubevirt.io/kubevirt/pkg/pointer"
)

// Option represents an action that enables an option.
type Option func(vmi *v1.VirtualMachineInstance)

// New instantiates a new VMI configuration,
// building its properties based on the specified With* options.
func New(opts ...Option) *v1.VirtualMachineInstance {
	vmi := baseVmi(randName())

	WithTerminationGracePeriod(0)(vmi)
	for _, f := range opts {
		f(vmi)
	}

	return vmi
}

var defaultOptions []Option

func RegisterDefaultOption(opt Option) {
	defaultOptions = append(defaultOptions, opt)
}

// randName returns a random name for a virtual machine
func randName() string {
	const randomPostfixLen = 5
	return "testvmi" + "-" + rand.String(randomPostfixLen)
}

// WithLabel sets a label with specified value
func WithLabel(key, value string) Option {
	return func(vmi *v1.VirtualMachineInstance) {
		if vmi.Labels == nil {
			vmi.Labels = map[string]string{}
		}
		vmi.Labels[key] = value
	}
}

// WithAnnotation adds an annotation with specified value
func WithAnnotation(key, value string) Option {
	return func(vmi *v1.VirtualMachineInstance) {
		if vmi.Annotations == nil {
			vmi.Annotations = map[string]string{}
		}
		vmi.Annotations[key] = value
	}
}

func WithNamespace(namespace string) Option {
	return func(vmi *v1.VirtualMachineInstance) {
		vmi.Namespace = namespace
	}
}

// WithTerminationGracePeriod specifies the termination grace period in seconds.
func WithTerminationGracePeriod(seconds int64) Option {
	return func(vmi *v1.VirtualMachineInstance) {
		vmi.Spec.TerminationGracePeriodSeconds = &seconds
	}
}

// WithRng adds `rng` to the vmi devices.
func WithRng() Option {
	return func(vmi *v1.VirtualMachineInstance) {
		vmi.Spec.Domain.Devices.Rng = &v1.Rng{}
	}
}

// WithWatchdog adds a watchdog to the vmi devices.
func WithWatchdog(action v1.WatchdogAction) Option {
	return func(vmi *v1.VirtualMachineInstance) {
		vmi.Spec.Domain.Devices.Watchdog = &v1.Watchdog{
			Name: "watchdog",
			WatchdogDevice: v1.WatchdogDevice{
				I6300ESB: &v1.I6300ESBWatchdog{
					Action: action,
				},
			},
		}
	}
}

// WithResourceMemory specifies the vmi memory resource.
func WithResourceMemory(value string) Option {
	return func(vmi *v1.VirtualMachineInstance) {
		if vmi.Spec.Domain.Resources.Requests == nil {
			vmi.Spec.Domain.Resources.Requests = k8sv1.ResourceList{}
		}
		vmi.Spec.Domain.Resources.Requests[k8sv1.ResourceMemory] = resource.MustParse(value)
	}
}

// WithResourceCPU specifies the vmi CPU resource.
func WithResourceCPU(value string) Option {
	return func(vmi *v1.VirtualMachineInstance) {
		if vmi.Spec.Domain.Resources.Requests == nil {
			vmi.Spec.Domain.Resources.Requests = k8sv1.ResourceList{}
		}
		vmi.Spec.Domain.Resources.Requests[k8sv1.ResourceCPU] = resource.MustParse(value)
	}
}

// WithLimitMemory specifies the VMI memory limit.
func WithLimitMemory(value string) Option {
	return func(vmi *v1.VirtualMachineInstance) {
		if vmi.Spec.Domain.Resources.Limits == nil {
			vmi.Spec.Domain.Resources.Limits = k8sv1.ResourceList{}
		}
		vmi.Spec.Domain.Resources.Limits[k8sv1.ResourceMemory] = resource.MustParse(value)
	}
}

// WithLimitCPU specifies the VMI CPU limit.
func WithLimitCPU(value string) Option {
	return func(vmi *v1.VirtualMachineInstance) {
		if vmi.Spec.Domain.Resources.Limits == nil {
			vmi.Spec.Domain.Resources.Limits = k8sv1.ResourceList{}
		}
		vmi.Spec.Domain.Resources.Limits[k8sv1.ResourceCPU] = resource.MustParse(value)
	}
}

func WithDownwardMetricsVolume(volumeName string) Option {
	return func(vmi *v1.VirtualMachineInstance) {
		vmi.Spec.Volumes = append(vmi.Spec.Volumes, v1.Volume{
			Name: volumeName,
			VolumeSource: v1.VolumeSource{
				DownwardMetrics: &v1.DownwardMetricsVolumeSource{},
			},
		})

		vmi.Spec.Domain.Devices.Disks = append(vmi.Spec.Domain.Devices.Disks, v1.Disk{
			Name: volumeName,
			DiskDevice: v1.DiskDevice{
				Disk: &v1.DiskTarget{
					Bus: v1.DiskBusVirtio,
				},
			},
		})
	}
}

func WithDownwardMetricsChannel() Option {
	return func(vmi *v1.VirtualMachineInstance) {
		vmi.Spec.Domain.Devices.DownwardMetrics = &v1.DownwardMetrics{}
	}
}

// WithUefi configures EFI bootloader and SecureBoot.
func WithUefi(secureBoot bool) Option {
	return func(vmi *v1.VirtualMachineInstance) {
		if vmi.Spec.Domain.Firmware == nil {
			vmi.Spec.Domain.Firmware = &v1.Firmware{}
		}
		if vmi.Spec.Domain.Firmware.Bootloader == nil {
			vmi.Spec.Domain.Firmware.Bootloader = &v1.Bootloader{}
		}
		if vmi.Spec.Domain.Firmware.Bootloader.EFI == nil {
			vmi.Spec.Domain.Firmware.Bootloader.EFI = &v1.EFI{}
		}
		vmi.Spec.Domain.Firmware.Bootloader.EFI.SecureBoot = pointer.P(secureBoot)
		// secureBoot Requires SMM to be enabled
		if secureBoot {
			if vmi.Spec.Domain.Features == nil {
				vmi.Spec.Domain.Features = &v1.Features{}
			}
			if vmi.Spec.Domain.Features.SMM == nil {
				vmi.Spec.Domain.Features.SMM = &v1.FeatureState{}
			}
			vmi.Spec.Domain.Features.SMM.Enabled = pointer.P(secureBoot)
		}
	}
}

// WithSEV adds `launchSecurity` with `sev`.
func WithSEV(isESEnabled bool) Option {
	return func(vmi *v1.VirtualMachineInstance) {
		vmi.Spec.Domain.LaunchSecurity = &v1.LaunchSecurity{
			SEV: &v1.SEV{
				Policy: &v1.SEVPolicy{
					EncryptedState: &isESEnabled,
				},
			},
		}
	}
}

func WithSEVAttestation() Option {
	return func(vmi *v1.VirtualMachineInstance) {
		startStrategy := v1.StartStrategyPaused
		vmi.Spec.StartStrategy = &startStrategy
		if vmi.Spec.Domain.LaunchSecurity == nil {
			vmi.Spec.Domain.LaunchSecurity = &v1.LaunchSecurity{}
		}
		if vmi.Spec.Domain.LaunchSecurity.SEV == nil {
			vmi.Spec.Domain.LaunchSecurity.SEV = &v1.SEV{}
		}
		vmi.Spec.Domain.LaunchSecurity.SEV.Attestation = &v1.SEVAttestation{}
	}
}

func WithCPUFeature(featureName, policy string) Option {
	return func(vmi *v1.VirtualMachineInstance) {
		if vmi.Spec.Domain.CPU == nil {
			vmi.Spec.Domain.CPU = &v1.CPU{}
		}

		vmi.Spec.Domain.CPU.Features = append(vmi.Spec.Domain.CPU.Features, v1.CPUFeature{
			Name:   featureName,
			Policy: policy,
		})
	}
}

func WithEvictionStrategy(evictionStrategy v1.EvictionStrategy) Option {
	return func(vmi *v1.VirtualMachineInstance) {
		vmi.Spec.EvictionStrategy = &evictionStrategy
	}
}

func WithStartStrategy(startStrategy v1.StartStrategy) Option {
	return func(vmi *v1.VirtualMachineInstance) {
		vmi.Spec.StartStrategy = &startStrategy
	}
}

func WithoutSerialConsole() Option {
	return func(vmi *v1.VirtualMachineInstance) {
		enabled := false
		vmi.Spec.Domain.Devices.AutoattachSerialConsole = &enabled
	}
}

func baseVmi(name string) *v1.VirtualMachineInstance {
	vmi := v1.NewVMIReferenceFromNameWithNS("", name)
	vmi.Spec = v1.VirtualMachineInstanceSpec{Domain: v1.DomainSpec{}}
	vmi.TypeMeta = k8smetav1.TypeMeta{
		APIVersion: v1.GroupVersion.String(),
		Kind:       "VirtualMachineInstance",
	}

	for _, opt := range defaultOptions {
		opt(vmi)
	}

	return vmi
}
