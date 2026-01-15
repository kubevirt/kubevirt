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
	"os"
	"path"
	"path/filepath"
	"strconv"
	"strings"
	"sync"

	"github.com/fsnotify/fsnotify"
	"kubevirt.io/client-go/log"

	"kubevirt.io/kubevirt/pkg/safepath"
	"kubevirt.io/kubevirt/pkg/util"
	pluginapi "kubevirt.io/kubevirt/pkg/virt-handler/device-manager/deviceplugin/v1beta1"
	"kubevirt.io/kubevirt/pkg/virt-handler/selinux"
)

type SocketDevicePlugin struct {
	*DevicePluginBase
	socketRoot    string
	socketDir     string
	socket        string
	executor      selinux.Executor
	p             permissionManager
	healthChecks  bool
	hostRootMount string
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

func NewSocketDevicePlugin(socketName, socketDir, socket string, maxDevices int, executor selinux.Executor, p permissionManager, useHostRootMount bool) (*SocketDevicePlugin, error) {
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
		executor:     executor,
		p:            p,
		healthChecks: true,
	}
	dpi.healthCheck = dpi.healthCheckFunc

	dpi.deviceNameByID = dpi.deviceNameByIDFunc
	dpi.allocateDP = dpi.allocateDPFunc

	for i := 0; i < maxDevices; i++ {
		deviceId := socketName + strconv.Itoa(i)
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
func NewOptionalSocketDevicePlugin(socketName, socketDir, socket string, maxDevices int, executor selinux.Executor, p permissionManager, useHostRootMount bool) *SocketDevicePlugin {
	dpi, _ := NewSocketDevicePlugin(socketName, socketDir, socket, maxDevices, executor, p, useHostRootMount)
	dpi.healthChecks = false
	return dpi
}

func (dpi *SocketDevicePlugin) allocateDPFunc(ctx context.Context, r *pluginapi.AllocateRequest) (*pluginapi.AllocateResponse, error) {
	log.DefaultLogger().Infof("Socket Allocate: resourceName: %s", dpi.resourceName)
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

func (dpi *SocketDevicePlugin) healthCheckFunc() error {
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
		logger.Warningf("device '%s' is not present, the device plugin can't expose it.", dpi.resourceName)
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
					logger.Infof("monitored device %s appeared", dpi.resourceName)
					dpi.sendHealthUpdate(true)
					if err := dpi.setSocketDirectoryPermissions(); err != nil {
						logger.Warningf("failed to set directory permissions for socket device %s", dpi.resourceName)
					}
					if err := dpi.setSocketPermissions(); err != nil {
						logger.Warningf("failed to set socket permissions for socket device %s", dpi.resourceName)
					}
				} else if (event.Op == fsnotify.Remove) || (event.Op == fsnotify.Rename) {
					logger.Infof("monitored device %s disappeared", dpi.resourceName)
					dpi.sendHealthUpdate(false)
				}
			} else if event.Name == dpi.socketPath && event.Op == fsnotify.Remove {
				logger.Infof("device socket file for device %s was removed, kubelet probably restarted.", dpi.resourceName)
				return nil
			}
		}
	}
}

func (dpi *SocketDevicePlugin) deviceNameByIDFunc(_ string) string {
	devicePath := filepath.Join(dpi.deviceRoot, dpi.devicePath)
	return fmt.Sprintf("socket device (%s)", devicePath)
}
