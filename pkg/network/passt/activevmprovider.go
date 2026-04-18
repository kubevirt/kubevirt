/*
Copyright The KubeVirt Authors.
SPDX-License-Identifier: Apache-2.0
*/

package passt

import (
	"sync"

	"k8s.io/apimachinery/pkg/types"
	v1 "kubevirt.io/api/core/v1"
)

type activeVMProvider struct {
	activeVMs map[types.UID]struct{}
	mutex     sync.Mutex
}

func newActiveVMProvider() *activeVMProvider {
	return &activeVMProvider{
		activeVMs: map[types.UID]struct{}{},
	}
}

func (a *activeVMProvider) TestAndSetActive(vmi *v1.VirtualMachineInstance) bool {
	var isAlreadyActive bool
	a.mutex.Lock()
	defer a.mutex.Unlock()

	if _, isActive := a.activeVMs[vmi.UID]; !isActive {
		a.activeVMs[vmi.UID] = struct{}{}
		isAlreadyActive = false
		return isAlreadyActive
	}
	isAlreadyActive = true
	return isAlreadyActive
}

func (a *activeVMProvider) SetInactive(vmi *v1.VirtualMachineInstance) {
	a.mutex.Lock()
	defer a.mutex.Unlock()
	delete(a.activeVMs, vmi.UID)
}
