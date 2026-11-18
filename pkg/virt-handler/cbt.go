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
 */

package virthandler

import (
	v1 "kubevirt.io/api/core/v1"

	"kubevirt.io/kubevirt/pkg/storage/cbt"
	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/api"
)

type CBTHandler struct{}

func NewCBTHandler() *CBTHandler {
	return &CBTHandler{}
}

// HandleChangedBlockTracking updates CBT status based on domain state.
func (h *CBTHandler) HandleChangedBlockTracking(vmi *v1.VirtualMachineInstance, domain *api.Domain) error {
	if domain == nil || !cbt.CBTStateInitializing(vmi.Status.ChangedBlockTracking) {
		return nil
	}

	if !h.allDisksHaveDataStore(vmi, domain) {
		return nil
	}

	cbt.SetCBTState(&vmi.Status.ChangedBlockTracking, v1.ChangedBlockTrackingEnabled)
	return nil
}

func (h *CBTHandler) allDisksHaveDataStore(vmi *v1.VirtualMachineInstance, domain *api.Domain) bool {
	for _, volume := range vmi.Spec.Volumes {
		if !cbt.IsCBTEligibleVolume(&volume) {
			continue
		}
		found := false
		for _, disk := range domain.Spec.Devices.Disks {
			if disk.Alias.GetName() == volume.Name {
				found = true
				if disk.Source.DataStore == nil {
					return false
				}
				break
			}
		}
		if !found {
			return false
		}
	}
	return true
}
