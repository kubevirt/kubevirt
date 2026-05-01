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
	"strings"
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

type devicePluginContract interface {
	setupMonitoredDevices(watcher *fsnotify.Watcher, monitoredDevices map[string]string) error   // REQUIRED function to set up the devices that are being monitored and update map such that key contains absolute paths to watch, and value contains the device id that path corresponds to.
	allocateDP(context.Context, *pluginapi.AllocateRequest) (*pluginapi.AllocateResponse, error) // REQUIRED function to allocate the device.
	setupDevicePlugin() error                                                                    // Optional function to perform additional setup steps that are not covered by the default implementation
	deviceNameByID(deviceID string) string                                                       // Optional function to convert device id to a human-readable name for logging
	configurePermissions(absoluteDevicePath *safepath.Path) error                                // Optional function to configure permissions for the device if needed. When present, device being marked healthy is contingent on the hook exiting without error.
	updateHealth(deviceID string, absoluteDevicePath string, healthy bool) (bool, error)         // Optional function to update the device health before it's sent via custom logic.

	getResourceName() string
	getDevices() []*pluginapi.Device
	getDevicePath() string
	getDeviceRoot() string
	getSocketPath() string
}

type DevicePluginBase struct {
	devs         []*pluginapi.Device
	socketPath   string
	resourceName string
	deviceRoot   string // Absolute base path for where this DP is inside virt-handler (typically intended to be either "/" or util.HostRootMount)
	devicePath   string // Device path on the host filesystem. When accessed from a virt-handler, it should be combined with deviceRoot.
}

// Optional, can be overridden
func (dpi *DevicePluginBase) setupDevicePlugin() error {
	return nil
}

// Optional, can be overridden
func (dpi *DevicePluginBase) deviceNameByID(deviceID string) string {
	return "device plugin (" + deviceID + ")"
}

// Optional, can be overridden
func (dpi *DevicePluginBase) configurePermissions(_ *safepath.Path) error {
	return nil
}

// Optional, can be overridden
func (dpi *DevicePluginBase) updateHealth(_ string, _ string, healthy bool) (bool, error) {
	return healthy, nil
}

func (dpi *DevicePluginBase) getResourceName() string {
	return dpi.resourceName
}

func (dpi *DevicePluginBase) getDevices() []*pluginapi.Device {
	return dpi.devs
}

func (dpi *DevicePluginBase) getDevicePath() string {
	return dpi.devicePath
}

func (dpi *DevicePluginBase) getDeviceRoot() string {
	return dpi.deviceRoot
}

func (dpi *DevicePluginBase) getSocketPath() string {
	return dpi.socketPath
}

type DevicePlugin struct {
	server              *grpc.Server
	stop                <-chan struct{}
	healthUpdateChan    chan struct{}
	done                chan struct{}
	initialized         bool
	skipDupHealthChecks bool // Should we skip propogating health updates if nothing changed (performance optimization; set to false only for tests)
	lock                *sync.Mutex
	deregistered        chan struct{} // Device path on the host filesystem. When accessed from a virt-handler, it should be combined with deviceRoot.
	contract            devicePluginContract
}

type healthCheckContext struct {
	// key: device path, value: corresponding device ID
	// Used to track the devices that are being monitored
	// When a corresponding device ID is empty (i.e. "") it means this device path represents ALL device IDs
	monitoredDevices map[string]string
	// key: device id, value: last known health
	// used to track the health of the devices
	lastKnownHealth map[string]string
	// watcher for the device plugin socket and the device directory
	watcher *fsnotify.Watcher
	// parent dirs of monitored devices; used for fast lookups on Create events to decide whether to add to watcher
	parentDirsToWatch map[string]struct{}
}

func (dpi *DevicePlugin) GetResourceName() string {
	return dpi.contract.getResourceName()
}

func newDevicePlugin(contract devicePluginContract) *DevicePlugin {
	dpi := &DevicePlugin{
		contract:            contract,
		initialized:         false,
		lock:                &sync.Mutex{},
		healthUpdateChan:    make(chan struct{}, 1),
		done:                make(chan struct{}),
		deregistered:        make(chan struct{}),
		skipDupHealthChecks: true,
	}
	return dpi
}

