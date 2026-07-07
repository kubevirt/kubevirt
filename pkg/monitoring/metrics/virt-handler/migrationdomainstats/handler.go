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
	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/api"
	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/stats"
)

const logVerbosityInfo = 3

type vmiQueue interface {
	addCompletedStats(stats.DomainJobInfo)
	all() ([]result, bool)
	startPolling()
}

type handler struct {
	sync.Mutex

	vmiStore cache.Store
	vmiStats map[string]vmiQueue
}

func newHandler(vmiInformer cache.SharedIndexInformer, domainInformer cache.SharedInformer) (*handler, error) {
	h := handler{
		vmiStore: vmiInformer.GetStore(),
		vmiStats: make(map[string]vmiQueue),
	}

	_, err := vmiInformer.AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc:    h.handleVmiAdd,
		UpdateFunc: h.handleVmiUpdate,
	})
	if err != nil {
		return nil, err
	}

	if domainInformer != nil {
		_, err = domainInformer.AddEventHandler(cache.ResourceEventHandlerFuncs{
			AddFunc:    h.handleDomainAdd,
			UpdateFunc: h.handleDomainUpdate,
		})
	}

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
			log.Log.V(logVerbosityInfo).Infof("deleting queue for VMI %s", key)
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

func (h *handler) handleVmiAdd(obj interface{}) {
	vmi := obj.(*v1.VirtualMachineInstance)

	if vmi.Status.MigrationState == nil || vmi.Status.MigrationState.Completed {
		return
	}

	h.addMigration(vmi)
}

func (h *handler) handleDomainUpdate(_oldObj, newObj interface{}) {
	h.handleDomainAdd(newObj)
}

func (h *handler) handleDomainAdd(obj interface{}) {
	domain := obj.(*api.Domain)
	if domain.Status.MigrationStats == nil || !hasCompletedDowntimeStats(*domain.Status.MigrationStats) {
		return
	}

	key := controller.NamespacedKey(domain.Namespace, domain.Name)

	h.Lock()
	q, ok := h.vmiStats[key]
	h.Unlock()
	if !ok {
		vmi, exists := h.vmiForCompletedStats(key)
		if !exists {
			log.Log.V(logVerbosityInfo).Infof("dropping completed migration stats for VMI %s: VMI not found", key)
			return
		}

		q = newQueue(h.vmiStore, vmi)

		h.Lock()
		if existingQueue, exists := h.vmiStats[key]; exists {
			q = existingQueue
		} else {
			h.vmiStats[key] = q
		}
		h.Unlock()
	}

	q.addCompletedStats(*domain.Status.MigrationStats)
}

func (h *handler) vmiForCompletedStats(key string) (*v1.VirtualMachineInstance, bool) {
	if h.vmiStore == nil {
		return nil, false
	}

	obj, exists, err := h.vmiStore.GetByKey(key)
	if err != nil {
		log.Log.Reason(err).Errorf("failed to look up VMI %s for completed migration stats", key)
		return nil, false
	}
	if !exists {
		return nil, false
	}

	vmi, ok := obj.(*v1.VirtualMachineInstance)
	if !ok {
		log.Log.Errorf("failed to look up VMI %s for completed migration stats: unexpected object type %T", key, obj)
		return nil, false
	}

	return vmi, true
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
