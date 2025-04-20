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

package vsock

import (
	"fmt"
	"math"
	"math/rand"
	"sync"

	virtv1 "kubevirt.io/api/core/v1"

	"kubevirt.io/kubevirt/pkg/controller"
)

type Allocator interface {
	Sync(vmis []*virtv1.VirtualMachineInstance)
	Allocate(vmi *virtv1.VirtualMachineInstance) error
	Remove(key string)
}

type randCIDFunc func() uint32
type nextCIDFunc func(uint32) uint32

type cidsMap struct {
	mu      sync.Mutex
	cids    map[string]uint32
	reverse map[uint32]string
	randCID randCIDFunc
	nextCID nextCIDFunc
}

func NewCIDsMap() *cidsMap {
	return &cidsMap{
		cids:    make(map[string]uint32),
		reverse: make(map[uint32]string),
		randCID: func() uint32 {
			cid := rand.Uint32()
			if cid < 3 {
				// The guest CID will start from 3
				cid += 3
			}
			return cid
		},
		nextCID: func(cur uint32) uint32 {
			if cur == math.MaxUint32 {
				return 3
			}
			return cur + 1
		},
	}
}

// Sync loads the allocated CIDs from VMIs.
func (m *cidsMap) Sync(vmis []*virtv1.VirtualMachineInstance) {
	m.mu.Lock()
	defer m.mu.Unlock()
	for _, vmi := range vmis {
		if vmi.Status.VSOCKCID == nil {
			continue
		}
		key := controller.VirtualMachineInstanceKey(vmi)
		m.cids[key] = *vmi.Status.VSOCKCID
		m.reverse[*vmi.Status.VSOCKCID] = key
	}
}

// Allocate select a new CID and set it to the status of the given VMI.
func (m *cidsMap) Allocate(vmi *virtv1.VirtualMachineInstance) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	key := controller.VirtualMachineInstanceKey(vmi)
	if cid, exist := m.cids[key]; exist {
		vmi.Status.VSOCKCID = &cid
		return nil
	}
	start := m.randCID()
	assigned := start
	for {
		if _, exist := m.reverse[assigned]; !exist {
			break
		}
		assigned = m.nextCID(assigned)
		if assigned == start {
			// Run out of CIDs. Practically this shouldn't happen.
			return fmt.Errorf("CIDs exhausted")
		}
	}
	m.cids[key] = assigned
	m.reverse[assigned] = key
	vmi.Status.VSOCKCID = &assigned
	return nil
}

// Remove cleans the CID for given VMI.
func (m *cidsMap) Remove(key string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if cid, exist := m.cids[key]; exist {
		delete(m.reverse, cid)
		delete(m.cids, key)
	}
}