func (dpi *DevicePlugin) Start(stop <-chan struct{}) (err error) {
	logger := log.DefaultLogger()
	dpi.stop = stop
	dpi.done = make(chan struct{})
	dpi.deregistered = make(chan struct{})

	if err := dpi.cleanup(); err != nil {
		return err
	}

	// If a custom setupDevicePlugin hook is implemented, call it
	// for additional setup steps that are not covered by the default implementation
	if err = dpi.contract.setupDevicePlugin(); err != nil {
		return err
	}

	sock, err := net.Listen("unix", dpi.contract.getSocketPath())
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

	if err = waitForGRPCServer(dpi.contract.getSocketPath(), connectionTimeout); err != nil {
		return fmt.Errorf("error starting the GRPC server: %v", err)
	}

	if err = dpi.register(); err != nil {
		return fmt.Errorf("error registering with device plugin manager: %v", err)
	}

	// synchronously setup the health check context before
	// we mark device initialized so we don't miss any events
	healthCheckContext, err := dpi.setupHealthCheckContext()
	if err != nil {
		return fmt.Errorf("error setting up health check context: %v", err)
	}

	go func() {
		errChan <- dpi.healthCheck(healthCheckContext)
	}()

	dpi.setInitialized(true)
	logger.Infof("%s device plugin started", dpi.contract.getResourceName())
	err = <-errChan

	return err
}

func (dpi *DevicePlugin) ListAndWatch(_ *pluginapi.Empty, s pluginapi.DevicePlugin_ListAndWatchServer) error {
	if err := s.Send(&pluginapi.ListAndWatchResponse{Devices: dpi.cloneDevs()}); err != nil {
		return fmt.Errorf("error sending initial ListAndWatchResponse: %v", err)
	}

	done := false
	for {
		select {
		case <-dpi.healthUpdateChan:
			if err := s.Send(&pluginapi.ListAndWatchResponse{Devices: dpi.cloneDevs()}); err != nil {
				return fmt.Errorf("error sending ListAndWatchResponse: %v", err)
			}
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
		log.DefaultLogger().Reason(err).Infof("%s device plugin failed to deregister", dpi.contract.getResourceName())
	}
	close(dpi.deregistered)
	return nil
}

// clones devices to avoid hoarding lock if kubelet does not respond immediately
// to the gRPC allowing health checks to continue.
func (dpi *DevicePlugin) cloneDevs() []*pluginapi.Device {
	dpi.lock.Lock()
	defer dpi.lock.Unlock()

	devs := make([]*pluginapi.Device, 0, len(dpi.contract.getDevices()))
	for _, dev := range dpi.contract.getDevices() {
		copiedDev := *dev
		devs = append(devs, &copiedDev)
	}
	return devs
}

func (dpi *DevicePlugin) setupHealthCheckContext() (healthCheckContext, error) {
	healthCtx := healthCheckContext{
		monitoredDevices:  make(map[string]string),
		lastKnownHealth:   make(map[string]string),
		watcher:           nil,
		parentDirsToWatch: make(map[string]struct{}),
	}

	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return healthCtx, fmt.Errorf("failed to create a fsnotify watcher: %v", err)
	}
	healthCtx.watcher = watcher

	// Set up monitored device paths
	// Should watch before stat'ing the device path to avoid race conditions
	if err = dpi.contract.setupMonitoredDevices(watcher, healthCtx.monitoredDevices); err != nil {
		watcher.Close()
		return healthCtx, err
	}

	// Precompute parent dirs of all monitored paths for O(1) Create-event handling.
	for monitoredPath := range healthCtx.monitoredDevices {
		dir := filepath.Dir(monitoredPath)
		for {
			healthCtx.parentDirsToWatch[dir] = struct{}{}
			parent := filepath.Dir(dir)
			if parent == "/" {
				break
			}
			dir = parent
		}
	}

	// Check the device plugin socket to ensure we can communicate with it
	dirName := filepath.Dir(dpi.contract.getSocketPath())
	if err = watcher.Add(dirName); err != nil {
		watcher.Close()
		return healthCtx, fmt.Errorf("failed to add the device-plugin kubelet path to the watcher: %v", err)
	}
	if _, err = os.Stat(dpi.contract.getSocketPath()); err != nil {
		watcher.Close()
		return healthCtx, fmt.Errorf("failed to stat the device-plugin socket: %v", err)
	}

	return healthCtx, nil
}

