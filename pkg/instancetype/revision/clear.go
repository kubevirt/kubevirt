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
package revision

import (
	"context"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	virtv1 "kubevirt.io/api/core/v1"

	"kubevirt.io/kubevirt/pkg/apimachinery/patch"
)

func (h *revisionHandler) Clear(vm *virtv1.VirtualMachine) error {
	patchSet := patch.New()
	if vm.Spec.Instancetype == nil && vm.Status.InstancetypeRef != nil {
		patchSet.AddOption(patch.WithRemove(instancetypeStatusRefPath))
	}
	if vm.Spec.Preference == nil && vm.Status.PreferenceRef != nil {
		patchSet.AddOption(patch.WithRemove(preferenceStatusRefPath))
	}
	if patchSet.IsEmpty() {
		return nil
	}
	statusPatch, err := patchSet.GeneratePayload()
	if err != nil {
		return err
	}
	patchedVM, err := h.virtClient.VirtualMachine(vm.Namespace).PatchStatus(
		context.Background(), vm.Name, types.JSONPatchType, statusPatch, metav1.PatchOptions{})
	if err != nil {
		return err
	}
	// Update the local vm ObjectMeta with the response to ensure ResourceVersion and other server-side fields are current
	vm.ObjectMeta = patchedVM.ObjectMeta
	vm.Status = patchedVM.Status

	return nil
}
