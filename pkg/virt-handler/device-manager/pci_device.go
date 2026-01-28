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
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/fsnotify/fsnotify"
	v1 "kubevirt.io/api/core/v1"
	"kubevirt.io/client-go/log"

	"kubevirt.io/kubevirt/pkg/util"
	pluginapi "kubevirt.io/kubevirt/pkg/virt-handler/device-manager/deviceplugin/v1beta1"
)

const (
	vfioDevicePath = "/dev/vfio/"
	vfioMount      = "/dev/vfio/vfio"
	pciBasePath    = "/sys/bus/pci/devices"
)

type PCIDevice struct {
	pciID      string
	driver     string
	pciAddress string
	iommuGroup string
	numaNode   int
}

type PCIDevicePlugin struct {
	*DevicePluginBase
	iommuToPCIMap map[string]string
}

func NewPCIDevicePlugin(pciDevices []*PCIDevice, resourceName string) *PCIDevicePlugin {
	serverSock := SocketPath(strings.Replace(resourceName, "/", "-", -1))
	iommuToPCIMap := make(map[string]string)

	devs := constructDPIdevices(pciDevices, iommuToPCIMap)

	dpi := &PCIDevicePlugin{
		DevicePluginBase: &DevicePluginBase{
			devs:         devs,
			initialized:  false,
			lock:         &sync.Mutex{},
			socketPath:   serverSock,
			devicePath:   vfioDevicePath,
			resourceName: resourceName,
			deviceRoot:   util.HostRootMount,
			healthUpdate: make(chan struct{}, 1),
			done:         make(chan struct{}),
			deregistered: make(chan struct{}),
		},
		iommuToPCIMap: iommuToPCIMap,
	}
	dpi.setupMonitoredDevices = dpi.setupMonitoredDevicesFunc
	dpi.allocateDP = dpi.allocateDPFunc
	dpi.deviceNameByID = dpi.deviceNameByIDFunc
	return dpi
}

func constructDPIdevices(pciDevices []*PCIDevice, iommuToPCIMap map[string]string) (devs []*pluginapi.Device) {
	for _, pciDevice := range pciDevices {
		iommuToPCIMap[pciDevice.iommuGroup] = pciDevice.pciAddress
		dpiDev := &pluginapi.Device{
			ID:     pciDevice.iommuGroup,
			Health: pluginapi.Unhealthy,
		}
		if pciDevice.numaNode >= 0 {
			numaInfo := &pluginapi.NUMANode{
				ID: int64(pciDevice.numaNode),
			}
			dpiDev.Topology = &pluginapi.TopologyInfo{
				Nodes: []*pluginapi.NUMANode{numaInfo},
			}
		}
		devs = append(devs, dpiDev)
	}
	return
}

func (dpi *PCIDevicePlugin) allocateDPFunc(_ context.Context, r *pluginapi.AllocateRequest) (*pluginapi.AllocateResponse, error) {
	resourceNameEnvVar := util.ResourceNameToEnvVar(v1.PCIResourcePrefix, dpi.resourceName)
	allocatedDevices := []string{}
	resp := new(pluginapi.AllocateResponse)
	containerResponse := new(pluginapi.ContainerAllocateResponse)

	for _, request := range r.ContainerRequests {
		deviceSpecs := make([]*pluginapi.DeviceSpec, 0)
		for _, devID := range request.DevicesIDs {
			// translate device's iommu group to its pci address
			devPCIAddress, exist := dpi.iommuToPCIMap[devID]
			if !exist {
				continue
			}
			allocatedDevices = append(allocatedDevices, devPCIAddress)
			deviceSpecs = append(deviceSpecs, formatVFIODeviceSpecs(devID)...)
		}
		containerResponse.Devices = deviceSpecs
		envVar := make(map[string]string)
		envVar[resourceNameEnvVar] = strings.Join(allocatedDevices, ",")

		containerResponse.Envs = envVar
		resp.ContainerResponses = append(resp.ContainerResponses, containerResponse)
	}
	return resp, nil
}

func (dpi *PCIDevicePlugin) deviceNameByIDFunc(monDevId string) string {
	pciID, ok := dpi.iommuToPCIMap[monDevId]
	if !ok {
		pciID = "not recognized"
	}
	return fmt.Sprintf("PCI device (pciAddr=%s, id=%s)", pciID, monDevId)
}

func setupVFIOMonitoredDevices(deviceRoot, devicePath string, devs []*pluginapi.Device, watcher *fsnotify.Watcher, monitoredDevices map[string]string) error {
	fullDevicePath := filepath.Join(deviceRoot, devicePath)
	// for pci and mediated devices, devices are added directly into the devicePath directory
	if err := watcher.Add(fullDevicePath); err != nil {
		log.DefaultLogger().Warningf("failed to add device path %s to the watcher: %v", fullDevicePath, err)
	}
	deviceDirPath := filepath.Dir(fullDevicePath)
	if err := watcher.Add(deviceDirPath); err != nil {
		// unrecoverable error
		return fmt.Errorf("failed to add device directory %s to the watcher", deviceDirPath)
	}
	// mark devices to be tracked by the watcher
	for _, dev := range devs {
		vfioDevice := filepath.Join(fullDevicePath, dev.ID)
		monitoredDevices[vfioDevice] = dev.ID
	}
	return nil
}

func (dpi *PCIDevicePlugin) setupMonitoredDevicesFunc(watcher *fsnotify.Watcher, monitoredDevices map[string]string) error {
	return setupVFIOMonitoredDevices(dpi.deviceRoot, dpi.devicePath, dpi.devs, watcher, monitoredDevices)
}

func discoverPermittedHostPCIDevices(supportedPCIDeviceMap map[string]string) map[string][]*PCIDevice {
	pciDevicesMap := make(map[string][]*PCIDevice)
	err := filepath.Walk(pciBasePath, func(path string, info os.FileInfo, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if info.IsDir() {
			return nil
		}
		pciID, err := handler.GetDevicePCIID(pciBasePath, info.Name())
		if err != nil {
			log.DefaultLogger().Reason(err).Errorf("failed get vendor:device ID for device: %s", info.Name())
			return nil
		}
		if resourceName, supported := supportedPCIDeviceMap[pciID]; supported {
			// check device driver
			driver, err := handler.GetDeviceDriver(pciBasePath, info.Name())
			if err != nil || driver != "vfio-pci" {
				return nil
			}

			pcidev := &PCIDevice{
				pciID:      pciID,
				pciAddress: info.Name(),
			}
			iommuGroup, err := handler.GetDeviceIOMMUGroup(pciBasePath, info.Name())
			if err != nil {
				return nil
			}
			pcidev.iommuGroup = iommuGroup
			pcidev.driver = driver
			pcidev.numaNode = handler.GetDeviceNumaNode(pciBasePath, info.Name())
			pciDevicesMap[resourceName] = append(pciDevicesMap[resourceName], pcidev)
		}
		return nil
	})
	if err != nil {
		log.DefaultLogger().Reason(err).Errorf("failed to discover host devices")
	}
	return pciDevicesMap
}
