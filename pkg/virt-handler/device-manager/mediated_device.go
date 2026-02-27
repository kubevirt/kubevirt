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

package device_manager

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/fsnotify/fsnotify"
	v1 "kubevirt.io/api/core/v1"
	"kubevirt.io/client-go/log"

	"kubevirt.io/kubevirt/pkg/util"
	pluginapi "kubevirt.io/kubevirt/pkg/virt-handler/device-manager/deviceplugin/v1beta1"
)

// Not a const for static test purposes
var mdevBasePath string = "/sys/bus/mdev/devices"

type MDEV struct {
	UUID             string
	typeName         string
	parentPciAddress string
	iommuGroup       string
	numaNode         int
}

type MediatedDevicePlugin struct {
	*DevicePluginBase
	iommuToMDEVMap map[string]string
}

func NewMediatedDevicePlugin(mdevs []*MDEV, resourceName string) *MediatedDevicePlugin {
	s := strings.Split(resourceName, "/")
	mdevTypeName := s[1]
	iommuToMDEVMap := make(map[string]string)

	devs := constructDPIdevicesFromMdev(mdevs, iommuToMDEVMap)

	dpi := &MediatedDevicePlugin{
		DevicePluginBase: newDevicePluginBase(
			devs,
			mdevTypeName,
			util.HostRootMount,
			vfioDevicePath,
			resourceName,
		),
		iommuToMDEVMap: iommuToMDEVMap,
	}
	dpi.setupMonitoredDevices = dpi.setupMonitoredDevicesFunc
	dpi.deviceNameByID = dpi.deviceNameByIDFunc
	dpi.allocateDP = dpi.allocateDPFunc
	return dpi
}

func constructDPIdevicesFromMdev(mdevs []*MDEV, iommuToMDEVMap map[string]string) (devs []*pluginapi.Device) {
	for _, mdev := range mdevs {
		iommuToMDEVMap[mdev.iommuGroup] = mdev.UUID
		dpiDev := &pluginapi.Device{
			ID:     mdev.iommuGroup,
			Health: pluginapi.Unhealthy,
		}
		if mdev.numaNode >= 0 {
			numaInfo := &pluginapi.NUMANode{
				ID: int64(mdev.numaNode),
			}
			dpiDev.Topology = &pluginapi.TopologyInfo{
				Nodes: []*pluginapi.NUMANode{numaInfo},
			}
		}
		devs = append(devs, dpiDev)
	}
	return
}

func (dpi *MediatedDevicePlugin) allocateDPFunc(_ context.Context, r *pluginapi.AllocateRequest) (*pluginapi.AllocateResponse, error) {
	log.DefaultLogger().Infof("Allocate: resourceName: %s", dpi.resourceName)
	log.DefaultLogger().Infof("Allocate: iommuMap: %v", dpi.iommuToMDEVMap)
	resourceNameEnvVar := util.ResourceNameToEnvVar(v1.MDevResourcePrefix, dpi.resourceName)
	log.DefaultLogger().Infof("Allocate: resourceNameEnvVar: %s", resourceNameEnvVar)
	allocatedDevices := []string{}
	resp := new(pluginapi.AllocateResponse)
	containerResponse := new(pluginapi.ContainerAllocateResponse)

	for _, request := range r.ContainerRequests {
		log.DefaultLogger().Infof("Allocate: request: %v", request)
		deviceSpecs := make([]*pluginapi.DeviceSpec, 0)
		for _, devID := range request.DevicesIDs {
			log.DefaultLogger().Infof("Allocate: devID: %s", devID)
			if mdevUUID, exist := dpi.iommuToMDEVMap[devID]; exist {
				log.DefaultLogger().Infof("Allocate: got devID: %s for uuid: %s", devID, mdevUUID)
				allocatedDevices = append(allocatedDevices, mdevUUID)

				// Perform check that node didn't disappear
				_, err := os.Stat(filepath.Join(dpi.deviceRoot, dpi.devicePath, devID))
				if err != nil {
					if errors.Is(err, os.ErrNotExist) {
						log.DefaultLogger().Errorf("Mediated device %s with id %s for resource %s disappeared", mdevUUID, devID, dpi.resourceName)
					}
					return resp, fmt.Errorf("failed to allocate resource for resourceName: %s", dpi.resourceName)
				}

				formattedVFIO := formatVFIODeviceSpecs(devID)
				log.DefaultLogger().Infof("Allocate: formatted vfio: %v", formattedVFIO)
				deviceSpecs = append(deviceSpecs, formattedVFIO...)
			}
		}
		envVar := make(map[string]string)
		envVar[resourceNameEnvVar] = strings.Join(allocatedDevices, ",")
		log.DefaultLogger().Infof("Allocate: allocatedDevices: %v", allocatedDevices)
		containerResponse.Envs = envVar
		containerResponse.Devices = deviceSpecs
		log.DefaultLogger().Infof("Allocate: Envs: %v", envVar)
		log.DefaultLogger().Infof("Allocate: Devices: %v", deviceSpecs)
		resp.ContainerResponses = append(resp.ContainerResponses, containerResponse)
		if len(deviceSpecs) == 0 {
			return resp, fmt.Errorf("failed to allocate resource for resourceName: %s", dpi.resourceName)
		}
	}
	return resp, nil
}

