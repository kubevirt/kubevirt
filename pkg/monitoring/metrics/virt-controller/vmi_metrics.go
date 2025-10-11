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

package virt_controller

import (
	"strings"
	"sync"
	"time"

	ioprometheusclient "github.com/prometheus/client_model/go"
	"github.com/rhobs/operator-observability-toolkit/pkg/operatormetrics"

	"k8s.io/apimachinery/pkg/api/resource"

	v1 "kubevirt.io/api/core/v1"
)

type EphemeralStatus struct {
	timestamp int64
	confirmed bool
}

type VolumeTracker struct {
	sync.RWMutex
	// keys are namespace/vmi_name/volume_name
	volumes map[string]EphemeralStatus
}

var (
	vmiMetrics = []operatormetrics.Metric{
		vmiLauncherMemoryOverhead,
	}

	ephemeralVolumeMetrics = operatormetrics.Collector{
		Metrics: []operatormetrics.Metric{
			vmiEphemeralHotplugVolumeTotal,
		},
		CollectCallback: EphemeralVolumeMetricsCallback,
	}

	vmiLauncherMemoryOverhead = operatormetrics.NewGaugeVec(
		operatormetrics.MetricOpts{
			Name: "kubevirt_vmi_launcher_memory_overhead_bytes",
			Help: "Estimation of the memory amount required for virt-launcher's infrastructure components (e.g. libvirt, QEMU).",
		},
		[]string{"namespace", "name"},
	)

	vmiEphemeralHotplugVolumeTotal = operatormetrics.NewGaugeVec(
		operatormetrics.MetricOpts{
			Name: "kubevirt_vmi_ephemeral_hotplug_volume_total",
			Help: "Total number of ephemeral hotplug volumes added to the VMI.",
		},
		[]string{"namespace", "vmi_name", "volume_name"},
	)

	volumeTracker = &VolumeTracker{
		volumes: make(map[string]EphemeralStatus),
	}
)

func UpdateEphemeralVolumeCount(vmi *v1.VirtualMachineInstance, vm *v1.VirtualMachine) {
	volumeTracker.Lock()
	defer volumeTracker.Unlock()

	vmVolumeMap := make(map[string]v1.Volume)
	if vmi == nil || vm == nil {
		return
	}

	for _, volume := range vm.Spec.Template.Spec.Volumes {
		vmVolumeMap[volume.Name] = volume
	}

	// store vmi volumes so we can check for potential unplugged volumes
	vmiVolumeMap := make(map[string]v1.Volume)

	// check if the vmi has any volumes that are not in the vm spec
	for _, volume := range vmi.Spec.Volumes {
		if !isHotplugVolume(volume) {
			continue
		}
		vmiVolumeMap[volume.Name] = volume
		trackerKey := vmi.Namespace + "/" + vmi.Name + "/" + volume.Name
		if _, exists := vmVolumeMap[volume.Name]; !exists {
			// only set timestamp on first detection
			if _, exists := volumeTracker.volumes[trackerKey]; exists {
				continue
			}
			// set timestamp for potential ephemeral volume
			volumeTracker.volumes[trackerKey] = EphemeralStatus{
				timestamp: time.Now().Unix(),
				confirmed: false,
			}
		} else {
			// volume exists in both specs
			volumeStatus, exists := volumeTracker.volumes[trackerKey]
			if !exists {
				continue
			}

			// if we previously marked this as ephemeral, check if it was added recently to spec (within 60s)
			// then it's actually a persistent hotplug and we can remove the metric
			timeDiff := time.Now().Unix() - volumeStatus.timestamp
			if timeDiff <= 60 {
				delete(volumeTracker.volumes, trackerKey)
			}
		}
	}

	// resets metric for any ephemeral volumes that were unplugged
	// i.e. volumes that used to be in vmi spec but are no longer
	for key, volumeStatus := range volumeTracker.volumes {
		_, _, volumeName := parseVolumeKey(key)
		if _, exists := vmiVolumeMap[volumeName]; !exists {
			delete(volumeTracker.volumes, key)
		} else {
			// check if we have tracked this volume for more than x seconds,
			// if so we can confirm it as an ephemeral volume
			timePassed := time.Now().Unix() - volumeStatus.timestamp

			// TODO: this is probably a poor approrach since we could accidentally confirm non-ephemeral volumes
			// this is ultimately trying to prevent increasing the metric in previous loop
			// and then having to remove it in subsequent iterations.
			timeThreshold := int64(2)
			if timePassed > timeThreshold {
				volumeStatus.confirmed = true
				volumeTracker.volumes[key] = volumeStatus
			}
		}

	}

}

func isHotplugVolume(volume v1.Volume) bool {
	return (volume.VolumeSource.PersistentVolumeClaim != nil &&
		volume.VolumeSource.PersistentVolumeClaim.Hotpluggable) ||
		(volume.VolumeSource.DataVolume != nil && volume.VolumeSource.DataVolume.Hotpluggable)
}

func EphemeralVolumeMetricsCallback() []operatormetrics.CollectorResult {
	volumeTracker.RLock()
	defer volumeTracker.RUnlock()

	// TODO: decide whether we care to track volume, or just increment total per VMI
	results := []operatormetrics.CollectorResult{}
	for key, volumeStatus := range volumeTracker.volumes {
		// only report confirmed volumes
		if !volumeStatus.confirmed {
			continue
		}
		namespace, vmiName, volumeName := parseVolumeKey(key)
		results = append(results, operatormetrics.CollectorResult{
			Metric: vmiEphemeralHotplugVolumeTotal,
			Labels: []string{namespace, vmiName, volumeName},
			Value:  float64(1),
		})
	}

	return results
}

func parseVolumeKey(key string) (string, string, string) {
	parts := strings.Split(key, "/")
	return parts[0], parts[1], parts[2]
}

func SetVmiLaucherMemoryOverhead(vmi *v1.VirtualMachineInstance, memoryOverhead resource.Quantity) {
	vmiLauncherMemoryOverhead.
		WithLabelValues(vmi.Namespace, vmi.Name).
		Set(float64(memoryOverhead.Value()))
}

func GetVmiLaucherMemoryOverhead(vmi *v1.VirtualMachineInstance) (float64, error) {
	dto := &ioprometheusclient.Metric{}
	if err := vmiLauncherMemoryOverhead.WithLabelValues(vmi.Namespace, vmi.Name).Write(dto); err != nil {
		return -1, err
	}

	return *dto.Gauge.Value, nil
}
