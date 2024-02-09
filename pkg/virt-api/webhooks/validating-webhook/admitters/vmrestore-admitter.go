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
	"strings"

	admissionv1 "k8s.io/api/admission/v1"
	"k8s.io/apimachinery/pkg/api/equality"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	k8sfield "k8s.io/apimachinery/pkg/util/validation/field"
	"k8s.io/client-go/tools/cache"

	"kubevirt.io/api/core"

	v1 "kubevirt.io/api/core/v1"
	snapshotv1 "kubevirt.io/api/snapshot/v1alpha1"
	"kubevirt.io/client-go/kubecli"

	webhookutils "kubevirt.io/kubevirt/pkg/util/webhooks"
	virtconfig "kubevirt.io/kubevirt/pkg/virt-config"
)

// VMRestoreAdmitter validates VirtualMachineRestores
type VMRestoreAdmitter struct {
	Config            *virtconfig.ClusterConfig
	Client            kubecli.KubevirtClient
	VMRestoreInformer cache.SharedIndexInformer
}

// NewVMRestoreAdmitter creates a VMRestoreAdmitter
func NewVMRestoreAdmitter(config *virtconfig.ClusterConfig, client kubecli.KubevirtClient, vmRestoreInformer cache.SharedIndexInformer) *VMRestoreAdmitter {
	return &VMRestoreAdmitter{
		Config:            config,
		Client:            client,
		VMRestoreInformer: vmRestoreInformer,
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
	var targetVMExists bool

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
			case core.GroupName:
				switch vmRestore.Spec.Target.Kind {
				case "VirtualMachine":
					causes, targetUID, targetVMExists, err = admitter.validateCreateVM(k8sfield.NewPath("spec"), vmRestore)
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
			targetVMExists,
		)
		if err != nil {
			return webhookutils.ToAdmissionResponseError(err)
		}

		objects, err := admitter.VMRestoreInformer.GetIndexer().ByIndex(cache.NamespaceIndex, ar.Request.Namespace)
		if err != nil {
			return webhookutils.ToAdmissionResponseError(err)
		}

		for _, obj := range objects {
			r := obj.(*snapshotv1.VirtualMachineRestore)
			if equality.Semantic.DeepEqual(r.Spec.Target, vmRestore.Spec.Target) &&
				(r.Status == nil || r.Status.Complete == nil || !*r.Status.Complete) {
				cause := metav1.StatusCause{
					Type:    metav1.CauseTypeFieldValueInvalid,
					Message: fmt.Sprintf("VirtualMachineRestore %q in progress", r.Name),
					Field:   targetField.String(),
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

		if !equality.Semantic.DeepEqual(prevObj.Spec, vmRestore.Spec) {
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

func (admitter *VMRestoreAdmitter) validateCreateVM(field *k8sfield.Path, vmRestore *snapshotv1.VirtualMachineRestore) (causes []metav1.StatusCause, uid *types.UID, targetVMExists bool, err error) {
	vmName := vmRestore.Spec.Target.Name
	namespace := vmRestore.Namespace

	causes = admitter.validatePatches(vmRestore.Spec.Patches, field.Child("patches"))

	vm, err := admitter.Client.VirtualMachine(namespace).Get(context.Background(), vmName, &metav1.GetOptions{})
	if errors.IsNotFound(err) {
		// If the target VM does not exist it would be automatically created by the restore controller
		return nil, nil, false, nil
	}

	if err != nil {
		return nil, nil, false, err
	}

	rs, err := vm.RunStrategy()
	if err != nil {
		return nil, nil, true, err
	}

	if rs != v1.RunStrategyHalted {
		var cause metav1.StatusCause
		targetField := field.Child("target")
		if vm.Spec.Running != nil && *vm.Spec.Running {
			cause = metav1.StatusCause{
				Type:    metav1.CauseTypeFieldValueInvalid,
				Message: fmt.Sprintf("VirtualMachine %q is not stopped", vmName),
				Field:   targetField.String(),
			}
		} else {
			cause = metav1.StatusCause{
				Type:    metav1.CauseTypeFieldValueInvalid,
				Message: fmt.Sprintf("VirtualMachine %q run strategy has to be %s", vmName, v1.RunStrategyHalted),
				Field:   targetField.String(),
			}
		}
		causes = append(causes, cause)
	}

	return causes, &vm.UID, true, nil
}

func (admitter *VMRestoreAdmitter) validatePatches(patches []string, field *k8sfield.Path) (causes []metav1.StatusCause) {
	// Validate patches are either on labels/annotations or on elements under "/spec/" path only
	for _, patch := range patches {
		for _, patchKeyValue := range strings.Split(strings.Trim(patch, "{}"), ",") {
			// For example, if the original patch is {"op": "replace", "path": "/metadata/name", "value": "someValue"}
			// now we're iterating on [`"op": "replace"`, `"path": "/metadata/name"`, `"value": "someValue"`]
			keyValSlice := strings.Split(patchKeyValue, ":")
			if len(keyValSlice) != 2 {
				causes = append(causes, metav1.StatusCause{
					Type:    metav1.CauseTypeFieldValueInvalid,
					Message: fmt.Sprintf(`patch format is not valid - one ":" expected in a single key-value json patch: %s`, patchKeyValue),
					Field:   field.String(),
				})
				continue
			}

			key := strings.TrimSpace(keyValSlice[0])
			value := strings.TrimSpace(keyValSlice[1])

			if key == `"path"` {
				if strings.HasPrefix(value, `"/metadata/labels/`) || strings.HasPrefix(value, `"/metadata/annotations/`) {
					continue
				}
				if !strings.HasPrefix(value, `"/spec/`) {
					causes = append(causes, metav1.StatusCause{
						Type:    metav1.CauseTypeFieldValueInvalid,
						Message: fmt.Sprintf("patching is valid only for elements under /spec/ only: %s", patchKeyValue),
						Field:   field.String(),
					})
				}
			}
		}
	}

	return causes
}

func (admitter *VMRestoreAdmitter) validateSnapshot(field *k8sfield.Path, namespace, name string, targetUID *types.UID, targetVMExists bool) ([]metav1.StatusCause, error) {
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

	if snapshot.Status != nil && snapshot.Status.Phase == snapshotv1.Failed {
		cause := metav1.StatusCause{
			Type:    metav1.CauseTypeFieldValueInvalid,
			Message: fmt.Sprintf("VirtualMachineSnapshot %q has failed and is invalid to use", name),
			Field:   field.String(),
		}
		causes = append(causes, cause)
	}

	if snapshot.Status == nil || snapshot.Status.ReadyToUse == nil || !*snapshot.Status.ReadyToUse {
		cause := metav1.StatusCause{
			Type:    metav1.CauseTypeFieldValueInvalid,
			Message: fmt.Sprintf("VirtualMachineSnapshot %q is not ready to use", name),
			Field:   field.String(),
		}
		causes = append(causes, cause)
	}

	sourceTargetVmsAreDifferent := targetUID != nil && snapshot.Status != nil && snapshot.Status.SourceUID != nil && *targetUID != *snapshot.Status.SourceUID
	if sourceTargetVmsAreDifferent && targetVMExists {
		cause := metav1.StatusCause{
			Type:    metav1.CauseTypeFieldValueInvalid,
			Message: fmt.Sprintf("when snapshot source and restore target VMs are different - target VM must not exist"),
			Field:   field.String(),
		}
		causes = append(causes, cause)
	}

	return causes, nil
}
