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
	"context"
	"encoding/json"
	"fmt"
	"reflect"

	admissionv1 "k8s.io/api/admission/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	k8sfield "k8s.io/apimachinery/pkg/util/validation/field"
	"k8s.io/client-go/tools/cache"

	v1 "kubevirt.io/client-go/api/v1"
	snapshotv1 "kubevirt.io/client-go/apis/snapshot/v1alpha1"
	"kubevirt.io/client-go/kubecli"
	webhookutils "kubevirt.io/kubevirt/pkg/util/webhooks"
	"kubevirt.io/kubevirt/pkg/virt-api/webhooks"
	virtconfig "kubevirt.io/kubevirt/pkg/virt-config"
)

// VMRestoreAdmitter validates VirtualMachineRestores
type VMRestoreAdmitter struct {
	Config *virtconfig.ClusterConfig
	Client kubecli.KubevirtClient
}

// NewVMRestoreAdmitter creates a VMRestoreAdmitter
func NewVMRestoreAdmitter(config *virtconfig.ClusterConfig, client kubecli.KubevirtClient) *VMRestoreAdmitter {
	return &VMRestoreAdmitter{
		Config: config,
		Client: client,
	}
}

// Admit validates an AdmissionReview
func (admitter *VMRestoreAdmitter) Admit(ar *admissionv1.AdmissionReview) *admissionv1.AdmissionResponse {
	if ar.Request.Resource.Group != snapshotv1.SchemeGroupVersion.Group ||
		ar.Request.Resource.Resource != "virtualmachinerestores" {
		return webhookutils.ToAdmissionResponseError(fmt.Errorf("unexpected resource %+v", ar.Request.Resource))
	}

	if ar.Request.Operation == admissionv1.Create && !admitter.Config.SnapshotEnabled() {
		return webhookutils.ToAdmissionResponseError(fmt.Errorf("Snapshot/Restore feature gate not enabled"))
	}

	vmRestore := &snapshotv1.VirtualMachineRestore{}
	// TODO ideally use UniversalDeserializer here
	err := json.Unmarshal(ar.Request.Object.Raw, vmRestore)
	if err != nil {
		return webhookutils.ToAdmissionResponseError(err)
	}

	var causes []metav1.StatusCause

	switch ar.Request.Operation {
	case admissionv1.Create:
		var targetUID *types.UID
		targetField := k8sfield.NewPath("spec", "target")

		if vmRestore.Spec.Target.APIGroup == nil {
			causes = []metav1.StatusCause{
				{
					Type:    metav1.CauseTypeFieldValueNotFound,
					Message: "missing apiGroup",
					Field:   targetField.Child("apiGroup").String(),
				},
			}
		} else {
			switch *vmRestore.Spec.Target.APIGroup {
			case v1.GroupName:
				switch vmRestore.Spec.Target.Kind {
				case "VirtualMachine":
					causes, targetUID, err = admitter.validateCreateVM(targetField.Child("name"), ar.Request.Namespace, vmRestore.Spec.Target.Name)
					if err != nil {
						return webhookutils.ToAdmissionResponseError(err)
					}
				default:
					causes = []metav1.StatusCause{
						{
							Type:    metav1.CauseTypeFieldValueInvalid,
							Message: "invalid kind",
							Field:   targetField.Child("kind").String(),
						},
					}
				}
			default:
				causes = []metav1.StatusCause{
					{
						Type:    metav1.CauseTypeFieldValueInvalid,
						Message: "invalid apiGroup",
						Field:   targetField.Child("apiGroup").String(),
					},
				}
			}
		}

		snapshotCauses, err := admitter.validateSnapshot(
			k8sfield.NewPath("spec", "virtualMachineSnapshotName"),
			ar.Request.Namespace,
			vmRestore.Spec.VirtualMachineSnapshotName,
			targetUID,
		)
		if err != nil {
			return webhookutils.ToAdmissionResponseError(err)
		}

		informers := webhooks.GetInformers()
		objects, err := informers.VMRestoreInformer.GetIndexer().ByIndex(cache.NamespaceIndex, ar.Request.Namespace)
		if err != nil {
			return webhookutils.ToAdmissionResponseError(err)
		}

		for _, obj := range objects {
			r := obj.(*snapshotv1.VirtualMachineRestore)
			if reflect.DeepEqual(r.Spec.Target, vmRestore.Spec.Target) &&
				(r.Status == nil || r.Status.Complete == nil || !*r.Status.Complete) {
				cause := metav1.StatusCause{
					Type:    metav1.CauseTypeFieldValueInvalid,
					Message: fmt.Sprintf("VirtualMachineRestore %q in progress", r.Name),
					Field:   targetField.Child("name").String(),
				}
				causes = append(causes, cause)
			}
		}

		causes = append(causes, snapshotCauses...)

	case admissionv1.Update:
		prevObj := &snapshotv1.VirtualMachineRestore{}
		err = json.Unmarshal(ar.Request.OldObject.Raw, prevObj)
		if err != nil {
			return webhookutils.ToAdmissionResponseError(err)
		}

		if !reflect.DeepEqual(prevObj.Spec, vmRestore.Spec) {
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

func (admitter *VMRestoreAdmitter) validateCreateVM(field *k8sfield.Path, namespace, name string) ([]metav1.StatusCause, *types.UID, error) {
	vm, err := admitter.Client.VirtualMachine(namespace).Get(name, &metav1.GetOptions{})
	if errors.IsNotFound(err) {
		return []metav1.StatusCause{
			{
				Type:    metav1.CauseTypeFieldValueInvalid,
				Message: fmt.Sprintf("VirtualMachine %q does not exist", name),
				Field:   field.String(),
			},
		}, nil, nil
	}

	if err != nil {
		return nil, nil, err
	}

	var causes []metav1.StatusCause

	rs, err := vm.RunStrategy()
	if err != nil {
		return nil, nil, err
	}

	if rs != v1.RunStrategyHalted {
		cause := metav1.StatusCause{
			Type:    metav1.CauseTypeFieldValueInvalid,
			Message: fmt.Sprintf("VirtualMachine %q is running", name),
			Field:   field.String(),
		}
		causes = append(causes, cause)
	}

	return causes, &vm.UID, nil
}

func (admitter *VMRestoreAdmitter) validateSnapshot(field *k8sfield.Path, namespace, name string, targetUID *types.UID) ([]metav1.StatusCause, error) {
	snapshot, err := admitter.Client.VirtualMachineSnapshot(namespace).Get(context.Background(), name, metav1.GetOptions{})
	if errors.IsNotFound(err) {
		return []metav1.StatusCause{
			{
				Type:    metav1.CauseTypeFieldValueInvalid,
				Message: fmt.Sprintf("VirtualMachineSnapshot %q does not exist", name),
				Field:   field.String(),
			},
		}, nil
	}

	if err != nil {
		return nil, err
	}

	var causes []metav1.StatusCause

	if snapshot.Status == nil || snapshot.Status.ReadyToUse == nil || !*snapshot.Status.ReadyToUse {
		cause := metav1.StatusCause{
			Type:    metav1.CauseTypeFieldValueInvalid,
			Message: fmt.Sprintf("VirtualMachineSnapshot %q is not ready to use", name),
			Field:   field.String(),
		}
		causes = append(causes, cause)
	}

	if targetUID != nil && snapshot.Status != nil && snapshot.Status.SourceUID != nil && *targetUID != *snapshot.Status.SourceUID {
		cause := metav1.StatusCause{
			Type:    metav1.CauseTypeFieldValueInvalid,
			Message: fmt.Sprintf("VirtualMachineSnapshot source UID is %q but target UID is %q", *snapshot.Status.SourceUID, *targetUID),
			Field:   field.String(),
		}
		causes = append(causes, cause)
	}

	return causes, nil
}
