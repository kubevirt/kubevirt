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
 * Copyright 2020 Red Hat, Inc.
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
	MDEV_RESOURCE_PREFIX = "MDEV_PCI_RESOURCE"
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
	devs           []*pluginapi.Device
	server         *grpc.Server
	socketPath     string
	stop           chan struct{}
	health         chan string
	devicePath     string
	deviceName     string
	resourceName   string
	done           chan struct{}
	deviceRoot     string
	healthy        chan string
	unhealthy      chan string
	iommuToMDEVMap map[string]string
	initialized    bool
	lock           *sync.Mutex
}

func NewMediatedDevicePlugin(mdevs []*MDEV, resourceName string) *MediatedDevicePlugin {
	s := strings.Split(resourceName, "/")
	mdevTypeName := s[1]
	serverSock := SocketPath(mdevTypeName)
	iommuToMDEVMap := make(map[string]string)

	initHandler()

	devs := constructDPIdevicesFromMdev(mdevs, iommuToMDEVMap)
	dpi := &MediatedDevicePlugin{
		devs:           devs,
		socketPath:     serverSock,
		health:         make(chan string),
		deviceName:     resourceName,
		resourceName:   resourceName,
		devicePath:     vfioDevicePath,
		deviceRoot:     util.HostRootMount,
		healthy:        make(chan string),
		unhealthy:      make(chan string),
		iommuToMDEVMap: iommuToMDEVMap,
		initialized:    false,
		lock:           &sync.Mutex{},
	}

	return dpi
}

func constructDPIdevicesFromMdev(mdevs []*MDEV, iommuToMDEVMap map[string]string) (devs []*pluginapi.Device) {
	for _, mdev := range mdevs {
		iommuToMDEVMap[mdev.iommuGroup] = mdev.UUID
		dpiDev := &pluginapi.Device{
			ID:     string(mdev.iommuGroup),
			Health: pluginapi.Healthy,
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

// Start starts the device plugin
func (dpi *MediatedDevicePlugin) Start(stop chan struct{}) (err error) {
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

func (dpi *MediatedDevicePlugin) GetDevicePath() string {
	return dpi.devicePath
}

func (dpi *MediatedDevicePlugin) GetDeviceName() string {
	return dpi.deviceName
}

func (dpi *MediatedDevicePlugin) Allocate(_ context.Context, r *pluginapi.AllocateRequest) (*pluginapi.AllocateResponse, error) {
	resourceName := dpi.deviceName
	log.DefaultLogger().Infof("Allocate: resourceName: %s", dpi.deviceName)
	log.DefaultLogger().Infof("Allocate: iommuMap: %v", dpi.iommuToMDEVMap)
	resourceNameEnvVar := util.ResourceNameToEnvVar(MDEV_RESOURCE_PREFIX, resourceName)
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
				formattedVFIO := formatVFIODeviceSpecs(devID)
				log.DefaultLogger().Infof("Allocate: formatted vfio: %v", formattedVFIO)
				deviceSpecs = append(deviceSpecs, formatVFIODeviceSpecs(devID)...)
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
			return resp, fmt.Errorf("failed to allocate resource for resourceName: %s", resourceName)
		}
	}
	return resp, nil
}

// Stop stops the gRPC server
func (dpi *MediatedDevicePlugin) Stop() error {
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
func (dpi *MediatedDevicePlugin) Register() error {
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

func (dpi *MediatedDevicePlugin) ListAndWatch(_ *pluginapi.Empty, s pluginapi.DevicePlugin_ListAndWatchServer) error {
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

func (dpi *MediatedDevicePlugin) cleanup() error {
	if err := os.Remove(dpi.socketPath); err != nil && !os.IsNotExist(err) {
		return err
	}

	return nil
}

func (dpi *MediatedDevicePlugin) GetDevicePluginOptions(_ context.Context, _ *pluginapi.Empty) (*pluginapi.DevicePluginOptions, error) {
	options := &pluginapi.DevicePluginOptions{
		PreStartRequired: false,
	}
	return options, nil
}

func (dpi *MediatedDevicePlugin) PreStartContainer(_ context.Context, _ *pluginapi.PreStartContainerRequest) (*pluginapi.PreStartContainerResponse, error) {
	res := &pluginapi.PreStartContainerResponse{}
	return res, nil
}

func discoverPermittedHostMediatedDevices(supportedMdevsMap map[string]string) map[string][]*MDEV {
	initHandler()

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
			parentPCIAddr, err := Handler.GetMdevParentPCIAddr(info.Name())
			if err != nil {
				log.DefaultLogger().Reason(err).Errorf("failed parent PCI address for mdev: %s", info.Name())
				continue
			}
			mdev.parentPciAddress = parentPCIAddr

			mdev.numaNode = Handler.GetDeviceNumaNode(pciBasePath, parentPCIAddr)
			iommuGroup, err := Handler.GetDeviceIOMMUGroup(mdevBasePath, info.Name())
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

func (dpi *MediatedDevicePlugin) healthCheck() error {
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

func getMdevTypeName(mdevUUID string) (string, error) {
	// #nosec No risk for path injection. Path is composed from static base  "mdevBasePath" and static components
	rawName, err := os.ReadFile(filepath.Join(mdevBasePath, mdevUUID, "mdev_type/name"))
	if err != nil {
		if os.IsNotExist(err) {
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

func (dpi *MediatedDevicePlugin) GetInitialized() bool {
	dpi.lock.Lock()
	defer dpi.lock.Unlock()
	return dpi.initialized
}

func (dpi *MediatedDevicePlugin) setInitialized(initialized bool) {
	dpi.lock.Lock()
	dpi.initialized = initialized
	dpi.lock.Unlock()
}
