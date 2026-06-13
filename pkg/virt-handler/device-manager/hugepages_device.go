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
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/fsnotify/fsnotify"
	"google.golang.org/grpc"

	"kubevirt.io/client-go/log"

	"kubevirt.io/kubevirt/pkg/util"
	pluginapi "kubevirt.io/kubevirt/pkg/virt-handler/device-manager/deviceplugin/v1beta1"
)

const (
	dynamicHugepageResourcePrefix = "dynamic-hugepages"

	// Kernel sysfs path (relative to host root).
	hugepagesSysfsBase = "sys/kernel/mm/hugepages"

	procCmdline = "proc/cmdline"

	overcommitFile = "nr_overcommit_hugepages"
	nrFile         = "nr_hugepages"

	// Pages with order > giganticOrderThreshold require CMA for runtime
	// allocation. Order = log2(page_size_bytes / 4096).
	// Order 10 = 4MiB (still buddy-allocatable). 1Gi = order 18 (CMA).
	giganticOrderThreshold = 10
	basePageSizeBytes      = 4096
)

// HugepagesDevicePlugin exposes dynamically-allocatable hugepages as a
// Kubernetes device plugin resource. One instance per page size.
// Each "device" represents one hugepage of that size.
//
// For non-gigantic pages (order < 10, e.g. 2Mi): capacity comes from
// nr_overcommit_hugepages (buddy allocator surplus).
//
// For gigantic pages (order >= 10, e.g. 1Gi): capacity comes from
// hugetlb_cma reservation and requires both hugetlb_cma and
// hugetlb_cma_only kernel params.
type HugepagesDevicePlugin struct {
	devs         []*pluginapi.Device
	server       *grpc.Server
	socketPath   string
	stop         <-chan struct{}
	health       chan deviceHealth
	resourceName string
	done         chan struct{}
	initialized  bool
	lock         *sync.Mutex
	deregistered chan struct{}

	pageSizeKB int64  // page size in kB (e.g., 2048, 1048576)
	pageLabel  string // normalized label (e.g., "2mi", "1gi")
	sysfsDir   string // kernel directory name (e.g., "hugepages-2048kB")
	hostRoot   string // host filesystem root
}

// DiscoverHugepageDevicePlugins scans the host for dynamically-allocatable
// hugepage capacity and returns a device plugin for each supported page size.
//
//   - Non-gigantic pages (order < 10): enabled if nr_overcommit_hugepages > 0
//   - Gigantic pages (order >= 10): enabled only if hugetlb_cma AND
//     hugetlb_cma_only are both set in kernel cmdline
func DiscoverHugepageDevicePlugins(hostRoot string) []*HugepagesDevicePlugin {
	if hostRoot == "" {
		hostRoot = util.HostRootMount
	}

	cmaConfig := detectCMAConfig(hostRoot)

	sysfsBase := filepath.Join(hostRoot, hugepagesSysfsBase)
	entries, err := os.ReadDir(sysfsBase)
	if err != nil {
		log.DefaultLogger().V(4).Infof("failed to read hugepages sysfs at %s: %v", sysfsBase, err)
		return nil
	}

	var plugins []*HugepagesDevicePlugin
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		dirName := entry.Name()
		pageSizeKB, ok := parseHugepageDirName(dirName)
		if !ok {
			continue
		}

		plugin := newDynamicHugepagesPlugin(pageSizeKB, dirName, hostRoot, cmaConfig)
		if plugin == nil || len(plugin.devs) == 0 {
			continue
		}
		plugins = append(plugins, plugin)
	}

	return plugins
}

// cmaConfiguration holds the parsed hugetlb CMA kernel parameters.
type cmaConfiguration struct {
	sizeMiB int64 // hugetlb_cma size in MiB (0 if not set)
	cmaOnly bool  // hugetlb_cma_only is present
}

// detectCMAConfig reads the kernel command line for hugetlb_cma= and
// hugetlb_cma_only parameters.
func detectCMAConfig(hostRoot string) cmaConfiguration {
	cmdlinePath := filepath.Join(hostRoot, procCmdline)
	data, err := os.ReadFile(cmdlinePath)
	if err != nil {
		log.DefaultLogger().V(4).Infof("failed to read %s: %v", cmdlinePath, err)
		return cmaConfiguration{}
	}

	var config cmaConfiguration
	for _, param := range strings.Fields(string(data)) {
		if strings.HasPrefix(param, "hugetlb_cma=") {
			sizeStr := strings.TrimPrefix(param, "hugetlb_cma=")
			config.sizeMiB = parseMemorySize(sizeStr)
		}
		if param == "hugetlb_cma_only" {
			config.cmaOnly = true
		}
	}

	return config
}

// isGigantic returns true if the page size requires CMA for runtime allocation
// (order > 10, i.e., page size > 4MiB with 4kB base pages).
func isGigantic(pageSizeKB int64) bool {
	pageSizeBytes := pageSizeKB * 1024
	order := 0
	for s := int64(basePageSizeBytes); s < pageSizeBytes; s <<= 1 {
		order++
	}
	return order > giganticOrderThreshold
}

