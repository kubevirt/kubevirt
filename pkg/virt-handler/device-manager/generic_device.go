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
	"strconv"
	"time"

	"github.com/fsnotify/fsnotify"
	"golang.org/x/net/context"
	"google.golang.org/grpc"

	"kubevirt.io/kubevirt/pkg/log"
	pluginapi "kubevirt.io/kubevirt/pkg/virt-handler/device-manager/deviceplugin/v1beta1"
)

const (
	DeviceNamespace   = "devices.kubevirt.io"
	connectionTimeout = 5 * time.Second
)

type GenericDevice interface {
	Start(chan struct{}) error
	GetDevicePath() string
	GetDeviceName() string
}

type GenericDevicePlugin struct {
	counter    int
	devs       []*pluginapi.Device
	server     *grpc.Server
	socketPath string
	stop       chan struct{}
	health     chan string
	devicePath string
	deviceName string
}

func NewGenericDevicePlugin(deviceName string, devicePath string, maxDevices int) *GenericDevicePlugin {
	serverSock := fmt.Sprintf(pluginapi.DevicePluginPath+"kubevirt-%s.sock", deviceName)
	dpi := &GenericDevicePlugin{
		counter:    0,
		devs:       []*pluginapi.Device{},
		socketPath: serverSock,
		health:     make(chan string),
		deviceName: deviceName,
		devicePath: devicePath,
	}
	for i := 0; i < maxDevices; i++ {
		dpi.addNewGenericDevice()
	}

	return dpi
}

func waitForGrpcServer(socketPath string, timeout time.Duration) error {
	conn, err := connect(socketPath, timeout)
	if err != nil {
		return err
	}
	conn.Close()
	return nil
}

// dial establishes the gRPC communication with the registered device plugin.
func connect(socketPath string, timeout time.Duration) (*grpc.ClientConn, error) {
	c, err := grpc.Dial(socketPath,
		grpc.WithInsecure(),
		grpc.WithBlock(),
		grpc.WithTimeout(timeout),
		grpc.WithDialer(func(addr string, timeout time.Duration) (net.Conn, error) {
			return net.DialTimeout("unix", addr, timeout)
		}),
	)

	if err != nil {
		return nil, err
	}

	return c, nil
}

func (dpi *GenericDevicePlugin) GetDevicePath() string {
	return dpi.devicePath
}

func (dpi *GenericDevicePlugin) GetDeviceName() string {
	return dpi.deviceName
}

// Start starts the gRPC server of the device plugin
func (dpi *GenericDevicePlugin) Start(stop chan struct{}) error {
	if dpi.server != nil {
		return fmt.Errorf("gRPC server already started")
	}

	logger := log.DefaultLogger()
	dpi.stop = stop

	err := dpi.cleanup()
	if err != nil {
		return err
	}

	sock, err := net.Listen("unix", dpi.socketPath)
	if err != nil {
		logger.Errorf("[%s] Error creating GRPC server socket: %v", dpi.deviceName, err)
		return err
	}

	dpi.server = grpc.NewServer([]grpc.ServerOption{}...)
	pluginapi.RegisterDevicePluginServer(dpi.server, dpi)

	err = dpi.Register()
	if err != nil {
		logger.Errorf("[%s] Error registering with device plugin manager: %v", dpi.deviceName, err)
		return err
	}

	go dpi.server.Serve(sock)

	err = waitForGrpcServer(dpi.socketPath, connectionTimeout)
	if err != nil {
		// this err is returned at the end of the Start function
		logger.Errorf("[%s] Error connecting to GRPC server: %v", dpi.deviceName, err)
	}

	go dpi.healthCheck()

	logger.V(3).Infof("[%s] Device plugin server ready", dpi.deviceName)

	return err
}

// Stop stops the gRPC server
func (dpi *GenericDevicePlugin) Stop() error {
	if dpi.server == nil {
		return nil
	}

	dpi.server.Stop()
	dpi.server = nil

	return dpi.cleanup()
}

// Register registers the device plugin for the given resourceName with Kubelet.
func (dpi *GenericDevicePlugin) Register() error {
	conn, err := connect(pluginapi.KubeletSocket, connectionTimeout)
	if err != nil {
		return err
	}
	defer conn.Close()

	client := pluginapi.NewRegistrationClient(conn)
	reqt := &pluginapi.RegisterRequest{
		Version:      pluginapi.Version,
		Endpoint:     path.Base(dpi.socketPath),
		ResourceName: fmt.Sprintf("%s/%s", DeviceNamespace, dpi.deviceName),
	}

	_, err = client.Register(context.Background(), reqt)
	if err != nil {
		return err
	}
	return nil
}

