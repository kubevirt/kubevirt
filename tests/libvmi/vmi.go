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
	"k8s.io/utils/pointer"

	kvirtv1 "kubevirt.io/api/core/v1"
	v1 "kubevirt.io/api/core/v1"
)

const defaultTestGracePeriod int64 = 0

var (
	arch string
)

func Setup(ar string) {
	arch = ar
}

func isARM64() bool {
	return arch == "arm64"
}

// Option represents an action that enables an option.
type Option func(vmi *kvirtv1.VirtualMachineInstance)

// New instantiates a new VMI configuration,
// building its properties based on the specified With* options.
func New(name string, opts ...Option) *kvirtv1.VirtualMachineInstance {
	vmi := baseVmi(name, "")
	return applyOptions(vmi, opts...)
}

// NewWithNamespace instantiates a new VMI configuration,
// building its properties based on the specified With* options.
func NewWithNamespace(namespace string, opts ...Option) *kvirtv1.VirtualMachineInstance {
	vmi := baseVmi(RandName(DefaultVmiName), namespace)
	return applyOptions(vmi, opts...)
}

func applyOptions(vmi *kvirtv1.VirtualMachineInstance, opts ...Option) *kvirtv1.VirtualMachineInstance {
	for _, f := range opts {
		f(vmi)
	}

	return vmi
}

// RandName returns a random name by concatenating the given name with a hyphen and a random string.
func RandName(name string) string {
	return name + "-" + rand.String(5)
}

// WithLabel sets a label with specified value
func WithLabel(key, value string) Option {
	return func(vmi *kvirtv1.VirtualMachineInstance) {
		if vmi.Labels == nil {
			vmi.Labels = map[string]string{}
		}
		vmi.Labels[key] = value
	}
}

// WithAnnotation adds an annotation with specified value
func WithAnnotation(key, value string) Option {
	return func(vmi *kvirtv1.VirtualMachineInstance) {
		if vmi.Annotations == nil {
			vmi.Annotations = map[string]string{}
		}
		vmi.Annotations[key] = value
	}
}

// WithTerminationGracePeriod specifies the termination grace period in seconds.
func WithTerminationGracePeriod(seconds int64) Option {
	return func(vmi *kvirtv1.VirtualMachineInstance) {
		vmi.Spec.TerminationGracePeriodSeconds = &seconds
	}
}

// WithRng adds `rng` to the the vmi devices.
func WithRng() Option {
	return func(vmi *kvirtv1.VirtualMachineInstance) {
		vmi.Spec.Domain.Devices.Rng = &kvirtv1.Rng{}
	}
}

// WithResourceMemory specifies the vmi memory resource.
func WithResourceMemory(value string) Option {
	return func(vmi *kvirtv1.VirtualMachineInstance) {
		vmi.Spec.Domain.Resources.Requests = k8sv1.ResourceList{
			k8sv1.ResourceMemory: resource.MustParse(value),
		}
	}
}

// WithNodeSelectorFor ensures that the VMI gets scheduled on the specified node
func WithNodeSelectorFor(node *k8sv1.Node) Option {
	return func(vmi *kvirtv1.VirtualMachineInstance) {
		if vmi.Spec.NodeSelector == nil {
			vmi.Spec.NodeSelector = map[string]string{}
		}
		vmi.Spec.NodeSelector["kubernetes.io/hostname"] = node.Name
	}
}

// WithUefi configures EFI bootloader and SecureBoot.
func WithUefi(secureBoot bool) Option {
	return func(vmi *kvirtv1.VirtualMachineInstance) {
		vmi.Spec.Domain.Firmware = &v1.Firmware{
			Bootloader: &v1.Bootloader{
				EFI: &v1.EFI{
					SecureBoot: pointer.BoolPtr(secureBoot),
				},
			},
		}
	}
}

// WithSEV adds `launchSecurity` with `sev`.
func WithSEV() Option {
	return func(vmi *kvirtv1.VirtualMachineInstance) {
		vmi.Spec.Domain.LaunchSecurity = &v1.LaunchSecurity{
			SEV: &v1.SEV{},
		}
	}
}

func baseVmi(name, namespace string) *kvirtv1.VirtualMachineInstance {
	vmi := kvirtv1.NewVMIReferenceFromNameWithNS(namespace, name)
	vmi.Spec = kvirtv1.VirtualMachineInstanceSpec{Domain: kvirtv1.DomainSpec{}}
	vmi.TypeMeta = k8smetav1.TypeMeta{
		APIVersion: kvirtv1.GroupVersion.String(),
		Kind:       "VirtualMachineInstance",
	}

	t := defaultTestGracePeriod
	vmi.Spec.TerminationGracePeriodSeconds = &t

	// To avoid mac address issue in the tests change the pod interface binding to masquerade
	// https://github.com/kubevirt/kubevirt/issues/1494
	vmi.Spec.Domain.Devices = v1.Devices{Interfaces: []v1.Interface{{Name: "default",
		InterfaceBindingMethod: v1.InterfaceBindingMethod{
			Masquerade: &v1.InterfaceMasquerade{}}}}}

	vmi.Spec.Networks = []v1.Network{*v1.DefaultPodNetwork()}
	var resourceMemory resource.Quantity
	if isARM64() {
		// Cirros image need 256M to boot on ARM64,
		// this issue is traced in https://github.com/kubevirt/kubevirt/issues/6363
		resourceMemory = resource.MustParse("256Mi")
	} else {
		resourceMemory = resource.MustParse("128Mi")
	}
	vmi.Spec.Domain.Resources.Requests = k8sv1.ResourceList{
		k8sv1.ResourceMemory: resourceMemory,
	}

	return vmi
}
