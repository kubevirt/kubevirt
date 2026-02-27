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
	"bufio"
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/fsnotify/fsnotify"
	"k8s.io/apimachinery/pkg/util/rand"
	v1 "kubevirt.io/api/core/v1"
	"kubevirt.io/client-go/log"

	"kubevirt.io/kubevirt/pkg/safepath"
	"kubevirt.io/kubevirt/pkg/util"
	pluginapi "kubevirt.io/kubevirt/pkg/virt-handler/device-manager/deviceplugin/v1beta1"
)

var (
	pathToUSBDevices = "/sys/bus/usb/devices"
)

var discoverLocalUSBDevicesFunc = discoverPluggedUSBDevices

// The sysfs metadata wrapper for the USB devices
type USBDevice struct {
	Name         string
	Manufacturer string
	Vendor       int
	Product      int
	BCD          int
	Bus          int
	DeviceNumber int
	Serial       string
	DevicePath   string
	Healthy      bool
}

// The uniqueness in the system comes from bus and device number but having the vendor:product
// information can help a lot. Not all usb devices provide or export a serial number.
func (dev *USBDevice) GetID() string {
	return fmt.Sprintf("%04x:%04x-%02d:%02d", dev.Vendor, dev.Product, dev.Bus, dev.DeviceNumber)
}

// The actual plugin
type USBDevicePlugin struct {
	*DevicePluginBase
	devices []*PluginDevices
	p       permissionManager
	logger  *log.FilteredLogger
}

type PluginDevices struct {
	ID      string
	Devices []*USBDevice
}

func newPluginDevices(resourceName string, index int, usbdevs []*USBDevice) *PluginDevices {
	return &PluginDevices{
		ID:      fmt.Sprintf("%s-%s-%d", resourceName, rand.String(4), index),
		Devices: usbdevs,
	}
}

func (pd *PluginDevices) toKubeVirtDevicePlugin() *pluginapi.Device {
	return &pluginapi.Device{
		ID:       pd.ID,
		Topology: nil,
		Health:   pluginapi.Unhealthy,
	}
}

func (plugin *USBDevicePlugin) FindDevice(pluginDeviceID string) *PluginDevices {
	for _, pd := range plugin.devices {
		if pd.ID == pluginDeviceID {
			return pd
		}
	}
	return nil
}

func devicesToKubeVirtDevicePlugin(pluginDevs []*PluginDevices) []*pluginapi.Device {
	devices := make([]*pluginapi.Device, 0, len(pluginDevs))
	for _, pluginDevices := range pluginDevs {
		devices = append(devices, pluginDevices.toKubeVirtDevicePlugin())
	}
	return devices
}

func (plugin *USBDevicePlugin) setupMonitoredDevicesFunc(watcher *fsnotify.Watcher, monitoredDevices map[string]string) error {
	watchedDirs := make(map[string]struct{})
	for _, pd := range plugin.devices {
		for _, usb := range pd.Devices {
			usbDevicePath := filepath.Join(plugin.deviceRoot, usb.DevicePath)
			usbDeviceParentPath := filepath.Dir(usbDevicePath)
			if _, exists := watchedDirs[usbDeviceParentPath]; !exists {
				if err := watcher.Add(usbDeviceParentPath); err != nil {
					return fmt.Errorf("failed to watch device %s's directory: %s", usbDevicePath, err)
				}
				watchedDirs[usbDeviceParentPath] = struct{}{}
				// e.g., watch /dev/bus/usb in case a bus dir is added/removed
				usbDeviceGrandParentPath := filepath.Dir(usbDeviceParentPath)
				if _, exists := watchedDirs[usbDeviceGrandParentPath]; !exists {
					if err := watcher.Add(usbDeviceGrandParentPath); err != nil {
						return fmt.Errorf("failed to watch device %s's super directory: %s", usbDevicePath, err)
					}
					watchedDirs[usbDeviceGrandParentPath] = struct{}{}
				}
			}

			monitoredDevices[usbDevicePath] = pd.ID
		}
	}
	return nil
}

func (plugin *USBDevicePlugin) mutateHealthUpdateFunc(deviceID string, devicePath string, healthy bool) (bool, error) {
	// a device is healthy when all devices in the usb device group are healthy
	pluginDevices := plugin.FindDevice(deviceID)
	if pluginDevices == nil {
		return false, fmt.Errorf("usb_device was unable to find a deviceID=%s corresponding to devicePath=%s", deviceID, devicePath)
	}
	for _, usbDev := range pluginDevices.Devices {
		expectedUsbDevicePath := filepath.Join(plugin.deviceRoot, usbDev.DevicePath)
		if devicePath == expectedUsbDevicePath {
			usbDev.Healthy = healthy
		}
	}
	// if any of the devices in the usb device group is unhealthy, the usb device group is unhealthy
	for _, usbDev := range pluginDevices.Devices {
		if !usbDev.Healthy {
			return false, nil
		}
	}
	return healthy, nil
}

