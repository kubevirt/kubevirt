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

package kvm_monitor

import (
	"net"
	"os"
	"path"
	"strconv"
	"time"

	"golang.org/x/net/context"

	"google.golang.org/grpc"
	pluginapi "k8s.io/kubernetes/pkg/kubelet/apis/deviceplugin/v1alpha"

	"github.com/fsnotify/fsnotify"

	"kubevirt.io/kubevirt/pkg/log"
)

const (
	KVMPath           = "/dev/kvm"
	KVMName           = "kvm"
	KvmDevice         = "devices.kubevirt.io/kvm"
	connectionTimeout = 5 * time.Second
	serverSock        = pluginapi.DevicePluginPath + "kubevirt-kvm.sock"
)

type KVMDevicePlugin struct {
	counter    int
	devs       []*pluginapi.Device
	update     chan struct{}
	server     *grpc.Server
	socketPath string
	stop       chan struct{}
	health     chan string
}

func NewKVMDevicePlugin() *KVMDevicePlugin {
	dpi := &KVMDevicePlugin{
		counter:    0,
		devs:       []*pluginapi.Device{},
		socketPath: serverSock,
		update:     make(chan struct{}),
		health:     make(chan string),
	}
	dpi.addNewKVMDevice()
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

// Start starts the gRPC server of the device plugin
func (dpi *KVMDevicePlugin) Start(stop chan struct{}) error {
	// FIXME: decide if starting twice is normal or error
	if dpi.server != nil {
		dpi.Stop()
		// return fmt.Errorf("gRPC server already started")
	}

	dpi.stop = stop

	err := dpi.cleanup()
	if err != nil {
		return err
	}

	sock, err := net.Listen("unix", dpi.socketPath)
	if err != nil {
		return err
	}

	dpi.server = grpc.NewServer([]grpc.ServerOption{}...)
	pluginapi.RegisterDevicePluginServer(dpi.server, dpi)

	dpi.Register()

	go dpi.server.Serve(sock)

	err = waitForGrpcServer(dpi.socketPath, connectionTimeout)

	go dpi.healthCheck()

	return err
}

// Stop stops the gRPC server
func (dpi *KVMDevicePlugin) Stop() error {
	if dpi.server == nil {
		return nil
	}

	dpi.server.Stop()
	dpi.server = nil

	return dpi.cleanup()
}

// Register registers the device plugin for the given resourceName with Kubelet.
func (dpi *KVMDevicePlugin) Register() error {
	conn, err := connect(pluginapi.KubeletSocket, connectionTimeout)
	if err != nil {
		return err
	}
	defer conn.Close()

	client := pluginapi.NewRegistrationClient(conn)
	reqt := &pluginapi.RegisterRequest{
		Version:      pluginapi.Version,
		Endpoint:     path.Base(dpi.socketPath),
		ResourceName: KvmDevice,
	}

	_, err = client.Register(context.Background(), reqt)
	if err != nil {
		return err
	}
	return nil
}

func (dpi *KVMDevicePlugin) addNewKVMDevice() {
	deviceId := KVMName + strconv.Itoa(dpi.counter)
	dpi.devs = append(dpi.devs, &pluginapi.Device{
		ID:     deviceId,
		Health: pluginapi.Healthy,
	})

	logger := log.DefaultLogger()
	logger.Infof("Allocated new KVM device: %s", deviceId)

	dpi.counter += 1
}

func (dpi *KVMDevicePlugin) ListAndWatch(e *pluginapi.Empty, s pluginapi.DevicePlugin_ListAndWatchServer) error {
	s.Send(&pluginapi.ListAndWatchResponse{Devices: dpi.devs})

	for {
		select {
		case health := <-dpi.health:
			// There's only one shared kvm device
			// so update each plugin device to reflect overall device health
			for _, dev := range dpi.devs {
				dev.Health = health
			}
			s.Send(&pluginapi.ListAndWatchResponse{Devices: dpi.devs})
		case <-dpi.update:
			s.Send(&pluginapi.ListAndWatchResponse{Devices: dpi.devs})
		case <-dpi.stop:
			return nil
		}
	}
}

// We can only allocate new devices. There is no provision to de-allocate
func (dpi *KVMDevicePlugin) Allocate(ctx context.Context, r *pluginapi.AllocateRequest) (*pluginapi.AllocateResponse, error) {
	// FIXME: This isn't threadsafe... Maybe tell ListAndWatch to make the device?
	// Or maybe add a mutex?
	dpi.addNewKVMDevice()
	dpi.update <- struct{}{}

	var response pluginapi.AllocateResponse
	dev := new(pluginapi.DeviceSpec)
	dev.HostPath = KVMPath
	dev.ContainerPath = KVMPath
	dev.Permissions = "rw"
	response.Devices = append(response.Devices, dev)

	// FIXME: This does not belong here.
	tundev := new(pluginapi.DeviceSpec)
	tundev.HostPath = "/dev/net/tun"
	tundev.ContainerPath = "/dev/net/tun"
	tundev.Permissions = "rw"
	response.Devices = append(response.Devices, tundev)

	return &response, nil
}

func (dpi *KVMDevicePlugin) cleanup() error {
	if err := os.Remove(dpi.socketPath); err != nil && !os.IsNotExist(err) {
		return err
	}

	return nil
}

func (dpi *KVMDevicePlugin) healthCheck() error {
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return nil
	}
	defer watcher.Close()

	watcher.Add(KVMPath)

	healthy := pluginapi.Healthy
	for {
		select {

		case event := <-watcher.Events:
			// Health in this case is if the KVM device actually exists
			if event.Op == fsnotify.Create {
				healthy = pluginapi.Healthy
			} else if event.Op == fsnotify.Remove {
				healthy = pluginapi.Unhealthy
			}
			dpi.health <- healthy
		case <-dpi.stop:
			return nil
		}
	}
}
