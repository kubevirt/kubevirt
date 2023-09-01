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

package v1beta1

import (
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/rand"

	instancetypev1beta1 "kubevirt.io/api/instancetype/v1beta1"
)

type InstancetypeSpecOption func(*instancetypev1beta1.VirtualMachineInstancetypeSpec)
type InstancetypeOption func(*instancetypev1beta1.VirtualMachineInstancetype)
type ClusterInstancetypeOption func(*instancetypev1beta1.VirtualMachineClusterInstancetype)

func NewInstancetypeSpec(opts ...InstancetypeSpecOption) instancetypev1beta1.VirtualMachineInstancetypeSpec {
	spec := &instancetypev1beta1.VirtualMachineInstancetypeSpec{}
	for _, f := range opts {
		f(spec)
	}
	return *spec
}

func WithCPUs(vcpus uint32) InstancetypeSpecOption {
	return func(spec *instancetypev1beta1.VirtualMachineInstancetypeSpec) {
		spec.CPU.Guest = vcpus
	}
}

func WithMemory(memory resource.Quantity) InstancetypeSpecOption {
	return func(spec *instancetypev1beta1.VirtualMachineInstancetypeSpec) {
		spec.Memory.Guest = memory
	}
}

func NewInstancetype(opts ...InstancetypeOption) *instancetypev1beta1.VirtualMachineInstancetype {
	instancetype := baseInstancetype(randInstancetypeName())
	for _, f := range opts {
		f(instancetype)
	}
	return instancetype
}

func WithInstancetypeSpec(spec instancetypev1beta1.VirtualMachineInstancetypeSpec) InstancetypeOption {
	return func(instancetype *instancetypev1beta1.VirtualMachineInstancetype) {
		instancetype.Spec = spec
	}
}

func NewClusterInstancetype(opts ...ClusterInstancetypeOption) *instancetypev1beta1.VirtualMachineClusterInstancetype {
	instancetype := baseClusterInstancetype(randInstancetypeName())
	for _, f := range opts {
		f(instancetype)
	}
	return instancetype
}

func WithClusterInstancetypeSpec(spec instancetypev1beta1.VirtualMachineInstancetypeSpec) ClusterInstancetypeOption {
	return func(instancetype *instancetypev1beta1.VirtualMachineClusterInstancetype) {
		instancetype.Spec = spec
	}
}

func baseInstancetype(name string) *instancetypev1beta1.VirtualMachineInstancetype {
	return &instancetypev1beta1.VirtualMachineInstancetype{
		ObjectMeta: metav1.ObjectMeta{
			Name: name,
		},
	}
}

func baseClusterInstancetype(name string) *instancetypev1beta1.VirtualMachineClusterInstancetype {
	return &instancetypev1beta1.VirtualMachineClusterInstancetype{
		ObjectMeta: metav1.ObjectMeta{
			Name: name,
		},
	}
}

func randInstancetypeName() string {
	const randomPostfixLen = 5
	return "instancetype" + "-" + rand.String(randomPostfixLen)
}
