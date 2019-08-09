/*
 * This file is part of the CDI project
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
 * Copyright 2019 Red Hat, Inc.
 *
 */

package webhooks

import (
	"encoding/json"
	"fmt"
	"net/url"

	"k8s.io/api/admission/v1beta1"
	v1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	k8sfield "k8s.io/apimachinery/pkg/util/validation/field"
	"k8s.io/client-go/kubernetes"
	"k8s.io/klog"

	cdicorev1alpha1 "kubevirt.io/containerized-data-importer/pkg/apis/core/v1alpha1"
	"kubevirt.io/containerized-data-importer/pkg/controller"
)

type dataVolumeValidatingWebhook struct {
	client kubernetes.Interface
}

func validateSourceURL(sourceURL string) string {
	if sourceURL == "" {
		return "source URL is empty"
	}
	url, err := url.ParseRequestURI(sourceURL)
	if err != nil {
		return fmt.Sprintf("Invalid source URL: %s", sourceURL)
	}
	if url.Scheme != "http" && url.Scheme != "https" {
		return fmt.Sprintf("Invalid source URL scheme: %s", sourceURL)
	}
	return ""
}

func (wh *dataVolumeValidatingWebhook) validateDataVolumeSpec(request *v1beta1.AdmissionRequest, field *k8sfield.Path, spec *cdicorev1alpha1.DataVolumeSpec) []metav1.StatusCause {
	var causes []metav1.StatusCause
	var url string
	var sourceType string
	// spec source field should not be empty
	if &spec.Source == nil || (spec.Source.HTTP == nil && spec.Source.S3 == nil && spec.Source.PVC == nil && spec.Source.Upload == nil &&
		spec.Source.Blank == nil && spec.Source.Registry == nil) {
		causes = append(causes, metav1.StatusCause{
			Type:    metav1.CauseTypeFieldValueInvalid,
			Message: fmt.Sprintf("Missing Data volume source"),
			Field:   field.Child("source").String(),
		})
		return causes
	}

	if (spec.Source.HTTP != nil && (spec.Source.S3 != nil || spec.Source.PVC != nil || spec.Source.Upload != nil || spec.Source.Blank != nil || spec.Source.Registry != nil)) ||
		(spec.Source.S3 != nil && (spec.Source.PVC != nil || spec.Source.Upload != nil || spec.Source.Blank != nil || spec.Source.Registry != nil)) ||
		(spec.Source.PVC != nil && (spec.Source.Upload != nil || spec.Source.Blank != nil || spec.Source.Registry != nil)) ||
		(spec.Source.Upload != nil && (spec.Source.Blank != nil || spec.Source.Registry != nil)) ||
		(spec.Source.Blank != nil && spec.Source.Registry != nil) {
		causes = append(causes, metav1.StatusCause{
			Type:    metav1.CauseTypeFieldValueInvalid,
			Message: fmt.Sprintf("Multiple Data volume sources"),
			Field:   field.Child("source").String(),
		})
		return causes
	}
	// if source types are HTTP or S3, check if URL is valid
	if spec.Source.HTTP != nil || spec.Source.S3 != nil {
		if spec.Source.HTTP != nil {
			url = spec.Source.HTTP.URL
			sourceType = field.Child("source", "HTTP", "url").String()
		} else if spec.Source.S3 != nil {
			url = spec.Source.S3.URL
			sourceType = field.Child("source", "S3", "url").String()
		}
		err := validateSourceURL(url)
		if err != "" {
			causes = append(causes, metav1.StatusCause{
				Type:    metav1.CauseTypeFieldValueInvalid,
				Message: fmt.Sprintf("%s %s", field.Child("source").String(), err),
				Field:   sourceType,
			})
			return causes
		}
	}

	// Make sure contentType is either empty (kubevirt), or kubevirt or archive
	if spec.ContentType != "" && string(spec.ContentType) != string(cdicorev1alpha1.DataVolumeKubeVirt) && string(spec.ContentType) != string(cdicorev1alpha1.DataVolumeArchive) {
		sourceType = field.Child("contentType").String()
		causes = append(causes, metav1.StatusCause{
			Type:    metav1.CauseTypeFieldValueInvalid,
			Message: fmt.Sprintf("ContentType not one of: %s, %s", cdicorev1alpha1.DataVolumeKubeVirt, cdicorev1alpha1.DataVolumeArchive),
			Field:   sourceType,
		})
		return causes
	}

	if spec.Source.Blank != nil && string(spec.ContentType) == string(cdicorev1alpha1.DataVolumeArchive) {
		sourceType = field.Child("contentType").String()
		causes = append(causes, metav1.StatusCause{
			Type:    metav1.CauseTypeFieldValueInvalid,
			Message: fmt.Sprintf("SourceType cannot be blank and the contentType be archive"),
			Field:   sourceType,
		})
		return causes
	}

	if spec.Source.Registry != nil && spec.ContentType != "" && string(spec.ContentType) != string(cdicorev1alpha1.DataVolumeKubeVirt) {
		sourceType = field.Child("contentType").String()
		causes = append(causes, metav1.StatusCause{
			Type:    metav1.CauseTypeFieldValueInvalid,
			Message: fmt.Sprintf("ContentType must be " + string(cdicorev1alpha1.DataVolumeKubeVirt) + " when Source is Registry"),
			Field:   sourceType,
		})
		return causes
	}

	if spec.Source.PVC != nil {
		if spec.Source.PVC.Namespace == "" || spec.Source.PVC.Name == "" {
			causes = append(causes, metav1.StatusCause{
				Type:    metav1.CauseTypeFieldValueInvalid,
				Message: fmt.Sprintf("%s source PVC is not valid", field.Child("source", "PVC").String()),
				Field:   field.Child("source", "PVC").String(),
			})
			return causes
		}

		if wh.client != nil && request.Operation == v1beta1.Create {
			sourcePVC, err := wh.client.CoreV1().PersistentVolumeClaims(spec.Source.PVC.Namespace).Get(spec.Source.PVC.Name, metav1.GetOptions{})
			if err != nil {
				if k8serrors.IsNotFound(err) {
					causes = append(causes, metav1.StatusCause{
						Type:    metav1.CauseTypeFieldValueNotFound,
						Message: fmt.Sprintf("Source PVC %s/%s doesn't exist", spec.Source.PVC.Namespace, spec.Source.PVC.Name),
						Field:   field.Child("source", "PVC").String(),
					})
					return causes
				}
			}
			err = controller.ValidateCanCloneSourceAndTargetSpec(&sourcePVC.Spec, spec.PVC)
			if err != nil {
				causes = append(causes, metav1.StatusCause{
					Type:    metav1.CauseTypeFieldValueInvalid,
					Message: err.Error(),
					Field:   field.Child("PVC").String(),
				})
				return causes
			}
		}
	}

	if spec.PVC == nil {
		causes = append(causes, metav1.StatusCause{
			Type:    metav1.CauseTypeFieldValueInvalid,
			Message: fmt.Sprintf("Missing Data volume PVC"),
			Field:   field.Child("PVC").String(),
		})
		return causes
	}
	if pvcSize, ok := spec.PVC.Resources.Requests["storage"]; ok {
		if pvcSize.IsZero() || pvcSize.Value() < 0 {
			causes = append(causes, metav1.StatusCause{
				Type:    metav1.CauseTypeFieldValueInvalid,
				Message: fmt.Sprintf("PVC size can't be equal or less than zero"),
				Field:   field.Child("PVC", "resources", "requests", "size").String(),
			})
			return causes
		}
	} else {
		causes = append(causes, metav1.StatusCause{
			Type:    metav1.CauseTypeFieldValueInvalid,
			Message: fmt.Sprintf("PVC size is missing"),
			Field:   field.Child("PVC", "resources", "requests", "size").String(),
		})
		return causes
	}

	accessModes := spec.PVC.AccessModes
	if len(accessModes) > 1 {
		causes = append(causes, metav1.StatusCause{
			Type:    metav1.CauseTypeFieldValueInvalid,
			Message: fmt.Sprintf("PVC multiple accessModes"),
			Field:   field.Child("PVC", "accessModes").String(),
		})
		return causes
	}
	// We know we have one access mode
	if accessModes[0] != v1.ReadWriteOnce && accessModes[0] != v1.ReadOnlyMany && accessModes[0] != v1.ReadWriteMany {
		causes = append(causes, metav1.StatusCause{
			Type:    metav1.CauseTypeFieldValueInvalid,
			Message: fmt.Sprintf("Unsupported value: \"%s\": supported values: \"ReadOnlyMany\", \"ReadWriteMany\", \"ReadWriteOnce\"", string(accessModes[0])),
			Field:   field.Child("PVC", "accessModes").String(),
		})
		return causes
	}
	return causes
}

