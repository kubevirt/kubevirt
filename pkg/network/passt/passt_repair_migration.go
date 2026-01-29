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
 *
 */

package passt

import (
	"sync"

	"k8s.io/apimachinery/pkg/types"
	v1 "kubevirt.io/api/core/v1"
	"kubevirt.io/client-go/log"

	"kubevirt.io/kubevirt/pkg/network/vmispec"
)

type socketBasedExecutor interface {
	RunCommand(string, *v1.VirtualMachineInstance)
	FindSocket(string) (string, error)
}

type clusterConfigurer interface {
	GetNetworkBindings() map[string]v1.InterfaceBindingPlugin
}

type PasstRepairMigrationCoordinator struct {
	passtRepairHandler socketBasedExecutor
	clusterConfigurer  clusterConfigurer
	activeVMs          map[types.UID]struct{}
	mutex              sync.Mutex
}

func NewPasstRepairMigrationCoordinator(clusterConfigurer clusterConfigurer) *PasstRepairMigrationCoordinator {
	return NewPasstRepairMigrationCoordinatorWithOptions(newRepairHandler(), clusterConfigurer)
}

func NewPasstRepairMigrationCoordinatorWithOptions(repairManager socketBasedExecutor, clusterConfigurer clusterConfigurer) *PasstRepairMigrationCoordinator {
	return &PasstRepairMigrationCoordinator{
		passtRepairHandler: repairManager,
		clusterConfigurer:  clusterConfigurer,
		activeVMs:          map[types.UID]struct{}{},
	}
}

func (r *PasstRepairMigrationCoordinator) MigrationSourceRun(vmi *v1.VirtualMachineInstance, socketDirFunc func(*v1.VirtualMachineInstance) (string, error)) error {
	// In migration source the socket already exists, need to pass the file itself to passt-repair
	socketFileFunc := func(vmi *v1.VirtualMachineInstance) (string, error) {
		socketDir, err := socketDirFunc(vmi)
		if err != nil {
			return "", err
		}
		return r.passtRepairHandler.FindSocket(socketDir)
	}
	return r.run(vmi, socketFileFunc)
}

func (r *PasstRepairMigrationCoordinator) MigrationTargetRun(vmi *v1.VirtualMachineInstance, socketDirFunc func(*v1.VirtualMachineInstance) (string, error)) error {
	return r.run(vmi, socketDirFunc)
}

func (r *PasstRepairMigrationCoordinator) run(vmi *v1.VirtualMachineInstance, socketDirFunc func(*v1.VirtualMachineInstance) (string, error)) error {
	if !shouldRunPasstRepair(vmi, r.clusterConfigurer.GetNetworkBindings()) {
		return nil
	}

	passtRepairArg, err := socketDirFunc(vmi)
	if err != nil {
		return err
	}

	go func() {
		if isPasstRepairRunning := r.testAndSetActive(vmi.UID); isPasstRepairRunning {
			log.DefaultLogger().Warningf("passt-repair already running for VMI %v, skipping execution", vmi.Name)
			return
		}
		defer r.setInactive(vmi.UID)
		r.passtRepairHandler.RunCommand(passtRepairArg, vmi)
	}()
	return nil
}

func (r *PasstRepairMigrationCoordinator) testAndSetActive(vmiUID types.UID) bool {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	_, isActive := r.activeVMs[vmiUID]
	if !isActive {
		r.activeVMs[vmiUID] = struct{}{}
	}
	return isActive
}

func (r *PasstRepairMigrationCoordinator) setInactive(vmiUID types.UID) {
	r.mutex.Lock()
	defer r.mutex.Unlock()
	delete(r.activeVMs, vmiUID)
}

func shouldRunPasstRepair(vmi *v1.VirtualMachineInstance, registeredPlugins map[string]v1.InterfaceBindingPlugin) bool {
	podNetwork := vmispec.LookUpDefaultNetwork(vmi.Spec.Networks)
	if podNetwork == nil {
		return false
	}

	iface := vmispec.LookupInterfaceByName(vmi.Spec.Domain.Devices.Interfaces, podNetwork.Name)
	if iface == nil {
		return false
	}

	binding := iface.Binding
	if binding == nil {
		return false
	}

	registeredPlugin, exists := registeredPlugins[binding.Name]
	if !exists || registeredPlugin.DomainAttachmentType != "" {
		return false
	}

	return true
}
