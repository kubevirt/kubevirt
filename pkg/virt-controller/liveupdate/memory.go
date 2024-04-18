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
 * Copyright 2024 The KubeVirt Authors.
 *
 */

package liveupdate

import (
	"context"
	"fmt"

	v1 "kubevirt.io/api/core/v1"
	"kubevirt.io/client-go/kubecli"
	"kubevirt.io/client-go/log"

	"kubevirt.io/kubevirt/pkg/controller"
	"kubevirt.io/kubevirt/pkg/util/migrations"

	k8score "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
)

type MemoryHotplugHandler struct {
	clientset kubecli.KubevirtClient
}

func NewMemoryHotplugHandler(clientset kubecli.KubevirtClient) *MemoryHotplugHandler {
	return &MemoryHotplugHandler{
		clientset: clientset,
	}
}

func (m *MemoryHotplugHandler) GetManagedFields() []string {
	return []string{"/Spec/Template/Spec/Domain/Memory/Guest"}
}

func (m *MemoryHotplugHandler) HandleLiveUpdate(vm *v1.VirtualMachine, vmi *v1.VirtualMachineInstance) error {
	if vmi == nil || vmi.DeletionTimestamp != nil {
		return nil
	}

	if vm.Spec.Template.Spec.Domain.Memory == nil || vmi.Spec.Domain.Memory == nil {
		return nil
	}

	guestMemory := vmi.Spec.Domain.Memory.Guest

	if vmi.Status.Memory == nil ||
		vmi.Status.Memory.GuestCurrent == nil ||
		vm.Spec.Template.Spec.Domain.Memory.Guest.Equal(*guestMemory) {
		return nil
	}

	conditionManager := controller.NewVirtualMachineInstanceConditionManager()
	if conditionManager.HasConditionWithStatus(vmi,
		v1.VirtualMachineInstanceMemoryChange, k8score.ConditionTrue) {
		return fmt.Errorf("another memory hotplug is in progress")
	}

	if migrations.IsMigrating(vmi) {
		return fmt.Errorf("memory hotplug is not allowed while VMI is migrating")
	}

	if vm.Spec.Template.Spec.Domain.Memory.Guest != nil && vmi.Status.Memory.GuestAtBoot != nil &&
		vm.Spec.Template.Spec.Domain.Memory.Guest.Cmp(*vmi.Status.Memory.GuestAtBoot) == -1 {
		vmConditions := controller.NewVirtualMachineConditionManager()
		vmConditions.UpdateCondition(vm, &v1.VirtualMachineCondition{
			Type:               v1.VirtualMachineRestartRequired,
			LastTransitionTime: metav1.Now(),
			Status:             k8score.ConditionTrue,
			Message:            "memory updated in template spec to a value lower than what the VM started with",
		})
		return nil
	}

	// If the following is true, MaxGuest was calculated, not manually specified (or the validation webhook would have rejected the change).
	// Since we're here, we can also assume MaxGuest was not changed in the VM spec since last boot.
	// Therefore, bumping Guest to a value higher than MaxGuest is fine, it just requires a reboot.
	if vm.Spec.Template.Spec.Domain.Memory.Guest != nil && vmi.Spec.Domain.Memory.MaxGuest != nil &&
		vm.Spec.Template.Spec.Domain.Memory.Guest.Cmp(*vmi.Spec.Domain.Memory.MaxGuest) == 1 {
		vmConditions := controller.NewVirtualMachineConditionManager()
		vmConditions.UpdateCondition(vm, &v1.VirtualMachineCondition{
			Type:               v1.VirtualMachineRestartRequired,
			LastTransitionTime: metav1.Now(),
			Status:             k8score.ConditionTrue,
			Message:            "memory updated in template spec to a value higher than what's available",
		})
		return nil
	}

	memoryDelta := resource.NewQuantity(vm.Spec.Template.Spec.Domain.Memory.Guest.Value()-vmi.Status.Memory.GuestCurrent.Value(), resource.BinarySI)

	newMemoryReq := vmi.Spec.Domain.Resources.Requests.Memory().DeepCopy()
	newMemoryReq.Add(*memoryDelta)

	// checking if the new memory req are at least equal to the memory being requested in the handleMemoryHotplugRequest
	// this is necessary as weirdness can arise after hot-unplugs as not all memory is guaranteed to be released when doing hot-unplug.
	if newMemoryReq.Cmp(*vm.Spec.Template.Spec.Domain.Memory.Guest) == -1 {
		newMemoryReq = *vm.Spec.Template.Spec.Domain.Memory.Guest
		// adjusting memoryDelta too for the new limits computation (if required)
		memoryDelta = resource.NewQuantity(vm.Spec.Template.Spec.Domain.Memory.Guest.Value()-newMemoryReq.Value(), resource.BinarySI)
	}

	patches := []string{
		fmt.Sprintf(`{ "op": "test", "path": "/spec/domain/memory/guest", "value": "%s"}`, vmi.Spec.Domain.Memory.Guest.String()),
		fmt.Sprintf(`{ "op": "replace", "path": "/spec/domain/memory/guest", "value": "%s"}`, vm.Spec.Template.Spec.Domain.Memory.Guest.String()),
		fmt.Sprintf(`{ "op": "test", "path": "/spec/domain/resources/requests/memory", "value": "%s"}`, vmi.Spec.Domain.Resources.Requests.Memory().String()),
		fmt.Sprintf(`{ "op": "replace", "path": "/spec/domain/resources/requests/memory", "value": "%s"}`, newMemoryReq.String()),
	}

	if !vm.Spec.Template.Spec.Domain.Resources.Limits.Memory().IsZero() {
		newMemoryLimit := vmi.Spec.Domain.Resources.Limits.Memory().DeepCopy()
		newMemoryLimit.Add(*memoryDelta)

		patches = append(patches, fmt.Sprintf(`{ "op": "test", "path": "/spec/domain/resources/limits/memory", "value": "%s"}`, vmi.Spec.Domain.Resources.Limits.Memory().String()))
		patches = append(patches, fmt.Sprintf(`{ "op": "replace", "path": "/spec/domain/resources/limits/memory", "value": "%s"}`, newMemoryLimit.String()))
	}

	memoryPatch := controller.GeneratePatchBytes(patches)

	_, err := m.clientset.VirtualMachineInstance(vmi.Namespace).Patch(context.Background(), vmi.Name, types.JSONPatchType, memoryPatch, metav1.PatchOptions{})
	if err != nil {
		return err
	}

	log.Log.Object(vmi).V(4).Infof("hotplugging memory to %s", vm.Spec.Template.Spec.Domain.Memory.Guest.String())

	return nil
}
