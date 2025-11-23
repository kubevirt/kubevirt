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

package passtrefactor

import (
	"sync"

	"k8s.io/apimachinery/pkg/types"
	v1 "kubevirt.io/api/core/v1"
	"kubevirt.io/client-go/log"

	"kubevirt.io/kubevirt/pkg/network/vmispec"
)

type handler interface {
	RunCommand(string, *v1.VirtualMachineInstance)
	FindSocket(string) (string, error)
}

type clusterConfigurer interface {
	GetNetworkBindings() map[string]v1.InterfaceBindingPlugin
}

type PasstRepairController struct {
	passtRepairHandler handler
	clusterConfigurer  clusterConfigurer
	activeVMs          map[types.UID]struct{}
	mutex              sync.Mutex
}

func NewPasstRepairController(clusterConfigurer clusterConfigurer) *PasstRepairController {
	return NewPasstRepairControllerWithOptions(newRepairHandler(), clusterConfigurer)
}

func NewPasstRepairControllerWithOptions(repairManager handler, clusterConfigurer clusterConfigurer) *PasstRepairController {
	return &PasstRepairController{
		passtRepairHandler: repairManager,
		clusterConfigurer:  clusterConfigurer,
		activeVMs:          map[types.UID]struct{}{},
	}
}

func (r *PasstRepairController) Run(vmi *v1.VirtualMachineInstance, isMigrationSource bool, socketDirFunc func(*v1.VirtualMachineInstance) (string, error)) error {
	if !shouldRunPasstRepair(vmi, r.clusterConfigurer.GetNetworkBindings()) {
		return nil
	}

	socketDir, err := socketDirFunc(vmi)
	if err != nil {
		return err
	}

	passtRepairArg := socketDir
	if isMigrationSource {
		var passtDir string
		passtDir, err = r.passtRepairHandler.FindSocket(socketDir)
		if err != nil {
			return err
		}
		passtRepairArg = passtDir
	}

	go func() {
		if isPasstRepairRunning := r.testAndSetActive(vmi); isPasstRepairRunning {
			log.DefaultLogger().Warningf("passt-repair already running for VMI %v, skipping execution", vmi.Name)
			return
		}
		defer r.setInactive(vmi)
		r.passtRepairHandler.RunCommand(passtRepairArg, vmi)
	}()
	return nil
}

func (r *PasstRepairController) testAndSetActive(vmi *v1.VirtualMachineInstance) bool {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	_, isActive := r.activeVMs[vmi.UID]
	if !isActive {
		r.activeVMs[vmi.UID] = struct{}{}
	}
	return isActive
}

func (r *PasstRepairController) setInactive(vmi *v1.VirtualMachineInstance) {
	r.mutex.Lock()
	defer r.mutex.Unlock()
	delete(r.activeVMs, vmi.UID)
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
