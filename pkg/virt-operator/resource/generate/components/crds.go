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
	"fmt"
	"strings"

	corev1 "k8s.io/api/core/v1"
	schedulingv1 "k8s.io/api/scheduling/v1"
	extv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	k8syaml "k8s.io/apimachinery/pkg/util/yaml"

	virtv1 "kubevirt.io/client-go/api/v1"
	flavorv1alpha1 "kubevirt.io/client-go/apis/flavor/v1alpha1"
	snapshotv1 "kubevirt.io/client-go/apis/snapshot/v1alpha1"
)

const (
	creationTimestampJSONPath = ".metadata.creationTimestamp"
	errorMessageJSONPath      = ".status.error.message"
)

var (
	VIRTUALMACHINE                   = "virtualmachines." + virtv1.VirtualMachineInstanceGroupVersionKind.Group
	VIRTUALMACHINEINSTANCE           = "virtualmachineinstances." + virtv1.VirtualMachineInstanceGroupVersionKind.Group
	VIRTUALMACHINEINSTANCEPRESET     = "virtualmachineinstancepresets." + virtv1.VirtualMachineInstancePresetGroupVersionKind.Group
	VIRTUALMACHINEINSTANCEREPLICASET = "virtualmachineinstancereplicasets." + virtv1.VirtualMachineInstanceReplicaSetGroupVersionKind.Group
	VIRTUALMACHINEINSTANCEMIGRATION  = "virtualmachineinstancemigrations." + virtv1.VirtualMachineInstanceMigrationGroupVersionKind.Group
	KUBEVIRT                         = "kubevirts." + virtv1.KubeVirtGroupVersionKind.Group
	VIRTUALMACHINESNAPSHOT           = "virtualmachinesnapshots." + snapshotv1.SchemeGroupVersion.Group
	VIRTUALMACHINESNAPSHOTCONTENT    = "virtualmachinesnapshotcontents." + snapshotv1.SchemeGroupVersion.Group
	PreserveUnknownFieldsFalse       = false
)

func addFieldsToVersion(version *extv1.CustomResourceDefinitionVersion, fields ...interface{}) error {
	for _, field := range fields {
		switch v := field.(type) {
		case []extv1.CustomResourceColumnDefinition:
			version.AdditionalPrinterColumns = v
		case *extv1.CustomResourceSubresources:
			version.Subresources = v
		case *extv1.CustomResourceValidation:
			version.Schema = v
		default:
			return fmt.Errorf("cannot add field of type %T to a CustomResourceDefinitionVersion", v)
		}
	}
	return nil
}

func addFieldsToAllVersions(crd *extv1.CustomResourceDefinition, fields ...interface{}) error {
	for i := range crd.Spec.Versions {
		if err := addFieldsToVersion(&crd.Spec.Versions[i], fields...); err != nil {
			return err
		}
	}
	return nil
}

func patchValidation(crd *extv1.CustomResourceDefinition, version *extv1.CustomResourceDefinitionVersion) error {
	name := crd.Spec.Names.Singular

	crd.Spec.PreserveUnknownFields = PreserveUnknownFieldsFalse
	validation, ok := CRDsValidation[name]
	if !ok {
		return nil
	}
	crvalidation := extv1.CustomResourceValidation{}
	err := k8syaml.NewYAMLToJSONDecoder(strings.NewReader(validation)).Decode(&crvalidation)
	if err != nil {
		return fmt.Errorf("Could not decode validation for %s, %v", name, err)
	}
	if err = addFieldsToVersion(version, &crvalidation); err != nil {
		return err
	}
	return nil
}

func patchValidationForAllVersions(crd *extv1.CustomResourceDefinition) error {
	for i := range crd.Spec.Versions {
		if err := patchValidation(crd, &crd.Spec.Versions[i]); err != nil {
			return err
		}
	}
	return nil
}

func newBlankCrd() *extv1.CustomResourceDefinition {
	return &extv1.CustomResourceDefinition{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "apiextensions.k8s.io/v1",
			Kind:       "CustomResourceDefinition",
		},
		ObjectMeta: metav1.ObjectMeta{
			Labels: map[string]string{
				virtv1.AppLabel: "",
			},
		},
	}
}

func newCRDVersions() []extv1.CustomResourceDefinitionVersion {
	versions := make([]extv1.CustomResourceDefinitionVersion, len(virtv1.ApiSupportedVersions))
	copy(versions, virtv1.ApiSupportedVersions)
	return versions
}

