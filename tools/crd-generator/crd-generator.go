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

package main

import (
	"flag"
	"fmt"

	crdutils "github.com/ant31/crd-validation/pkg"
	extensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	v1 "kubevirt.io/kubevirt/pkg/api/v1"
)

func generateBlankCrd() *extensionsv1.CustomResourceDefinition {
	return &extensionsv1.CustomResourceDefinition{
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

func generateVirtualMachineCrd() {
	crd := generateBlankCrd()

	crd.ObjectMeta.Name = "virtualmachines." + v1.VirtualMachineGroupVersionKind.Group
	crd.Spec = extensionsv1.CustomResourceDefinitionSpec{
		Group:   v1.VirtualMachineGroupVersionKind.Group,
		Version: v1.VirtualMachineGroupVersionKind.Version,
		Scope:   "Namespaced",

		Names: extensionsv1.CustomResourceDefinitionNames{
			Plural:     "virtualmachines",
			Singular:   "virtualmachine",
			Kind:       v1.VirtualMachineGroupVersionKind.Kind,
			ShortNames: []string{"vm", "vms"},
		},
		AdditionalPrinterColumns: []extensionsv1.CustomResourceColumnDefinition{
			{Name: "Age", Type: "date", JSONPath: ".metadata.creationTimestamp"},
			{Name: "Running", Type: "boolean", JSONPath: ".spec.running"},
			{Name: "Volume", Description: "Primary Volume", Type: "string", JSONPath: ".spec.volumes[0].name"},
		},
	}

	crdutils.MarshallCrd(crd, "yaml")
}

func generatePresetCrd() {
	crd := generateBlankCrd()

	crd.ObjectMeta.Name = "virtualmachineinstancepresets." + v1.VirtualMachineInstancePresetGroupVersionKind.Group
	crd.Spec = extensionsv1.CustomResourceDefinitionSpec{
		Group:   v1.VirtualMachineInstancePresetGroupVersionKind.Group,
		Version: v1.VirtualMachineInstancePresetGroupVersionKind.Version,
		Scope:   "Namespaced",

		Names: extensionsv1.CustomResourceDefinitionNames{
			Plural:     "virtualmachineinstancepresets",
			Singular:   "virtualmachineinstancepreset",
			Kind:       v1.VirtualMachineInstancePresetGroupVersionKind.Kind,
			ShortNames: []string{"vmipreset", "vmipresets"},
		},
	}

	crdutils.MarshallCrd(crd, "yaml")
}

func generateReplicaSetCrd() {
	crd := generateBlankCrd()

	crd.ObjectMeta.Name = "virtualmachineinstancereplicasets." + v1.VirtualMachineInstanceReplicaSetGroupVersionKind.Group
	crd.Spec = extensionsv1.CustomResourceDefinitionSpec{
		Group:   v1.VirtualMachineInstanceReplicaSetGroupVersionKind.Group,
		Version: v1.VirtualMachineInstanceReplicaSetGroupVersionKind.Version,
		Scope:   "Namespaced",

		Names: extensionsv1.CustomResourceDefinitionNames{
			Plural:     "virtualmachineinstancereplicasets",
			Singular:   "virtualmachineinstancereplicaset",
			Kind:       v1.VirtualMachineInstanceReplicaSetGroupVersionKind.Kind,
			ShortNames: []string{"vmirs", "vmirss"},
		},
		AdditionalPrinterColumns: []extensionsv1.CustomResourceColumnDefinition{
			{Name: "Desired", Type: "integer", JSONPath: ".spec.replicas",
				Description: "Number of desired VirtualMachineInstances"},
			{Name: "Current", Type: "integer", JSONPath: ".status.replicas",
				Description: "Number of managed and not final or deleted VirtualMachineInstances"},
			{Name: "Ready", Type: "integer", JSONPath: ".status.readyReplicas",
				Description: "Number of managed VirtualMachineInstances which are ready to receive traffic"},
			{Name: "Age", Type: "date", JSONPath: ".metadata.creationTimestamp"},
		},
	}

	crdutils.MarshallCrd(crd, "yaml")
}

func generateVirtualMachineInstanceMigrationCrd() {
	crd := generateBlankCrd()

	crd.ObjectMeta.Name = "virtualmachineinstancemigrations." + v1.VirtualMachineInstanceMigrationGroupVersionKind.Group
	crd.Spec = extensionsv1.CustomResourceDefinitionSpec{
		Group:   v1.VirtualMachineInstanceMigrationGroupVersionKind.Group,
		Version: v1.VirtualMachineInstanceMigrationGroupVersionKind.Version,
		Scope:   "Namespaced",

		Names: extensionsv1.CustomResourceDefinitionNames{
			Plural:     "virtualmachineinstancemigrations",
			Singular:   "virtualmachineinstancemigration",
			Kind:       v1.VirtualMachineInstanceMigrationGroupVersionKind.Kind,
			ShortNames: []string{"vmim", "vmims"},
		},
	}

	crdutils.MarshallCrd(crd, "yaml")
}

func generateVirtualMachineInstanceCrd() {
	crd := generateBlankCrd()

	crd.ObjectMeta.Name = "virtualmachineinstances." + v1.VirtualMachineInstanceGroupVersionKind.Group
	crd.Spec = extensionsv1.CustomResourceDefinitionSpec{
		Group:   v1.VirtualMachineInstanceGroupVersionKind.Group,
		Version: v1.VirtualMachineInstanceGroupVersionKind.Version,
		Scope:   "Namespaced",

		Names: extensionsv1.CustomResourceDefinitionNames{
			Plural:     "virtualmachineinstances",
			Singular:   "virtualmachineinstance",
			Kind:       v1.VirtualMachineInstanceGroupVersionKind.Kind,
			ShortNames: []string{"vmi", "vmis"},
		},
		AdditionalPrinterColumns: []extensionsv1.CustomResourceColumnDefinition{
			{Name: "Age", Type: "date", JSONPath: ".metadata.creationTimestamp"},
			{Name: "Phase", Type: "string", JSONPath: ".status.phase"},
			{Name: "IP", Type: "string", JSONPath: ".status.interfaces[0].ipAddress"},
			{Name: "NodeName", Type: "string", JSONPath: ".status.nodeName"},
		},
	}

	crdutils.MarshallCrd(crd, "yaml")
}

func main() {
	crdType := flag.String("crd-type", "", "Type of crd to generate. vmi | vmipreset | vmirs | vm | vmim")
	flag.Parse()

	switch *crdType {
	case "vmi":
		generateVirtualMachineInstanceCrd()
	case "vmipreset":
		generatePresetCrd()
	case "vmirs":
		generateReplicaSetCrd()
	case "vm":
		generateVirtualMachineCrd()
	case "vmim":
		generateVirtualMachineInstanceMigrationCrd()
	default:
		panic(fmt.Errorf("unknown crd type %s", *crdType))
	}
}
