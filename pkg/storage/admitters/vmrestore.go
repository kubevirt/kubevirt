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
	"context"
	"encoding/json"
	"fmt"
	"strings"

	admissionv1 "k8s.io/api/admission/v1"
	"k8s.io/apimachinery/pkg/api/equality"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	k8sfield "k8s.io/apimachinery/pkg/util/validation/field"
	"k8s.io/client-go/tools/cache"

	"kubevirt.io/api/core"

	snapshotv1 "kubevirt.io/api/snapshot/v1beta1"
	"kubevirt.io/client-go/kubecli"

	backendstorage "kubevirt.io/kubevirt/pkg/storage/backend-storage"
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
func (admitter *VMRestoreAdmitter) Admit(ctx context.Context, ar *admissionv1.AdmissionReview) *admissionv1.AdmissionResponse {
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
					causes, err = admitter.validateTargetVM(ctx, k8sfield.NewPath("spec"), vmRestore)
					if err != nil {
						return webhookutils.ToAdmissionResponseError(err)
					}

					newCauses := admitter.validateVolumeOverrides(ctx, vmRestore)
					if newCauses != nil {
						causes = append(causes, newCauses...)
					}

					newCauses = admitter.validateVolumeRestorePolicy(ctx, vmRestore)
					if newCauses != nil {
						causes = append(causes, newCauses...)
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

func (admitter *VMRestoreAdmitter) validateTargetVM(ctx context.Context, field *k8sfield.Path, vmRestore *snapshotv1.VirtualMachineRestore) (causes []metav1.StatusCause, err error) {
	targetName := vmRestore.Spec.Target.Name
	namespace := vmRestore.Namespace

	causes = admitter.validatePatches(vmRestore.Spec.Patches, field.Child("patches"))

	vmSnapshot, err := admitter.Client.VirtualMachineSnapshot(namespace).Get(ctx, vmRestore.Spec.VirtualMachineSnapshotName, metav1.GetOptions{})
	if err != nil {
		if errors.IsNotFound(err) {
			return nil, nil
		}
		return nil, err
	}

	target, err := admitter.Client.VirtualMachine(namespace).Get(ctx, targetName, metav1.GetOptions{})
	if err != nil && !errors.IsNotFound(err) {
		return nil, err
	}

	sourceTargetVmsAreDifferent := errors.IsNotFound(err) || (vmSnapshot.Status.SourceUID != nil && target.UID != *vmSnapshot.Status.SourceUID)
	if sourceTargetVmsAreDifferent {
		contentName := vmSnapshot.Status.VirtualMachineSnapshotContentName
		if contentName == nil {
			return nil, fmt.Errorf("snapshot content name is nil in vmSnapshot status")
		}

		vmSnapshotContent, err := admitter.Client.VirtualMachineSnapshotContent(namespace).Get(ctx, *contentName, metav1.GetOptions{})
		if err != nil {
			return nil, err
		}

		snapshotVM := vmSnapshotContent.Spec.Source.VirtualMachine
		if snapshotVM == nil {
			return nil, fmt.Errorf("unexpected snapshot source")
		}

		if backendstorage.IsBackendStorageNeededForVMI(&snapshotVM.Spec.Template.Spec) {
			causes = append(causes, metav1.StatusCause{
				Type:    metav1.CauseTypeFieldValueInvalid,
				Message: "Restore to a different VM not supported when using backend storage",
				Field:   field.String(),
			})
		}
	}

	return causes, nil
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

func (admitter *VMRestoreAdmitter) validateVolumeOverrides(ctx context.Context, vmRestore *snapshotv1.VirtualMachineRestore) (causes []metav1.StatusCause) {
	// Cancel if there's no volume override
	if vmRestore.Spec.VolumeRestoreOverrides == nil {
		return nil
	}

	// Check each individual override
	for i, override := range vmRestore.Spec.VolumeRestoreOverrides {
		if override.VolumeName == "" {
			causes = append(causes, metav1.StatusCause{
				Type:    metav1.CauseTypeFieldValueRequired,
				Message: fmt.Sprintf("must provide a volume name"),
				Field: k8sfield.NewPath("spec").
					Child("volumeRestoreOverrides").
					Index(i).Child("volumeName").
					String(),
			})
		}

		if override.RestoreName == "" && override.Annotations == nil && override.Labels == nil {
			causes = append(causes, metav1.StatusCause{
				Type:    metav1.CauseTypeFieldValueRequired,
				Message: fmt.Sprintf("must provide at least one overriden field"),
				Field:   k8sfield.NewPath("spec").Child("volumeRestoreOverrides").Index(i).String(),
			})
		}
	}

	return causes
}

func (admitter *VMRestoreAdmitter) validateVolumeRestorePolicy(ctx context.Context, vmRestore *snapshotv1.VirtualMachineRestore) (causes []metav1.StatusCause) {
	// Cancel if there's no volume restore policy
	if vmRestore.Spec.VolumeRestorePolicy == nil {
		return nil
	}

	policy := *vmRestore.Spec.VolumeRestorePolicy

	// Verify the policy provided is among the ones that are allowed
	switch policy {
	case snapshotv1.VolumeRestorePolicyInPlace:
	case snapshotv1.VolumeRestorePolicyRandomizeNames:
		return nil
	default:
		causes = append(causes, metav1.StatusCause{
			Type:    metav1.CauseTypeFieldValueInvalid,
			Message: fmt.Sprintf("volume restore policy \"%s\" doesn't exist", policy),
			Field: k8sfield.NewPath("spec").
				Child("volumeRestorePolicy").
				String(),
		})
	}

	return causes
}