func (wh *dataVolumeValidatingWebhook) Admit(ar v1beta1.AdmissionReview) *v1beta1.AdmissionResponse {
	if err := validateDataVolumeResource(ar); err != nil {
		return toAdmissionResponseError(err)
	}

	raw := ar.Request.Object.Raw
	dv := cdicorev1alpha1.DataVolume{}

	err := json.Unmarshal(raw, &dv)

	if err != nil {
		return toAdmissionResponseError(err)
	}

	if wh.client != nil && ar.Request.Operation == v1beta1.Create {
		pvc, err := wh.client.CoreV1().PersistentVolumeClaims(dv.GetNamespace()).Get(dv.GetName(), metav1.GetOptions{})
		if err != nil && !k8serrors.IsNotFound(err) {
			return toAdmissionResponseError(err)
		}
		if pvc != nil && pvc.Name != "" {
			klog.Errorf("destination PVC %s/%s already exists", dv.GetNamespace(), dv.GetName())
			var causes []metav1.StatusCause
			causes = append(causes, metav1.StatusCause{
				Type:    metav1.CauseTypeFieldValueDuplicate,
				Message: fmt.Sprintf("Destination PVC already exists"),
				Field:   k8sfield.NewPath("DataVolume").Child("Name").String(),
			})
			return toRejectedAdmissionResponse(causes)
		}
	}

	causes := wh.validateDataVolumeSpec(ar.Request, k8sfield.NewPath("spec"), &dv.Spec)
	if len(causes) > 0 {
		klog.Infof("rejected DataVolume admission")
		return toRejectedAdmissionResponse(causes)
	}

	reviewResponse := v1beta1.AdmissionResponse{}
	reviewResponse.Allowed = true
	return &reviewResponse
}
