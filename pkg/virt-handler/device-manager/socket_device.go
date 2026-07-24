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
	"path"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/fsnotify/fsnotify"
	"kubevirt.io/client-go/log"

	"kubevirt.io/kubevirt/pkg/safepath"
	"kubevirt.io/kubevirt/pkg/util"
	pluginapi "kubevirt.io/kubevirt/pkg/virt-handler/device-manager/deviceplugin/v1beta1"
	"kubevirt.io/kubevirt/pkg/virt-handler/selinux"
)

type SocketDevicePlugin struct {
	*DevicePluginBase
	p            permissionManager
	executor     selinux.Executor
	healthChecks bool
}

func (dpi *SocketDevicePlugin) setSocketPermissions() error {
	if dpi.p == nil {
		return nil
	}
	prSock, err := safepath.JoinAndResolveWithRelativeRoot(dpi.deviceRoot, dpi.devicePath)
	if err != nil {
		return fmt.Errorf("error opening the socket %s: %v", path.Join(dpi.deviceRoot, dpi.devicePath), err)
	}
	err = dpi.p.ChownAtNoFollow(prSock, util.NonRootUID, util.NonRootUID)
	if err != nil {
		return fmt.Errorf("error setting the permission the socket %s: %v", path.Join(dpi.deviceRoot, dpi.devicePath), err)
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
	socketDir := filepath.Dir(dpi.devicePath)
	dir, err := safepath.JoinAndResolveWithRelativeRoot(dpi.deviceRoot, socketDir)
	log.DefaultLogger().Infof("setting socket directory permissions for %s", path.Join(dpi.deviceRoot, dpi.devicePath))
	if err != nil {
		return fmt.Errorf("error opening the socket dir %s: %v", path.Join(dpi.deviceRoot, dpi.devicePath), err)
	}
	err = dpi.p.ChownAtNoFollow(dir, util.NonRootUID, util.NonRootUID)
	if err != nil {
		return fmt.Errorf("error setting the permission the socket dir %s: %v", path.Join(dpi.deviceRoot, dpi.devicePath), err)
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

func newSocketDevicePlugin(socketName, socketDir, socketFile string, maxDevices int, executor selinux.Executor, p permissionManager, useHostRootMount bool, healthChecks bool) *DevicePlugin {
	deviceRoot := "/"
	if useHostRootMount {
		deviceRoot = util.HostRootMount
	}
	dpi := &SocketDevicePlugin{
		DevicePluginBase: &DevicePluginBase{
			devs:         []*pluginapi.Device{},
			resourceName: fmt.Sprintf("%s/%s", DeviceNamespace, socketName),
			deviceRoot:   deviceRoot,
			devicePath:   filepath.Join(socketDir, socketFile),
			socketPath:   SocketPath(strings.Replace(socketName, "/", "-", -1)),
		},
		p:            p,
		executor:     executor,
		healthChecks: healthChecks,
	}

	health := pluginapi.Unhealthy
	if !healthChecks {
		health = pluginapi.Healthy
	}

	for i := range maxDevices {
		deviceId := socketName + strconv.Itoa(i)
		dpi.devs = append(dpi.devs, &pluginapi.Device{
			ID:     deviceId,
			Health: health,
		})
	}

	return newDevicePlugin(dpi)
}

func NewSocketDevicePlugin(socketName, socketDir, socketFile string, maxDevices int, executor selinux.Executor, p permissionManager, useHostRootMount bool) *DevicePlugin {
	return newSocketDevicePlugin(socketName, socketDir, socketFile, maxDevices, executor, p, useHostRootMount, true)
}

// NewOptionalSocketDevicePlugin creates a SocketDevicePlugin where health checks are overriden to always return healthy
func NewOptionalSocketDevicePlugin(socketName, socketDir, socketFile string, maxDevices int, executor selinux.Executor, p permissionManager, useHostRootMount bool) *DevicePlugin {
	return newSocketDevicePlugin(socketName, socketDir, socketFile, maxDevices, executor, p, useHostRootMount, false)
}

func (dpi *SocketDevicePlugin) configurePermissions(_ *safepath.Path) error {
	if dpi.p == nil || dpi.executor == nil {
		return fmt.Errorf("permission manager and executor are not provided")
	}
	// Set directory permissions first
	if err := dpi.setSocketDirectoryPermissions(); err != nil {
		return err
	}
	// Then set socket permissions
	return dpi.setSocketPermissions()
}

func (dpi *SocketDevicePlugin) updateHealth(_ string, _ string, healthy bool) (bool, error) {
	if dpi.healthChecks {
		return healthy, nil
	} else {
		return true, nil
	}
}

func (dpi *SocketDevicePlugin) allocateDP(_ context.Context, r *pluginapi.AllocateRequest) (*pluginapi.AllocateResponse, error) {
	log.DefaultLogger().Infof("Socket Allocate: resourceName: %s", dpi.resourceName)
	log.DefaultLogger().Infof("Socket Allocate: request: %v", r.ContainerRequests)

	response := pluginapi.AllocateResponse{}
	containerResponse := new(pluginapi.ContainerAllocateResponse)
	socketDir := filepath.Dir(dpi.devicePath)

	m := new(pluginapi.Mount)
	m.HostPath = socketDir
	m.ContainerPath = socketDir
	m.ReadOnly = false
	containerResponse.Mounts = []*pluginapi.Mount{m}

	response.ContainerResponses = []*pluginapi.ContainerAllocateResponse{containerResponse}

	return &response, nil
}

func (dpi *SocketDevicePlugin) setupMonitoredDevices(watcher *fsnotify.Watcher, monitoredDevices map[string]string) error {
	logger := log.DefaultLogger()
	devicePath := filepath.Join(dpi.deviceRoot, dpi.devicePath)
	socketDir := filepath.Dir(devicePath)
	if err := watcher.Add(socketDir); err != nil {
		logger.Warningf("failed to add the device directory %s to the watcher: %v", socketDir, err)
	}
	parentDir := filepath.Dir(socketDir)
	if err := watcher.Add(parentDir); err != nil {
		// unrecoverable error
		return fmt.Errorf("failed to add the device parent directory to the watcher: %v", err)
	}
	monitoredDevices[devicePath] = ""
	return nil
}

func (dpi *SocketDevicePlugin) deviceNameByID(_ string) string {
	devicePath := filepath.Join(dpi.deviceRoot, dpi.devicePath)
	return fmt.Sprintf("socket device (%s)", devicePath)
}
