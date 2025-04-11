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
 * Copyright The KubeVirt Authors.
 *
 */

package admitters

import (
	"encoding/json"
	"fmt"
	"reflect"

	admissionv1 "k8s.io/api/admission/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/equality"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/validation/field"
	k8sfield "k8s.io/apimachinery/pkg/util/validation/field"
	v1 "kubevirt.io/api/core/v1"

	cdiv1 "kubevirt.io/containerized-data-importer-api/pkg/apis/core/v1beta1"
)

func (a Admitter) validateVirtualMachineDataVolumeTemplateNamespace() ([]metav1.StatusCause, error) {
	var causes []metav1.StatusCause

	if a.ar.Operation == admissionv1.Update || a.ar.Operation == admissionv1.Delete {
		oldVM := &v1.VirtualMachine{}
		if err := json.Unmarshal(a.ar.OldObject.Raw, oldVM); err != nil {
			return []metav1.StatusCause{{
				Type:    metav1.CauseTypeUnexpectedServerResponse,
				Message: "Could not fetch old VM",
			}}, nil
		}

		if equality.Semantic.DeepEqual(oldVM.Spec.DataVolumeTemplates, a.vm.Spec.DataVolumeTemplates) {
			return nil, nil
		}
	}

	for idx, dataVolume := range a.vm.Spec.DataVolumeTemplates {
		targetNamespace := a.vm.Namespace
		if targetNamespace == "" {
			targetNamespace = a.ar.Namespace
		}
		if dataVolume.Namespace != "" && dataVolume.Namespace != targetNamespace {
			causes = append(causes, metav1.StatusCause{
				Type:    metav1.CauseTypeFieldValueInvalid,
				Message: fmt.Sprintf("Embedded DataVolume namespace %s differs from VM namespace %s", dataVolume.Namespace, targetNamespace),
				Field:   k8sfield.NewPath("spec", "dataVolumeTemplates").Index(idx).String(),
			})

			continue
		}
	}

	return causes, nil
}

func ValidateDataVolumeTemplate(field *k8sfield.Path, spec *v1.VirtualMachineSpec) (causes []metav1.StatusCause) {
	for idx, dataVolume := range spec.DataVolumeTemplates {
		cause := validateDataVolume(field.Child("dataVolumeTemplate").Index(idx), dataVolume)
		if cause != nil {
			causes = append(causes, cause...)
			continue
		}

		dataVolumeRefFound := false
		for _, volume := range spec.Template.Spec.Volumes {
			if volume.VolumeSource.PersistentVolumeClaim != nil && volume.VolumeSource.PersistentVolumeClaim.ClaimName == dataVolume.Name ||
				volume.VolumeSource.DataVolume != nil && volume.VolumeSource.DataVolume.Name == dataVolume.Name {
				dataVolumeRefFound = true
				break
			}
		}

		if !dataVolumeRefFound {
			causes = append(causes, metav1.StatusCause{
				Type:    metav1.CauseTypeFieldValueRequired,
				Message: fmt.Sprintf("DataVolumeTemplate entry %s must be referenced in the VMI template's 'volumes' list", field.Child("dataVolumeTemplate").Index(idx).String()),
				Field:   field.Child("dataVolumeTemplate").Index(idx).String(),
			})
		}
	}
	return causes
}

func validateDataVolume(field *k8sfield.Path, dataVolume v1.DataVolumeTemplateSpec) []metav1.StatusCause {
	if dataVolume.Name == "" {
		return []metav1.StatusCause{{
			Type:    metav1.CauseTypeFieldValueRequired,
			Message: fmt.Sprintf("'name' field must not be empty for DataVolumeTemplate entry %s.", field.Child("name").String()),
			Field:   field.Child("name").String(),
		}}
	}
	if dataVolume.Spec.PVC == nil && dataVolume.Spec.Storage == nil {
		return []metav1.StatusCause{{
			Type:    metav1.CauseTypeFieldValueInvalid,
			Message: "Missing Data volume PVC or Storage",
			Field:   field.Child("PVC", "Storage").String(),
		}}
	}
	if dataVolume.Spec.PVC != nil && dataVolume.Spec.Storage != nil {
		return []metav1.StatusCause{{
			Type:    metav1.CauseTypeFieldValueInvalid,
			Message: "Duplicate storage definition, both target storage and target pvc defined",
			Field:   field.Child("PVC", "Storage").String(),
		}}
	}

	var dataSourceRef *corev1.TypedObjectReference
	var dataSource *corev1.TypedLocalObjectReference
	if dataVolume.Spec.PVC != nil {
		dataSourceRef = dataVolume.Spec.PVC.DataSourceRef
		dataSource = dataVolume.Spec.PVC.DataSource
	} else if dataVolume.Spec.Storage != nil {
		dataSourceRef = dataVolume.Spec.Storage.DataSourceRef
		dataSource = dataVolume.Spec.Storage.DataSource
	}

	// dataVolume is externally populated
	if (dataSourceRef != nil || dataSource != nil) &&
		(dataVolume.Spec.Source != nil || dataVolume.Spec.SourceRef != nil) {
		return []metav1.StatusCause{{
			Type:    metav1.CauseTypeFieldValueInvalid,
			Message: "External population is incompatible with Source and SourceRef",
			Field:   field.Child("source").String(),
		}}
	}

	if (dataSourceRef == nil && dataSource == nil) &&
		(dataVolume.Spec.Source == nil && dataVolume.Spec.SourceRef == nil) {
		return []metav1.StatusCause{{
			Type:    metav1.CauseTypeFieldValueInvalid,
			Message: "Data volume should have either Source, SourceRef, or be externally populated",
			Field:   field.Child("source", "sourceRef").String(),
		}}
	}

	if dataVolume.Spec.Source != nil {
		return validateNumberOfSources(field, dataVolume.Spec.Source)
	}

	return nil
}

func validateNumberOfSources(field *field.Path, source *cdiv1.DataVolumeSource) []metav1.StatusCause {
	numberOfSources := 0
	s := reflect.ValueOf(source).Elem()
	for i := 0; i < s.NumField(); i++ {
		if !reflect.ValueOf(s.Field(i).Interface()).IsNil() {
			numberOfSources++
		}
	}
	if numberOfSources == 0 {
		return []metav1.StatusCause{{
			Type:    metav1.CauseTypeFieldValueInvalid,
			Message: "Missing dataVolume valid source",
			Field:   field.Child("source").String(),
		}}
	}
	if numberOfSources > 1 {
		return []metav1.StatusCause{{
			Type:    metav1.CauseTypeFieldValueInvalid,
			Message: "Multiple dataVolume sources",
			Field:   field.Child("source").String(),
		}}
	}
	return nil
}