func (dpi *DevicePlugin) healthCheck(healthCtx healthCheckContext) error {
	logger := log.DefaultLogger()
	defer healthCtx.watcher.Close()

	// Do an initial health check by stat'ing all device paths
	if err := dpi.doStaticHealthCheck(healthCtx.monitoredDevices, healthCtx.lastKnownHealth, ""); err != nil {
		return err
	}

	// Loop and watch for device changes
	for {
		select {
		case <-dpi.stop:
			return nil
		case err := <-healthCtx.watcher.Errors:
			logger.Reason(err).Errorf("error watching devices and device plugin directory")
		case event := <-healthCtx.watcher.Events:
			logger.V(4).Infof("health Event: %v", event)
			if monDevId, exist := healthCtx.monitoredDevices[event.Name]; exist {
				friendlyName := dpi.contract.deviceNameByID(monDevId)
				switch event.Op {
				case fsnotify.Create:
					logger.Infof("monitored device '%s' with resource %s appeared", friendlyName, dpi.contract.getResourceName())
					// Try to configure permissions before marking the device as healthy.
					succeeded := dpi.configurePermissionsAndReportSuccess(event.Name)
					if !succeeded {
						logger.Warningf("failed to configure permissions for monitored device '%s' with resource %s", friendlyName, dpi.contract.getResourceName())
					}
					dpi.reportHealth(friendlyName, monDevId, event.Name, succeeded, healthCtx.lastKnownHealth)
				case fsnotify.Remove:
					logger.Infof("monitored device '%s' with resource %s was deleted", friendlyName, dpi.contract.getResourceName())
					dpi.reportHealth(friendlyName, monDevId, event.Name, false, healthCtx.lastKnownHealth)
				case fsnotify.Rename:
					logger.Infof("monitored device '%s' with resource %s was renamed", friendlyName, dpi.contract.getResourceName())
					dpi.reportHealth(friendlyName, monDevId, event.Name, false, healthCtx.lastKnownHealth)
				case fsnotify.Chmod:
					logger.Infof("monitored device '%s' with resource %s had its permissions modified", friendlyName, dpi.contract.getResourceName())
				}
			} else if event.Op == fsnotify.Create {
				// If the created path is a parent of any monitored device, add it to the watcher
				// so we receive events when the device node appears (e.g. USB bus dir created after plugin start).
				if _, isParentDir := healthCtx.parentDirsToWatch[event.Name]; isParentDir {
					if addErr := healthCtx.watcher.Add(event.Name); addErr != nil {
						logger.Reason(addErr).Warningf("failed to add directory %s to watcher", event.Name)
					}
					// It's possible we missed events under this path if the events occurred before we could
					// add this directory to the watcher so we must manually stat devices.
					_ = dpi.doStaticHealthCheck(healthCtx.monitoredDevices, healthCtx.lastKnownHealth, event.Name)
				}
			} else if event.Name == dpi.contract.getSocketPath() && event.Op == fsnotify.Remove {
				logger.Infof("device socket file for device '%s' was removed, kubelet probably restarted.", dpi.contract.getResourceName())
				return nil
			}
		}
	}
}

func (dpi *DevicePlugin) PreStartContainer(_ context.Context, _ *pluginapi.PreStartContainerRequest) (*pluginapi.PreStartContainerResponse, error) {
	res := &pluginapi.PreStartContainerResponse{}
	return res, nil
}

func (dpi *DevicePlugin) GetDevicePluginOptions(_ context.Context, _ *pluginapi.Empty) (*pluginapi.DevicePluginOptions, error) {
	options := &pluginapi.DevicePluginOptions{
		PreStartRequired: false,
	}
	return options, nil
}

func (dpi *DevicePlugin) Allocate(ctx context.Context, r *pluginapi.AllocateRequest) (*pluginapi.AllocateResponse, error) {
	return dpi.contract.allocateDP(ctx, r)
}