// Interface to allocate requested Device, exported by ListAndWatch
func (plugin *USBDevicePlugin) allocateDPFunc(_ context.Context, allocRequest *pluginapi.AllocateRequest) (*pluginapi.AllocateResponse, error) {
	allocResponse := new(pluginapi.AllocateResponse)
	env := make(map[string]string)
	for _, request := range allocRequest.ContainerRequests {
		containerResponse := &pluginapi.ContainerAllocateResponse{}
		for _, id := range request.DevicesIDs {
			plugin.logger.V(2).Infof("usb device id: %s", id)

			pluginDevices := plugin.FindDevice(id)
			if pluginDevices == nil {
				plugin.logger.V(2).Infof("usb disappeared: %s", id)
				continue
			}

			deviceSpecs := []*pluginapi.DeviceSpec{}
			for _, dev := range pluginDevices.Devices {
				spath, err := safepath.JoinAndResolveWithRelativeRoot(plugin.deviceRoot, dev.DevicePath)
				if err != nil {
					return nil, fmt.Errorf("error opening the device %s: %v", dev.DevicePath, err)
				}
				if plugin.configurePermissions != nil {
					if err = plugin.configurePermissions(spath); err != nil {
						return nil, fmt.Errorf("error configuring the permission the device %s during allocation: %v", dev.DevicePath, err)
					}
				}
				// We might have more than one USB device per resource name
				key := util.ResourceNameToEnvVar(v1.USBResourcePrefix, plugin.resourceName)
				value := fmt.Sprintf("%d:%d", dev.Bus, dev.DeviceNumber)
				if previous, exist := env[key]; exist {
					env[key] = fmt.Sprintf("%s,%s", previous, value)
				} else {
					env[key] = value
				}

				deviceSpecs = append(deviceSpecs, &pluginapi.DeviceSpec{
					ContainerPath: dev.DevicePath,
					HostPath:      dev.DevicePath,
					Permissions:   "mrw",
				})
			}
			containerResponse.Envs = env
			containerResponse.Devices = append(containerResponse.Devices, deviceSpecs...)
		}
		allocResponse.ContainerResponses = append(allocResponse.ContainerResponses, containerResponse)
	}

	return allocResponse, nil
}

func parseSysUeventFile(path string) *USBDevice {
	// Grab all details we are interested from uevent
	file, err := os.Open(filepath.Join(path, "uevent"))
	if err != nil {
		log.Log.Reason(err).Infof("Unable to access %s/%s", path, "uevent")
		return nil
	}
	defer file.Close()

	u := USBDevice{Healthy: false}
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
		values := strings.Split(line, "=")
		if len(values) != 2 {
			log.Log.Infof("Skipping %s due not being key=value", line)
			continue
		}
		switch values[0] {
		case "BUSNUM":
			val, err := strconv.ParseInt(values[1], 10, 32)
			if err != nil {
				return nil
			}
			u.Bus = int(val)
		case "DEVNUM":
			val, err := strconv.ParseInt(values[1], 10, 32)
			if err != nil {
				return nil
			}
			u.DeviceNumber = int(val)
		case "PRODUCT":
			products := strings.Split(values[1], "/")
			if len(products) != 3 {
				return nil
			}

			val, err := strconv.ParseInt(products[0], 16, 32)
			if err != nil {
				return nil
			}
			u.Vendor = int(val)

			val, err = strconv.ParseInt(products[1], 16, 32)
			if err != nil {
				return nil
			}
			u.Product = int(val)

			val, err = strconv.ParseInt(products[2], 16, 32)
			if err != nil {
				return nil
			}
			u.BCD = int(val)
		case "DEVNAME":
			u.DevicePath = filepath.Join("/dev", values[1])
		default:
			log.Log.V(5).Infof("Skipping unhandled line: %s", line)
		}
	}
	return &u
}

type LocalDevices struct {
	// For quicker indexing, map devices based on vendor string
	devices map[int][]*USBDevice
}

// finds by vendor and product
func (l *LocalDevices) find(vendor, product int) *USBDevice {
	if devices, exist := l.devices[vendor]; exist {
		for _, local := range devices {
			if local.Product == product {
				return local
			}
		}
	}
	return nil
}

// remove all cached elements
func (l *LocalDevices) remove(usbdevs []*USBDevice) {
	for _, dev := range usbdevs {
		devices, exists := l.devices[dev.Vendor]
		if !exists {
			continue
		}

		for i, usb := range devices {
			if usb.GetID() == dev.GetID() {
				devices = append(devices[:i], devices[i+1:]...)
				break
			}
		}

		l.devices[dev.Vendor] = devices
		if len(devices) == 0 {
			delete(l.devices, dev.Vendor)
		}
	}
}

