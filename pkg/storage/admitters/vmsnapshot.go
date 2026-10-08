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

	admissionv1 "k8s.io/api/admission/v1"
	"k8s.io/apimachinery/pkg/api/equality"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	k8sfield "k8s.io/apimachinery/pkg/util/validation/field"

	"kubevirt.io/api/core"
	v1 "kubevirt.io/api/core/v1"

	snapshotv1 "kubevirt.io/api/snapshot/v1beta1"
	"kubevirt.io/client-go/kubecli"

	webhookutils "kubevirt.io/kubevirt/pkg/util/webhooks"
)

type vmSnapshotConfigChecker interface {
	SnapshotEnabled() bool
	ExternalVMSnapshotEnabled() bool
	UtilityVolumesEnabled() bool
}

// VMSnapshotAdmitter validates VirtualMachineSnapshots
type VMSnapshotAdmitter struct {
	config vmSnapshotConfigChecker
	Client kubecli.KubevirtClient
}

// NewVMSnapshotAdmitter creates a VMSnapshotAdmitter
func NewVMSnapshotAdmitter(config vmSnapshotConfigChecker, client kubecli.KubevirtClient) *VMSnapshotAdmitter {
	return &VMSnapshotAdmitter{
		config: config,
		Client: client,
	}
}

func externalMode(vmSnapshot *snapshotv1.VirtualMachineSnapshot) bool {
	return vmSnapshot.Spec.SnapshotMode != nil && *vmSnapshot.Spec.SnapshotMode == snapshotv1.SnapshotModeExternal
}

// validateExternalSource rejects an External mode snapshot of a stopped VM, since
// the overlays are put in place through the running domain, and of a VM with a
// disk libvirt cannot put an overlay on.
func (admitter *VMSnapshotAdmitter) validateExternalSource(ctx context.Context, vmSnapshot *snapshotv1.VirtualMachineSnapshot, namespace string, sourceField *k8sfield.Path) (*metav1.StatusCause, error) {
	if !externalMode(vmSnapshot) {
		return nil, nil
	}

	notRunning := &metav1.StatusCause{
		Type:    metav1.CauseTypeFieldValueInvalid,
		Message: fmt.Sprintf("%s mode requires a running VirtualMachine", snapshotv1.SnapshotModeExternal),
		Field:   sourceField.Child("name").String(),
	}

	vmi, err := admitter.Client.VirtualMachineInstance(namespace).Get(ctx, vmSnapshot.Spec.Source.Name, metav1.GetOptions{})
	if err != nil {
		if k8serrors.IsNotFound(err) {
			return notRunning, nil
		}
		return nil, err
	}
	if !vmi.IsRunning() {
		return notRunning, nil
	}
	for _, disk := range vmi.Spec.Domain.Devices.Disks {
		if kind := overlayUnsupported(disk); kind != "" {
			return &metav1.StatusCause{
				Type:    metav1.CauseTypeFieldValueInvalid,
				Message: fmt.Sprintf("%s mode does not support the %s disk %s", snapshotv1.SnapshotModeExternal, kind, disk.Name),
				Field:   sourceField.Child("name").String(),
			}, nil
		}
	}
	return nil, nil
}

// overlayUnsupported names the disks libvirt refuses a live external snapshot of:
// SCSI passthrough, and writable shared disks, since a qcow2 overlay cannot be shared
func overlayUnsupported(disk v1.Disk) string {
	switch {
	case disk.LUN != nil:
		return "LUN"
	case disk.Disk != nil && !disk.Disk.ReadOnly && disk.Shareable != nil && *disk.Shareable:
		return "shareable"
	}
	return ""
}

// Admit validates an AdmissionReview
func (admitter *VMSnapshotAdmitter) Admit(ctx context.Context, ar *admissionv1.AdmissionReview) *admissionv1.AdmissionResponse {
	if ar.Request.Resource.Group != snapshotv1.SchemeGroupVersion.Group ||
		ar.Request.Resource.Resource != "virtualmachinesnapshots" {
		return webhookutils.ToAdmissionResponseError(fmt.Errorf("unexpected resource %+v", ar.Request.Resource))
	}

	if ar.Request.Operation == admissionv1.Create && !admitter.config.SnapshotEnabled() {
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
		if externalMode(vmSnapshot) {
			if !admitter.config.ExternalVMSnapshotEnabled() {
				return webhookutils.ToAdmissionResponseError(fmt.Errorf("ExternalVMSnapshot feature gate not enabled"))
			}
			// The overlays are kept on a utility volume
			if !admitter.config.UtilityVolumesEnabled() {
				return webhookutils.ToAdmissionResponseError(fmt.Errorf("ExternalVMSnapshot requires the UtilityVolumes feature gate"))
			}
		}

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
				break
			}

			cause, err := admitter.validateExternalSource(ctx, vmSnapshot, ar.Request.Namespace, sourceField)
			if err != nil {
				return webhookutils.ToAdmissionResponseError(err)
			}
			if cause != nil {
				causes = []metav1.StatusCause{*cause}
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
