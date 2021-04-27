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
 * Copyright 2018 Red Hat, Inc.
 *
 */

package device_manager

import (
	"fmt"
	"net"
	"os"
	"path"
	"path/filepath"
	"strings"
	"sync"

	"github.com/fsnotify/fsnotify"
	"golang.org/x/net/context"
	"google.golang.org/grpc"

	"kubevirt.io/client-go/log"
	"kubevirt.io/kubevirt/pkg/util"
	pluginapi "kubevirt.io/kubevirt/pkg/virt-handler/device-manager/deviceplugin/v1beta1"
)

const (
	vfioDevicePath      = "/dev/vfio/"
	vfioMount           = "/dev/vfio/vfio"
	pciBasePath         = "/sys/bus/pci/devices"
	PCI_RESOURCE_PREFIX = "PCI_RESOURCE"
)

type PCIDevice struct {
	pciID      string
	driver     string
	pciAddress string
	iommuGroup string
	numaNode   int
}

type PCIDevicePlugin struct {
	devs          []*pluginapi.Device
	server        *grpc.Server
	socketPath    string
	stop          chan struct{}
	health        chan string
	devicePath    string
	deviceName    string
	resourceName  string
	done          chan struct{}
	deviceRoot    string
	healthy       chan string
	unhealthy     chan string
	iommuToPCIMap map[string]string
	initialized   bool
	lock          *sync.Mutex
}

func NewPCIDevicePlugin(pciDevices []*PCIDevice, resourceName string) *PCIDevicePlugin {
	deviceIDStr := strings.Replace(pciDevices[0].pciID, ":", "-", -1)
	serverSock := SocketPath(deviceIDStr)
	iommuToPCIMap := make(map[string]string)

	initHandler()

	devs := constructDPIdevices(pciDevices, iommuToPCIMap)
	dpi := &PCIDevicePlugin{
		devs:          devs,
		socketPath:    serverSock,
		deviceName:    resourceName,
		resourceName:  resourceName,
		devicePath:    vfioDevicePath,
		deviceRoot:    util.HostRootMount,
		iommuToPCIMap: iommuToPCIMap,
		healthy:       make(chan string),
		unhealthy:     make(chan string),
		initialized:   false,
		lock:          &sync.Mutex{},
	}
	return dpi
}

