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

	kvirtv1 "kubevirt.io/client-go/api/v1"
)

// Option represents an action that enables an option.
type Option func(vmi *kvirtv1.VirtualMachineInstance)

// New instantiates a new VMI configuration,
// building its properties based on the specified With* options.
func New(name string, opts ...Option) *kvirtv1.VirtualMachineInstance {
	vmi := baseVmi(name)

	for _, f := range opts {
		f(vmi)
	}

	return vmi
}

// RandName returns a random name by concatenating the given name with a hyphen and a random string.
func RandName(name string) string {
	return name + "-" + rand.String(5)
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

func baseVmi(name string) *kvirtv1.VirtualMachineInstance {
	vmi := kvirtv1.NewVMIReferenceFromNameWithNS("", name)
	vmi.Spec = kvirtv1.VirtualMachineInstanceSpec{Domain: kvirtv1.DomainSpec{}}
	vmi.TypeMeta = k8smetav1.TypeMeta{
		APIVersion: kvirtv1.GroupVersion.String(),
		Kind:       "VirtualMachineInstance",
	}
	return vmi
}
