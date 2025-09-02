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
	"net"
	"os"
	"strconv"
	"sync"
	"time"

	"google.golang.org/grpc"

	"kubevirt.io/client-go/log"

	"kubevirt.io/kubevirt/pkg/util"
	pluginapi "kubevirt.io/kubevirt/pkg/virt-handler/device-manager/deviceplugin/v1beta1"
)

const (
	DeviceNamespace   = "devices.kubevirt.io"
	connectionTimeout = 5 * time.Second
)

type Device interface {
	Start(stop <-chan struct{}) error
	ListAndWatch(*pluginapi.Empty, pluginapi.DevicePlugin_ListAndWatchServer) error
	PreStartContainer(context.Context, *pluginapi.PreStartContainerRequest) (*pluginapi.PreStartContainerResponse, error)
	Allocate(context.Context, *pluginapi.AllocateRequest) (*pluginapi.AllocateResponse, error)
	GetDeviceName() string
	GetInitialized() bool
}

type GenericDevicePlugin struct {
	*DevicePluginBase
	deviceName  string
	preOpen     bool
	permissions string
}

func NewGenericDevicePlugin(deviceName string, devicePath string, maxDevices int, permissions string, preOpen bool) *GenericDevicePlugin {
	serverSock := SocketPath(deviceName)
	dpi := &GenericDevicePlugin{
		DevicePluginBase: &DevicePluginBase{
			devs:         []*pluginapi.Device{},
			socketPath:   serverSock,
			health:       make(chan deviceHealth),
			devicePath:   devicePath,
			deviceRoot:   util.HostRootMount,
			resourceName: fmt.Sprintf("%s/%s", DeviceNamespace, deviceName),
			initialized:  false,
			lock:         &sync.Mutex{},
		},
		deviceName:  deviceName,
		preOpen:     preOpen,
		permissions: permissions,
	}

	for i := 0; i < maxDevices; i++ {
		deviceId := dpi.deviceName + strconv.Itoa(i)
		dpi.devs = append(dpi.devs, &pluginapi.Device{
			ID:     deviceId,
			Health: pluginapi.Healthy,
		})
	}

	return dpi
}

func (dpi *GenericDevicePlugin) GetDeviceName() string {
	return dpi.deviceName
}

// Start starts the device plugin
func (dpi *GenericDevicePlugin) Start(stop <-chan struct{}) (err error) {
	logger := log.DefaultLogger()
	dpi.stop = stop
	dpi.done = make(chan struct{})
	dpi.deregistered = make(chan struct{})

	err = dpi.cleanup()
	if err != nil {
		return err
	}

	// The kernel module(s) for some devices, like tun and vhost-net, auto-load when needed.
	// That need is identified by the first access to their main device node.
	// Opening and closing the device nodes here will trigger any necessary modprobe.
	if dpi.preOpen {
		devnode, err := os.Open(dpi.devicePath)
		if err == nil {
			devnode.Close()
		}
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
	logger.Infof("%s device plugin started", dpi.deviceName)
	err = <-errChan

	return err
}

func (dpi *GenericDevicePlugin) Allocate(ctx context.Context, r *pluginapi.AllocateRequest) (*pluginapi.AllocateResponse, error) {
	ar, err := dpi.DevicePluginBase.Allocate(ctx, r)
	ar.ContainerResponses[0].Devices[0].Permissions = dpi.permissions
	return ar, err
}