func discoverPermittedHostMediatedDevices(supportedMdevsMap map[string]string) map[string][]*MDEV {
	mdevsMap := make(map[string][]*MDEV)
	files, err := os.ReadDir(mdevBasePath)
	for _, info := range files {
		if info.Type()&os.ModeSymlink == 0 {
			continue
		}
		mdevTypeName, err := getMdevTypeName(info.Name())
		if err != nil {
			log.DefaultLogger().Reason(err).Errorf("failed read type name for mdev: %s", info.Name())
			continue
		}
		if _, supported := supportedMdevsMap[mdevTypeName]; supported {

			mdev := &MDEV{
				typeName: mdevTypeName,
				UUID:     info.Name(),
			}
			parentPCIAddr, err := handler.GetMdevParentPCIAddr(info.Name())
			if err != nil {
				log.DefaultLogger().Reason(err).Errorf("failed parent PCI address for mdev: %s", info.Name())
				continue
			}
			mdev.parentPciAddress = parentPCIAddr

			mdev.numaNode = handler.GetDeviceNumaNode(pciBasePath, parentPCIAddr)
			iommuGroup, err := handler.GetDeviceIOMMUGroup(mdevBasePath, info.Name())
			if err != nil {
				log.DefaultLogger().Reason(err).Errorf("failed to get iommu group of mdev: %s", info.Name())
				continue
			}
			mdev.iommuGroup = iommuGroup
			mdevsMap[mdevTypeName] = append(mdevsMap[mdevTypeName], mdev)
		}
	}
	if err != nil {
		log.DefaultLogger().Reason(err).Errorf("failed to discover mediated devices")
	}
	return mdevsMap
}

func (dpi *MediatedDevicePlugin) deviceNameByIDFunc(monDevId string) string {
	mdev, ok := dpi.iommuToMDEVMap[monDevId]
	if !ok {
		mdev = "not recognized"
	}
	return fmt.Sprintf("mediated device (mdev=%s, id=%s)", mdev, monDevId)
}

func (dpi *MediatedDevicePlugin) setupMonitoredDevicesFunc(watcher *fsnotify.Watcher, monitoredDevices map[string]string) error {
	// setupVFIOMonitoredDevices is a helper function defined in pci_device.go
	return setupVFIOMonitoredDevices(dpi.deviceRoot, dpi.devicePath, dpi.devs, watcher, monitoredDevices)
}

func getMdevTypeName(mdevUUID string) (string, error) {
	// #nosec No risk for path injection. Path is composed from static base  "mdevBasePath" and static components
	rawName, err := os.ReadFile(filepath.Join(mdevBasePath, mdevUUID, "mdev_type/name"))
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			originFile, err := os.Readlink(filepath.Join(mdevBasePath, mdevUUID, "mdev_type"))
			if err != nil {
				return "", err
			}
			rawName = []byte(filepath.Base(originFile))
		} else {
			return "", err
		}
	}
	// The name usually contain spaces which should be replaced with _
	typeNameStr := strings.Replace(string(rawName), " ", "_", -1)
	typeNameStr = strings.TrimSpace(typeNameStr)
	return typeNameStr, nil
}
