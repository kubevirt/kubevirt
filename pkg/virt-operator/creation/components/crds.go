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
package components

import (
	extv1beta1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	virtv1 "kubevirt.io/kubevirt/pkg/api/v1"
)

func newBlankCrd() *extv1beta1.CustomResourceDefinition {
	return &extv1beta1.CustomResourceDefinition{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "apiextensions.k8s.io/v1beta1",
			Kind:       "CustomResourceDefinition",
		},
		ObjectMeta: metav1.ObjectMeta{
			Labels: map[string]string{
				"kubevirt.io": "",
			},
		},
	}
}

func NewVirtualMachineInstanceCrd() *extv1beta1.CustomResourceDefinition {
	crd := newBlankCrd()

	crd.ObjectMeta.Name = "virtualmachineinstances." + virtv1.VirtualMachineInstanceGroupVersionKind.Group
	crd.Spec = extv1beta1.CustomResourceDefinitionSpec{
		Group:   virtv1.VirtualMachineInstanceGroupVersionKind.Group,
		Version: virtv1.VirtualMachineInstanceGroupVersionKind.Version,
		Scope:   "Namespaced",

		Names: extv1beta1.CustomResourceDefinitionNames{
			Plural:     "virtualmachineinstances",
			Singular:   "virtualmachineinstance",
			Kind:       virtv1.VirtualMachineInstanceGroupVersionKind.Kind,
			ShortNames: []string{"vmi", "vmis"},
		},
		AdditionalPrinterColumns: []extv1beta1.CustomResourceColumnDefinition{
			{Name: "Age", Type: "date", JSONPath: ".metadata.creationTimestamp"},
			{Name: "Phase", Type: "string", JSONPath: ".status.phase"},
			{Name: "IP", Type: "string", JSONPath: ".status.interfaces[0].ipAddress"},
			{Name: "NodeName", Type: "string", JSONPath: ".status.nodeName"},
		},
	}

	return crd
}

func NewVirtualMachineCrd() *extv1beta1.CustomResourceDefinition {
	crd := newBlankCrd()

	crd.ObjectMeta.Name = "virtualmachines." + virtv1.VirtualMachineGroupVersionKind.Group
	crd.Spec = extv1beta1.CustomResourceDefinitionSpec{
		Group:   virtv1.VirtualMachineGroupVersionKind.Group,
		Version: virtv1.VirtualMachineGroupVersionKind.Version,
		Scope:   "Namespaced",

		Names: extv1beta1.CustomResourceDefinitionNames{
			Plural:     "virtualmachines",
			Singular:   "virtualmachine",
			Kind:       virtv1.VirtualMachineGroupVersionKind.Kind,
			ShortNames: []string{"vm", "vms"},
		},
		AdditionalPrinterColumns: []extv1beta1.CustomResourceColumnDefinition{
			{Name: "Age", Type: "date", JSONPath: ".metadata.creationTimestamp"},
			{Name: "Running", Type: "boolean", JSONPath: ".spec.running"},
			{Name: "Volume", Description: "Primary Volume", Type: "string", JSONPath: ".spec.volumes[0].name"},
		},
	}

	return crd
}

func NewPresetCrd() *extv1beta1.CustomResourceDefinition {
	crd := newBlankCrd()

	crd.ObjectMeta.Name = "virtualmachineinstancepresets." + virtv1.VirtualMachineInstancePresetGroupVersionKind.Group
	crd.Spec = extv1beta1.CustomResourceDefinitionSpec{
		Group:   virtv1.VirtualMachineInstancePresetGroupVersionKind.Group,
		Version: virtv1.VirtualMachineInstancePresetGroupVersionKind.Version,
		Scope:   "Namespaced",

		Names: extv1beta1.CustomResourceDefinitionNames{
			Plural:     "virtualmachineinstancepresets",
			Singular:   "virtualmachineinstancepreset",
			Kind:       virtv1.VirtualMachineInstancePresetGroupVersionKind.Kind,
			ShortNames: []string{"vmipreset", "vmipresets"},
		},
	}

	return crd
}

func NewReplicaSetCrd() *extv1beta1.CustomResourceDefinition {
	crd := newBlankCrd()

	crd.ObjectMeta.Name = "virtualmachineinstancereplicasets." + virtv1.VirtualMachineInstanceReplicaSetGroupVersionKind.Group
	crd.Spec = extv1beta1.CustomResourceDefinitionSpec{
		Group:   virtv1.VirtualMachineInstanceReplicaSetGroupVersionKind.Group,
		Version: virtv1.VirtualMachineInstanceReplicaSetGroupVersionKind.Version,
		Scope:   "Namespaced",

		Names: extv1beta1.CustomResourceDefinitionNames{
			Plural:     "virtualmachineinstancereplicasets",
			Singular:   "virtualmachineinstancereplicaset",
			Kind:       virtv1.VirtualMachineInstanceReplicaSetGroupVersionKind.Kind,
			ShortNames: []string{"vmirs", "vmirss"},
		},
		AdditionalPrinterColumns: []extv1beta1.CustomResourceColumnDefinition{
			{Name: "Desired", Type: "integer", JSONPath: ".spec.replicas",
				Description: "Number of desired VirtualMachineInstances"},
			{Name: "Current", Type: "integer", JSONPath: ".status.replicas",
				Description: "Number of managed and not final or deleted VirtualMachineInstances"},
			{Name: "Ready", Type: "integer", JSONPath: ".status.readyReplicas",
				Description: "Number of managed VirtualMachineInstances which are ready to receive traffic"},
			{Name: "Age", Type: "date", JSONPath: ".metadata.creationTimestamp"},
		},
	}

	return crd
}

func NewVirtualMachineInstanceMigrationCrd() *extv1beta1.CustomResourceDefinition {
	crd := newBlankCrd()

	crd.ObjectMeta.Name = "virtualmachineinstancemigrations." + virtv1.VirtualMachineInstanceMigrationGroupVersionKind.Group
	crd.Spec = extv1beta1.CustomResourceDefinitionSpec{
		Group:   virtv1.VirtualMachineInstanceMigrationGroupVersionKind.Group,
		Version: virtv1.VirtualMachineInstanceMigrationGroupVersionKind.Version,
		Scope:   "Namespaced",

		Names: extv1beta1.CustomResourceDefinitionNames{
			Plural:     "virtualmachineinstancemigrations",
			Singular:   "virtualmachineinstancemigration",
			Kind:       virtv1.VirtualMachineInstanceMigrationGroupVersionKind.Kind,
			ShortNames: []string{"vmim", "vmims"},
		},
	}

	return crd
}
