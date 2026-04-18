/*
Copyright The KubeVirt Authors.
SPDX-License-Identifier: Apache-2.0
*/

package admitters

import (
	"context"
	"encoding/json"
	"fmt"

	admissionv1 "k8s.io/api/admission/v1"
	"k8s.io/apimachinery/pkg/api/equality"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	k8sfield "k8s.io/apimachinery/pkg/util/validation/field"

	"kubevirt.io/api/core"

	snapshotv1 "kubevirt.io/api/snapshot/v1beta1"
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
func (admitter *VMSnapshotAdmitter) Admit(ctx context.Context, ar *admissionv1.AdmissionReview) *admissionv1.AdmissionResponse {
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
		case core.GroupName:
			if vmSnapshot.Spec.Source.Kind != "VirtualMachine" {
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

		if !equality.Semantic.DeepEqual(prevObj.Spec, vmSnapshot.Spec) {
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