func constructDPIdevices(pciDevices []*PCIDevice, iommuToPCIMap map[string]string) (devs []*pluginapi.Device) {
	for _, pciDevice := range pciDevices {
		iommuToPCIMap[pciDevice.iommuGroup] = pciDevice.pciAddress
		dpiDev := &pluginapi.Device{
			ID:     string(pciDevice.iommuGroup),
			Health: pluginapi.Healthy,
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

// Start starts the device plugin
func (dpi *PCIDevicePlugin) Start(stop chan struct{}) (err error) {
	logger := log.DefaultLogger()
	dpi.stop = stop
	dpi.done = make(chan struct{})

	err = dpi.cleanup()
	if err != nil {
		return err
	}

	sock, err := net.Listen("unix", dpi.socketPath)
	if err != nil {
		return fmt.Errorf("error creating GRPC server socket: %v", err)
	}

	dpi.server = grpc.NewServer([]grpc.ServerOption{}...)
	defer dpi.Stop()

	pluginapi.RegisterDevicePluginServer(dpi.server, dpi)
	err = dpi.Register()
	if err != nil {
		return fmt.Errorf("error registering with device plugin manager: %v", err)
	}

	errChan := make(chan error, 2)

	go func() {
		errChan <- dpi.server.Serve(sock)
	}()

	err = waitForGrpcServer(dpi.socketPath, connectionTimeout)
	if err != nil {
		return fmt.Errorf("error starting the GRPC server: %v", err)
	}

	go func() {
		errChan <- dpi.healthCheck()
	}()

	dpi.setInitialized(true)
	logger.Infof("%s device plugin started", dpi.deviceName)
	err = <-errChan

	return err
}

func (dpi *PCIDevicePlugin) ListAndWatch(_ *pluginapi.Empty, s pluginapi.DevicePlugin_ListAndWatchServer) error {
	// FIXME: sending an empty list up front should not be needed. This is a workaround for:
	// https://github.com/kubevirt/kubevirt/issues/1196
	// This can safely be removed once supported upstream Kubernetes is 1.10.3 or higher.
	emptyList := []*pluginapi.Device{}
	s.Send(&pluginapi.ListAndWatchResponse{Devices: emptyList})

	s.Send(&pluginapi.ListAndWatchResponse{Devices: dpi.devs})

	for {
		select {
		case unhealthy := <-dpi.unhealthy:
			for _, dev := range dpi.devs {
				if unhealthy == dev.ID {
					dev.Health = pluginapi.Unhealthy
				}
			}
			s.Send(&pluginapi.ListAndWatchResponse{Devices: dpi.devs})
		case healthy := <-dpi.healthy:
			for _, dev := range dpi.devs {
				if healthy == dev.ID {
					dev.Health = pluginapi.Healthy
				}
			}
			s.Send(&pluginapi.ListAndWatchResponse{Devices: dpi.devs})
		case <-dpi.stop:
			return nil
		case <-dpi.done:
			return nil
		}
	}
}

func formatVFIODeviceSpecs(devID string) []*pluginapi.DeviceSpec {
	// always add /dev/vfio/vfio device as well
	devSpecs := make([]*pluginapi.DeviceSpec, 0)
	devSpecs = append(devSpecs, &pluginapi.DeviceSpec{
		HostPath:      vfioMount,
		ContainerPath: vfioMount,
		Permissions:   "mrw",
	})

	vfioDevice := filepath.Join(vfioDevicePath, devID)
	devSpecs = append(devSpecs, &pluginapi.DeviceSpec{
		HostPath:      vfioDevice,
		ContainerPath: vfioDevice,
		Permissions:   "mrw",
	})
	return devSpecs
}

func (dpi *PCIDevicePlugin) Allocate(_ context.Context, r *pluginapi.AllocateRequest) (*pluginapi.AllocateResponse, error) {
	resourceName := dpi.deviceName
	resourceNameEnvVar := util.ResourceNameToEnvVar(PCI_RESOURCE_PREFIX, resourceName)
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

func (dpi *PCIDevicePlugin) healthCheck() error {
	logger := log.DefaultLogger()
	monitoredDevices := make(map[string]string)
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return fmt.Errorf("failed to creating a fsnotify watcher: %v", err)
	}
	defer watcher.Close()

	// This way we don't have to mount /dev from the node
	devicePath := filepath.Join(dpi.deviceRoot, dpi.devicePath)

	// Start watching the files before we check for their existence to avoid races
	dirName := filepath.Dir(devicePath)
	err = watcher.Add(dirName)
	if err != nil {
		return fmt.Errorf("failed to add the device root path to the watcher: %v", err)
	}

	_, err = os.Stat(devicePath)
	if err != nil {
		if !os.IsNotExist(err) {
			return fmt.Errorf("could not stat the device: %v", err)
		}
	}

	// probe all devices
	for _, dev := range dpi.devs {
		vfioDevice := filepath.Join(devicePath, dev.ID)
		err = watcher.Add(vfioDevice)
		if err != nil {
			return fmt.Errorf("failed to add the device %s to the watcher: %v", vfioDevice, err)
		}
		monitoredDevices[vfioDevice] = dev.ID
	}

	dirName = filepath.Dir(dpi.socketPath)
	err = watcher.Add(dirName)

	if err != nil {
		return fmt.Errorf("failed to add the device-plugin kubelet path to the watcher: %v", err)
	}
	_, err = os.Stat(dpi.socketPath)
	if err != nil {
		return fmt.Errorf("failed to stat the device-plugin socket: %v", err)
	}

	for {
		select {
		case <-dpi.stop:
			return nil
		case err := <-watcher.Errors:
			logger.Reason(err).Errorf("error watching devices and device plugin directory")
		case event := <-watcher.Events:
			logger.V(4).Infof("health Event: %v", event)
			if monDevId, exist := monitoredDevices[event.Name]; exist {
				// Health in this case is if the device path actually exists
				if event.Op == fsnotify.Create {
					logger.Infof("monitored device %s appeared", dpi.deviceName)
					dpi.healthy <- monDevId
				} else if (event.Op == fsnotify.Remove) || (event.Op == fsnotify.Rename) {
					logger.Infof("monitored device %s disappeared", dpi.deviceName)
					dpi.unhealthy <- monDevId
				}
			} else if event.Name == dpi.socketPath && event.Op == fsnotify.Remove {
				logger.Infof("device socket file for device %s was removed, kubelet probably restarted.", dpi.deviceName)
				return nil
			}
		}
	}
}

func (dpi *PCIDevicePlugin) GetDevicePath() string {
	return dpi.devicePath
}

func (dpi *PCIDevicePlugin) GetDeviceName() string {
	return dpi.deviceName
}

// Stop stops the gRPC server
func (dpi *PCIDevicePlugin) Stop() error {
	defer func() {
		if !IsChanClosed(dpi.done) {
			close(dpi.done)
		}
	}()
	dpi.server.Stop()
	dpi.setInitialized(false)
	return dpi.cleanup()
}

// Register registers the device plugin for the given resourceName with Kubelet.
func (dpi *PCIDevicePlugin) Register() error {
	conn, err := connect(pluginapi.KubeletSocket, connectionTimeout)
	if err != nil {
		return err
	}
	defer conn.Close()

	client := pluginapi.NewRegistrationClient(conn)
	reqt := &pluginapi.RegisterRequest{
		Version:      pluginapi.Version,
		Endpoint:     path.Base(dpi.socketPath),
		ResourceName: dpi.resourceName,
	}

	_, err = client.Register(context.Background(), reqt)
	if err != nil {
		return err
	}
	return nil
}

func (dpi *PCIDevicePlugin) cleanup() error {
	if err := os.Remove(dpi.socketPath); err != nil && !os.IsNotExist(err) {
		return err
	}

	return nil
}

func (dpi *PCIDevicePlugin) GetDevicePluginOptions(_ context.Context, _ *pluginapi.Empty) (*pluginapi.DevicePluginOptions, error) {
	options := &pluginapi.DevicePluginOptions{
		PreStartRequired: false,
	}
	return options, nil
}

func (dpi *PCIDevicePlugin) PreStartContainer(_ context.Context, _ *pluginapi.PreStartContainerRequest) (*pluginapi.PreStartContainerResponse, error) {
	res := &pluginapi.PreStartContainerResponse{}
	return res, nil
}

func discoverPermittedHostPCIDevices(supportedPCIDeviceMap map[string]string) map[string][]*PCIDevice {
	initHandler()

	pciDevicesMap := make(map[string][]*PCIDevice)
	err := filepath.Walk(pciBasePath, func(path string, info os.FileInfo, err error) error {
		if info.IsDir() {
			return nil
		}
		pciID, err := Handler.GetDevicePCIID(pciBasePath, info.Name())
		if err != nil {
			log.DefaultLogger().Reason(err).Errorf("failed get vendor:device ID for device: %s", info.Name())
			return nil
		}
		if _, supported := supportedPCIDeviceMap[pciID]; supported {
			// check device driver
			driver, err := Handler.GetDeviceDriver(pciBasePath, info.Name())
			if err != nil || driver != "vfio-pci" {
				return nil
			}

			pcidev := &PCIDevice{
				pciID:      pciID,
				pciAddress: info.Name(),
			}
			iommuGroup, err := Handler.GetDeviceIOMMUGroup(pciBasePath, info.Name())
			if err != nil {
				return nil
			}
			pcidev.iommuGroup = iommuGroup
			pcidev.driver = driver
			pcidev.numaNode = Handler.GetDeviceNumaNode(pciBasePath, info.Name())
			pciDevicesMap[pciID] = append(pciDevicesMap[pciID], pcidev)
		}
		return nil
	})
	if err != nil {
		log.DefaultLogger().Reason(err).Errorf("failed to discover host devices")
	}
	return pciDevicesMap
}

func (dpi *PCIDevicePlugin) GetInitialized() bool {
	dpi.lock.Lock()
	defer dpi.lock.Unlock()
	return dpi.initialized
}

func (dpi *PCIDevicePlugin) setInitialized(initialized bool) {
	dpi.lock.Lock()
	dpi.initialized = initialized
	dpi.lock.Unlock()
}
