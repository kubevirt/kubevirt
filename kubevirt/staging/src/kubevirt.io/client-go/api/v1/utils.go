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
 * Copyright 2017 Red Hat, Inc.
 *
 */

package v1

import (
	k8sv1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	k8smetav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"kubevirt.io/client-go/precond"
)

// This is meant for testing
func NewMinimalVMI(name string) *VirtualMachineInstance {
	return NewMinimalVMIWithNS(k8sv1.NamespaceDefault, name)
}

// This is meant for testing
func NewMinimalVMIWithNS(namespace, name string) *VirtualMachineInstance {
	precond.CheckNotEmpty(name)
	vmi := NewVMIReferenceFromNameWithNS(namespace, name)
	vmi.Spec = VirtualMachineInstanceSpec{Domain: DomainSpec{}}
	vmi.Spec.Domain.Resources.Requests = k8sv1.ResourceList{
		k8sv1.ResourceMemory: resource.MustParse("8192Ki"),
	}
	vmi.TypeMeta = k8smetav1.TypeMeta{
		APIVersion: GroupVersion.String(),
		Kind:       "VirtualMachineInstance",
	}
	return vmi
}