func NewVirtualMachineInstanceCrd() (*extv1.CustomResourceDefinition, error) {
	crd := newBlankCrd()

	crd.ObjectMeta.Name = VIRTUALMACHINEINSTANCE
	crd.Spec = extv1.CustomResourceDefinitionSpec{
		Group:    virtv1.VirtualMachineInstanceGroupVersionKind.Group,
		Versions: newCRDVersions(),
		Scope:    "Namespaced",

		Names: extv1.CustomResourceDefinitionNames{
			Plural:     "virtualmachineinstances",
			Singular:   "virtualmachineinstance",
			Kind:       virtv1.VirtualMachineInstanceGroupVersionKind.Kind,
			ShortNames: []string{"vmi", "vmis"},
			Categories: []string{
				"all",
			},
		},
	}
	err := addFieldsToAllVersions(crd, []extv1.CustomResourceColumnDefinition{
		{Name: "Age", Type: "date", JSONPath: creationTimestampJSONPath},
		{Name: "Phase", Type: "string", JSONPath: ".status.phase"},
		{Name: "IP", Type: "string", JSONPath: ".status.interfaces[0].ipAddress"},
		{Name: "NodeName", Type: "string", JSONPath: ".status.nodeName"},
		{Name: "Ready", Type: "string", JSONPath: ".status.conditions[?(@.type=='Ready')].status"},
		{Name: "Live-Migratable", Type: "string", JSONPath: ".status.conditions[?(@.type=='LiveMigratable')].status", Priority: 1},
		{Name: "Paused", Type: "string", JSONPath: ".status.conditions[?(@.type=='Paused')].status", Priority: 1},
	})
	if err != nil {
		return nil, err
	}

	if err := patchValidationForAllVersions(crd); err != nil {
		return nil, err
	}
	return crd, nil
}

func NewVirtualMachineCrd() (*extv1.CustomResourceDefinition, error) {
	crd := newBlankCrd()

	crd.ObjectMeta.Name = VIRTUALMACHINE
	crd.Spec = extv1.CustomResourceDefinitionSpec{
		Group:    virtv1.VirtualMachineGroupVersionKind.Group,
		Versions: newCRDVersions(),
		Scope:    "Namespaced",

		Names: extv1.CustomResourceDefinitionNames{
			Plural:     "virtualmachines",
			Singular:   "virtualmachine",
			Kind:       virtv1.VirtualMachineGroupVersionKind.Kind,
			ShortNames: []string{"vm", "vms"},
			Categories: []string{
				"all",
			},
		},
	}
	err := addFieldsToAllVersions(crd, []extv1.CustomResourceColumnDefinition{
		{Name: "Age", Type: "date", JSONPath: creationTimestampJSONPath},
		{Name: "Status", Description: "Human Readable Status", Type: "string", JSONPath: ".status.printableStatus"},
		{Name: "Ready", Type: "string", JSONPath: ".status.conditions[?(@.type=='Ready')].status"},
	}, &extv1.CustomResourceSubresources{
		Status: &extv1.CustomResourceSubresourceStatus{}})
	if err != nil {
		return nil, err
	}

	if err = patchValidationForAllVersions(crd); err != nil {
		return nil, err
	}
	return crd, nil
}

func NewPresetCrd() (*extv1.CustomResourceDefinition, error) {
	crd := newBlankCrd()

	crd.ObjectMeta.Name = VIRTUALMACHINEINSTANCEPRESET
	crd.Spec = extv1.CustomResourceDefinitionSpec{
		Group:    virtv1.VirtualMachineInstancePresetGroupVersionKind.Group,
		Versions: newCRDVersions(),
		Scope:    "Namespaced",

		Names: extv1.CustomResourceDefinitionNames{
			Plural:     "virtualmachineinstancepresets",
			Singular:   "virtualmachineinstancepreset",
			Kind:       virtv1.VirtualMachineInstancePresetGroupVersionKind.Kind,
			ShortNames: []string{"vmipreset", "vmipresets"},
			Categories: []string{
				"all",
			},
		},
	}

	if err := patchValidationForAllVersions(crd); err != nil {
		return nil, err
	}
	return crd, nil
}

