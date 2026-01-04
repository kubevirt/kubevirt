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
 */

package migrationdomainstats

import (
	"sync"

	"k8s.io/client-go/tools/cache"
	v1 "kubevirt.io/api/core/v1"
	"kubevirt.io/client-go/log"

	"kubevirt.io/kubevirt/pkg/controller"
)

type vmiQueue interface {
	all() ([]result, bool)
	startPolling()
}

type handler struct {
	sync.Mutex

	vmiStore cache.Store
	vmiStats map[string]vmiQueue
}

func newHandler(vmiInformer cache.SharedIndexInformer) (*handler, error) {
	h := handler{
		vmiStore: vmiInformer.GetStore(),
		vmiStats: make(map[string]vmiQueue),
	}

	_, err := vmiInformer.AddEventHandler(cache.ResourceEventHandlerFuncs{
		UpdateFunc: h.handleVmiUpdate,
	})

	return &h, err
}

func (h *handler) Collect() []result {
	var allResults []result

	h.Lock()
	defer h.Unlock()

	for key, q := range h.vmiStats {
		vmiResults, isFinished := q.all()
		allResults = append(allResults, vmiResults...)

		if isFinished {
			log.Log.V(3).Infof("deleting queue for VMI %s", key)
			delete(h.vmiStats, key)
		}
	}

	return allResults
}

func (h *handler) handleVmiUpdate(_oldObj, newObj interface{}) {
	newVmi := newObj.(*v1.VirtualMachineInstance)

	if newVmi.Status.MigrationState == nil || newVmi.Status.MigrationState.Completed {
		return
	}

	h.addMigration(newVmi)
}

func (h *handler) addMigration(vmi *v1.VirtualMachineInstance) {
	key := controller.NamespacedKey(vmi.Namespace, vmi.Name)

	h.Lock()
	defer h.Unlock()

	if _, ok := h.vmiStats[key]; ok {
		return
	}

	q := newQueue(h.vmiStore, vmi)
	q.startPolling()
	h.vmiStats[key] = q
}
