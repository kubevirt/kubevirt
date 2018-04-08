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

	"kubevirt.io/kubevirt/pkg/api/v1"
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

func generateOfflineVirtualMachineCrd() {
	crd := generateBlankCrd()

	crd.ObjectMeta.Name = "offlinevirtualmachines." + v1.OfflineVirtualMachineGroupVersionKind.Group
	crd.Spec = extensionsv1.CustomResourceDefinitionSpec{
		Group:   v1.OfflineVirtualMachineGroupVersionKind.Group,
		Version: v1.OfflineVirtualMachineGroupVersionKind.Version,
		Scope:   "Namespaced",

		Names: extensionsv1.CustomResourceDefinitionNames{
			Plural:     "offlinevirtualmachines",
			Singular:   "offlinevirtualmachine",
			Kind:       v1.OfflineVirtualMachineGroupVersionKind.Kind,
			ShortNames: []string{"ovm", "ovms"},
		},
		Validation: crdutils.GetCustomResourceValidation("kubevirt.io/kubevirt/pkg/api/v1.OfflineVirtualMachine", v1.GetOpenAPIDefinitions),
	}

	crdutils.MarshallCrd(crd, "yaml")
}

func generatePresetCrd() {
	crd := generateBlankCrd()

	crd.ObjectMeta.Name = "virtualmachinepresets." + v1.VirtualMachinePresetGroupVersionKind.Group
	crd.Spec = extensionsv1.CustomResourceDefinitionSpec{
		Group:   v1.VirtualMachinePresetGroupVersionKind.Group,
		Version: v1.VirtualMachinePresetGroupVersionKind.Version,
		Scope:   "Namespaced",

		Names: extensionsv1.CustomResourceDefinitionNames{
			Plural:     "virtualmachinepresets",
			Singular:   "virtualmachinepreset",
			Kind:       v1.VirtualMachinePresetGroupVersionKind.Kind,
			ShortNames: []string{"vmpreset", "vmpresets"},
		},
		Validation: crdutils.GetCustomResourceValidation("kubevirt.io/kubevirt/pkg/api/v1.VirtualMachinePreset", v1.GetOpenAPIDefinitions),
	}

	crdutils.MarshallCrd(crd, "yaml")
}

func generateReplicaSetCrd() {
	crd := generateBlankCrd()

	crd.ObjectMeta.Name = "virtualmachinereplicasets." + v1.VMReplicaSetGroupVersionKind.Group
	crd.Spec = extensionsv1.CustomResourceDefinitionSpec{
		Group:   v1.VMReplicaSetGroupVersionKind.Group,
		Version: v1.VMReplicaSetGroupVersionKind.Version,
		Scope:   "Namespaced",

		Names: extensionsv1.CustomResourceDefinitionNames{
			Plural:     "virtualmachinereplicasets",
			Singular:   "virtualmachinereplicaset",
			Kind:       v1.VMReplicaSetGroupVersionKind.Kind,
			ShortNames: []string{"vmrs", "vmrss"},
		},
		Validation: crdutils.GetCustomResourceValidation("kubevirt.io/kubevirt/pkg/api/v1.VirtualMachineReplicaSet", v1.GetOpenAPIDefinitions),
	}

	crdutils.MarshallCrd(crd, "yaml")
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
		Validation: crdutils.GetCustomResourceValidation("kubevirt.io/kubevirt/pkg/api/v1.VirtualMachine", v1.GetOpenAPIDefinitions),
	}

	crdutils.MarshallCrd(crd, "yaml")
}

func main() {
	crdType := flag.String("crd-type", "", "Type of crd to generate. vm | vmpreset | vmrs | ovm")
	flag.Parse()

	switch *crdType {
	case "vm":
		generateVirtualMachineCrd()
	case "vmpreset":
		generatePresetCrd()
	case "vmrs":
		generateReplicaSetCrd()
	case "ovm":
		generateOfflineVirtualMachineCrd()
	default:
		panic(fmt.Errorf("unknown crd type %s", *crdType))
	}
}
