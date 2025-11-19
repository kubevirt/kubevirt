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
	"sync"
	"time"

	"github.com/fsnotify/fsnotify"
	"google.golang.org/grpc"
	"kubevirt.io/client-go/log"

	"kubevirt.io/kubevirt/pkg/safepath"
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
	GetResourceName() string
	GetInitialized() bool
}

type DevicePluginBase struct {
	devs                  []*pluginapi.Device
	server                *grpc.Server
	socketPath            string
	stop                  <-chan struct{}
	health                chan deviceHealth
	resourceName          string // The kubernetes resource name for this device plugin
	done                  chan struct{}
	initialized           bool
	lock                  *sync.Mutex
	deregistered          chan struct{}
	deviceRoot            string                                                                       // Root directory where the device is located
	devicePath            string                                                                       // Relative path to the device from the device root
	SetupMonitoredDevices func(*fsnotify.Watcher, map[string]string) error                             // REQUIRED function to set up the devices that are being monitored and update map such that key contains absolute paths to watch, and value contains the device id.
	SetupDevicePlugin     func() error                                                                 // Optional function to perform additional setup steps that are not covered by the default implementation
	GetIDDeviceName       func(string) string                                                          // Optional function to convert device id to a human readable name for logging
	ConfigurePermissions  func(*safepath.Path) error                                                   // Optional function to configure permissions for the device if needed. When present, device being marked healthy is contingent on the hook exiting with out error.
	CustomReportHealth    func(deviceID string, absoluteDevicePath string, healthy bool) (bool, error) // Optional function for plugin devices that require custom logic to handle health reports.
}

func (dpi *DevicePluginBase) GetResourceName() string {
	return dpi.resourceName
}

