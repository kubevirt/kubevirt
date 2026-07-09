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

package resources

import (
	k8sv1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	"k8s.io/client-go/tools/cache"

	v1 "kubevirt.io/api/core/v1"
	"kubevirt.io/client-go/log"

	"kubevirt.io/kubevirt/pkg/controller"
	"kubevirt.io/kubevirt/pkg/storage/cbt"
	storagetypes "kubevirt.io/kubevirt/pkg/storage/types"
)

const (
	bitmapGranularity = "64Ki"
	bufferOverhead    = "20Mi"
)

type MemoryCalculator struct {
	pvcStore              cache.Store
	backupTrackerInformer cache.SharedIndexInformer
}

func NewMemoryCalculator(pvcStore cache.Store, backupTrackerInformer cache.SharedIndexInformer) MemoryCalculator {
	return MemoryCalculator{
		pvcStore:              pvcStore,
		backupTrackerInformer: backupTrackerInformer,
	}
}

func (mc MemoryCalculator) Calculate(vmi *v1.VirtualMachineInstance) resource.Quantity {
	totalMemory := resource.Quantity{}

	if !cbt.HasCBTStateEnabled(vmi.Status.ChangedBlockTracking) {
		return totalMemory
	}

	bitmapGranularityMemory := resource.MustParse(bitmapGranularity)
	// TODO: Currently we multiply by the number of trackers because
	// we prune the bitmaps per disk per tracker to one but for multi-checkpoint support
	// with a retention policy we should sum the number of checkpoint retained per tracker
	// and multiply by that instead.
	trackerCount := mc.backupTrackerCountForVMI(vmi.Name, vmi.Namespace)
	logger := log.Log.Object(vmi)
	for _, q := range mc.cbtVolumeCapacities(vmi.Spec.Volumes, vmi.Namespace) {
		// QEMU allocates one bit per granularity-sized block for each dirty bitmap:
		// https://gitlab.com/qemu-project/qemu/-/blob/master/util/hbitmap.c#L803
		// e.g., a 1TiB disk will warrant an overhead of 2MiB memory per associated bitmap.
		bitmapBytes := (q.Value() / bitmapGranularityMemory.Value() / 8) * trackerCount
		totalMemory.Add(*resource.NewQuantity(bitmapBytes, resource.BinarySI))
	}
	// bufferOverhead is a conservative safety margin to account for
	// bitmap growth from operations like disk resize, disk hotplug,
	// or simply a new checkpoint creation, that may occur between
	// memory overhead calculations, therefore we add it regardless
	// of whether there are existing trackers or not.
	totalMemory.Add(resource.MustParse(bufferOverhead))

	logger.V(3).Infof("CBT memory overhead for VMI %s/%s: %s (including %s buffer)", vmi.Namespace, vmi.Name, totalMemory.String(), bufferOverhead)
	return totalMemory
}

func (mc MemoryCalculator) cbtVolumeCapacities(volumes []v1.Volume, namespace string) []resource.Quantity {
	var capacities []resource.Quantity
	for _, vol := range volumes {
		if !cbt.IsCBTEligibleVolume(&vol) {
			continue
		}
		pvcName := storagetypes.PVCNameFromVirtVolume(&vol)
		if pvcName == "" {
			continue
		}
		key := controller.NamespacedKey(namespace, pvcName)
		obj, exists, err := mc.pvcStore.GetByKey(key)
		if err != nil {
			log.Log.V(3).Infof("failed to retrieve PVC %s from store: %v", key, err)
			continue
		}
		if !exists {
			continue
		}
		if q, ok := obj.(*k8sv1.PersistentVolumeClaim).Status.Capacity[k8sv1.ResourceStorage]; ok {
			capacities = append(capacities, q)
		}
	}
	return capacities
}

func (mc MemoryCalculator) backupTrackerCountForVMI(name, namespace string) int64 {
	trackers, err := mc.backupTrackerInformer.GetIndexer().ByIndex("vmi", controller.NamespacedKey(namespace, name))
	if err != nil {
		log.Log.V(3).Infof("failed to retrieve backup trackers for VMI %s/%s: %v", namespace, name, err)
		return 0
	}
	return int64(len(trackers))
}
