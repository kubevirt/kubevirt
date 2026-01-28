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
	"strconv"
	"sync"

	"github.com/fsnotify/fsnotify"
	"kubevirt.io/client-go/log"

	"kubevirt.io/kubevirt/pkg/util"
	pluginapi "kubevirt.io/kubevirt/pkg/virt-handler/device-manager/deviceplugin/v1beta1"
)

type GenericDevicePlugin struct {
	*DevicePluginBase
	preOpen     bool
	permissions string
}

func NewGenericDevicePlugin(deviceName string, devicePath string, maxDevices int, permissions string, preOpen bool) *GenericDevicePlugin {
	serverSock := SocketPath(deviceName)
	dpi := &GenericDevicePlugin{
		DevicePluginBase: &DevicePluginBase{
			devs:         []*pluginapi.Device{},
			socketPath:   serverSock,
			deviceRoot:   util.HostRootMount,
			devicePath:   devicePath,
			healthUpdate: make(chan struct{}, 1),
			done:         make(chan struct{}),
			deregistered: make(chan struct{}),
			resourceName: fmt.Sprintf("%s/%s", DeviceNamespace, deviceName),
			initialized:  false,
			lock:         &sync.Mutex{},
		},
		preOpen:     preOpen,
		permissions: permissions,
	}

	for i := range maxDevices {
		deviceId := deviceName + strconv.Itoa(i)
		dpi.devs = append(dpi.devs, &pluginapi.Device{
			ID:     deviceId,
			Health: pluginapi.Unhealthy,
		})
	}
	dpi.deviceNameByID = dpi.deviceNameByIDFunc
	dpi.setupMonitoredDevices = dpi.setupMonitoredDevicesFunc
	dpi.setupDevicePlugin = dpi.setupDevicePluginFunc
	dpi.allocateDP = dpi.allocateDPFunc
	return dpi
}

func (dpi *GenericDevicePlugin) setupDevicePluginFunc() error {
	// The kernel module(s) for some devices, like tun and vhost-net, auto-load when needed.
	// That need is identified by the first access to their main device node.
	// Opening and closing the device nodes here will trigger any necessary modprobe.
	if dpi.preOpen {
		devicePath := filepath.Join(dpi.deviceRoot, dpi.devicePath)
		devnode, err := os.Open(devicePath)
		if err == nil {
			devnode.Close()
		}
	}
	return nil
}

func (dpi *GenericDevicePlugin) allocateDPFunc(ctx context.Context, r *pluginapi.AllocateRequest) (*pluginapi.AllocateResponse, error) {
	log.DefaultLogger().Infof("Generic Allocate: resourceName: %s", dpi.resourceName)
	log.DefaultLogger().Infof("Generic Allocate: request: %v", r.ContainerRequests)
	response := pluginapi.AllocateResponse{}
	containerResponse := new(pluginapi.ContainerAllocateResponse)

	dev := new(pluginapi.DeviceSpec)
	dev.HostPath = dpi.devicePath
	dev.ContainerPath = dpi.devicePath
	dev.Permissions = dpi.permissions
	containerResponse.Devices = []*pluginapi.DeviceSpec{dev}

	response.ContainerResponses = []*pluginapi.ContainerAllocateResponse{containerResponse}

	return &response, nil
}

func (dpi *GenericDevicePlugin) setupMonitoredDevicesFunc(watcher *fsnotify.Watcher, monitoredDevices map[string]string) error {
	devicePath := filepath.Join(dpi.deviceRoot, dpi.devicePath)
	deviceDirPath := filepath.Dir(devicePath)
	// Note: Directly watching the device path is not recommended and instead we watch the directory path
	// as this is more stable (as per fsnotify documentation)
	if err := watcher.Add(deviceDirPath); err != nil {
		return fmt.Errorf("failed to add the device path to the watcher: %v", err)
	}
	monitoredDevices[devicePath] = ""
	return nil
}

func (dpi *GenericDevicePlugin) deviceNameByIDFunc(_ string) string {
	devicePath := filepath.Join(dpi.deviceRoot, dpi.devicePath)
	return fmt.Sprintf("generic device (%s)", devicePath)
}