func (dpi *DevicePluginBase) Start(stop <-chan struct{}) (err error) {
	logger := log.DefaultLogger()
	dpi.stop = stop
	dpi.done = make(chan struct{})
	dpi.deregistered = make(chan struct{})

	if err := dpi.cleanup(); err != nil {
		return err
	}

	// If a custom SetupDevicePlugin hook is implemented, call it
	// for additional setup steps that are not covered by the default implementation
	if dpi.SetupDevicePlugin != nil {
		if err = dpi.SetupDevicePlugin(); err != nil {
			return err
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
	logger.Infof("%s device plugin started", dpi.resourceName)
	err = <-errChan

	return err
}

func (dpi *DevicePluginBase) ListAndWatch(_ *pluginapi.Empty, s pluginapi.DevicePlugin_ListAndWatchServer) error {
	s.Send(&pluginapi.ListAndWatchResponse{Devices: dpi.devs})

	done := false
	for {
		select {
		case devHealth := <-dpi.health:
			for _, dev := range dpi.devs {
				// If the devHealth.DevId is empty, it was not set by the device plugin, so we update all devices
				if devHealth.DevId == dev.ID || devHealth.DevId == "" {
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
	emptyList := []*pluginapi.Device{}
	if err := s.Send(&pluginapi.ListAndWatchResponse{Devices: emptyList}); err != nil {
		log.DefaultLogger().Reason(err).Infof("%s device plugin failed to deregister", dpi.resourceName)
	}
	close(dpi.deregistered)
	return nil
}

func (dpi *DevicePluginBase) healthCheck() error {
	logger := log.DefaultLogger()

	// key: device path, value: corresponding device ID
	// Used to track the devices that are being monitored
	// When a corresponding device ID is empty it means this device path represents ALL device IDs
	monitoredDevices := make(map[string]string)
	// key: device path, value: last known health
	lastKnownHealth := make(map[string]string)

	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return fmt.Errorf("failed to creating a fsnotify watcher: %v", err)
	}
	defer watcher.Close()

	devicePath := filepath.Join(dpi.deviceRoot, dpi.devicePath)
	deviceDirPath := filepath.Dir(devicePath)

	// Set up monitored device paths
	// Should watch before stat'ing the device path to avoid race conditions
	if dpi.SetupMonitoredDevices == nil {
		return fmt.Errorf("SetupMonitoredDevices is not implemented")
	}
	err = dpi.SetupMonitoredDevices(watcher, monitoredDevices)
	if err != nil {
		return err
	}

	// Check the device plugin socket to ensure we can communicate with it
	dirName := filepath.Dir(dpi.socketPath)
	if err = watcher.Add(dirName); err != nil {
		return fmt.Errorf("failed to add the device-plugin kubelet path to the watcher: %v", err)
	}
	if _, err = os.Stat(dpi.socketPath); err != nil {
		return fmt.Errorf("failed to stat the device-plugin socket: %v", err)
	}

	// Define helper functions for health checks
	//  - configurePermissionsAndReportSuccess: configures permissions for a device and reports success
	configurePermissionsAndReportSuccess := func(devicePath string) bool {
		success := true
		// If the ConfigurePermissions hook is not implemented, we consider the operation successful
		if dpi.ConfigurePermissions != nil {
			logger.V(4).Infof("ensuring permissions for device %s", devicePath)
			// Since devicePath is an absolute path, we need to get the relative path from deviceRoot to enforce containment
			relPath, err := filepath.Rel(dpi.deviceRoot, devicePath)
			if err != nil {
				logger.Reason(err).Warningf("failed to get relative path for device %s", devicePath)
				return false
			}
			// Use JoinAndResolveWithRelativeRoot to ensure path stays within deviceRoot
			dp, err := safepath.JoinAndResolveWithRelativeRoot("/", dpi.deviceRoot, relPath)
			if err != nil {
				logger.Reason(err).Warningf("failed to create safepath for device %s", devicePath)
				return false
			}
			if err := dpi.ConfigurePermissions(dp); err != nil {
				logger.Reason(err).Warningf("failed to ensure permissions for device %s", devicePath)
				success = false
			}
		}
		return success
	}
	//  - reportHealth: reports the health of a device
	reportHealth := func(deviceID string, absoluteDevicePath string, healthy bool) {
		if dpi.CustomReportHealth != nil {
			var err error
			healthy, err = dpi.CustomReportHealth(deviceID, absoluteDevicePath, healthy)
			if err != nil {
				logger.Reason(err).Warningf("failed to report health for device %s", absoluteDevicePath)
				healthy = false
			}
		}
		newHealthStatus := pluginapi.Unhealthy
		if healthy {
			newHealthStatus = pluginapi.Healthy
		}
		// only update the health if it is different from the current health or if this a new report
		if oldHealthStatus, exists := lastKnownHealth[absoluteDevicePath]; !exists || newHealthStatus != oldHealthStatus {
			lastKnownHealth[absoluteDevicePath] = newHealthStatus
			dpi.health <- deviceHealth{
				DevId:  deviceID,
				Health: newHealthStatus,
			}
		}
	}

	// Do initial health check by stat'ing the device paths
	for idDevicePath, deviceID := range monitoredDevices {

		// Stat the device path first to check if it exists
		_, err = os.Stat(idDevicePath)
		if err != nil {
			if errors.Is(err, os.ErrNotExist) {
				logger.Warningf("device %s is not present, waiting for it to be created", idDevicePath)
				reportHealth(deviceID, idDevicePath, false)
				continue
			} else {
				return fmt.Errorf("could not stat the device: %v", err)
			}
		}

		// Device exists, try to configure permissions before marking as healthy
		isHealthy := configurePermissionsAndReportSuccess(idDevicePath)
		if isHealthy {
			logger.Infof("device %s is present, marking device as healthy", idDevicePath)
		} else {
			logger.Warningf("device %s is present but permissions could not be configured, marking device as unhealthy", idDevicePath)
		}
		reportHealth(deviceID, idDevicePath, isHealthy)
	}

	// Loop and watch for device changes
	for {
		select {
		case <-dpi.stop:
			return nil
		case err := <-watcher.Errors:
			logger.Reason(err).Errorf("error watching devices and device plugin directory")
		case event := <-watcher.Events:
			logger.V(4).Infof("health Event: %v", event)
			if event.Name == deviceDirPath && event.Op == fsnotify.Create {
				// If event for the parent directory add a watcher for the device directory
				// This code path can only be triggered if parent directory was added to the watcher.
				logger.Infof("device directory %s was created, adding watcher", deviceDirPath)
				if err := watcher.Add(deviceDirPath); err != nil {
					// can happen if device was immediately removed after creation
					logger.Reason(err).Errorf("failed to add device directory to watcher")
				}
			} else if monDevId, exist := monitoredDevices[event.Name]; exist {
				// If the event is for a monitored device, update its health and fix permissions if needed.

				var friendlyName string
				if dpi.GetIDDeviceName == nil {
					friendlyName = "generic device"
				} else {
					friendlyName = dpi.GetIDDeviceName(monDevId)
				}

				// Health in this case is if the device path actually exists
				switch event.Op {
				case fsnotify.Create:
					logger.Infof("monitored device \"%s\" with resource %s appeared", friendlyName, dpi.resourceName)
					// Try to configure permissions before marking the device as healthy.
					isHealthy := configurePermissionsAndReportSuccess(event.Name)
					if isHealthy {
						logger.Infof("monitored device \"%s\" with resource %s is healthy", friendlyName, dpi.resourceName)
					} else {
						logger.Warningf("failed to configure permissions for monitored device \"%s\" with resource %s, marking as unhealthy", friendlyName, dpi.resourceName)
					}
					reportHealth(monDevId, event.Name, isHealthy)
				case fsnotify.Remove:
					logger.Infof("monitored device \"%s\" with resource %s was deleted, marking device as unhealthy", friendlyName, dpi.resourceName)
					reportHealth(monDevId, event.Name, false)
				case fsnotify.Rename:
					logger.Infof("monitored device \"%s\" with resource %s was renamed, marking device as unhealthy", friendlyName, dpi.resourceName)
					reportHealth(monDevId, event.Name, false)
				case fsnotify.Chmod:
					logger.Infof("monitored device \"%s\" with resource %s had its permissions modified", friendlyName, dpi.resourceName)
				}
			} else if event.Name == dpi.socketPath && event.Op == fsnotify.Remove {
				logger.Infof("device socket file for device %s was removed, kubelet probably restarted.", dpi.resourceName)
				return nil
			}
		}
	}
}

func (dpi *DevicePluginBase) PreStartContainer(_ context.Context, _ *pluginapi.PreStartContainerRequest) (*pluginapi.PreStartContainerResponse, error) {
	res := &pluginapi.PreStartContainerResponse{}
	return res, nil
}

func (dpi *DevicePluginBase) GetDevicePluginOptions(_ context.Context, _ *pluginapi.Empty) (*pluginapi.DevicePluginOptions, error) {
	options := &pluginapi.DevicePluginOptions{
		PreStartRequired: false,
	}
	return options, nil
}

func (dpi *DevicePluginBase) Allocate(ctx context.Context, r *pluginapi.AllocateRequest) (*pluginapi.AllocateResponse, error) {
	return nil, fmt.Errorf("not implemented")
}

func (dpi *DevicePluginBase) stopDevicePlugin() error {
	defer func() {
		if !IsChanClosed(dpi.done) {
			close(dpi.done)
		}
	}()
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

func (dpi *DevicePluginBase) cleanup() error {
	if err := os.Remove(dpi.socketPath); err != nil && !errors.Is(err, os.ErrNotExist) {
		return err
	}
	return nil
}

func (dpi *DevicePluginBase) GetInitialized() bool {
	dpi.lock.Lock()
	defer dpi.lock.Unlock()
	return dpi.initialized
}

func (dpi *DevicePluginBase) setInitialized(initialized bool) {
	dpi.lock.Lock()
	dpi.initialized = initialized
	dpi.lock.Unlock()
}

func (dpi *DevicePluginBase) register() error {
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