// return a list of USBDevices while removing it from the list of local devices
func (l *LocalDevices) fetch(selectors []v1.USBSelector) ([]*USBDevice, bool) {
	usbdevs := []*USBDevice{}

	// we have to find all devices under this resource name
	for _, selector := range selectors {
		vendor, product, err := parseSelector(&selector)
		if err != nil {
			log.Log.Reason(err).Warningf("Failed to convert selector: %+v", selector)
			return nil, false
		}

		local := l.find(vendor, product)
		if local == nil {
			return nil, false
		}

		usbdevs = append(usbdevs, local)
	}

	// To avoid mapping the same usb device to different k8s plugins
	l.remove(usbdevs)
	return usbdevs, true
}

func discoverPluggedUSBDevices() *LocalDevices {
	usbDevices := make(map[int][]*USBDevice)
	err := filepath.Walk(pathToUSBDevices, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		// Ignore named usb controllers
		if strings.HasPrefix(info.Name(), "usb") {
			return nil
		}
		// We are interested in actual USB devices information that
		// contains idVendor and idProduct. We can skip all others.
		if _, err := os.Stat(filepath.Join(path, "idVendor")); err != nil {
			return nil
		}

		// Get device information
		if device := parseSysUeventFile(path); device != nil {
			usbDevices[device.Vendor] = append(usbDevices[device.Vendor], device)
		}
		return nil
	})

	if err != nil {
		log.Log.Reason(err).Error("Failed when walking usb devices tree")
	}
	return &LocalDevices{devices: usbDevices}
}

func parseSelector(s *v1.USBSelector) (int, int, error) {
	val, err := strconv.ParseInt(s.Vendor, 16, 32)
	if err != nil {
		return -1, -1, err
	}
	vendor := int(val)

	val, err = strconv.ParseInt(s.Product, 16, 32)
	if err != nil {
		return -1, -1, err
	}
	product := int(val)

	return vendor, product, nil
}

func discoverAllowedUSBDevices(usbs []v1.USBHostDevice) map[string][]*PluginDevices {
	// The return value: USB Device Plugins found and permitted to be exposed
	plugins := make(map[string][]*PluginDevices)
	// All USB devices found plugged in the Node
	localDevices := discoverLocalUSBDevicesFunc()
	for _, usbConfig := range usbs {
		resourceName := usbConfig.ResourceName
		if usbConfig.ExternalResourceProvider {
			log.Log.V(6).Infof("Skipping discovery of %s. To be handled by external device-plugin",
				resourceName)
			continue
		}
		index := 0
		usbdevs, foundAll := localDevices.fetch(usbConfig.Selectors)
		for foundAll {
			// Create new USB Device Plugin with found USB Devices for this resource name
			pluginDevices := newPluginDevices(resourceName, index, usbdevs)
			plugins[resourceName] = append(plugins[resourceName], pluginDevices)
			index++
			usbdevs, foundAll = localDevices.fetch(usbConfig.Selectors)
		}
	}
	return plugins
}

func NewUSBDevicePlugin(resourceName string, deviceRoot string, pluginDevices []*PluginDevices, p permissionManager) *USBDevicePlugin {
	s := strings.Split(resourceName, "/")
	resourceID := s[0]
	if len(s) > 1 {
		resourceID = s[1]
	}
	resourceID = fmt.Sprintf("usb-%s", resourceID)
	devs := devicesToKubeVirtDevicePlugin(pluginDevices)
	usb := &USBDevicePlugin{
		DevicePluginBase: newDevicePluginBase(
			devs,
			resourceID,
			deviceRoot,
			pathToUSBDevices,
			resourceName,
		),
		devices: pluginDevices,
		p:       p,
		logger:  log.Log.With("subcomponent", resourceID),
	}
	usb.setupMonitoredDevices = usb.setupMonitoredDevicesFunc
	usb.deviceNameByID = usb.deviceNameByIDFunc
	// If permission manager is not provided, we assume that device doesn't need any permissions configured.
	if p != nil {
		usb.configurePermissions = func(dp *safepath.Path) error {
			err := usb.p.ChownAtNoFollow(dp, util.NonRootUID, util.NonRootUID)
			if err != nil {
				return fmt.Errorf("error setting the ownership of the device: %v", err)
			}
			return nil
		}
	}
	usb.allocateDP = usb.allocateDPFunc
	usb.mutateHealthUpdate = usb.mutateHealthUpdateFunc
	return usb
}

func (plugin *USBDevicePlugin) deviceNameByIDFunc(devGroupID string) string {
	return fmt.Sprintf("USB device group (%s)", devGroupID)
}
