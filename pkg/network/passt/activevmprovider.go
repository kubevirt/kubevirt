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
