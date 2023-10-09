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
 * Copyright 2023 Red Hat, Inc.
 *
 */

package device_manager

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"net"
	"os"
	"path"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/fsnotify/fsnotify"
	"google.golang.org/grpc"
	"k8s.io/apimachinery/pkg/util/rand"
	v1 "kubevirt.io/api/core/v1"
	"kubevirt.io/client-go/log"

	"kubevirt.io/kubevirt/pkg/safepath"
	"kubevirt.io/kubevirt/pkg/util"
	pluginapi "kubevirt.io/kubevirt/pkg/virt-handler/device-manager/deviceplugin/v1beta1"
)

const (
	PathToUSBDevices = "/sys/bus/usb/devices"
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
}

// The uniqueness in the system comes from bus and device number but having the vendor:product
// information can help a lot. Not all usb devices provide or export a serial number.
func (dev *USBDevice) GetID() string {
	return fmt.Sprintf("%04x:%04x-%02d:%02d", dev.Vendor, dev.Product, dev.Bus, dev.DeviceNumber)
}

// The actual plugin
type USBDevicePlugin struct {
	socketPath   string
	stop         <-chan struct{}
	update       chan struct{}
	done         chan struct{}
	deregistered chan struct{}
	server       *grpc.Server
	resourceName string
	devices      []*PluginDevices
	logger       *log.FilteredLogger

	initialized bool
	lock        *sync.Mutex
}

type PluginDevices struct {
	ID        string
	isHealthy bool
	Devices   []*USBDevice
}

func newPluginDevices(resourceName string, index int, usbdevs []*USBDevice) *PluginDevices {
	return &PluginDevices{
		ID:        fmt.Sprintf("%s-%s-%d", resourceName, rand.String(4), index),
		isHealthy: true,
		Devices:   usbdevs,
	}
}

