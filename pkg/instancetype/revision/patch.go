/*
Copyright The KubeVirt Authors.
SPDX-License-Identifier: Apache-2.0
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
	patchedVM, err := h.virtClient.VirtualMachine(vm.Namespace).PatchStatus(
		context.Background(), vm.Name, types.JSONPatchType, revisionPatch, metav1.PatchOptions{},
	)
	if err != nil {
		logger().Reason(err).Error("Failed to update VirtualMachine with instancetype and preference ControllerRevision references.")
		return err
	}
	// Update the local vm ObjectMeta with the response to ensure ResourceVersion and other server-side fields are current
	vm.ObjectMeta = patchedVM.ObjectMeta
	vm.Status = patchedVM.Status

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
