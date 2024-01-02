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
 * Copyright 2024 The KubeVirt Authors
 *
 */

package admitters

import (
	"encoding/json"
	"fmt"

	admissionv1 "k8s.io/api/admission/v1"
	"k8s.io/apimachinery/pkg/api/equality"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	k8sfield "k8s.io/apimachinery/pkg/util/validation/field"
	virtstorage "kubevirt.io/api/storage/v1alpha1"
	"kubevirt.io/client-go/kubecli"

	webhookutils "kubevirt.io/kubevirt/pkg/util/webhooks"
	virtconfig "kubevirt.io/kubevirt/pkg/virt-config"
)

// VolumeMigrationAdmitter validates VolumeMigrations
type VolumeMigrationAdmitter struct {
	Config *virtconfig.ClusterConfig
	Client kubecli.KubevirtClient
}

// NewVolumeMigrationAdmitter creates a VolumeMigrationAdmitter
func NewVolumeMigrationAdmitter(config *virtconfig.ClusterConfig, client kubecli.KubevirtClient) *VolumeMigrationAdmitter {
	return &VolumeMigrationAdmitter{
		Config: config,
		Client: client,
	}
}

// Admit validates an AdmissionReview
func (admitter *VolumeMigrationAdmitter) Admit(ar *admissionv1.AdmissionReview) *admissionv1.AdmissionResponse {
	var causes []metav1.StatusCause
	if ar.Request.Resource.Group != virtstorage.SchemeGroupVersion.Group ||
		ar.Request.Resource.Resource != "volumemigrations" {
		return webhookutils.ToAdmissionResponseError(fmt.Errorf("unexpected resource %+v", ar.Request.Resource))
	}

	if ar.Request.Operation == admissionv1.Create && !admitter.Config.VolumeMigrationEnabled() {
		return webhookutils.ToAdmissionResponseError(fmt.Errorf("VolumeMigration feature gate not enabled"))
	}
	volMig := &virtstorage.VolumeMigration{}
	// TODO ideally use UniversalDeserializer here
	err := json.Unmarshal(ar.Request.Object.Raw, volMig)
	if err != nil {
		return webhookutils.ToAdmissionResponseError(err)
	}

	switch ar.Request.Operation {
	case admissionv1.Create:
		c, err := admitter.validateVolumeMigration(volMig)
		if err != nil {
			return webhookutils.ToAdmissionResponseError(err)
		}
		causes = append(causes, c...)
	case admissionv1.Update:
		prevObj := &virtstorage.VolumeMigration{}
		err = json.Unmarshal(ar.Request.OldObject.Raw, prevObj)
		if err != nil {
			return webhookutils.ToAdmissionResponseError(err)
		}

		if !equality.Semantic.DeepEqual(prevObj.Spec, volMig.Spec) {
			causes = append(causes, metav1.StatusCause{
				Type:    metav1.CauseTypeFieldValueInvalid,
				Message: "spec is immutable after creation",
				Field:   k8sfield.NewPath("spec").String(),
			})
		}
	default:
		return &admissionv1.AdmissionResponse{
			Allowed: true,
		}
	}
	if len(causes) > 0 {
		return webhookutils.ToAdmissionResponse(causes)
	}

	return &admissionv1.AdmissionResponse{
		Allowed: true,
	}
}

func validatePVC(name, typeVol, volMigName string, existingVols map[string]bool) *metav1.StatusCause {
	_, ok := existingVols[name]
	switch {
	case name == "":
		return &metav1.StatusCause{
			Type:    metav1.CauseTypeFieldValueInvalid,
			Message: fmt.Sprintf("VolumeMigration %s has an empty %s PVC name", volMigName, typeVol),
			Field:   "spec.migratedVolume",
		}
	case ok:
		return &metav1.StatusCause{
			Type:    metav1.CauseTypeFieldValueInvalid,
			Message: fmt.Sprintf("The %s volume %s has already been declared", typeVol, name),
			Field:   "spec.migratedVolume",
		}
	}

	return nil
}

func (admitter *VolumeMigrationAdmitter) validateVolumeMigration(volMig *virtstorage.VolumeMigration) ([]metav1.StatusCause, error) {
	var causes []metav1.StatusCause
	const (
		srcString = "source"
		dstString = "destination"
	)
	srcVols := make(map[string]bool)
	dstVols := make(map[string]bool)

	if len(volMig.Spec.MigratedVolume) < 1 {
		causes = append(causes, metav1.StatusCause{
			Type:    metav1.CauseTypeFieldValueInvalid,
			Message: "Empty migrated volumes",
			Field:   "spec.migratedVolume",
		})
	}

	for _, v := range volMig.Spec.MigratedVolume {
		if c := validatePVC(v.SourceClaim, srcString, volMig.Name, srcVols); c == nil {
			srcVols[v.SourceClaim] = true
		} else {
			causes = append(causes, *c)

		}

		if c := validatePVC(v.DestinationClaim, dstString, volMig.Name, dstVols); c == nil {
			dstVols[v.DestinationClaim] = true
		} else {
			causes = append(causes, *c)
		}
	}

	return causes, nil
}
