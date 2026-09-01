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
	"time"

	"k8s.io/client-go/tools/cache"
	v1 "kubevirt.io/api/core/v1"
	"kubevirt.io/client-go/log"

	"kubevirt.io/kubevirt/pkg/controller"
	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/api"
)

const logVerbosityInfo = 3

type vmiQueue interface {
	all() ([]result, bool)
	startPolling()
}

type handler struct {
	sync.Mutex

	sourceVMIStore cache.Store
	globalVMIStore cache.Store
	vmiStats       map[string]vmiQueue
	nodeName       string

	// Completed downtime follows the VMI across node handoff. It is removed when
	// the VMI disappears or a newer migration succeeds.
	completedMigrationStats map[string]completedMigrationResult
}

type completedMigrationResult struct {
	result
	vmiUID                  string
	migrationUID            string
	migrationStartTimestamp time.Time
}

func newHandler(
	nodeName string,
	sourceVMIInformer, globalVMIInformer cache.SharedIndexInformer,
	domainInformer cache.SharedInformer,
) (*handler, error) {
	h := handler{
		sourceVMIStore:          sourceVMIInformer.GetStore(),
		globalVMIStore:          globalVMIInformer.GetStore(),
		vmiStats:                make(map[string]vmiQueue),
		nodeName:                nodeName,
		completedMigrationStats: make(map[string]completedMigrationResult),
	}

	_, err := sourceVMIInformer.AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc:    h.handleVmiAdd,
		UpdateFunc: h.handleVmiUpdate,
	})
	if err != nil {
		return nil, err
	}

	if domainInformer != nil {
		_, err = domainInformer.AddEventHandler(cache.ResourceEventHandlerFuncs{
			AddFunc: h.handleDomainCompletedMigrationStats,
			UpdateFunc: func(_oldObj, newObj interface{}) {
				h.handleDomainCompletedMigrationStats(newObj)
			},
		})
		if err != nil {
			return nil, err
		}
	}

	return &h, nil
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

	for key, completedStats := range h.completedMigrationStats {
		obj, exists, err := h.globalVMIStore.GetByKey(key)
		if err != nil {
			log.Log.Reason(err).Errorf("failed to look up VMI %s for completed migration stats", key)
			continue
		}
		if !exists {
			delete(h.completedMigrationStats, key)
			continue
		}

		vmi, ok := obj.(*v1.VirtualMachineInstance)
		if !ok || string(vmi.UID) != completedStats.vmiUID || hasNewerSuccessfulMigration(vmi, completedStats.migrationUID, completedStats.migrationStartTimestamp) {
			delete(h.completedMigrationStats, key)
			continue
		}

		allResults = append(allResults, completedStats.result)
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

func (h *handler) handleDomainCompletedMigrationStats(obj interface{}) {
	domain := obj.(*api.Domain)
	if domain.Status.CompletedMigrationStats == nil || !domain.Status.CompletedMigrationStats.DowntimeSet {
		return
	}

	key := controller.NamespacedKey(domain.Namespace, domain.Name)
	vmi, exists := h.vmiForCompletedStats(key)
	if !exists {
		log.Log.V(logVerbosityInfo).Infof("dropping completed migration stats for VMI %s: VMI not found", key)
		return
	}
	domainUID := domain.Spec.Metadata.KubeVirt.UID
	if domainUID == "" {
		log.Log.Errorf("dropping completed migration stats for VMI %s: domain UID is missing", key)
		return
	}
	if domainUID != vmi.UID {
		log.Log.V(logVerbosityInfo).Infof("dropping stale completed migration stats for VMI %s: domain UID %q does not match VMI UID %q", key, domainUID, vmi.UID)
		return
	}

	completedStats := domain.Status.CompletedMigrationStats
	migration := domain.Spec.Metadata.KubeVirt.Migration
	if migration == nil || migration.UID == "" || migration.StartTimestamp == nil {
		log.Log.Errorf("dropping completed migration stats for VMI %s: incomplete migration metadata", key)
		return
	}
	migrationUID := string(migration.UID)
	migrationStartTimestamp := migration.StartTimestamp.Time
	if hasNewerSuccessfulMigration(vmi, migrationUID, migrationStartTimestamp) {
		log.Log.V(logVerbosityInfo).Infof("dropping stale completed migration stats for VMI %s and migration %s", key, migrationUID)
		return
	}
	r := result{
		vmi:         domain.Name,
		namespace:   domain.Namespace,
		node:        h.nodeName,
		downtimeSet: completedStats.DowntimeSet,
		downtime:    completedStats.Downtime,
	}

	h.Lock()
	defer h.Unlock()

	previousStats, exists := h.completedMigrationStats[key]
	if exists && previousStats.vmiUID == string(vmi.UID) {
		if previousStats.migrationUID == migrationUID {
			return
		}
		if !migrationStartTimestamp.After(previousStats.migrationStartTimestamp) {
			log.Log.V(logVerbosityInfo).Infof("dropping out-of-order completed migration stats for VMI %s and migration %s", key, migrationUID)
			return
		}
	}
	h.completedMigrationStats[key] = completedMigrationResult{
		result:                  r,
		vmiUID:                  string(vmi.UID),
		migrationUID:            migrationUID,
		migrationStartTimestamp: migrationStartTimestamp,
	}
}

func hasNewerSuccessfulMigration(vmi *v1.VirtualMachineInstance, migrationUID string, migrationStartTimestamp time.Time) bool {
	migrationState := vmi.Status.MigrationState
	// Migration UIDs are opaque, so informer ordering is resolved by the source timestamps.
	return migrationState != nil &&
		migrationState.Completed &&
		!migrationState.Failed &&
		string(migrationState.MigrationUID) != migrationUID &&
		migrationState.StartTimestamp != nil &&
		migrationState.StartTimestamp.Time.After(migrationStartTimestamp)
}

func (h *handler) vmiForCompletedStats(key string) (*v1.VirtualMachineInstance, bool) {
	if h.globalVMIStore == nil {
		return nil, false
	}

	obj, exists, err := h.globalVMIStore.GetByKey(key)
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

	q := newQueue(h.sourceVMIStore, vmi)
	q.startPolling()
	h.vmiStats[key] = q
}
