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
	"bufio"
	"bytes"
	"context"
	"errors"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"

	"github.com/fsnotify/fsnotify"
	"google.golang.org/grpc"

	"kubevirt.io/client-go/log"

	"kubevirt.io/kubevirt/pkg/util"
	pluginapi "kubevirt.io/kubevirt/pkg/virt-handler/device-manager/deviceplugin/v1beta1"
)

const (
	CapacityPath = "/sys/fs/cgroup/misc.capacity"

	// Use the same resource names as CoCo's extended resources
	TDXResourceName = "tdx.intel.com/keys"
	SNPResourceName = "sev-snp.amd.com/esids"
)

type SecureGuestCapacityKeyType string

const (
	TDX_CAPACITY_KEY SecureGuestCapacityKeyType = "tdx"
	// SEV-SNP shares sev_es capacity in misc.capacity
	// Note the SEV/SEVES is not considered here because SNP is the target
	SNP_CAPACITY_KEY SecureGuestCapacityKeyType = "sev_es"
)

type SecureGuestType string

const (
	TDX SecureGuestType = "tdx"
	SNP SecureGuestType = "snp"
)

type SecureGuestCapacityDevicePlugin struct {
	*DevicePluginBase
	secureGuestType SecureGuestType
	capacity        int
}

func NewSecureGuestCapacityDevicePlugin() (*SecureGuestCapacityDevicePlugin, error) {
	return newSecureGuestCapacityDevicePlugin(filepath.Join(util.HostRootMount, CapacityPath))
}

func newSecureGuestCapacityDevicePlugin(capacityPath string) (*SecureGuestCapacityDevicePlugin, error) {
	capacityKey, capacity, err := detectSecureGuestCapacity(capacityPath)
	if err != nil {
		return nil, fmt.Errorf("Failed to detect secure guest capacity: %v", err)
	}

	if capacity == 0 {
		return nil, fmt.Errorf("No secure guest capacity available on this node")
	}

	var resourceName string
	var secureGuestType SecureGuestType
	switch capacityKey {
	case TDX_CAPACITY_KEY:
		resourceName = TDXResourceName
		secureGuestType = TDX
	case SNP_CAPACITY_KEY:
		resourceName = SNPResourceName
		secureGuestType = SNP
	}

	plugin := &SecureGuestCapacityDevicePlugin{
		DevicePluginBase: &DevicePluginBase{
			resourceName: resourceName,
			initialized:  false,
			socketPath:   SocketPath(strings.ReplaceAll(resourceName, "/", "-")),
			health:       make(chan deviceHealth),
			lock:         &sync.Mutex{},
			done:         make(chan struct{}),
			deregistered: make(chan struct{}),
		},
		secureGuestType: secureGuestType,
		capacity:        capacity,
	}

	plugin.devs = make([]*pluginapi.Device, capacity)
	for i := 0; i < capacity; i++ {
		plugin.devs[i] = &pluginapi.Device{
			ID:     fmt.Sprintf("secure-guest-slot-%d", i),
			Health: pluginapi.Healthy,
		}
	}

	return plugin, nil
}

func detectSecureGuestCapacity(capacityPath string) (SecureGuestCapacityKeyType, int, error) {
	logger := log.DefaultLogger()

	// Read /sys/fs/cgroup/misc.capacity to get the capacity, e.g.
	// "tdx 15" or "sev_es 99", here sev_es and snp share the same capacity 99
	content, err := os.ReadFile(capacityPath)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			logger.V(4).Infof("%s not found, secure guest capacity not available", capacityPath)
			return "", 0, nil
		}
		return "", 0, fmt.Errorf("Failed to read misc.capacity: %v", err)
	}

	capacity := 0
	capacityKey := SecureGuestCapacityKeyType("")
	scanner := bufio.NewScanner(bytes.NewReader(content))

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}

		fields := strings.Fields(line)
		if len(fields) != 2 {
			continue
		}

		tmpKey := SecureGuestCapacityKeyType(fields[0])
		if tmpKey != TDX_CAPACITY_KEY && tmpKey != SNP_CAPACITY_KEY {
			logger.V(4).Infof("Skipped unsupported secure guest capacity key: %s", tmpKey)
			continue
		}
		capacityKey = tmpKey

		capacity, err = strconv.Atoi(fields[1])
		if err != nil {
			logger.Warningf("Failed to read capacity for %s: %v", capacityKey, err)
			break
		}

		if capacity > 0 {
			logger.V(4).Infof("Detected secure guest capacity key %s with capacity %d", capacityKey, capacity)
			break
		}
	}

	if err := scanner.Err(); err != nil {
		err = fmt.Errorf("Error parsing misc.capacity: %v", err)
	}

	return capacityKey, capacity, err
}

