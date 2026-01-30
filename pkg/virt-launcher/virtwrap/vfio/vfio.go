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

package vfio

import (
	"path/filepath"

	"kubevirt.io/client-go/log"

	cmdv1 "kubevirt.io/kubevirt/pkg/handler-launcher-com/cmd/v1"
	"kubevirt.io/kubevirt/pkg/util/hardware"
)

type VFIOSpec struct {
	iommufdAvailable bool
}

func NewVFIOSpec(options *cmdv1.VirtualMachineOptions) *VFIOSpec {
	iommufdAvailable := false
	if options != nil && options.HostDevIOMMUFDCap {
		exist, err := hardware.IOMMUFDExists()
		if err != nil {
			log.Log.Reason(err).Error("failed to detect the presence of iommufd")
		}
		if exist {
			iommufdAvailable = true
		}
	}
	if iommufdAvailable == true {
		log.Log.V(2).Info("iommufd is avaiable")
	}

	return &VFIOSpec{
		iommufdAvailable: iommufdAvailable,
	}
}

func (v *VFIOSpec) IsPCIAssignableViaIOMMUFD(pciAddress string) bool {
	devPath := filepath.Join(hardware.PCIDeviceBasePath, pciAddress)
	return v.isDeviceAssignableViaIOMMUFD(devPath)
}

func (v *VFIOSpec) IsMDevAssignableViaIOMMUFD(mdevUUID string) bool {
	devPath := filepath.Join(hardware.MDevDeviceBasePath, mdevUUID)
	return v.isDeviceAssignableViaIOMMUFD(devPath)
}

func (v *VFIOSpec) isDeviceAssignableViaIOMMUFD(devPath string) bool {
	if !v.iommufdAvailable {
		return false
	}

	devName, err := hardware.GetDeviceVFIOCDevName(devPath)
	if err != nil {
		log.Log.Reason(err).Errorf("failed to get vfio cdev name of %s", devPath)
		return false
	}

	result, err := hardware.VFIOCDevExists(devName)
	if err != nil {
		log.Log.Reason(err).Errorf("failed to detect the presence of vfio cdev %s", devName)
	}
	return result
}
