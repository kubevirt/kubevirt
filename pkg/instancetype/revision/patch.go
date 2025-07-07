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
 * Copyright The KubeVirt Authors
 *
 */
package revision

import (
	"context"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	virtv1 "kubevirt.io/api/core/v1"
	"kubevirt.io/client-go/log"

	"kubevirt.io/kubevirt/pkg/apimachinery/patch"
)

func (h *revisionHandler) patchVM(
	instancetypeStatusRef, preferenceStatusRef *virtv1.InstancetypeStatusRef,
	vm *virtv1.VirtualMachine,
) error {
	// Batch any writes to the VirtualMachine into a single PatchStatus() call to avoid races in the controller.
	logger := func() *log.FilteredLogger { return log.Log.Object(vm) }
	revisionPatch, err := GeneratePatch(instancetypeStatusRef, preferenceStatusRef)
	if err != nil || len(revisionPatch) == 0 {
		return err
	}
	if _, err := h.virtClient.VirtualMachine(vm.Namespace).PatchStatus(
		context.Background(), vm.Name, types.JSONPatchType, revisionPatch, metav1.PatchOptions{},
	); err != nil {
		logger().Reason(err).Error("Failed to update VirtualMachine with instancetype and preference ControllerRevision references.")
		return err
	}
	return nil
}

func GeneratePatch(instancetypeStatusRef, preferenceStatusRef *virtv1.InstancetypeStatusRef) ([]byte, error) {
	patchSet := patch.New()
	if instancetypeStatusRef != nil {
		patchSet.AddOption(
			patch.WithAdd("/status/instancetypeRef", instancetypeStatusRef),
		)
	}

	if preferenceStatusRef != nil {
		patchSet.AddOption(
			patch.WithAdd("/status/preferenceRef", preferenceStatusRef),
		)
	}

	if patchSet.IsEmpty() {
		return nil, nil
	}

	return patchSet.GeneratePayload()
}
