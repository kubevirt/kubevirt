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
	"path/filepath"
	"strconv"
	"sync"

	"github.com/fsnotify/fsnotify"
	"google.golang.org/grpc"

	"kubevirt.io/client-go/log"

	pluginapi "kubevirt.io/kubevirt/pkg/virt-handler/device-manager/deviceplugin/v1beta1"
)

// DummyDevicePlugin advertises a fixed amount of a resource that has no device
// node behind it, e.g. the secure guest capacity of a confidential computing
// host. It only performs resource housekeeping so the scheduler cannot place
// more consumers on the node than the advertised capacity; Allocate hands out
// no devices and no mounts.
type DummyDevicePlugin struct {
	*DevicePluginBase
}

func NewDummyDevicePlugin(deviceName string, capacity int) *DummyDevicePlugin {
	dpi := &DummyDevicePlugin{
		DevicePluginBase: &DevicePluginBase{
			devs:         []*pluginapi.Device{},
			socketPath:   SocketPath(deviceName),
			health:       make(chan deviceHealth),
			deviceName:   deviceName,
			resourceName: fmt.Sprintf("%s/%s", DeviceNamespace, deviceName),
			initialized:  false,
			lock:         &sync.Mutex{},
		},
	}

	for i := 0; i < capacity; i++ {
		dpi.devs = append(dpi.devs, &pluginapi.Device{
			ID:     deviceName + strconv.Itoa(i),
			Health: pluginapi.Healthy,
		})
	}

	return dpi
}

func (dpi *DummyDevicePlugin) Start(stop <-chan struct{}) (err error) {
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
	logger.Infof("%s device plugin started", dpi.deviceName)
	err = <-errChan

	return err
}

// Allocate accounts for the requested devices without exposing anything to the
// container, since there is no device node to mount.
func (dpi *DummyDevicePlugin) Allocate(_ context.Context, r *pluginapi.AllocateRequest) (*pluginapi.AllocateResponse, error) {
	log.DefaultLogger().Infof("Dummy Allocate: resourceName: %s", dpi.deviceName)
	log.DefaultLogger().Infof("Dummy Allocate: request: %v", r.ContainerRequests)

	response := pluginapi.AllocateResponse{}
	for range r.ContainerRequests {
		response.ContainerResponses = append(response.ContainerResponses, &pluginapi.ContainerAllocateResponse{})
	}

	return &response, nil
}

// healthCheck only monitors the device plugin socket, there is no device node
// whose health could be tracked.
func (dpi *DummyDevicePlugin) healthCheck() error {
	logger := log.DefaultLogger()
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return fmt.Errorf("failed to creating a fsnotify watcher: %v", err)
	}
	defer watcher.Close()

	err = watcher.Add(filepath.Dir(dpi.socketPath))
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
			logger.Reason(err).Errorf("error watching device plugin directory")
		case event := <-watcher.Events:
			logger.V(4).Infof("health Event: %v", event)
			if event.Name == dpi.socketPath && event.Op == fsnotify.Remove {
				logger.Infof("device socket file for device %s was removed, kubelet probably restarted.", dpi.deviceName)
				return nil
			}
		}
	}
}

func (dpi *DummyDevicePlugin) GetDeviceName() string {
	return dpi.deviceName
}
