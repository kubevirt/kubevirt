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
 * Copyright the KubeVirt Authors.
 */

package migrationdomainstats

import (
	"k8s.io/client-go/tools/cache"
	v1 "kubevirt.io/api/core/v1"
	"kubevirt.io/client-go/log"

	"kubevirt.io/kubevirt/pkg/controller"
)

type Handler struct {
	migrationInformer cache.SharedIndexInformer
	vmiInformer       cache.SharedIndexInformer

	vmimStats                map[string]*queue
	vmimScheduledForDeletion map[string]bool
}

func NewHandler(vmiInformer cache.SharedIndexInformer, migrationInformer cache.SharedIndexInformer) (Handler, error) {
	h := Handler{
		migrationInformer: migrationInformer,
		vmiInformer:       vmiInformer,

		vmimStats:                make(map[string]*queue),
		vmimScheduledForDeletion: make(map[string]bool),
	}

	_, err := migrationInformer.AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc:    h.addMigration,
		DeleteFunc: h.deleteMigration,
	})
	if err != nil {
		return Handler{}, err
	}

	return h, nil
}

func (h *Handler) Collect() []Result {
	var results []Result

	for _, q := range h.vmimStats {
		results = append(results, q.all()...)
	}

	// After collecting all the pending results, remove VMIMs that were already deleted
	for key := range h.vmimScheduledForDeletion {
		delete(h.vmimStats, key)
		delete(h.vmimScheduledForDeletion, key)
	}

	return results
}

func (h *Handler) addMigration(obj interface{}) {
	vmim := obj.(*v1.VirtualMachineInstanceMigration)

	vmi, exists, err := h.getVmi(vmim)
	if err != nil {
		log.Log.Reason(err).Errorf("Failed to get VMI for VMIM %s", namespacedNameKey(vmim))
		return
	}
	if !exists {
		log.Log.Warningf("VMI for VMIM %s not found", namespacedNameKey(vmim))
		return
	}

	q := newQueue(vmi, vmim)
	q.startPolling()
	h.vmimStats[namespacedNameKey(vmim)] = q
}

func (h *Handler) deleteMigration(obj interface{}) {
	vmim := obj.(*v1.VirtualMachineInstanceMigration)

	q := h.vmimStats[namespacedNameKey(vmim)]
	q.stopPolling()

	h.vmimScheduledForDeletion[namespacedNameKey(vmim)] = true
}

func namespacedNameKey(vmim *v1.VirtualMachineInstanceMigration) string {
	return vmim.Namespace + "/" + vmim.Name
}

func (h *Handler) getVmi(vmim *v1.VirtualMachineInstanceMigration) (*v1.VirtualMachineInstance, bool, error) {
	key := controller.NamespacedKey(vmim.Namespace, vmim.Spec.VMIName)
	obj, exists, err := h.vmiInformer.GetStore().GetByKey(key)
	if err != nil || !exists {
		return nil, exists, err
	}
	return obj.(*v1.VirtualMachineInstance), exists, nil
}
