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
 */

package gpuinfo

import (
	"context"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/rhobs/operator-observability-toolkit/pkg/operatormetrics"
	"k8s.io/client-go/tools/cache"

	v1 "kubevirt.io/api/core/v1"
	"kubevirt.io/client-go/log"

	"kubevirt.io/kubevirt/pkg/monitoring/metrics/virt-handler/gpuinfo/podresources"
	"kubevirt.io/kubevirt/pkg/util"
	"kubevirt.io/kubevirt/pkg/util/net/dns"
	kvgrpc "kubevirt.io/kubevirt/pkg/util/net/grpc"
)

// launcherPodSuffix matches the random suffix Kubernetes appends to a
// virt-launcher pod's GenerateName ("virt-launcher-<hostname>-<suffix>").
var launcherPodSuffix = regexp.MustCompile(`^[a-z0-9]{5}$`)

const (
	podResourcesSocket = util.KubeletRoot + "/pod-resources/kubelet.sock"
	refreshInterval    = 25 * time.Second
	grpcTimeout        = 10 * time.Second
)

var (
	Collector = operatormetrics.Collector{
		Metrics:         []operatormetrics.Metric{vmiGPUInfo},
		CollectCallback: collectCallback,
	}

	gpuCache *gpuInfoCache
)

type GPUAllocation struct {
	Namespace string
	PodName   string
	VMIName   string
	Resource  string
	UUID      string
}

type gpuInfoCache struct {
	mu          sync.Mutex
	allocations []GPUAllocation
	nodeName    string
	vmiInformer cache.SharedIndexInformer
	lastRefresh time.Time
}

func Setup(nodeName string, vmiInformer cache.SharedIndexInformer) {
	gpuCache = &gpuInfoCache{
		nodeName:    nodeName,
		vmiInformer: vmiInformer,
	}
}

func (c *gpuInfoCache) get() []GPUAllocation {
	c.mu.Lock()
	defer c.mu.Unlock()

	if time.Since(c.lastRefresh) > refreshInterval {
		allocations, err := c.fetchGPUAllocations()
		if err != nil {
			log.Log.Warningf("Failed to fetch GPU allocations from Pod Resources API: %v", err)
		} else {
			c.allocations = allocations
			c.lastRefresh = time.Now()
		}
	}

	result := make([]GPUAllocation, len(c.allocations))
	copy(result, c.allocations)
	return result
}

func (c *gpuInfoCache) fetchGPUAllocations() ([]GPUAllocation, error) {
	if c.vmiInformer == nil {
		return nil, nil
	}

	conn, err := kvgrpc.DialSocketWithTimeout(podResourcesSocket, int(grpcTimeout.Seconds()))
	if err != nil {
		return nil, err
	}
	defer conn.Close()

	client := podresources.NewPodResourcesListerClient(conn)
	resp, err := client.List(context.Background(), &podresources.ListPodResourcesRequest{})
	if err != nil {
		return nil, err
	}

	var allocations []GPUAllocation
	for _, pod := range resp.PodResources {
		if !strings.HasPrefix(pod.Name, "virt-launcher-") {
			continue
		}

		vmiName := resolveVMIName(pod.Namespace, pod.Name, c.vmisInNamespace(pod.Namespace))

		for _, container := range pod.Containers {
			for _, device := range container.Devices {
				if !strings.HasPrefix(device.ResourceName, "nvidia.com") {
					continue
				}
				for _, uuid := range device.DeviceIds {
					allocations = append(allocations, GPUAllocation{
						Namespace: pod.Namespace,
						PodName:   pod.Name,
						VMIName:   vmiName,
						Resource:  device.ResourceName,
						UUID:      uuid,
					})
				}
			}
		}
	}

	return allocations, nil
}

// vmisInNamespace returns the node-local VMIs in the given namespace, using the
// informer's namespace index to avoid scanning the whole store. The returned
// values are pointers into the shared informer cache and must not be mutated.
func (c *gpuInfoCache) vmisInNamespace(namespace string) []*v1.VirtualMachineInstance {
	objs, err := c.vmiInformer.GetIndexer().ByIndex(cache.NamespaceIndex, namespace)
	if err != nil {
		log.Log.Warningf("Failed to look up VMIs in namespace %s: %v", namespace, err)
		return nil
	}

	vmis := make([]*v1.VirtualMachineInstance, 0, len(objs))
	for _, obj := range objs {
		if vmi, ok := obj.(*v1.VirtualMachineInstance); ok {
			vmis = append(vmis, vmi)
		}
	}
	return vmis
}

// resolveVMIName returns the name of the VMI owning the given virt-launcher
// pod, by recomputing each node-local VMI's expected pod-name prefix with the
// same sanitization KubeVirt uses to build it and validating the trailing
// GenerateName suffix. Returns "" when no VMI matches.
func resolveVMIName(namespace, podName string, vmis []*v1.VirtualMachineInstance) string {
	for _, vmi := range vmis {
		if vmi.Namespace != namespace {
			continue
		}
		prefix := "virt-launcher-" + dns.SanitizeHostname(vmi) + "-"
		if !strings.HasPrefix(podName, prefix) {
			continue
		}
		if launcherPodSuffix.MatchString(podName[len(prefix):]) {
			return vmi.Name
		}
	}
	return ""
}

func collectCallback() []operatormetrics.CollectorResult {
	if gpuCache == nil {
		return nil
	}

	allocations := gpuCache.get()
	results := make([]operatormetrics.CollectorResult, 0, len(allocations))

	for _, alloc := range allocations {
		results = append(results, operatormetrics.CollectorResult{
			Metric: vmiGPUInfo,
			ConstLabels: map[string]string{
				"node":      gpuCache.nodeName,
				"namespace": alloc.Namespace,
				"pod":       alloc.PodName,
				"name":      alloc.VMIName,
				"resource":  alloc.Resource,
				"uuid":      alloc.UUID,
			},
			Value: 1,
		})
	}

	return results
}