func NewReplicaSetCrd() (*extv1.CustomResourceDefinition, error) {
	crd := newBlankCrd()
	labelSelector := ".status.labelSelector"

	crd.ObjectMeta.Name = VIRTUALMACHINEINSTANCEREPLICASET
	crd.Spec = extv1.CustomResourceDefinitionSpec{
		Group:    virtv1.VirtualMachineInstanceReplicaSetGroupVersionKind.Group,
		Versions: newCRDVersions(),
		Scope:    "Namespaced",

		Names: extv1.CustomResourceDefinitionNames{
			Plural:     "virtualmachineinstancereplicasets",
			Singular:   "virtualmachineinstancereplicaset",
			Kind:       virtv1.VirtualMachineInstanceReplicaSetGroupVersionKind.Kind,
			ShortNames: []string{"vmirs", "vmirss"},
			Categories: []string{
				"all",
			},
		},
	}
	err := addFieldsToAllVersions(crd,
		[]extv1.CustomResourceColumnDefinition{
			{Name: "Desired", Type: "integer", JSONPath: ".spec.replicas",
				Description: "Number of desired VirtualMachineInstances"},
			{Name: "Current", Type: "integer", JSONPath: ".status.replicas",
				Description: "Number of managed and not final or deleted VirtualMachineInstances"},
			{Name: "Ready", Type: "integer", JSONPath: ".status.readyReplicas",
				Description: "Number of managed VirtualMachineInstances which are ready to receive traffic"},
			{Name: "Age", Type: "date", JSONPath: creationTimestampJSONPath},
		}, &extv1.CustomResourceSubresources{
			Scale: &extv1.CustomResourceSubresourceScale{
				SpecReplicasPath:   ".spec.replicas",
				StatusReplicasPath: ".status.replicas",
				LabelSelectorPath:  &labelSelector,
			},
			Status: &extv1.CustomResourceSubresourceStatus{},
		})
	if err != nil {
		return nil, err
	}
	if err := patchValidationForAllVersions(crd); err != nil {
		return nil, err
	}
	return crd, nil
}

func NewVirtualMachineInstanceMigrationCrd() (*extv1.CustomResourceDefinition, error) {
	crd := newBlankCrd()

	crd.ObjectMeta.Name = VIRTUALMACHINEINSTANCEMIGRATION
	crd.Spec = extv1.CustomResourceDefinitionSpec{
		Group:    virtv1.VirtualMachineInstanceMigrationGroupVersionKind.Group,
		Versions: newCRDVersions(),
		Scope:    "Namespaced",

		Names: extv1.CustomResourceDefinitionNames{
			Plural:     "virtualmachineinstancemigrations",
			Singular:   "virtualmachineinstancemigration",
			Kind:       virtv1.VirtualMachineInstanceMigrationGroupVersionKind.Kind,
			ShortNames: []string{"vmim", "vmims"},
			Categories: []string{
				"all",
			},
		},
	}
	err := addFieldsToAllVersions(crd, &extv1.CustomResourceSubresources{
		Status: &extv1.CustomResourceSubresourceStatus{},
	})
	if err != nil {
		return nil, err
	}

	if err = patchValidationForAllVersions(crd); err != nil {
		return nil, err
	}
	return crd, nil
}

// Used by manifest generation
// If you change something here, you probably need to change the CSV manifest too,
// see /manifests/release/kubevirt.VERSION.csv.yaml.in
func NewKubeVirtCrd() (*extv1.CustomResourceDefinition, error) {

	// we use a different label here, so no newBlankCrd()
	crd := &extv1.CustomResourceDefinition{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "apiextensions.k8s.io/v1",
			Kind:       "CustomResourceDefinition",
		},
		ObjectMeta: metav1.ObjectMeta{
			Labels: map[string]string{
				"operator.kubevirt.io": "",
			},
		},
	}

	crd.ObjectMeta.Name = KUBEVIRT
	crd.Spec = extv1.CustomResourceDefinitionSpec{
		Group:    virtv1.KubeVirtGroupVersionKind.Group,
		Versions: newCRDVersions(),
		Scope:    "Namespaced",

		Names: extv1.CustomResourceDefinitionNames{
			Plural:     "kubevirts",
			Singular:   "kubevirt",
			Kind:       virtv1.KubeVirtGroupVersionKind.Kind,
			ShortNames: []string{"kv", "kvs"},
			Categories: []string{
				"all",
			},
		},
	}
	err := addFieldsToAllVersions(crd, []extv1.CustomResourceColumnDefinition{
		{Name: "Age", Type: "date", JSONPath: creationTimestampJSONPath},
		{Name: "Phase", Type: "string", JSONPath: ".status.phase"},
	}, &extv1.CustomResourceSubresources{
		Status: &extv1.CustomResourceSubresourceStatus{},
	})
	if err != nil {
		return nil, err
	}

	if err = patchValidationForAllVersions(crd); err != nil {
		return nil, err
	}
	return crd, nil
}

