/*
Copyright The KubeVirt Authors.
SPDX-License-Identifier: Apache-2.0
*/

package migrationdomainstats

import (
	"sync"

	"k8s.io/client-go/tools/cache"
	v1 "kubevirt.io/api/core/v1"
	"kubevirt.io/client-go/log"

	"kubevirt.io/kubevirt/pkg/controller"
)

const logVerbosityInfo = 3

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
		AddFunc:    h.handleVmiAdd,
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
