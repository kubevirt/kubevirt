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
	Start(chan struct{}) (err error)
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
	done       chan struct{}
	deviceRoot string
}

func NewGenericDevicePlugin(deviceName string, devicePath string, maxDevices int) *GenericDevicePlugin {
	serverSock := SocketPath(deviceName)
	dpi := &GenericDevicePlugin{
		counter:    0,
		devs:       []*pluginapi.Device{},
		socketPath: serverSock,
		health:     make(chan string),
		deviceName: deviceName,
		devicePath: devicePath,
		deviceRoot: "/proc/1/root/",
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

// Start starts the device plugin
func (dpi *GenericDevicePlugin) Start(stop chan struct{}) (err error) {
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

	logger.Infof("%s device plugin started", dpi.deviceName)
	err = <-errChan

	return err
}

// Stop stops the gRPC server
func (dpi *GenericDevicePlugin) Stop() error {
	defer close(dpi.done)
	dpi.server.Stop()
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
		case <-dpi.done:
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
		dpi.health <- pluginapi.Unhealthy
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
			if event.Name == devicePath {
				// Health in this case is if the device path actually exists
				if event.Op == fsnotify.Create {
					logger.Infof("monitored device %s appeared", dpi.deviceName)
					dpi.health <- pluginapi.Healthy
				} else if (event.Op == fsnotify.Remove) || (event.Op == fsnotify.Rename) {
					logger.Infof("monitored device %s disappeared", dpi.deviceName)
					dpi.health <- pluginapi.Unhealthy
				}
			} else if event.Name == dpi.socketPath && event.Op == fsnotify.Remove {
				logger.Infof("device socket file for device %s was removed, kubelet probably restarted.", dpi.deviceName)
				return nil
			}
		}
	}
}

func SocketPath(deviceName string) string {
	return filepath.Join(pluginapi.DevicePluginPath, fmt.Sprintf("kubevirt-%s.sock", deviceName))
}
