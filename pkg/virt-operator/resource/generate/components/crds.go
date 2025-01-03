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

	"kubevirt.io/api/clone"

	clonev1alpha1 "kubevirt.io/api/clone/v1alpha1"
	clonev1beta1 "kubevirt.io/api/clone/v1beta1"

	"kubevirt.io/api/instancetype"

	"kubevirt.io/api/migrations"

	migrationsv1 "kubevirt.io/api/migrations/v1alpha1"

	schedulingv1 "k8s.io/api/scheduling/v1"
	extv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	k8syaml "k8s.io/apimachinery/pkg/util/yaml"
	virtv1 "kubevirt.io/api/core/v1"
	exportv1alpha1 "kubevirt.io/api/export/v1alpha1"
	exportv1beta1 "kubevirt.io/api/export/v1beta1"
	instancetypev1alpha1 "kubevirt.io/api/instancetype/v1alpha1"
	instancetypev1alpha2 "kubevirt.io/api/instancetype/v1alpha2"
	instancetypev1beta1 "kubevirt.io/api/instancetype/v1beta1"
	poolv1 "kubevirt.io/api/pool/v1alpha1"
	snapshotv1alpha1 "kubevirt.io/api/snapshot/v1alpha1"
	snapshotv1beta1 "kubevirt.io/api/snapshot/v1beta1"

	"kubevirt.io/kubevirt/pkg/pointer"
)

const (
	creationTimestampJSONPath = ".metadata.creationTimestamp"
	errorMessageJSONPath      = ".status.error.message"
	phaseJSONPath             = ".status.phase"
)

