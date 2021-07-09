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

package admitters

import (
	"encoding/json"
	"fmt"
	"reflect"

	admissionv1 "k8s.io/api/admission/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	k8sfield "k8s.io/apimachinery/pkg/util/validation/field"

	v1 "kubevirt.io/client-go/api/v1"
	snapshotv1 "kubevirt.io/client-go/apis/snapshot/v1alpha1"
	"kubevirt.io/client-go/kubecli"
	webhookutils "kubevirt.io/kubevirt/pkg/util/webhooks"
	virtconfig "kubevirt.io/kubevirt/pkg/virt-config"
)

// VMSnapshotAdmitter validates VirtualMachineSnapshots
type VMSnapshotAdmitter struct {
	Config *virtconfig.ClusterConfig
	Client kubecli.KubevirtClient
}

// NewVMSnapshotAdmitter creates a VMSnapshotAdmitter
func NewVMSnapshotAdmitter(config *virtconfig.ClusterConfig, client kubecli.KubevirtClient) *VMSnapshotAdmitter {
	return &VMSnapshotAdmitter{
		Config: config,
		Client: client,
	}
}

// Admit validates an AdmissionReview
func (admitter *VMSnapshotAdmitter) Admit(ar *admissionv1.AdmissionReview) *admissionv1.AdmissionResponse {
	if ar.Request.Resource.Group != snapshotv1.SchemeGroupVersion.Group ||
		ar.Request.Resource.Resource != "virtualmachinesnapshots" {
		return webhookutils.ToAdmissionResponseError(fmt.Errorf("unexpected resource %+v", ar.Request.Resource))
	}

	if ar.Request.Operation == admissionv1.Create && !admitter.Config.SnapshotEnabled() {
		return webhookutils.ToAdmissionResponseError(fmt.Errorf("snapshot feature gate not enabled"))
	}

	vmSnapshot := &snapshotv1.VirtualMachineSnapshot{}
	// TODO ideally use UniversalDeserializer here
	err := json.Unmarshal(ar.Request.Object.Raw, vmSnapshot)
	if err != nil {
		return webhookutils.ToAdmissionResponseError(err)
	}

	var causes []metav1.StatusCause

	switch ar.Request.Operation {
	case admissionv1.Create:
		sourceField := k8sfield.NewPath("spec", "source")

		if vmSnapshot.Spec.Source.APIGroup == nil {
			causes = []metav1.StatusCause{
				{
					Type:    metav1.CauseTypeFieldValueNotFound,
					Message: "missing apiGroup",
					Field:   sourceField.Child("apiGroup").String(),
				},
			}
			break
		}

		switch *vmSnapshot.Spec.Source.APIGroup {
		case v1.GroupName:
			switch vmSnapshot.Spec.Source.Kind {
			case "VirtualMachine":
				causes, err = admitter.validateCreateVM(sourceField.Child("name"), ar.Request.Namespace, vmSnapshot.Spec.Source.Name)
				if err != nil {
					return webhookutils.ToAdmissionResponseError(err)
				}
			default:
				causes = []metav1.StatusCause{
					{
						Type:    metav1.CauseTypeFieldValueInvalid,
						Message: "invalid kind",
						Field:   sourceField.Child("kind").String(),
					},
				}
			}
		default:
			causes = []metav1.StatusCause{
				{
					Type:    metav1.CauseTypeFieldValueInvalid,
					Message: "invalid apiGroup",
					Field:   sourceField.Child("apiGroup").String(),
				},
			}
		}

	case admissionv1.Update:
		prevObj := &snapshotv1.VirtualMachineSnapshot{}
		err = json.Unmarshal(ar.Request.OldObject.Raw, prevObj)
		if err != nil {
			return webhookutils.ToAdmissionResponseError(err)
		}

		if !reflect.DeepEqual(prevObj.Spec, vmSnapshot.Spec) {
			causes = []metav1.StatusCause{
				{
					Type:    metav1.CauseTypeFieldValueInvalid,
					Message: "spec in immutable after creation",
					Field:   k8sfield.NewPath("spec").String(),
				},
			}
		}
	default:
		return webhookutils.ToAdmissionResponseError(fmt.Errorf("unexpected operation %s", ar.Request.Operation))
	}

	if len(causes) > 0 {
		return webhookutils.ToAdmissionResponse(causes)
	}

	reviewResponse := admissionv1.AdmissionResponse{
		Allowed: true,
	}
	return &reviewResponse
}

func (admitter *VMSnapshotAdmitter) validateCreateVM(field *k8sfield.Path, namespace, name string) ([]metav1.StatusCause, error) {
	_, err := admitter.Client.VirtualMachine(namespace).Get(name, &metav1.GetOptions{})
	if errors.IsNotFound(err) {
		return []metav1.StatusCause{
			{
				Type:    metav1.CauseTypeFieldValueInvalid,
				Message: fmt.Sprintf("VirtualMachine %q does not exist", name),
				Field:   field.String(),
			},
		}, nil
	}

	if err != nil {
		return nil, err
	}

	return []metav1.StatusCause{}, nil
}
