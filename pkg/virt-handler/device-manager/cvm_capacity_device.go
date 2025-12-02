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
	"strconv"
	"strings"
	"sync"

	"google.golang.org/grpc"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/client-go/tools/cache"

	"kubevirt.io/client-go/log"
	pluginapi "kubevirt.io/kubevirt/pkg/virt-handler/device-manager/deviceplugin/v1beta1"
)

const (
	CapacityPath = "/sys/fs/cgroup/misc.capacity"

	// Use the same resource names as CoCo's extended resources
	TDXResourceName    = "tdx.intel.com/keys"
	SEVSNPResourceName = "sev-snp.amd.com/esids"
)

type SecureGuestType string

const (
	TDX SecureGuestType = "tdx"
	SNP SecureGuestType = "sev_es" // SEV-SNP shares sev_es capacity in misc.capacity
)

type SecureGuestCapacityDevicePlugin struct {
	*DevicePluginBase
	secureGuestType SecureGuestType
	capacity        int
	isManaged       bool
	nodeStore       cache.Store
	nodeName        string
}

func NewSecureGuestCapacityDevicePlugin(nodeStore cache.Store, nodeName string) (*SecureGuestCapacityDevicePlugin, error) {
	logger := log.DefaultLogger()

	tempPlugin := &SecureGuestCapacityDevicePlugin{}

	cvmType, capacity, err := tempPlugin.detectSecureGuestCapacity()
	if err != nil {
		return nil, fmt.Errorf("failed to detect secure guest capacity: %v", err)
	}

	if capacity == 0 {
		return nil, fmt.Errorf("no secure guest capacity available on this node")
	}

	var resourceName string
	switch cvmType {
	case TDX:
		resourceName = TDXResourceName
	case SNP:
		resourceName = SEVSNPResourceName
	default:
		return nil, fmt.Errorf("unsupported secure guest type: %s", cvmType)
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
		nodeStore:       nodeStore,
		nodeName:        nodeName,
		isManaged:       false,
		secureGuestType: cvmType,
		capacity:        capacity,
	}

	// Check if the resources has been managed by others, e.g. CoCo
	isexisted, err := plugin.isResourceExisted()
	if err != nil {
		logger.Warningf("Failed to check if CoCo is managing secure guest capacity: %v", err)
	}

	if isexisted {
		logger.Infof("Secure guest capacity already existed, KubeVirt will not advertise %s", resourceName)
		plugin.isManaged = false
	} else {
		logger.Infof("Secure guest capacity device plugin will be managed by KubeVirt: type=%s, capacity=%d, resource=%s",
			cvmType, capacity, resourceName)
		plugin.isManaged = true
	}

	return plugin, nil
}

func (plugin *SecureGuestCapacityDevicePlugin) detectSecureGuestCapacity() (SecureGuestType, int, error) {
	logger := log.DefaultLogger()

	// Read /sys/fs/cgroup/misc.capacity to get the capacity, e.g.
	// "tdx 15" or "sev_es 99"
	content, err := os.ReadFile(CapacityPath)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			logger.V(4).Infof("%s not found, secure guest capacity not available", CapacityPath)
			return "", 0, nil
		}
		return "", 0, fmt.Errorf("Failed to read misc.capacity: %v", err)
	}

	capacities := make(map[SecureGuestType]int)
	scanner := bufio.NewScanner(bytes.NewReader(content))

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}

		fields := strings.Fields(line)
		if len(fields) != 2 {
			logger.V(4).Infof("Skipping invalid misc.capacity line: %s", line)
			continue
		}

		cvmType := SecureGuestType(fields[0])
		if cvmType != TDX && cvmType != SNP {
			logger.V(4).Infof("Skipping unsupported secure guest type: %s", cvmType)
			continue
		}

		capacity, err := strconv.Atoi(fields[1])
		if err != nil {
			logger.Warningf("Failed to parse capacity for %s: %v", cvmType, err)
			continue
		}

		if capacity > 0 {
			capacities[cvmType] = capacity
			logger.V(4).Infof("Detected secure guest type %s with capacity %d", cvmType, capacity)
		}
	}

	if err := scanner.Err(); err != nil {
		return "", 0, fmt.Errorf("error parsing misc.capacity: %v", err)
	}

	if cap, ok := capacities[TDX]; ok {
		return TDX, cap, nil
	}
	if cap, ok := capacities[SNP]; ok {
		return SNP, cap, nil
	}

	return "", 0, nil
}

func (plugin *SecureGuestCapacityDevicePlugin) GetInitialized() bool {
	return plugin.capacity > 0
}

func (plugin *SecureGuestCapacityDevicePlugin) Start(stop <-chan struct{}) error {
	logger := log.DefaultLogger()
	plugin.stop = stop

	// Don't start it because the secure guest capacity has already existed
	if !plugin.isManaged {
		logger.V(3).Infof("Skipping secure guest capacity device plugin: resource=%s existed",
			plugin.resourceName)
		<-stop
		return nil
	}

	logger.Infof("Starting secure guest capacity device plugin: resource=%s, type=%s, capacity=%d",
		plugin.resourceName, plugin.secureGuestType, plugin.capacity)

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

	plugin.devs = make([]*pluginapi.Device, plugin.capacity)

	for i := 0; i < plugin.capacity; i++ {
		plugin.devs[i] = &pluginapi.Device{
			ID:     fmt.Sprintf("secure-guest-slot-%d", i),
			Health: pluginapi.Healthy,
		}
	}

	logger.V(4).Infof("Advertising %d secure guest slots (type=%s, resource=%s)",
		plugin.capacity, plugin.secureGuestType, plugin.resourceName)

	if err := s.Send(&pluginapi.ListAndWatchResponse{Devices: plugin.devs}); err != nil {
		return fmt.Errorf("failed to send device list: %v", err)
	}

	<-plugin.stop

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

func (plugin *SecureGuestCapacityDevicePlugin) isResourceExisted() (bool, error) {
	existingResource, err := plugin.getExistingResourceName()
	return existingResource != "", err
}

func (plugin *SecureGuestCapacityDevicePlugin) getExistingResourceName() (string, error) {
	logger := log.DefaultLogger()

	nodeObj, exists, err := plugin.nodeStore.GetByKey(plugin.nodeName)
	if err != nil {
		return "", fmt.Errorf("failed to get node %s from store: %v", plugin.nodeName, err)
	}
	if !exists {
		return "", fmt.Errorf("node %s does not exist in store", plugin.nodeName)
	}

	node, ok := nodeObj.(*corev1.Node)
	if !ok {
		return "", fmt.Errorf("object is not a Node")
	}

	resourceMap := map[SecureGuestType]string{
		TDX: TDXResourceName,
		SNP: SEVSNPResourceName,
	}

	if resource, exists := resourceMap[plugin.secureGuestType]; exists {
		if capacity, found := node.Status.Capacity[corev1.ResourceName(resource)]; found {
			logger.V(4).Infof("Node %s has resource %s with capacity: %s",
				plugin.nodeName, resource, capacity.String())
			return resource, nil
		}
	}

	return "", nil
}