func newDynamicHugepagesPlugin(pageSizeKB int64, sysfsDir, hostRoot string, cma cmaConfiguration) *HugepagesDevicePlugin {
	pageLabel := pageSizeToLabel(pageSizeKB)
	deviceName := fmt.Sprintf("%s-%s", dynamicHugepageResourcePrefix, pageLabel)

	var capacity int64

	if isGigantic(pageSizeKB) {
		// Gigantic pages require both hugetlb_cma and hugetlb_cma_only
		if cma.sizeMiB <= 0 || !cma.cmaOnly {
			log.DefaultLogger().V(4).Infof(
				"skipping gigantic page size %s: requires hugetlb_cma and hugetlb_cma_only (cma=%dMiB, cma_only=%v)",
				pageLabel, cma.sizeMiB, cma.cmaOnly)
			return nil
		}
		pageSizeMiB := pageSizeKB / 1024
		capacity = cma.sizeMiB / pageSizeMiB
	} else {
		// Non-gigantic pages: use nr_overcommit_hugepages from sysfs
		sysfsPath := filepath.Join(hostRoot, hugepagesSysfsBase, sysfsDir)
		overcommit := readSysfsInt(filepath.Join(sysfsPath, overcommitFile))
		if overcommit <= 0 {
			log.DefaultLogger().V(4).Infof(
				"skipping page size %s: nr_overcommit_hugepages is 0", pageLabel)
			return nil
		}
		capacity = overcommit
	}

	if capacity <= 0 {
		return nil
	}

	dpi := &HugepagesDevicePlugin{
		devs:         make([]*pluginapi.Device, 0, capacity),
		socketPath:   SocketPath(deviceName),
		health:       make(chan deviceHealth),
		resourceName: fmt.Sprintf("%s/%s", DeviceNamespace, deviceName),
		initialized:  false,
		lock:         &sync.Mutex{},
		pageSizeKB:   pageSizeKB,
		pageLabel:    pageLabel,
		sysfsDir:     sysfsDir,
		hostRoot:     hostRoot,
	}

	for i := int64(0); i < capacity; i++ {
		dpi.devs = append(dpi.devs, &pluginapi.Device{
			ID:     fmt.Sprintf("%s-%d", deviceName, i),
			Health: pluginapi.Healthy,
		})
	}

	return dpi
}

// parseMemorySize parses a kernel memory size string like "16G", "4096M",
// "8388608K" into MiB.
func parseMemorySize(s string) int64 {
	s = strings.TrimSpace(s)
	if len(s) == 0 {
		return 0
	}

	suffix := s[len(s)-1]
	numStr := s[:len(s)-1]

	switch suffix {
	case 'G', 'g':
		val, err := strconv.ParseInt(numStr, 10, 64)
		if err != nil {
			return 0
		}
		return val * 1024
	case 'M', 'm':
		val, err := strconv.ParseInt(numStr, 10, 64)
		if err != nil {
			return 0
		}
		return val
	case 'K', 'k':
		val, err := strconv.ParseInt(numStr, 10, 64)
		if err != nil {
			return 0
		}
		return val / 1024
	default:
		val, err := strconv.ParseInt(s, 10, 64)
		if err != nil {
			return 0
		}
		return val / (1024 * 1024)
	}
}

// parseHugepageDirName extracts the page size in kB from "hugepages-2048kB" → 2048.
func parseHugepageDirName(dirName string) (int64, bool) {
	if !strings.HasPrefix(dirName, "hugepages-") || !strings.HasSuffix(dirName, "kB") {
		return 0, false
	}
	sizeStr := dirName[len("hugepages-") : len(dirName)-len("kB")]
	size, err := strconv.ParseInt(sizeStr, 10, 64)
	if err != nil || size <= 0 {
		return 0, false
	}
	return size, true
}

// pageSizeToLabel converts a page size in kB to a Kubernetes-friendly label.
// Examples: 2048 → "2mi", 1048576 → "1gi", 32768 → "32mi"
func pageSizeToLabel(pageSizeKB int64) string {
	sizeMiB := pageSizeKB / 1024
	if sizeMiB >= 1024 && sizeMiB%1024 == 0 {
		return fmt.Sprintf("%dgi", sizeMiB/1024)
	}
	return fmt.Sprintf("%dmi", sizeMiB)
}

func readSysfsInt(path string) int64 {
	data, err := os.ReadFile(path)
	if err != nil {
		log.DefaultLogger().V(4).Infof("failed to read %s: %v", path, err)
		return 0
	}
	val, err := strconv.ParseInt(strings.TrimSpace(string(data)), 10, 64)
	if err != nil {
		log.DefaultLogger().V(4).Infof("failed to parse %s: %v", path, err)
		return 0
	}
	return val
}

func (dpi *HugepagesDevicePlugin) GetDeviceName() string {
	return fmt.Sprintf("%s-%s", dynamicHugepageResourcePrefix, dpi.pageLabel)
}