func (pd *PluginDevices) toKubeVirtDevicePlugin() *pluginapi.Device {
	healthStr := pluginapi.Healthy
	if !pd.isHealthy {
		healthStr = pluginapi.Unhealthy
	}
	return &pluginapi.Device{
		ID:       pd.ID,
		Health:   healthStr,
		Topology: nil,
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

func (plugin *USBDevicePlugin) FindDeviceByUSBID(usbID string) *PluginDevices {
	for _, pd := range plugin.devices {
		for _, usb := range pd.Devices {
			if usb.GetID() == usbID {
				return pd
			}
		}
	}
	return nil
}

func (plugin *USBDevicePlugin) setDeviceHealth(usbID string, isHealthy bool) {
	pd := plugin.FindDeviceByUSBID(usbID)
	isDifferent := pd.isHealthy != isHealthy
	pd.isHealthy = isHealthy
	if isDifferent {
		plugin.update <- struct{}{}
	}
}

func (plugin *USBDevicePlugin) devicesToKubeVirtDevicePlugin() []*pluginapi.Device {
	devices := make([]*pluginapi.Device, 0, len(plugin.devices))
	for _, pluginDevices := range plugin.devices {
		devices = append(devices, pluginDevices.toKubeVirtDevicePlugin())
	}
	return devices
}

func (plugin *USBDevicePlugin) GetInitialized() bool {
	plugin.lock.Lock()
	defer plugin.lock.Unlock()
	return plugin.initialized
}

func (plugin *USBDevicePlugin) setInitialized(initialized bool) {
	plugin.lock.Lock()
	plugin.initialized = initialized
	plugin.lock.Unlock()
}

func (plugin *USBDevicePlugin) GetDeviceName() string {
	return plugin.resourceName
}

func (plugin *USBDevicePlugin) stopDevicePlugin() error {
	defer func() {
		select {
		case <-plugin.done:
			return
		default:
			close(plugin.done)
		}
	}()

	// Give the device plugin one second to properly deregister
	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()
	select {
	case <-plugin.deregistered:
	case <-ticker.C:
	}

	plugin.server.Stop()
	plugin.setInitialized(false)
	return plugin.cleanup()
}

func (plugin *USBDevicePlugin) Start(stop <-chan struct{}) error {
	plugin.stop = stop
	plugin.done = make(chan struct{})
	plugin.deregistered = make(chan struct{})

	err := plugin.cleanup()
	if err != nil {
		return fmt.Errorf("error on cleanup: %v", err)
	}

	sock, err := net.Listen("unix", plugin.socketPath)
	if err != nil {
		return fmt.Errorf("error creating GRPC server socket: %v", err)
	}

	plugin.server = grpc.NewServer([]grpc.ServerOption{}...)
	defer plugin.stopDevicePlugin()

	pluginapi.RegisterDevicePluginServer(plugin.server, plugin)

	errChan := make(chan error, 2)

	go func() {
		errChan <- plugin.server.Serve(sock)
	}()

	err = waitForGRPCServer(plugin.socketPath, 5*time.Second)
	if err != nil {
		return fmt.Errorf("error starting the GRPC server: %v", err)
	}

	err = plugin.register()
	if err != nil {
		return fmt.Errorf("error registering with device plugin manager: %v", err)
	}

	go func() {
		errChan <- plugin.healthCheck()
	}()

	plugin.setInitialized(true)
	plugin.logger.Infof("%s device plugin started", plugin.resourceName)
	err = <-errChan

	return err
}

func (plugin *USBDevicePlugin) healthCheck() error {
	monitoredDevices := make(map[string]string)
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return fmt.Errorf("failed to creating a fsnotify watcher: %v", err)
	}
	defer watcher.Close()

	watchedDirs := make(map[string]struct{})
	for _, pd := range plugin.devices {
		for _, usb := range pd.Devices {
			usbDevicePath := filepath.Join(util.HostRootMount, usb.DevicePath)
			usbDeviceDirPath := filepath.Dir(usbDevicePath)
			if _, exists := watchedDirs[usbDeviceDirPath]; !exists {
				if err := watcher.Add(usbDeviceDirPath); err != nil {
					return fmt.Errorf("failed to watch device %s parent directory: %s", usbDevicePath, err)
				}
				watchedDirs[usbDeviceDirPath] = struct{}{}
			}

			if err := watcher.Add(usbDevicePath); err != nil {
				return fmt.Errorf("failed to add the device %s to the watcher: %s", usbDevicePath, err)
			} else if _, err := os.Stat(usbDevicePath); err != nil {
				return fmt.Errorf("failed to validate device %s: %s", usbDevicePath, err)
			}
			monitoredDevices[usbDevicePath] = usb.GetID()
		}
	}

	dirName := filepath.Dir(plugin.socketPath)
	if err := watcher.Add(dirName); err != nil {
		return fmt.Errorf("failed to add the device-plugin kubelet path to the watcher: %v", err)
	} else if _, err = os.Stat(plugin.socketPath); err != nil {
		return fmt.Errorf("failed to stat the device-plugin socket: %v", err)
	}

	for {
		select {
		case <-plugin.stop:
			return nil
		case err := <-watcher.Errors:
			plugin.logger.Reason(err).Errorf("error watching devices and device plugin directory")
		case event := <-watcher.Events:
			plugin.logger.V(2).Infof("health Event: %v", event)
			if id, exist := monitoredDevices[event.Name]; exist {
				// Health in this case is if the device path actually exists
				if event.Op == fsnotify.Create {
					plugin.logger.Infof("monitored device %s appeared", plugin.resourceName)
					plugin.setDeviceHealth(id, true)
				} else if (event.Op == fsnotify.Remove) || (event.Op == fsnotify.Rename) {
					plugin.logger.Infof("monitored device %s disappeared", plugin.resourceName)
					plugin.setDeviceHealth(id, false)
				}
			} else if event.Name == plugin.socketPath && event.Op == fsnotify.Remove {
				plugin.logger.Infof("device socket file for device %s was removed, kubelet probably restarted.", plugin.resourceName)
				return nil
			}
		}
	}
}

func (plugin *USBDevicePlugin) cleanup() error {
	err := os.Remove(plugin.socketPath)
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		return err
	}
	return nil
}