func (dpi *GenericDevicePlugin) addNewGenericDevice() {
	deviceId := dpi.deviceName + strconv.Itoa(dpi.counter)
	dpi.devs = append(dpi.devs, &pluginapi.Device{
		ID:     deviceId,
		Health: pluginapi.Healthy,
	})

	dpi.counter += 1
}

func (dpi *GenericDevicePlugin) ListAndWatch(e *pluginapi.Empty, s pluginapi.DevicePlugin_ListAndWatchServer) error {
	// FIXME: sending an empty list up front should not be needed. This is a workaround for:
	// https://github.com/kubevirt/kubevirt/issues/1196
	// This can safely be removed once supported upstream Kubernetes is 1.10.3 or higher.
	emptyList := []*pluginapi.Device{}
	s.Send(&pluginapi.ListAndWatchResponse{Devices: emptyList})

	s.Send(&pluginapi.ListAndWatchResponse{Devices: dpi.devs})

	for {
		select {
		case health := <-dpi.health:
			// There's only one shared generic device
			// so update each plugin device to reflect overall device health
			for _, dev := range dpi.devs {
				dev.Health = health
			}
			s.Send(&pluginapi.ListAndWatchResponse{Devices: dpi.devs})
		case <-dpi.stop:
			return nil
		}
	}
}

func (dpi *GenericDevicePlugin) Allocate(ctx context.Context, r *pluginapi.AllocateRequest) (*pluginapi.AllocateResponse, error) {
	response := pluginapi.AllocateResponse{}
	containerResponse := new(pluginapi.ContainerAllocateResponse)

	dev := new(pluginapi.DeviceSpec)
	dev.HostPath = dpi.devicePath
	dev.ContainerPath = dpi.devicePath
	dev.Permissions = "rw"
	containerResponse.Devices = []*pluginapi.DeviceSpec{dev}

	response.ContainerResponses = []*pluginapi.ContainerAllocateResponse{containerResponse}

	return &response, nil
}

func (dpi *GenericDevicePlugin) cleanup() error {
	if err := os.Remove(dpi.socketPath); err != nil && !os.IsNotExist(err) {
		return err
	}

	return nil
}

func (dpi *GenericDevicePlugin) GetDevicePluginOptions(ctx context.Context, e *pluginapi.Empty) (*pluginapi.DevicePluginOptions, error) {
	options := &pluginapi.DevicePluginOptions{
		PreStartRequired: false,
	}
	return options, nil
}

func (dpi *GenericDevicePlugin) PreStartContainer(ctx context.Context, in *pluginapi.PreStartContainerRequest) (*pluginapi.PreStartContainerResponse, error) {
	res := &pluginapi.PreStartContainerResponse{}
	return res, nil
}

func (dpi *GenericDevicePlugin) healthCheck() error {
	logger := log.DefaultLogger()
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		logger.Errorf("Unable to create fsnotify watcher: %v", err)
		return nil
	}
	defer watcher.Close()

	healthy := pluginapi.Healthy
	_, err = os.Stat(dpi.devicePath)
	if err != nil {
		if !os.IsNotExist(err) {
			logger.Errorf("Unable to stat device: %v", err)
			return err
		}
		healthy = pluginapi.Unhealthy
	}

	dirName := filepath.Dir(dpi.devicePath)

	err = watcher.Add(dirName)
	if err != nil {
		logger.Errorf("Unable to add path to fsnotify watcher: %v", err)
		return err
	}

	for {
		select {
		case <-dpi.stop:
			return nil
		case event := <-watcher.Events:
			logger.Infof("health Event: %v", event)
			logger.Infof("health Event Name: %s", event.Name)
			if event.Name == dpi.devicePath {
				// Health in this case is if the device path actually exists
				if event.Op == fsnotify.Create {
					healthy = pluginapi.Healthy
				} else if (event.Op == fsnotify.Remove) || (event.Op == fsnotify.Rename) {
					healthy = pluginapi.Unhealthy
				}
				dpi.health <- healthy
			}
		}
	}
}
