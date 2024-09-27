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
 * Copyright 2024 Red Hat, Inc.
 *
 */
package revision

import (
	"context"
	"fmt"

	appsv1 "k8s.io/api/apps/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	virtv1 "kubevirt.io/api/core/v1"
	"kubevirt.io/client-go/log"

	"kubevirt.io/kubevirt/pkg/apimachinery/patch"
)

func (h *RevisionHandler) patchVM(instancetypeRevision, preferenceRevision *appsv1.ControllerRevision, vm *virtv1.VirtualMachine) error {
	// Batch any writes to the VirtualMachine into a single Patch() call to avoid races in the controller.
	logger := func() *log.FilteredLogger { return log.Log.Object(vm) }
	revisionPatch, err := GeneratePatch(instancetypeRevision, preferenceRevision)
	if err != nil || len(revisionPatch) == 0 {
		return err
	}
	if _, err := h.virtClient.VirtualMachine(vm.Namespace).Patch(
		context.Background(), vm.Name, types.JSONPatchType, revisionPatch, metav1.PatchOptions{},
	); err != nil {
		logger().Reason(err).Error("Failed to update VirtualMachine with instancetype and preference ControllerRevision references.")
		return err
	}
	return nil
}

func GeneratePatch(instancetypeRevision, preferenceRevision *appsv1.ControllerRevision) ([]byte, error) {
	patchSet := patch.New()
	if instancetypeRevision != nil {
		patchSet.AddOption(
			patch.WithTest("/spec/instancetype/revisionName", nil),
			patch.WithAdd("/spec/instancetype/revisionName", instancetypeRevision.Name),
		)
	}

	if preferenceRevision != nil {
		patchSet.AddOption(
			patch.WithTest("/spec/preference/revisionName", nil),
			patch.WithAdd("/spec/preference/revisionName", preferenceRevision.Name),
		)
	}

	if patchSet.IsEmpty() {
		return nil, nil
	}

	payload, err := patchSet.GeneratePayload()
	if err != nil {
		// This is a programmer's error and should not happen
		return nil, fmt.Errorf("failed to generate patch payload: %w", err)
	}

	return payload, nil
}