func NewVirtualMachineSnapshotCrd() (*extv1.CustomResourceDefinition, error) {
	crd := newBlankCrd()

	crd.ObjectMeta.Name = VIRTUALMACHINESNAPSHOT
	crd.Spec = extv1.CustomResourceDefinitionSpec{
		Group: snapshotv1.SchemeGroupVersion.Group,
		Versions: []extv1.CustomResourceDefinitionVersion{
			{
				Name:    snapshotv1.SchemeGroupVersion.Version,
				Served:  true,
				Storage: true,
			},
		},
		Scope: "Namespaced",
		Names: extv1.CustomResourceDefinitionNames{
			Plural:     "virtualmachinesnapshots",
			Singular:   "virtualmachinesnapshot",
			Kind:       "VirtualMachineSnapshot",
			ShortNames: []string{"vmsnapshot", "vmsnapshots"},
			Categories: []string{
				"all",
			},
		},
	}
	err := addFieldsToAllVersions(crd, []extv1.CustomResourceColumnDefinition{
		{Name: "SourceKind", Type: "string", JSONPath: ".spec.source.kind"},
		{Name: "SourceName", Type: "string", JSONPath: ".spec.source.name"},
		{Name: "Phase", Type: "string", JSONPath: ".status.phase"},
		{Name: "ReadyToUse", Type: "boolean", JSONPath: ".status.readyToUse"},
		{Name: "CreationTime", Type: "date", JSONPath: ".status.creationTime"},
		{Name: "Error", Type: "string", JSONPath: errorMessageJSONPath},
	})
	if err != nil {
		return nil, err
	}

	if err = patchValidationForAllVersions(crd); err != nil {
		return nil, err
	}
	return crd, nil
}

func NewVirtualMachineSnapshotContentCrd() (*extv1.CustomResourceDefinition, error) {
	crd := newBlankCrd()

	crd.ObjectMeta.Name = VIRTUALMACHINESNAPSHOTCONTENT
	crd.Spec = extv1.CustomResourceDefinitionSpec{
		Group: snapshotv1.SchemeGroupVersion.Group,
		Versions: []extv1.CustomResourceDefinitionVersion{
			{
				Name:    snapshotv1.SchemeGroupVersion.Version,
				Served:  true,
				Storage: true,
			},
		},
		Scope: "Namespaced",
		Names: extv1.CustomResourceDefinitionNames{
			Plural:     "virtualmachinesnapshotcontents",
			Singular:   "virtualmachinesnapshotcontent",
			Kind:       "VirtualMachineSnapshotContent",
			ShortNames: []string{"vmsnapshotcontent", "vmsnapshotcontents"},
			Categories: []string{
				"all",
			},
		},
	}
	err := addFieldsToAllVersions(crd, []extv1.CustomResourceColumnDefinition{
		{Name: "ReadyToUse", Type: "boolean", JSONPath: ".status.readyToUse"},
		{Name: "CreationTime", Type: "date", JSONPath: ".status.creationTime"},
		{Name: "Error", Type: "string", JSONPath: errorMessageJSONPath},
	})
	if err != nil {
		return nil, err
	}

	if err = patchValidationForAllVersions(crd); err != nil {
		return nil, err
	}
	return crd, nil
}

func NewVirtualMachineRestoreCrd() (*extv1.CustomResourceDefinition, error) {
	crd := newBlankCrd()

	crd.ObjectMeta.Name = "virtualmachinerestores." + snapshotv1.SchemeGroupVersion.Group
	crd.Spec = extv1.CustomResourceDefinitionSpec{
		Group: snapshotv1.SchemeGroupVersion.Group,
		Versions: []extv1.CustomResourceDefinitionVersion{
			{
				Name:    snapshotv1.SchemeGroupVersion.Version,
				Served:  true,
				Storage: true,
			},
		},
		Scope: "Namespaced",
		Names: extv1.CustomResourceDefinitionNames{
			Plural:     "virtualmachinerestores",
			Singular:   "virtualmachinerestore",
			Kind:       "VirtualMachineRestore",
			ShortNames: []string{"vmrestore", "vmrestores"},
			Categories: []string{
				"all",
			},
		},
	}
	err := addFieldsToAllVersions(crd, []extv1.CustomResourceColumnDefinition{
		{Name: "TargetKind", Type: "string", JSONPath: ".spec.target.kind"},
		{Name: "TargetName", Type: "string", JSONPath: ".spec.target.name"},
		{Name: "Complete", Type: "boolean", JSONPath: ".status.complete"},
		{Name: "RestoreTime", Type: "date", JSONPath: ".status.restoreTime"},
		{Name: "Error", Type: "string", JSONPath: errorMessageJSONPath},
	})
	if err != nil {
		return nil, err
	}

	if err = patchValidationForAllVersions(crd); err != nil {
		return nil, err
	}
	return crd, nil
}

