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
	"errors"
	"fmt"
	"net"
	"os"
	"path"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"context"

	"github.com/fsnotify/fsnotify"
	"google.golang.org/grpc"

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
	devs           []*pluginapi.Device
	server         *grpc.Server
	socketPath     string
	stop           <-chan struct{}
	health         chan deviceHealth
	devicePath     string
	resourceName   string
	done           chan struct{}
	deviceRoot     string
	iommuToMDEVMap map[string]string
	initialized    bool
	lock           *sync.Mutex
	deregistered   chan struct{}
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
		health:         make(chan deviceHealth),
		resourceName:   resourceName,
		devicePath:     vfioDevicePath,
		deviceRoot:     util.HostRootMount,
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
func (dpi *MediatedDevicePlugin) Start(stop <-chan struct{}) (err error) {
	logger := log.DefaultLogger()
	dpi.stop = stop
	dpi.done = make(chan struct{})
	dpi.deregistered = make(chan struct{})

	err = dpi.cleanup()
	if err != nil {
		return err
	}

	sock, err := net.Listen("unix", dpi.socketPath)
	if err != nil {
		return fmt.Errorf("error creating GRPC server socket: %v", err)
	}

	dpi.server = grpc.NewServer([]grpc.ServerOption{}...)
	defer dpi.stopDevicePlugin()

	pluginapi.RegisterDevicePluginServer(dpi.server, dpi)

	errChan := make(chan error, 2)

	go func() {
		errChan <- dpi.server.Serve(sock)
	}()

	err = waitForGRPCServer(dpi.socketPath, connectionTimeout)
	if err != nil {
		return fmt.Errorf("error starting the GRPC server: %v", err)
	}

	err = dpi.register()
	if err != nil {
		return fmt.Errorf("error registering with device plugin manager: %v", err)
	}

	go func() {
		errChan <- dpi.healthCheck()
	}()

	dpi.setInitialized(true)
	logger.Infof("%s device plugin started", dpi.resourceName)
	err = <-errChan

	return err
}

func (dpi *MediatedDevicePlugin) GetDeviceName() string {
	return dpi.resourceName
}

func (dpi *MediatedDevicePlugin) Allocate(_ context.Context, r *pluginapi.AllocateRequest) (*pluginapi.AllocateResponse, error) {
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
					return resp, fmt.Errorf("Failed to allocate resource %s", dpi.resourceName)
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

// Stop stops the gRPC server
func (dpi *MediatedDevicePlugin) stopDevicePlugin() error {
	defer func() {
		if !IsChanClosed(dpi.done) {
			close(dpi.done)
		}
	}()

	// Give the device plugin one second to properly deregister
	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()
	select {
	case <-dpi.deregistered:
	case <-ticker.C:
	}

	dpi.server.Stop()
	dpi.setInitialized(false)
	return dpi.cleanup()
}

// Register registers the device plugin for the given resourceName with Kubelet.
func (dpi *MediatedDevicePlugin) register() error {
	conn, err := gRPCConnect(pluginapi.KubeletSocket, connectionTimeout)
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
	s.Send(&pluginapi.ListAndWatchResponse{Devices: dpi.devs})

	done := false
	for {
		select {
		case devHealth := <-dpi.health:
			for _, dev := range dpi.devs {
				if devHealth.DevId == dev.ID {
					dev.Health = devHealth.Health
				}
			}
			s.Send(&pluginapi.ListAndWatchResponse{Devices: dpi.devs})
		case <-dpi.stop:
			done = true
		case <-dpi.done:
			done = true
		}
		if done {
			break
		}
	}
	// Send empty list to increase the chance that the kubelet acts fast on stopped device plugins
	// There exists no explicit way to deregister devices
	emptyList := []*pluginapi.Device{}
	if err := s.Send(&pluginapi.ListAndWatchResponse{Devices: emptyList}); err != nil {
		log.DefaultLogger().Reason(err).Infof("%s device plugin failed to deregister", dpi.resourceName)
	}
	close(dpi.deregistered)
	return nil
}

func (dpi *MediatedDevicePlugin) cleanup() error {
	if err := os.Remove(dpi.socketPath); err != nil && !errors.Is(err, os.ErrNotExist) {
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
		if !errors.Is(err, os.ErrNotExist) {
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
					logger.Infof("monitored device %s appeared", dpi.resourceName)
					dpi.health <- deviceHealth{
						DevId:  monDevId,
						Health: pluginapi.Healthy,
					}
				} else if (event.Op == fsnotify.Remove) || (event.Op == fsnotify.Rename) {
					mdev, ok := dpi.iommuToMDEVMap[monDevId]
					if !ok {
						mdev = " not recognized"
					}

					if event.Op == fsnotify.Rename {
						logger.Infof("Mediated device %s with id %s for resource %s was renamed", mdev, monDevId, dpi.resourceName)
					} else {
						logger.Infof("Mediated device %s with id %s for resource %s disappeared", mdev, monDevId, dpi.resourceName)
					}

					dpi.health <- deviceHealth{
						DevId:  monDevId,
						Health: pluginapi.Unhealthy,
					}
				}
			} else if event.Name == dpi.socketPath && event.Op == fsnotify.Remove {
				logger.Infof("device socket file for device %s was removed, kubelet probably restarted.", dpi.resourceName)
				return nil
			}
		}
	}
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
