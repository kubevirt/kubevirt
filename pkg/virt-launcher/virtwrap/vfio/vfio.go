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

//go:generate mockgen -source $GOFILE -package=$GOPACKAGE -destination=generated_mock_$GOFILE

package vfio

import (
	"errors"
	"os"
	"path/filepath"

	"kubevirt.io/client-go/log"

	"kubevirt.io/kubevirt/pkg/util/hardware"
)

var (
	pciDeviceBasePath  = "/sys/bus/pci/devices"
	mdevDeviceBasePath = "/sys/bus/mdev/devices"

	iommufdPath      = "/dev/iommu"
	vfioCDevBasePath = "/dev/vfio/devices"
)

type VFIOSpec interface {
	IsPCIAssignableViaIOMMUFD(pciAddress string) bool
	IsMDevAssignableViaIOMMUFD(mdevUUID string) bool
}

type vfioSpec struct {
	iommufdAvailable bool
}

func NewVFIOSpec(iommufdSupported bool) VFIOSpec {
	iommufdAvailable := false
	if iommufdSupported {
		exist, err := iommufdExists()
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

	return &vfioSpec{
		iommufdAvailable: iommufdAvailable,
	}
}

func iommufdExists() (bool, error) {
	if _, err := os.Stat(iommufdPath); err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return false, nil
		}
		return false, err
	}
	return true, nil
}

func (v *vfioSpec) IsPCIAssignableViaIOMMUFD(pciAddress string) bool {
	devPath := filepath.Join(pciDeviceBasePath, pciAddress)
	return v.isDeviceAssignableViaIOMMUFD(devPath)
}

func (v *vfioSpec) IsMDevAssignableViaIOMMUFD(mdevUUID string) bool {
	devPath := filepath.Join(mdevDeviceBasePath, mdevUUID)
	return v.isDeviceAssignableViaIOMMUFD(devPath)
}

func (v *vfioSpec) isDeviceAssignableViaIOMMUFD(devPath string) bool {
	if !v.iommufdAvailable {
		return false
	}

	devName, err := hardware.GetDeviceVFIOCDevName(devPath)
	if err != nil {
		log.Log.Reason(err).Errorf("failed to get vfio cdev name of %s", devPath)
		return false
	}

	result, err := vfioCDevExists(devName)
	if err != nil {
		log.Log.Reason(err).Errorf("failed to detect the presence of vfio cdev %s", devName)
	}
	return result
}

func vfioCDevExists(name string) (bool, error) {
	if name == "" {
		return false, nil
	}

	path := filepath.Join(vfioCDevBasePath, name)
	if _, err := os.Stat(path); err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return false, nil
		}
		return false, err
	}
	return true, nil
}
