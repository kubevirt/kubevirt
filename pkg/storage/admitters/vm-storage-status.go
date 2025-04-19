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

	admissionv1 "k8s.io/api/admission/v1"
	"k8s.io/apimachinery/pkg/api/equality"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	k8sfield "k8s.io/apimachinery/pkg/util/validation/field"
	v1 "kubevirt.io/api/core/v1"
)

func (a *Admitter) validateRestoreStatus() []metav1.StatusCause {
	if a.ar.Operation != admissionv1.Update || a.vm.Status.RestoreInProgress == nil {
		return nil
	}

	oldVM := &v1.VirtualMachine{}
	if err := json.Unmarshal(a.ar.OldObject.Raw, oldVM); err != nil {
		return []metav1.StatusCause{{
			Type:    metav1.CauseTypeUnexpectedServerResponse,
			Message: "Could not fetch old vm",
		}}
	}

	if !equality.Semantic.DeepEqual(oldVM.Spec, a.vm.Spec) {
		strategy, _ := a.vm.RunStrategy()
		if strategy != v1.RunStrategyHalted {
			return []metav1.StatusCause{{
				Type:    metav1.CauseTypeFieldValueNotSupported,
				Message: fmt.Sprintf("Cannot start VM until restore %q completes", *a.vm.Status.RestoreInProgress),
				Field:   k8sfield.NewPath("spec").String(),
			}}
		}
	}

	return nil
}

func (a *Admitter) validateSnapshotStatus() []metav1.StatusCause {
	if a.ar.Operation != admissionv1.Update || a.vm.Status.SnapshotInProgress == nil {
		return nil
	}

	oldVM := &v1.VirtualMachine{}
	if err := json.Unmarshal(a.ar.OldObject.Raw, oldVM); err != nil {
		return []metav1.StatusCause{{
			Type:    metav1.CauseTypeUnexpectedServerResponse,
			Message: "Could not fetch old vm",
		}}
	}

	if !compareVolumes(oldVM.Spec.Template.Spec.Volumes, a.vm.Spec.Template.Spec.Volumes) {
		return []metav1.StatusCause{{
			Type:    metav1.CauseTypeFieldValueNotSupported,
			Message: fmt.Sprintf("Cannot update vm disks or volumes until snapshot %q completes", *a.vm.Status.SnapshotInProgress),
			Field:   k8sfield.NewPath("spec").String(),
		}}
	}
	if !compareRunningSpec(&oldVM.Spec, &a.vm.Spec) {
		return []metav1.StatusCause{{
			Type:    metav1.CauseTypeFieldValueNotSupported,
			Message: fmt.Sprintf("Cannot update vm running state until snapshot %q completes", *a.vm.Status.SnapshotInProgress),
			Field:   k8sfield.NewPath("spec").String(),
		}}
	}

	return nil
}

func compareVolumes(old, new []v1.Volume) bool {
	if len(old) != len(new) {
		return false
	}

	for i, volume := range old {
		if !equality.Semantic.DeepEqual(volume, new[i]) {
			return false
		}
	}

	return true
}

func compareRunningSpec(old, new *v1.VirtualMachineSpec) bool {
	if old == nil || new == nil {
		// This should never happen, but just in case return false
		return false
	}

	// Its impossible to get here while both running and RunStrategy are nil.
	if old.Running != nil && new.Running != nil {
		return *old.Running == *new.Running
	}
	if old.RunStrategy != nil && new.RunStrategy != nil {
		return *old.RunStrategy == *new.RunStrategy
	}
	return false
}
