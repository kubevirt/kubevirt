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
	"errors"
	"fmt"
	"net"
	"os"
	"path"
	"path/filepath"
	"strconv"
	"strings"
	"sync"

	"github.com/fsnotify/fsnotify"
	"google.golang.org/grpc"

	"kubevirt.io/client-go/log"

	"kubevirt.io/kubevirt/pkg/safepath"
	"kubevirt.io/kubevirt/pkg/util"
	pluginapi "kubevirt.io/kubevirt/pkg/virt-handler/device-manager/deviceplugin/v1beta1"
	"kubevirt.io/kubevirt/pkg/virt-handler/selinux"
)

//go:generate mockgen -source $GOFILE -package=$GOPACKAGE -destination=generated_mock_$GOFILE

type PermissionManager interface {
	ChownAtNoFollow(path *safepath.Path, uid, gid int) error
}

type permissionManager struct{}

func NewPermissionManager() PermissionManager {
	return &permissionManager{}
}

func (p *permissionManager) ChownAtNoFollow(path *safepath.Path, uid, gid int) error {
	return safepath.ChownAtNoFollow(path, uid, gid)
}

type SocketDevicePlugin struct {
	*DevicePluginBase
	socketRoot    string
	socketDir     string
	socket        string
	socketName    string
	executor      selinux.Executor
	p             PermissionManager
	healthChecks  bool
	hostRootMount string
}

