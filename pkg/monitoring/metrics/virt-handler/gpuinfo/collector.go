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
	"strings"
	"sync"
	"time"

	"github.com/rhobs/operator-observability-toolkit/pkg/operatormetrics"
	"k8s.io/client-go/tools/cache"
	"kubevirt.io/client-go/log"

	"kubevirt.io/kubevirt/pkg/monitoring/metrics/virt-handler/gpuinfo/podresources"
	"kubevirt.io/kubevirt/pkg/util"
	kvgrpc "kubevirt.io/kubevirt/pkg/util/net/grpc"
)

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

func Setup(nodeName string) {
	gpuCache = &gpuInfoCache{
		nodeName: nodeName,
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

		for _, container := range pod.Containers {
			for _, device := range container.Devices {
				if !strings.HasPrefix(device.ResourceName, "nvidia.com") {
					continue
				}
				for _, uuid := range device.DeviceIds {
					allocations = append(allocations, GPUAllocation{
						Namespace: pod.Namespace,
						PodName:   pod.Name,
						Resource:  device.ResourceName,
						UUID:      uuid,
					})
				}
			}
		}
	}

	return allocations, nil
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
				"resource":  alloc.Resource,
				"uuid":      alloc.UUID,
			},
			Value: 1,
		})
	}

	return results
}
