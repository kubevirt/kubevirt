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
	// healthy is the last health reported to ListAndWatch; only the health check goroutine touches it.
	healthy bool
}

func (dpi *SocketDevicePlugin) Start(stop <-chan struct{}) (err error) {
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

// applyPermissions hands the socket and its directory over to the non-root user consumers run as.
func (dpi *SocketDevicePlugin) applyPermissions() error {
	if err := dpi.setSocketDirectoryPermissions(); err != nil {
		return err
	}
	return dpi.setSocketPermissions()
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
		// The devices start Unhealthy: they are only advertised once the initial reconcile in healthCheck has verified the socket and applied its permissions.
		// healthy must match their initial state, or the first report is deduplicated away; both survive a plugin restart, which the reconcile on start covers.
		healthy: false,
	}

	for i := 0; i < maxDevices; i++ {
		deviceId := dpi.socketName + strconv.Itoa(i)
		dpi.devs = append(dpi.devs, &pluginapi.Device{
			ID:     deviceId,
			Health: pluginapi.Unhealthy,
		})
	}
	if err := dpi.applyPermissions(); err != nil {
		return dpi, err
	}

	return dpi, nil
}

// NewOptionalSocketDevicePlugin creates a SocketDevicePlugin where health checks are disabled (so device is always healthy)
func NewOptionalSocketDevicePlugin(socketName, socketDir, socket string, maxDevices int, executor selinux.Executor, p PermissionManager, useHostRootMount bool) *SocketDevicePlugin {
	dpi, _ := NewSocketDevicePlugin(socketName, socketDir, socket, maxDevices, executor, p, useHostRootMount)
	dpi.healthChecks = false
	// With health checks disabled nothing would ever report Healthy, so the devices must start that way.
	dpi.healthy = true
	for _, dev := range dpi.devs {
		dev.Health = pluginapi.Healthy
	}
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

// sendHealthUpdate reports a health change to ListAndWatch; unchanged health is not resent, as every send blocks the draining of the fsnotify queue.
func (dpi *SocketDevicePlugin) sendHealthUpdate(healthy bool) {
	if !dpi.healthChecks || healthy == dpi.healthy {
		return
	}
	dpi.healthy = healthy
	if healthy {
		log.DefaultLogger().Infof("monitored device %s became healthy", dpi.socketName)
		dpi.health <- deviceHealth{Health: pluginapi.Healthy}
	} else {
		log.DefaultLogger().Infof("monitored device %s became unhealthy", dpi.socketName)
		dpi.health <- deviceHealth{Health: pluginapi.Unhealthy}
	}
}

// reconcileDeviceHealth re-arms the deviceDir watch and derives health from the filesystem rather than from events, which can be missed or dropped.
// A permission failure is an error: no event would ever retry it here, so it is left to the device controller, which restarts the plugin with backoff, as at construction.
func (dpi *SocketDevicePlugin) reconcileDeviceHealth(watcher *fsnotify.Watcher, deviceDir, devicePath string) error {
	logger := log.DefaultLogger()

	// Replaces the watch if deviceDir was recreated; a no-op if it is still alive.
	if err := watcher.Add(deviceDir); err != nil {
		if !errors.Is(err, os.ErrNotExist) {
			logger.Reason(err).Errorf("failed to watch the device directory '%s'", deviceDir)
		}
		dpi.sendHealthUpdate(false)
		return nil
	}
	if _, err := os.Stat(devicePath); err != nil {
		if !errors.Is(err, os.ErrNotExist) {
			logger.Reason(err).Errorf("could not stat the device '%s'", devicePath)
		}
		dpi.sendHealthUpdate(false)
		return nil
	}
	// Apply the socket's permissions before advertising it, so a newly scheduled consumer cannot race the chown; a socket whose permissions failed to apply is not advertised at all.
	if err := dpi.applyPermissions(); err != nil {
		return fmt.Errorf("failed to set permissions for socket device %s: %v", dpi.socketName, err)
	}
	dpi.sendHealthUpdate(true)
	return nil
}

// armDeviceDirWatch watches the device's directory and derives the initial health from the filesystem; a missing directory or socket is unhealthy, not fatal.
func (dpi *SocketDevicePlugin) armDeviceDirWatch(watcher *fsnotify.Watcher, deviceDir, devicePath string) error {
	logger := log.DefaultLogger()

	// Watch before the existence checks to avoid races; fsnotify passes the errno through untouched.
	if err := watcher.Add(deviceDir); err != nil {
		if !errors.Is(err, os.ErrNotExist) {
			return fmt.Errorf("failed to add the device root path to the watcher: %v", err)
		}
		logger.Warningf("device directory '%s' is not present, waiting for it to be created.", deviceDir)
		dpi.sendHealthUpdate(false)
		return nil
	}
	if _, err := os.Stat(devicePath); err != nil {
		if !errors.Is(err, os.ErrNotExist) {
			return fmt.Errorf("could not stat the device: %v", err)
		}
		logger.Warningf("device '%s' is not present, the device plugin can't expose it.", dpi.socketName)
		dpi.sendHealthUpdate(false)
		return nil
	}
	logger.Infof("device '%s' is present.", devicePath)
	// Stale health survives a plugin restart, and with the socket already present no Create event would fix it up.
	return dpi.reconcileDeviceHealth(watcher, deviceDir, devicePath)
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

	// deviceDir (typically a systemd RuntimeDirectory=) dies with service restarts, so watch its stable parent to re-arm on recreation; a missing parent is fatal.
	if err := watcher.Add(filepath.Dir(deviceDir)); err != nil {
		return fmt.Errorf("failed to add the parent of the device root path to the watcher: %v", err)
	}

	if err := watcher.Add(filepath.Dir(dpi.socketPath)); err != nil {
		return fmt.Errorf("failed to add the kubelet device-plugin directory to the watcher: %v", err)
	}
	_, err = os.Stat(dpi.socketPath)
	if err != nil {
		return fmt.Errorf("failed to stat the device-plugin socket: %v", err)
	}

	// This may block reporting the initial health until ListAndWatch drains it, so it goes last, after every watch is in place.
	if err := dpi.armDeviceDirWatch(watcher, deviceDir, devicePath); err != nil {
		return err
	}

	for {
		select {
		case <-dpi.stop:
			return nil
		case err := <-watcher.Errors:
			logger.Reason(err).Errorf("error watching devices and device plugin directory")
			// The kernel drops events on queue overflow, so re-check the filesystem instead of trusting events.
			// This cannot loop back here: watcher.Add failures are returned synchronously, never sent to watcher.Errors.
			if dpi.healthChecks {
				if err := dpi.reconcileDeviceHealth(watcher, deviceDir, devicePath); err != nil {
					return err
				}
			}
		case event := <-watcher.Events:
			logger.V(4).Infof("health Event: %v", event)
			if (event.Name == devicePath || event.Name == deviceDir) && dpi.healthChecks {
				// Health in this case is if the device path actually exists.
				// Only these ops are handled: reacting to the Chmod from applying permissions would loop.
				if event.Has(fsnotify.Create) {
					if err := dpi.reconcileDeviceHealth(watcher, deviceDir, devicePath); err != nil {
						return err
					}
				} else if event.Has(fsnotify.Remove) || event.Has(fsnotify.Rename) {
					logger.Infof("monitored device %s disappeared: %s was removed", dpi.socketName, event.Name)
					dpi.sendHealthUpdate(false)
				}
			} else if event.Name == dpi.socketPath && event.Has(fsnotify.Remove) {
				logger.Infof("device socket file for device %s was removed, kubelet probably restarted.", dpi.socketName)
				return nil
			}
		}
	}
}