func NewVirtualMachineFlavorCrd() (*extv1.CustomResourceDefinition, error) {
	crd := newBlankCrd()

	crd.Name = "virtualmachineflavors." + flavorv1alpha1.SchemeGroupVersion.Group
	crd.Spec = extv1.CustomResourceDefinitionSpec{
		Group: flavorv1alpha1.SchemeGroupVersion.Group,
		Names: extv1.CustomResourceDefinitionNames{
			Plural:     "virtualmachineflavors",
			Singular:   "virtualmachineflavor",
			ShortNames: []string{"vmflavor", "vmflavors"},
			Kind:       "VirtualMachineFlavor",
			Categories: []string{"all"},
		},
		Scope: extv1.NamespaceScoped,
		Versions: []extv1.CustomResourceDefinitionVersion{{
			Name:    flavorv1alpha1.SchemeGroupVersion.Version,
			Served:  true,
			Storage: true,
		}},
	}

	if err := patchValidationForAllVersions(crd); err != nil {
		return nil, err
	}
	return crd, nil
}

func NewVirtualMachineClusterFlavorCrd() (*extv1.CustomResourceDefinition, error) {
	crd := newBlankCrd()

	crd.Name = "virtualmachineclusterflavors." + flavorv1alpha1.SchemeGroupVersion.Group
	crd.Spec = extv1.CustomResourceDefinitionSpec{
		Group: flavorv1alpha1.SchemeGroupVersion.Group,
		Names: extv1.CustomResourceDefinitionNames{
			Plural:     "virtualmachineclusterflavors",
			Singular:   "virtualmachineclusterflavor",
			ShortNames: []string{"vmclusterflavor", "vmclusterflavors"},
			Kind:       "VirtualMachineClusterFlavor",
			Categories: []string{"all"},
		},
		Scope: extv1.ClusterScoped,
		Versions: []extv1.CustomResourceDefinitionVersion{{
			Name:    flavorv1alpha1.SchemeGroupVersion.Version,
			Served:  true,
			Storage: true,
		}},
	}

	if err := patchValidationForAllVersions(crd); err != nil {
		return nil, err
	}
	return crd, nil
}

// Used by manifest generation
func NewKubeVirtCR(namespace string, pullPolicy corev1.PullPolicy, featureGates string) *virtv1.KubeVirt {
	cr := &virtv1.KubeVirt{
		TypeMeta: metav1.TypeMeta{
			APIVersion: virtv1.GroupVersion.String(),
			Kind:       "KubeVirt",
		},
		ObjectMeta: metav1.ObjectMeta{
			Namespace: namespace,
			Name:      "kubevirt",
		},
		Spec: virtv1.KubeVirtSpec{
			ImagePullPolicy: pullPolicy,
		},
	}

	if featureGates != "" {
		cr.Spec.Configuration = virtv1.KubeVirtConfiguration{
			DeveloperConfiguration: &virtv1.DeveloperConfiguration{
				FeatureGates: strings.Split(featureGates, ","),
			},
		}
	}

	return cr
}

// NewKubeVirtPriorityClassCR is used for manifest generation
func NewKubeVirtPriorityClassCR() *schedulingv1.PriorityClass {
	return &schedulingv1.PriorityClass{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "scheduling.k8s.io/v1",
			Kind:       "PriorityClass",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: "kubevirt-cluster-critical",
		},
		// 1 billion is the highest value we can set
		// https://kubernetes.io/docs/concepts/configuration/pod-priority-preemption/#priorityclass
		Value:         1000000000,
		GlobalDefault: false,
		Description:   "This priority class should be used for KubeVirt core components only.",
	}
}
