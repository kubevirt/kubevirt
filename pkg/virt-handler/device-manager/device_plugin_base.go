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
	"time"

	"github.com/fsnotify/fsnotify"

	"kubevirt.io/kubevirt/pkg/safepath"
	pluginapi "kubevirt.io/kubevirt/pkg/virt-handler/device-manager/deviceplugin/v1beta1"
)

const (
	DeviceNamespace   = "devices.kubevirt.io"
	connectionTimeout = 5 * time.Second
)

type devicePluginContract interface {
	setupMonitoredDevices(watcher *fsnotify.Watcher, monitoredDevices map[string]string) error   // REQUIRED function to set up the devices that are being monitored and update map such that key contains absolute paths to watch, and value contains the device id that path corresponds to.
	allocateDP(context.Context, *pluginapi.AllocateRequest) (*pluginapi.AllocateResponse, error) // REQUIRED function to allocate the device.
	setupDevicePlugin() error                                                                    // Optional function to perform additional setup steps that are not covered by the default implementation
	deviceNameByID(deviceID string) string                                                       // Optional function to convert device id to a human-readable name for logging
	configurePermissions(absoluteDevicePath *safepath.Path) error                                // Optional function to configure permissions for the device if needed. When present, device being marked healthy is contingent on the hook exiting without error.
	updateHealth(deviceID string, absoluteDevicePath string, healthy bool) (bool, error)         // Optional function to update the device health before it's sent via custom logic.

	getResourceName() string
	getDevices() []*pluginapi.Device
	getDevicePath() string
	getDeviceRoot() string
	getSocketPath() string
}

type DevicePluginBase struct {
	devs         []*pluginapi.Device
	socketPath   string
	resourceName string
	deviceRoot   string // Absolute base path for where this DP is inside virt-handler (typically intended to be either "/" or util.HostRootMount)
	devicePath   string // Device path on the host filesystem. When accessed from a virt-handler, it should be combined with deviceRoot.
}

// Optional, can be overridden
func (dpi *DevicePluginBase) setupDevicePlugin() error {
	return nil
}

// Optional, can be overridden
func (dpi *DevicePluginBase) deviceNameByID(deviceID string) string {
	return "device plugin (" + deviceID + ")"
}

// Optional, can be overridden
func (dpi *DevicePluginBase) configurePermissions(_ *safepath.Path) error {
	return nil
}

// Optional, can be overridden
func (dpi *DevicePluginBase) updateHealth(_ string, _ string, healthy bool) (bool, error) {
	return healthy, nil
}

func (dpi *DevicePluginBase) getResourceName() string {
	return dpi.resourceName
}

func (dpi *DevicePluginBase) getDevices() []*pluginapi.Device {
	return dpi.devs
}

func (dpi *DevicePluginBase) getDevicePath() string {
	return dpi.devicePath
}

func (dpi *DevicePluginBase) getDeviceRoot() string {
	return dpi.deviceRoot
}

func (dpi *DevicePluginBase) getSocketPath() string {
	return dpi.socketPath
}
