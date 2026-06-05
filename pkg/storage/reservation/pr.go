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

package reservation

import (
	"fmt"
	"path/filepath"

	k8sv1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/tools/cache"
	v1 "kubevirt.io/api/core/v1"

	storagetypes "kubevirt.io/kubevirt/pkg/storage/types"
)

const (
	sourceDaemonsPath     = "/var/run/kubevirt/daemons"
	hostSourceDaemonsPath = "/proc/1/root" + sourceDaemonsPath
	prHelperDir           = "pr"
	prHelperSocket        = "pr-helper.sock"
	prResourceName        = "pr-helper"
)

func GetPrResourceName() string {
	return prResourceName
}

func GetPrHelperSocketDir() string {
	return filepath.Join(sourceDaemonsPath, prHelperDir)
}

func GetPrHelperHostSocketDir() string {
	return filepath.Join(hostSourceDaemonsPath, prHelperDir)
}

func GetPrHelperSocketPath() string {
	return filepath.Join(GetPrHelperSocketDir(), prHelperSocket)
}

func GetPrHelperSocket() string {
	return prHelperSocket
}

func HasVMIPersistentReservation(vmi *v1.VirtualMachineInstance) bool {
	return HasVMISpecPersistentReservation(&vmi.Spec)
}

func HasVMISpecPersistentReservation(vmiSpec *v1.VirtualMachineInstanceSpec) bool {
	for _, disk := range vmiSpec.Domain.Devices.Disks {
		if disk.DiskDevice.LUN != nil && disk.DiskDevice.LUN.Reservation {
			return true
		}
	}
	return false
}

// PersistentReservationPVCLabels returns labels identifying which PVC(s) are
// used with SCSI PersistentReservation by this VMI. Each label key contains
// the PVC's UID, allowing pod anti-affinity to prevent co-scheduling of VMs
// sharing the same PR LUN.
func PersistentReservationPVCLabels(vmi *v1.VirtualMachineInstance, pvcStore cache.Store) (map[string]string, error) {
	labels := map[string]string{}

	volumeByName := make(map[string]*v1.Volume, len(vmi.Spec.Volumes))
	for i := range vmi.Spec.Volumes {
		volumeByName[vmi.Spec.Volumes[i].Name] = &vmi.Spec.Volumes[i]
	}

	for _, disk := range vmi.Spec.Domain.Devices.Disks {
		if disk.DiskDevice.LUN == nil || !disk.DiskDevice.LUN.Reservation {
			continue
		}
		volume, found := volumeByName[disk.Name]
		if !found {
			continue
		}
		pvcName := storagetypes.PVCNameFromVirtVolume(volume)
		if pvcName == "" {
			continue
		}
		pvc, err := storagetypes.GetPersistentVolumeClaimFromCache(vmi.Namespace, pvcName, pvcStore)
		if err != nil {
			return nil, fmt.Errorf("failed to get PVC %s/%s: %v", vmi.Namespace, pvcName, err)
		}
		if pvc == nil {
			continue
		}
		labels[v1.PersistentReservationLabelPrefix+string(pvc.UID)] = ""
	}

	return labels, nil
}

// PersistentReservationPodAntiAffinityTerms returns pod anti-affinity terms
// that prevent co-scheduling with other pods that share the same PR PVC labels.
func PersistentReservationPodAntiAffinityTerms(labels map[string]string) []k8sv1.PodAffinityTerm {
	terms := make([]k8sv1.PodAffinityTerm, 0, len(labels))
	for labelKey := range labels {
		terms = append(terms, k8sv1.PodAffinityTerm{
			LabelSelector: &metav1.LabelSelector{
				MatchExpressions: []metav1.LabelSelectorRequirement{
					{
						Key:      labelKey,
						Operator: metav1.LabelSelectorOpExists,
					},
				},
			},
			TopologyKey: "kubernetes.io/hostname",
		})
	}
	return terms
}