func (plugin *SecureGuestCapacityDevicePlugin) Start(stop <-chan struct{}) error {
	logger := log.DefaultLogger()
	plugin.stop = stop

	err := plugin.cleanup()
	if err != nil {
		return err
	}

	sock, err := net.Listen("unix", plugin.socketPath)
	if err != nil {
		return fmt.Errorf("error creating GRPC server socket: %v", err)
	}

	plugin.server = grpc.NewServer([]grpc.ServerOption{}...)
	defer plugin.stopDevicePlugin()

	pluginapi.RegisterDevicePluginServer(plugin.server, plugin)

	errChan := make(chan error, 2)

	go func() {
		errChan <- plugin.server.Serve(sock)
	}()

	err = waitForGRPCServer(plugin.socketPath, connectionTimeout)
	if err != nil {
		return fmt.Errorf("error starting the GRPC server: %v", err)
	}

	err = plugin.register()
	if err != nil {
		return fmt.Errorf("error registering with device plugin manager: %v", err)
	}

	go func() {
		errChan <- plugin.healthCheck()
	}()

	plugin.setInitialized(true)
	logger.Infof("%s device plugin started", plugin.resourceName)
	err = <-errChan

	return err
}

func (plugin *SecureGuestCapacityDevicePlugin) ListAndWatch(e *pluginapi.Empty, s pluginapi.DevicePlugin_ListAndWatchServer) error {
	logger := log.DefaultLogger()

	logger.Infof("Advertising %d secure guest slots (type=%s, resource=%s)",
		plugin.capacity, plugin.secureGuestType, plugin.resourceName)

	if err := s.Send(&pluginapi.ListAndWatchResponse{Devices: plugin.devs}); err != nil {
		return fmt.Errorf("failed to send device list: %v", err)
	}

	// No need to check the devices' health here
	select {
	case <-plugin.stop:
	case <-plugin.done:
	}

	emptyList := []*pluginapi.Device{}
	if err := s.Send(&pluginapi.ListAndWatchResponse{Devices: emptyList}); err != nil {
		logger.Reason(err).Infof("%s device plugin failed to deregister", plugin.resourceName)
	}
	close(plugin.deregistered)
	return nil
}

func (plugin *SecureGuestCapacityDevicePlugin) Allocate(ctx context.Context, r *pluginapi.AllocateRequest) (*pluginapi.AllocateResponse, error) {
	logger := log.DefaultLogger()
	response := &pluginapi.AllocateResponse{}

	for _, req := range r.ContainerRequests {
		logger.Infof("Allocating %d secure guest slots (type=%s, resource=%s, IDs=%v)",
			len(req.DevicesIDs), plugin.secureGuestType, plugin.resourceName, req.DevicesIDs)

		containerResponse := &pluginapi.ContainerAllocateResponse{}
		response.ContainerResponses = append(response.ContainerResponses, containerResponse)
	}

	return response, nil
}

// Monitor the socket path only because there is no real device to be monitored
func (plugin *SecureGuestCapacityDevicePlugin) healthCheck() error {
	logger := log.DefaultLogger()
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return fmt.Errorf("Failed to create a fsnotify watcher: %v", err)
	}
	defer watcher.Close()

	dirName := filepath.Dir(plugin.socketPath)
	if err = watcher.Add(dirName); err != nil {
		return fmt.Errorf("Failed to add the device-plugin kubelet path to the watcher: %v", err)
	}

	if _, err = os.Stat(plugin.socketPath); err != nil {
		return fmt.Errorf("Failed to stat the device-plugin socket: %v", err)
	}

	for {
		select {
		case <-plugin.stop:
			return nil
		case err := <-watcher.Errors:
			logger.Reason(err).Errorf("Error watching device plugin socket directory")
		case event := <-watcher.Events:
			logger.V(4).Infof("health Event: %v", event)
			if event.Name == plugin.socketPath && event.Op == fsnotify.Remove {
				logger.Infof("Device plugin socket file for %s was removed, kubelet probably restarted.", plugin.resourceName)
				return nil
			}
		}
	}
}