func (dpi *SocketDevicePlugin) Start(stop <-chan struct{}) (err error) {
	logger := log.DefaultLogger()
	dpi.stop = stop

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

func (dpi *SocketDevicePlugin) setSocketPermissions() error {
	if dpi.p == nil {
		return nil
	}
	prSock, err := safepath.JoinAndResolveWithRelativeRoot(dpi.socketRoot, dpi.socketDir, dpi.socket)
	if err != nil {
		return fmt.Errorf("error opening the socket %s: %v", path.Join(dpi.socketRoot, dpi.socketDir, dpi.socket), err)
	}
	err = dpi.p.ChownAtNoFollow(prSock, util.NonRootUID, util.NonRootUID)
	if err != nil {
		return fmt.Errorf("error setting the permission the socket %s: %v", path.Join(dpi.socketRoot, dpi.socketDir, dpi.socket), err)
	}
	if se, exists, err := dpi.executor.NewSELinux(); err == nil && exists {
		if err := selinux.RelabelFilesUnprivileged(se.IsPermissive(), prSock); err != nil {
			return fmt.Errorf("error relabeling required files: %v", err)
		}
	} else if err != nil {
		return fmt.Errorf("failed to detect the presence of selinux: %v", err)
	}

	return nil
}

func (dpi *SocketDevicePlugin) setSocketDirectoryPermissions() error {
	if dpi.p == nil {
		return nil
	}
	dir, err := safepath.JoinAndResolveWithRelativeRoot(dpi.socketRoot, dpi.socketDir)
	log.DefaultLogger().Infof("setting socket directory permissions for %s", path.Join(dpi.socketRoot, dpi.socketDir))
	if err != nil {
		return fmt.Errorf("error opening the socket dir %s: %v", path.Join(dpi.socketRoot, dpi.socketDir), err)
	}
	err = dpi.p.ChownAtNoFollow(dir, util.NonRootUID, util.NonRootUID)
	if err != nil {
		return fmt.Errorf("error setting the permission the socket dir %s: %v", path.Join(dpi.socketRoot, dpi.socketDir), err)
	}
	if se, exists, err := dpi.executor.NewSELinux(); err == nil && exists {
		if err := selinux.RelabelFilesUnprivileged(se.IsPermissive(), dir); err != nil {
			return fmt.Errorf("error relabeling required files: %v", err)
		}
	} else if err != nil {
		return fmt.Errorf("failed to detect the presence of selinux: %v", err)
	}

	return nil
}

func NewSocketDevicePlugin(socketName, socketDir, socket string, maxDevices int, executor selinux.Executor, p PermissionManager, useHostRootMount bool) (*SocketDevicePlugin, error) {
	socketRoot := "/"
	if useHostRootMount {
		socketRoot = util.HostRootMount
	}
	dpi := &SocketDevicePlugin{
		DevicePluginBase: &DevicePluginBase{
			health:       make(chan deviceHealth),
			resourceName: fmt.Sprintf("%s/%s", DeviceNamespace, socketName),
			initialized:  false,
			lock:         &sync.Mutex{},
			done:         make(chan struct{}),
			deregistered: make(chan struct{}),
			socketPath:   SocketPath(strings.Replace(socketName, "/", "-", -1)),
		},
		socketRoot:   socketRoot,
		socket:       socket,
		socketDir:    socketDir,
		socketName:   socketName,
		executor:     executor,
		p:            p,
		healthChecks: true,
	}

	for i := 0; i < maxDevices; i++ {
		deviceId := dpi.socketName + strconv.Itoa(i)
		dpi.devs = append(dpi.devs, &pluginapi.Device{
			ID:     deviceId,
			Health: pluginapi.Healthy,
		})
	}
	if err := dpi.setSocketDirectoryPermissions(); err != nil {
		return dpi, err
	}
	if err := dpi.setSocketPermissions(); err != nil {
		return dpi, err
	}

	return dpi, nil
}

// NewOptionalSocketDevicePlugin creates a SocketDevicePlugin where health checks are disabled (so device is always healthy)
func NewOptionalSocketDevicePlugin(socketName, socketDir, socket string, maxDevices int, executor selinux.Executor, p PermissionManager, useHostRootMount bool) *SocketDevicePlugin {
	dpi, _ := NewSocketDevicePlugin(socketName, socketDir, socket, maxDevices, executor, p, useHostRootMount)
	dpi.healthChecks = false
	return dpi
}

// Register registers the device plugin for the given resourceName with Kubelet.
func (dpi *SocketDevicePlugin) register() error {
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

func (dpi *SocketDevicePlugin) Allocate(ctx context.Context, r *pluginapi.AllocateRequest) (*pluginapi.AllocateResponse, error) {
	log.DefaultLogger().Infof("Socket Allocate: resourceName: %s", dpi.socketName)
	log.DefaultLogger().Infof("Socket Allocate: request: %v", r.ContainerRequests)
	response := pluginapi.AllocateResponse{}
	containerResponse := new(pluginapi.ContainerAllocateResponse)

	m := new(pluginapi.Mount)
	m.HostPath = dpi.socketDir
	m.ContainerPath = dpi.socketDir
	m.ReadOnly = false
	containerResponse.Mounts = []*pluginapi.Mount{m}

	response.ContainerResponses = []*pluginapi.ContainerAllocateResponse{containerResponse}

	return &response, nil
}

func (dpi *SocketDevicePlugin) sendHealthUpdate(healthy bool) {
	if !dpi.healthChecks {
		return
	}
	if healthy {
		dpi.health <- deviceHealth{Health: pluginapi.Healthy}
	} else {
		dpi.health <- deviceHealth{Health: pluginapi.Unhealthy}
	}
}

func (dpi *SocketDevicePlugin) healthCheck() error {
	logger := log.DefaultLogger()
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return fmt.Errorf("failed to creating a fsnotify watcher: %v", err)
	}
	defer watcher.Close()

	deviceDir := filepath.Join(dpi.socketRoot, dpi.socketDir)
	devicePath := filepath.Join(deviceDir, dpi.socket)

	// Start watching the files before we check for their existence to avoid races
	err = watcher.Add(deviceDir)
	if err != nil {
		return fmt.Errorf("failed to add the device root path to the watcher: %v", err)
	}

	_, err = os.Stat(devicePath)
	if err != nil {
		if !errors.Is(err, os.ErrNotExist) {
			return fmt.Errorf("could not stat the device: %v", err)
		}
		logger.Warningf("device '%s' is not present, the device plugin can't expose it.", dpi.socketName)
		dpi.sendHealthUpdate(false)
	}
	logger.Infof("device '%s' is present.", devicePath)

	err = watcher.Add(deviceDir)

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
			if event.Name == devicePath && dpi.healthChecks {
				// Health in this case is if the device path actually exists
				if event.Op == fsnotify.Create {
					logger.Infof("monitored device %s appeared", dpi.socketName)
					dpi.sendHealthUpdate(true)
					if err := dpi.setSocketDirectoryPermissions(); err != nil {
						logger.Warningf("failed to set directory permissions for socket device %s", dpi.socketName)
					}
					if err := dpi.setSocketPermissions(); err != nil {
						logger.Warningf("failed to set socket permissions for socket device %s", dpi.socketName)
					}
				} else if (event.Op == fsnotify.Remove) || (event.Op == fsnotify.Rename) {
					logger.Infof("monitored device %s disappeared", dpi.socketName)
					dpi.sendHealthUpdate(false)
				}
			} else if event.Name == dpi.socketPath && event.Op == fsnotify.Remove {
				logger.Infof("device socket file for device %s was removed, kubelet probably restarted.", dpi.socketName)
				return nil
			}
		}
	}
}