var (
	VIRTUALMACHINE                   = "virtualmachines." + virtv1.VirtualMachineInstanceGroupVersionKind.Group
	VIRTUALMACHINEINSTANCE           = "virtualmachineinstances." + virtv1.VirtualMachineInstanceGroupVersionKind.Group
	VIRTUALMACHINEINSTANCEPRESET     = "virtualmachineinstancepresets." + virtv1.VirtualMachineInstancePresetGroupVersionKind.Group
	VIRTUALMACHINEINSTANCEREPLICASET = "virtualmachineinstancereplicasets." + virtv1.VirtualMachineInstanceReplicaSetGroupVersionKind.Group
	VIRTUALMACHINEINSTANCEMIGRATION  = "virtualmachineinstancemigrations." + virtv1.VirtualMachineInstanceMigrationGroupVersionKind.Group
	KUBEVIRT                         = "kubevirts." + virtv1.KubeVirtGroupVersionKind.Group
	VIRTUALMACHINEPOOL               = "virtualmachinepools." + poolv1.SchemeGroupVersion.Group
	VIRTUALMACHINESNAPSHOT           = "virtualmachinesnapshots." + snapshotv1beta1.SchemeGroupVersion.Group
	VIRTUALMACHINESNAPSHOTCONTENT    = "virtualmachinesnapshotcontents." + snapshotv1beta1.SchemeGroupVersion.Group
	VIRTUALMACHINEEXPORT             = "virtualmachineexports." + exportv1beta1.SchemeGroupVersion.Group
	MIGRATIONPOLICY                  = "migrationpolicies." + migrationsv1.MigrationPolicyKind.Group
	VIRTUALMACHINECLONE              = "virtualmachineclones." + clone.GroupName
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
		{Name: "Phase", Type: "string", JSONPath: phaseJSONPath},
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
		Group: virtv1.VirtualMachineInstancePresetGroupVersionKind.Group,
		Versions: []extv1.CustomResourceDefinitionVersion{
			{
				Name:               "v1",
				Served:             true,
				Storage:            false,
				Deprecated:         true,
				DeprecationWarning: pointer.P("kubevirt.io/v1 VirtualMachineInstancePresets is now deprecated and will be removed in v2."),
			},
			{
				Name:               "v1alpha3",
				Served:             true,
				Storage:            true,
				Deprecated:         true,
				DeprecationWarning: pointer.P("kubevirt.io/v1alpha3 VirtualMachineInstancePresets is now deprecated and will be removed in v2."),
			},
		},
		Scope: "Namespaced",

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
	err := addFieldsToAllVersions(crd,
		[]extv1.CustomResourceColumnDefinition{
			{Name: "Phase", Type: "string", JSONPath: phaseJSONPath,
				Description: "The current phase of VM instance migration"},
			{Name: "VMI", Type: "string", JSONPath: ".spec.vmiName",
				Description: "The name of the VMI to perform the migration on"},
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
		{Name: "Phase", Type: "string", JSONPath: phaseJSONPath},
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

func NewVirtualMachinePoolCrd() (*extv1.CustomResourceDefinition, error) {
	crd := newBlankCrd()
	labelSelector := ".status.labelSelector"

	crd.ObjectMeta.Name = VIRTUALMACHINEPOOL
	crd.Spec = extv1.CustomResourceDefinitionSpec{
		Group: poolv1.SchemeGroupVersion.Group,
		Versions: []extv1.CustomResourceDefinitionVersion{
			{
				Name:    poolv1.SchemeGroupVersion.Version,
				Served:  true,
				Storage: true,
			},
		},
		Scope: "Namespaced",
		Names: extv1.CustomResourceDefinitionNames{
			Plural:     "virtualmachinepools",
			Singular:   "virtualmachinepool",
			Kind:       "VirtualMachinePool",
			ShortNames: []string{"vmpool", "vmpools"},
			Categories: []string{
				"all",
			},
		},
	}

	err := addFieldsToAllVersions(crd,
		[]extv1.CustomResourceColumnDefinition{
			{Name: "Desired", Type: "integer", JSONPath: ".spec.replicas",
				Description: "Number of desired VirtualMachines"},
			{Name: "Current", Type: "integer", JSONPath: ".status.replicas",
				Description: "Number of managed and not final or deleted VirtualMachines"},
			{Name: "Ready", Type: "integer", JSONPath: ".status.readyReplicas",
				Description: "Number of managed VirtualMachines which are ready to receive traffic"},
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

func NewVirtualMachineSnapshotCrd() (*extv1.CustomResourceDefinition, error) {
	crd := newBlankCrd()

	crd.ObjectMeta.Name = VIRTUALMACHINESNAPSHOT
	crd.Spec = extv1.CustomResourceDefinitionSpec{
		Group: snapshotv1beta1.SchemeGroupVersion.Group,
		Versions: []extv1.CustomResourceDefinitionVersion{
			{
				Name:    snapshotv1alpha1.SchemeGroupVersion.Version,
				Served:  true,
				Storage: false,
				Subresources: &extv1.CustomResourceSubresources{
					Status: &extv1.CustomResourceSubresourceStatus{},
				},
			},
			{
				Name:    snapshotv1beta1.SchemeGroupVersion.Version,
				Served:  true,
				Storage: true,
				Subresources: &extv1.CustomResourceSubresources{
					Status: &extv1.CustomResourceSubresourceStatus{},
				},
			},
		},
		Scope: "Namespaced",
		Conversion: &extv1.CustomResourceConversion{
			Strategy: extv1.NoneConverter,
		},
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
		{Name: "Phase", Type: "string", JSONPath: phaseJSONPath},
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
		Group: snapshotv1beta1.SchemeGroupVersion.Group,
		Versions: []extv1.CustomResourceDefinitionVersion{
			{
				Name:    snapshotv1alpha1.SchemeGroupVersion.Version,
				Served:  true,
				Storage: false,
				Subresources: &extv1.CustomResourceSubresources{
					Status: &extv1.CustomResourceSubresourceStatus{},
				},
			},
			{
				Name:    snapshotv1beta1.SchemeGroupVersion.Version,
				Served:  true,
				Storage: true,
				Subresources: &extv1.CustomResourceSubresources{
					Status: &extv1.CustomResourceSubresourceStatus{},
				},
			},
		},
		Scope: "Namespaced",
		Conversion: &extv1.CustomResourceConversion{
			Strategy: extv1.NoneConverter,
		},
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

	crd.ObjectMeta.Name = "virtualmachinerestores." + snapshotv1beta1.SchemeGroupVersion.Group
	crd.Spec = extv1.CustomResourceDefinitionSpec{
		Group: snapshotv1beta1.SchemeGroupVersion.Group,
		Versions: []extv1.CustomResourceDefinitionVersion{
			{
				Name:    snapshotv1alpha1.SchemeGroupVersion.Version,
				Served:  true,
				Storage: false,
				Subresources: &extv1.CustomResourceSubresources{
					Status: &extv1.CustomResourceSubresourceStatus{},
				},
			},
			{
				Name:    snapshotv1beta1.SchemeGroupVersion.Version,
				Served:  true,
				Storage: true,
				Subresources: &extv1.CustomResourceSubresources{
					Status: &extv1.CustomResourceSubresourceStatus{},
				},
			},
		},
		Scope: "Namespaced",
		Conversion: &extv1.CustomResourceConversion{
			Strategy: extv1.NoneConverter,
		},
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

func NewVirtualMachineExportCrd() (*extv1.CustomResourceDefinition, error) {
	crd := newBlankCrd()

	crd.ObjectMeta.Name = "virtualmachineexports." + exportv1beta1.SchemeGroupVersion.Group
	crd.Spec = extv1.CustomResourceDefinitionSpec{
		Group: exportv1beta1.SchemeGroupVersion.Group,
		Versions: []extv1.CustomResourceDefinitionVersion{
			{
				Name:    exportv1alpha1.SchemeGroupVersion.Version,
				Served:  true,
				Storage: false,
				Subresources: &extv1.CustomResourceSubresources{
					Status: &extv1.CustomResourceSubresourceStatus{},
				},
			},
			{
				Name:    exportv1beta1.SchemeGroupVersion.Version,
				Served:  true,
				Storage: true,
				Subresources: &extv1.CustomResourceSubresources{
					Status: &extv1.CustomResourceSubresourceStatus{},
				},
			},
		},
		Scope: "Namespaced",
		Conversion: &extv1.CustomResourceConversion{
			Strategy: extv1.NoneConverter,
		},
		Names: extv1.CustomResourceDefinitionNames{
			Plural:     "virtualmachineexports",
			Singular:   "virtualmachineexport",
			Kind:       "VirtualMachineExport",
			ShortNames: []string{"vmexport", "vmexports"},
			Categories: []string{
				"all",
			},
		},
	}
	err := addFieldsToAllVersions(crd, []extv1.CustomResourceColumnDefinition{
		{Name: "SourceKind", Type: "string", JSONPath: ".spec.source.kind"},
		{Name: "SourceName", Type: "string", JSONPath: ".spec.source.name"},
		{Name: "Phase", Type: "string", JSONPath: phaseJSONPath},
	})
	if err != nil {
		return nil, err
	}

	if err = patchValidationForAllVersions(crd); err != nil {
		return nil, err
	}
	return crd, nil
}

func NewVirtualMachineInstancetypeCrd() (*extv1.CustomResourceDefinition, error) {
	crd := newBlankCrd()

	crd.Name = "virtualmachineinstancetypes." + instancetypev1beta1.SchemeGroupVersion.Group
	crd.Spec = extv1.CustomResourceDefinitionSpec{
		Group: instancetypev1beta1.SchemeGroupVersion.Group,
		Names: extv1.CustomResourceDefinitionNames{
			Plural:     instancetype.PluralResourceName,
			Singular:   instancetype.SingularResourceName,
			ShortNames: []string{"vminstancetype", "vminstancetypes", "vmf", "vmfs"},
			Kind:       "VirtualMachineInstancetype",
			Categories: []string{"all"},
		},
		Scope: extv1.NamespaceScoped,
		Conversion: &extv1.CustomResourceConversion{
			Strategy: extv1.NoneConverter,
		},
		Versions: []extv1.CustomResourceDefinitionVersion{{
			Name:               instancetypev1alpha1.SchemeGroupVersion.Version,
			Served:             true,
			Storage:            false,
			Deprecated:         true,
			DeprecationWarning: pointer.P("instancetype.kubevirt.io/v1alpha1 VirtualMachineInstancetypes is now deprecated and will be removed in v1."),
		}, {
			Name:               instancetypev1alpha2.SchemeGroupVersion.Version,
			Served:             true,
			Storage:            false,
			Deprecated:         true,
			DeprecationWarning: pointer.P("instancetype.kubevirt.io/v1alpha2 VirtualMachineInstancetypes is now deprecated and will be removed in v1."),
		}, {
			Name:    instancetypev1beta1.SchemeGroupVersion.Version,
			Served:  true,
			Storage: true,
		}},
	}

	if err := patchValidationForAllVersions(crd); err != nil {
		return nil, err
	}
	return crd, nil
}

func NewVirtualMachineClusterInstancetypeCrd() (*extv1.CustomResourceDefinition, error) {
	crd := newBlankCrd()

	crd.Name = "virtualmachineclusterinstancetypes." + instancetypev1beta1.SchemeGroupVersion.Group
	crd.Spec = extv1.CustomResourceDefinitionSpec{
		Group: instancetypev1beta1.SchemeGroupVersion.Group,
		Names: extv1.CustomResourceDefinitionNames{
			Plural:     instancetype.ClusterPluralResourceName,
			Singular:   instancetype.ClusterSingularResourceName,
			ShortNames: []string{"vmclusterinstancetype", "vmclusterinstancetypes", "vmcf", "vmcfs"},
			Kind:       "VirtualMachineClusterInstancetype",
		},
		Scope: extv1.ClusterScoped,
		Conversion: &extv1.CustomResourceConversion{
			Strategy: extv1.NoneConverter,
		},
		Versions: []extv1.CustomResourceDefinitionVersion{{
			Name:               instancetypev1alpha1.SchemeGroupVersion.Version,
			Served:             true,
			Storage:            false,
			Deprecated:         true,
			DeprecationWarning: pointer.P("instancetype.kubevirt.io/v1alpha1 VirtualMachineClusterInstanceTypes is now deprecated and will be removed in v1."),
		}, {
			Name:               instancetypev1alpha2.SchemeGroupVersion.Version,
			Served:             true,
			Storage:            false,
			Deprecated:         true,
			DeprecationWarning: pointer.P("instancetype.kubevirt.io/v1alpha2 VirtualMachineClusterInstanceTypes is now deprecated and will be removed in v1."),
		}, {
			Name:    instancetypev1beta1.SchemeGroupVersion.Version,
			Served:  true,
			Storage: true,
		}},
	}

	if err := patchValidationForAllVersions(crd); err != nil {
		return nil, err
	}
	return crd, nil
}

func NewVirtualMachinePreferenceCrd() (*extv1.CustomResourceDefinition, error) {
	crd := newBlankCrd()

	crd.Name = "virtualmachinepreferences." + instancetypev1beta1.SchemeGroupVersion.Group
	crd.Spec = extv1.CustomResourceDefinitionSpec{
		Group: instancetypev1beta1.SchemeGroupVersion.Group,
		Names: extv1.CustomResourceDefinitionNames{
			Plural:     instancetype.PluralPreferenceResourceName,
			Singular:   instancetype.SingularPreferenceResourceName,
			ShortNames: []string{"vmpref", "vmprefs", "vmp", "vmps"},
			Kind:       "VirtualMachinePreference",
			Categories: []string{"all"},
		},
		Scope: extv1.NamespaceScoped,
		Conversion: &extv1.CustomResourceConversion{
			Strategy: extv1.NoneConverter,
		},
		Versions: []extv1.CustomResourceDefinitionVersion{{
			Name:               instancetypev1alpha1.SchemeGroupVersion.Version,
			Served:             true,
			Storage:            false,
			Deprecated:         true,
			DeprecationWarning: pointer.P("instancetype.kubevirt.io/v1alpha1 VirtualMachinePreferences is now deprecated and will be removed in v1."),
		}, {
			Name:               instancetypev1alpha2.SchemeGroupVersion.Version,
			Served:             true,
			Storage:            false,
			Deprecated:         true,
			DeprecationWarning: pointer.P("instancetype.kubevirt.io/v1alpha2 VirtualMachinePreferences is now deprecated and will be removed in v1."),
		}, {
			Name:    instancetypev1beta1.SchemeGroupVersion.Version,
			Served:  true,
			Storage: true,
		}},
	}

	if err := patchValidationForAllVersions(crd); err != nil {
		return nil, err
	}
	return crd, nil
}

func NewVirtualMachineClusterPreferenceCrd() (*extv1.CustomResourceDefinition, error) {
	crd := newBlankCrd()

	crd.Name = "virtualmachineclusterpreferences." + instancetypev1beta1.SchemeGroupVersion.Group
	crd.Spec = extv1.CustomResourceDefinitionSpec{
		Group: instancetypev1beta1.SchemeGroupVersion.Group,
		Names: extv1.CustomResourceDefinitionNames{
			Plural:     instancetype.ClusterPluralPreferenceResourceName,
			Singular:   instancetype.ClusterSingularPreferenceResourceName,
			ShortNames: []string{"vmcp", "vmcps"},
			Kind:       "VirtualMachineClusterPreference",
		},
		Scope: extv1.ClusterScoped,
		Conversion: &extv1.CustomResourceConversion{
			Strategy: extv1.NoneConverter,
		},
		Versions: []extv1.CustomResourceDefinitionVersion{{
			Name:               instancetypev1alpha1.SchemeGroupVersion.Version,
			Served:             true,
			Storage:            false,
			Deprecated:         true,
			DeprecationWarning: pointer.P("instancetype.kubevirt.io/v1alpha1 VirtualMachineClusterPreferences is now deprecated and will be removed in v1."),
		}, {
			Name:               instancetypev1alpha2.SchemeGroupVersion.Version,
			Served:             true,
			Storage:            false,
			Deprecated:         true,
			DeprecationWarning: pointer.P("instancetype.kubevirt.io/v1alpha2 VirtualMachineClusterPreferences is now deprecated and will be removed in v1."),
		}, {
			Name:    instancetypev1beta1.SchemeGroupVersion.Version,
			Served:  true,
			Storage: true,
		}},
	}

	if err := patchValidationForAllVersions(crd); err != nil {
		return nil, err
	}
	return crd, nil
}

func NewMigrationPolicyCrd() (*extv1.CustomResourceDefinition, error) {
	crd := newBlankCrd()

	crd.ObjectMeta.Name = MIGRATIONPOLICY
	crd.Spec = extv1.CustomResourceDefinitionSpec{
		Group: migrationsv1.MigrationPolicyKind.Group,
		Versions: []extv1.CustomResourceDefinitionVersion{
			{
				Name:    migrationsv1.SchemeGroupVersion.Version,
				Served:  true,
				Storage: true,
			},
		},
		Scope: extv1.ClusterScoped,

		Names: extv1.CustomResourceDefinitionNames{
			Plural:   migrations.ResourceMigrationPolicies,
			Singular: "migrationpolicy",
			Kind:     migrationsv1.MigrationPolicyKind.Kind,
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

func NewVirtualMachineCloneCrd() (*extv1.CustomResourceDefinition, error) {
	crd := newBlankCrd()

	crd.ObjectMeta.Name = VIRTUALMACHINECLONE
	crd.Spec = extv1.CustomResourceDefinitionSpec{
		Group: clonev1alpha1.VirtualMachineCloneKind.Group,
		Versions: []extv1.CustomResourceDefinitionVersion{
			{
				Name:    clonev1alpha1.SchemeGroupVersion.Version,
				Served:  true,
				Storage: false,
			},
			{
				Name:    clonev1beta1.SchemeGroupVersion.Version,
				Served:  true,
				Storage: true,
			},
		},
		Scope: extv1.NamespaceScoped,

		Names: extv1.CustomResourceDefinitionNames{
			Plural:     clone.ResourceVMClonePlural,
			Singular:   clone.ResourceVMCloneSingular,
			ShortNames: []string{"vmclone", "vmclones"},
			Kind:       clonev1alpha1.VirtualMachineCloneKind.Kind,
			Categories: []string{
				"all",
			},
		},
	}
	err := addFieldsToAllVersions(crd,
		&extv1.CustomResourceSubresources{
			Status: &extv1.CustomResourceSubresourceStatus{},
		},
		[]extv1.CustomResourceColumnDefinition{
			{Name: "Phase", Type: "string", JSONPath: phaseJSONPath},
			{Name: "SourceVirtualMachine", Type: "string", JSONPath: ".spec.source.name"},
			{Name: "TargetVirtualMachine", Type: "string", JSONPath: ".spec.target.name"},
		},
	)
	if err != nil {
		return nil, err
	}

	if err = patchValidationForAllVersions(crd); err != nil {
		return nil, err
	}
	return crd, nil
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