func (plugin *USBDevicePlugin) register() error {
	conn, err := grpc.Dial(pluginapi.KubeletSocket,
		grpc.WithInsecure(),
		grpc.WithBlock(),
		grpc.WithTimeout(5*time.Second),
		grpc.WithDialer(func(addr string, timeout time.Duration) (net.Conn, error) {
			return net.DialTimeout("unix", addr, timeout)
		}),
	)
	if err != nil {
		return err
	}
	defer conn.Close()

	client := pluginapi.NewRegistrationClient(conn)
	reqt := &pluginapi.RegisterRequest{
		Version:      pluginapi.Version,
		Endpoint:     path.Base(plugin.socketPath),
		ResourceName: plugin.GetDeviceName(),
	}

	_, err = client.Register(context.Background(), reqt)
	if err != nil {
		return err
	}
	return nil
}

func (plugin *USBDevicePlugin) GetDevicePluginOptions(ctx context.Context, _ *pluginapi.Empty) (*pluginapi.DevicePluginOptions, error) {
	return &pluginapi.DevicePluginOptions{
		PreStartRequired: false,
	}, nil
}

// Interface to expose Devices: IDs, health and Topology
func (plugin *USBDevicePlugin) ListAndWatch(_ *pluginapi.Empty, lws pluginapi.DevicePlugin_ListAndWatchServer) error {
	sendUpdate := func(devices []*pluginapi.Device) error {
		response := pluginapi.ListAndWatchResponse{
			Devices: devices,
		}
		err := lws.Send(&response)
		if err != nil {
			plugin.logger.Reason(err).Warningf("Failed to send device plugin %s",
				plugin.resourceName)
		}
		return err
	}

	if err := sendUpdate(plugin.devicesToKubeVirtDevicePlugin()); err != nil {
		return err
	}
	done := false
	for !done {
		select {
		case <-plugin.update:
			if err := sendUpdate(plugin.devicesToKubeVirtDevicePlugin()); err != nil {
				return err
			}
		case <-plugin.stop:
			done = true
		}
	}

	if err := sendUpdate([]*pluginapi.Device{}); err != nil {
		plugin.logger.Reason(err).Warningf("Failed to deregister device plugin %s",
			plugin.resourceName)
	}
	close(plugin.deregistered)
	return nil
}

// Interface to allocate requested Device, exported by ListAndWatch
func (plugin *USBDevicePlugin) Allocate(_ context.Context, allocRequest *pluginapi.AllocateRequest) (*pluginapi.AllocateResponse, error) {
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
				spath, err := safepath.JoinAndResolveWithRelativeRoot(dev.DevicePath)
				if err != nil {
					return nil, fmt.Errorf("error opening the socket %s: %v", dev.DevicePath, err)
				}

				err = safepath.ChownAtNoFollow(spath, util.NonRootUID, util.NonRootUID)
				if err != nil {
					return nil, fmt.Errorf("error setting the permission the socket %s: %v", dev.DevicePath, err)
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

func (plugin *USBDevicePlugin) PreStartContainer(context.Context, *pluginapi.PreStartContainerRequest) (*pluginapi.PreStartContainerResponse, error) {
	return &pluginapi.PreStartContainerResponse{}, nil
}

func parseSysUeventFile(path string) *USBDevice {
	// Grab all details we are interested from uevent
	file, err := os.Open(filepath.Join(path, "uevent"))
	if err != nil {
		log.Log.Reason(err).Infof("Unable to access %s/%s", path, "uevent")
		return nil
	}
	defer file.Close()

	u := USBDevice{}
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
	usbDevices := make(map[int][]*USBDevice, 0)
	err := filepath.Walk(PathToUSBDevices, func(path string, info os.FileInfo, err error) error {
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

func NewUSBDevicePlugin(resourceName string, pluginDevices []*PluginDevices) *USBDevicePlugin {
	s := strings.Split(resourceName, "/")
	resourceID := s[0]
	if len(s) > 1 {
		resourceID = s[1]
	}
	resourceID = fmt.Sprintf("usb-%s", resourceID)
	return &USBDevicePlugin{
		socketPath:   SocketPath(resourceID),
		resourceName: resourceName,
		devices:      pluginDevices,
		logger:       log.Log.With("subcomponent", resourceID),
		initialized:  false,
		lock:         &sync.Mutex{},
	}
}