func (dpi *DevicePlugin) stopDevicePlugin() error {
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

func (dpi *DevicePlugin) cleanup() error {
	if err := os.Remove(dpi.contract.getSocketPath()); err != nil && !errors.Is(err, os.ErrNotExist) {
		return err
	}
	return nil
}

func (dpi *DevicePlugin) GetInitialized() bool {
	dpi.lock.Lock()
	defer dpi.lock.Unlock()
	return dpi.initialized
}

func (dpi *DevicePlugin) getDevHealthByIndex(index int) string {
	dpi.lock.Lock()
	defer dpi.lock.Unlock()
	if index < 0 || index >= len(dpi.contract.getDevices()) {
		return ""
	}
	return dpi.contract.getDevices()[index].Health
}

func (dpi *DevicePlugin) setInitialized(initialized bool) {
	dpi.lock.Lock()
	dpi.initialized = initialized
	dpi.lock.Unlock()
}

func (dpi *DevicePlugin) configurePermissionsAndReportSuccess(absoluteDevicePath string) bool {
	logger := log.DefaultLogger()

	logger.V(4).Infof("ensuring permissions for device %s", absoluteDevicePath)
	// Since absoluteDevicePath = deviceRoot + devicePath
	relDevicePath, err := filepath.Rel(dpi.contract.getDeviceRoot(), absoluteDevicePath)
	if err != nil {
		logger.Reason(err).Warningf("failed to get relative path for device %s", absoluteDevicePath)
		return false
	}
	// We derive relDevicePath to be able to enforce containment with JoinAndResolveWithRelativeRoot
	dp, err := safepath.JoinAndResolveWithRelativeRoot(dpi.contract.getDeviceRoot(), relDevicePath)
	if err != nil {
		logger.Reason(err).Warningf("failed to create safepath for device %s", absoluteDevicePath)
		return false
	}
	if err := dpi.contract.configurePermissions(dp); err != nil {
		logger.Reason(err).Warningf("failed to ensure permissions for device %s", absoluteDevicePath)
		return false
	}
	return true
}

func (dpi *DevicePlugin) notifyHealthUpdate() {
	select {
	case dpi.healthUpdateChan <- struct{}{}:
	default:
		// already a signal pending, discard update
	}
}

func (dpi *DevicePlugin) reportHealth(devFriendlyName string, deviceID string, absoluteDevicePath string, healthy bool, lastKnownHealth map[string]string) {
	logger := log.DefaultLogger()
	healthy, err := dpi.contract.updateHealth(deviceID, absoluteDevicePath, healthy)
	if err != nil {
		logger.Reason(err).Warningf("An error occurred while attempting to mutate health update for %s", devFriendlyName)
		healthy = false
	}
	newHealthStatus := pluginapi.Unhealthy
	if healthy {
		newHealthStatus = pluginapi.Healthy
	}
	logger.V(4).Infof("Attempting to update health status for device %s: %s", devFriendlyName, newHealthStatus)
	// only update the health if it is different from the current health or if this a new report
	if oldHealthStatus, exists := lastKnownHealth[deviceID]; !dpi.skipDupHealthChecks || !exists || newHealthStatus != oldHealthStatus {
		lastKnownHealth[deviceID] = newHealthStatus
		if newHealthStatus == pluginapi.Healthy {
			logger.Infof("device %s is now healthy", devFriendlyName)
		} else {
			logger.Warningf("device %s is now unhealthy", devFriendlyName)
		}

		dpi.lock.Lock()
		for _, dev := range dpi.contract.getDevices() {
			// If the devHealth.DevId is empty, it was not set by the device plugin, so we update all devices
			if deviceID == dev.ID || deviceID == "" {
				dev.Health = newHealthStatus
			}
		}
		dpi.lock.Unlock()
		dpi.notifyHealthUpdate()
	}
}

// doStaticHealthCheck stats device paths and reports health. scope limits which devices are checked:
// empty scope means all monitored devices; non-empty scope means only devices under that directory.
func (dpi *DevicePlugin) doStaticHealthCheck(monitoredDevices map[string]string, lastKnownHealth map[string]string, scope string) error {
	logger := log.DefaultLogger()
	scopePrefix := scope
	if scope != "" {
		scopePrefix = filepath.Clean(scope) + string(filepath.Separator)
	}
	for idDevicePath, deviceID := range monitoredDevices {
		if scope != "" && idDevicePath != scope && !strings.HasPrefix(filepath.Clean(idDevicePath), scopePrefix) {
			continue
		}
		friendlyName := dpi.contract.deviceNameByID(deviceID)
		// Stat the device path first to check if it exists
		_, err := os.Stat(idDevicePath)
		if err != nil {
			if errors.Is(err, os.ErrNotExist) {
				logger.Warningf("device '%s' is not present at '%s', waiting for it to be created", friendlyName, idDevicePath)
				dpi.reportHealth(friendlyName, deviceID, idDevicePath, false, lastKnownHealth)
				continue
			} else {
				return fmt.Errorf("could not stat the device '%s': %v", friendlyName, err)
			}
		}

		// Device exists, try to configure permissions before marking as healthy
		succeeded := dpi.configurePermissionsAndReportSuccess(idDevicePath)
		if succeeded {
			logger.Infof("device '%s' is present", idDevicePath)
		} else {
			logger.Warningf("device '%s' is present but permissions could not be configured", idDevicePath)
		}
		dpi.reportHealth(friendlyName, deviceID, idDevicePath, succeeded, lastKnownHealth)
	}
	return nil
}

func (dpi *DevicePlugin) register() error {
	conn, err := gRPCConnect(pluginapi.KubeletSocket, connectionTimeout)
	if err != nil {
		return err
	}
	defer conn.Close()

	client := pluginapi.NewRegistrationClient(conn)
	reqt := &pluginapi.RegisterRequest{
		Version:      pluginapi.Version,
		Endpoint:     path.Base(dpi.contract.getSocketPath()),
		ResourceName: dpi.contract.getResourceName(),
	}

	_, err = client.Register(context.Background(), reqt)
	if err != nil {
		return err
	}
	return nil
}