func (dpi *HugepagesDevicePlugin) Start(stop <-chan struct{}) (err error) {
	logger := log.DefaultLogger()
	dpi.stop = stop
	dpi.done = make(chan struct{})
	dpi.deregistered = make(chan struct{})

	if err := dpi.cleanup(); err != nil {
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

	if err := waitForGRPCServer(dpi.socketPath, connectionTimeout); err != nil {
		return fmt.Errorf("error starting the GRPC server: %v", err)
	}

	if err := dpi.register(); err != nil {
		return fmt.Errorf("error registering with device plugin manager: %v", err)
	}

	go func() {
		errChan <- dpi.healthCheck()
	}()

	dpi.setInitialized(true)
	logger.Infof("dynamic hugepages device plugin started: %s (pageSize=%s, gigantic=%v, capacity=%d pages)",
		dpi.resourceName, dpi.pageLabel, isGigantic(dpi.pageSizeKB), len(dpi.devs))

	err = <-errChan
	return err
}

func (dpi *HugepagesDevicePlugin) ListAndWatch(_ *pluginapi.Empty, s pluginapi.DevicePlugin_ListAndWatchServer) error {
	s.Send(&pluginapi.ListAndWatchResponse{Devices: dpi.devs})

	done := false
	for {
		select {
		case devHealth := <-dpi.health:
			for _, dev := range dpi.devs {
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
		log.DefaultLogger().Reason(err).Infof("%s device plugin failed to deregister", dpi.pageLabel)
	}
	close(dpi.deregistered)
	return nil
}

func (dpi *HugepagesDevicePlugin) Allocate(_ context.Context, r *pluginapi.AllocateRequest) (*pluginapi.AllocateResponse, error) {
	logger := log.DefaultLogger()
	logger.Infof("Dynamic Hugepages Allocate: resource=%s, requests=%d", dpi.resourceName, len(r.ContainerRequests))

	response := pluginapi.AllocateResponse{}
	for _, req := range r.ContainerRequests {
		logger.Infof("Dynamic Hugepages Allocate: %d x %s pages", len(req.DevicesIDs), dpi.pageLabel)
		response.ContainerResponses = append(response.ContainerResponses, &pluginapi.ContainerAllocateResponse{})
	}

	return &response, nil
}

func (dpi *HugepagesDevicePlugin) PreStartContainer(_ context.Context, _ *pluginapi.PreStartContainerRequest) (*pluginapi.PreStartContainerResponse, error) {
	return &pluginapi.PreStartContainerResponse{}, nil
}

func (dpi *HugepagesDevicePlugin) GetDevicePluginOptions(_ context.Context, _ *pluginapi.Empty) (*pluginapi.DevicePluginOptions, error) {
	return &pluginapi.DevicePluginOptions{PreStartRequired: false}, nil
}

func (dpi *HugepagesDevicePlugin) GetInitialized() bool {
	dpi.lock.Lock()
	defer dpi.lock.Unlock()
	return dpi.initialized
}

func (dpi *HugepagesDevicePlugin) setInitialized(initialized bool) {
	dpi.lock.Lock()
	defer dpi.lock.Unlock()
	dpi.initialized = initialized
}

func (dpi *HugepagesDevicePlugin) register() error {
	conn, err := gRPCConnect(pluginapi.KubeletSocket, connectionTimeout)
	if err != nil {
		return err
	}
	defer conn.Close()

	client := pluginapi.NewRegistrationClient(conn)
	reqt := &pluginapi.RegisterRequest{
		Version:      pluginapi.Version,
		Endpoint:     filepath.Base(dpi.socketPath),
		ResourceName: dpi.resourceName,
	}

	_, err = client.Register(context.Background(), reqt)
	return err
}

func (dpi *HugepagesDevicePlugin) stopDevicePlugin() error {
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

func (dpi *HugepagesDevicePlugin) cleanup() error {
	if err := os.Remove(dpi.socketPath); err != nil && !errors.Is(err, os.ErrNotExist) {
		return err
	}
	return nil
}

func (dpi *HugepagesDevicePlugin) healthCheck() error {
	logger := log.DefaultLogger()
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return fmt.Errorf("failed to create fsnotify watcher: %v", err)
	}
	defer watcher.Close()

	socketDir := filepath.Dir(dpi.socketPath)
	if err := watcher.Add(socketDir); err != nil {
		return fmt.Errorf("failed to watch socket dir: %v", err)
	}

	sysfsPath := filepath.Join(dpi.hostRoot, hugepagesSysfsBase, dpi.sysfsDir)
	if err := watcher.Add(sysfsPath); err != nil {
		logger.V(4).Infof("failed to watch sysfs hugepages dir %s: %v (non-fatal)", sysfsPath, err)
	}

	for {
		select {
		case <-dpi.stop:
			return nil
		case err := <-watcher.Errors:
			logger.Reason(err).Errorf("error watching dynamic hugepages device plugin")
		case event := <-watcher.Events:
			if event.Name == dpi.socketPath && event.Op == fsnotify.Remove {
				logger.Infof("dynamic hugepages device socket removed, kubelet probably restarted")
				return nil
			}
		}
	}
}
