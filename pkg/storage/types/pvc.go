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
 * Copyright 2018 Red Hat, Inc.
 *
 */

package types

import (
	"fmt"
	"strings"

	k8sv1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/tools/cache"

	virtv1 "kubevirt.io/api/core/v1"
	"kubevirt.io/client-go/log"
	cdiv1 "kubevirt.io/containerized-data-importer-api/pkg/apis/core/v1beta1"

	"kubevirt.io/kubevirt/pkg/controller"
)

const (
	MiB = 1024 * 1024

	allowClaimAdoptionAnnotation = "cdi.kubevirt.io/allowClaimAdoption"
)

type PvcNotFoundError struct {
	Reason string
}

func (e PvcNotFoundError) Error() string {
	return e.Reason
}

func IsPVCBlockFromStore(store cache.Store, namespace string, claimName string) (pvc *k8sv1.PersistentVolumeClaim, exists bool, isBlockDevice bool, err error) {
	obj, exists, err := store.GetByKey(namespace + "/" + claimName)
	if err != nil || !exists {
		return nil, exists, false, err
	}
	if pvc, ok := obj.(*k8sv1.PersistentVolumeClaim); ok {
		return obj.(*k8sv1.PersistentVolumeClaim), true, IsPVCBlock(pvc.Spec.VolumeMode), nil
	}
	return nil, false, false, fmt.Errorf("this is not a PVC! %v", obj)
}

func IsPVCBlock(volumeMode *k8sv1.PersistentVolumeMode) bool {
	// We do not need to consider the data in a PersistentVolume (as of Kubernetes 1.9)
	// If a PVC does not specify VolumeMode and the PV specifies VolumeMode = Block
	// the claim will not be bound. So for the sake of a boolean answer, if the PVC's
	// VolumeMode is Block, that unambiguously answers the question
	return volumeMode != nil && *volumeMode == k8sv1.PersistentVolumeBlock
}

func HasSharedAccessMode(accessModes []k8sv1.PersistentVolumeAccessMode) bool {
	for _, accessMode := range accessModes {
		if accessMode == k8sv1.ReadWriteMany {
			return true
		}
	}
	return false
}

func IsReadOnlyAccessMode(accessModes []k8sv1.PersistentVolumeAccessMode) bool {
	for _, accessMode := range accessModes {
		if accessMode == k8sv1.ReadOnlyMany {
			return true
		}
	}
	return false
}

func IsReadWriteOnceAccessMode(accessModes []k8sv1.PersistentVolumeAccessMode) bool {
	for _, accessMode := range accessModes {
		if accessMode == k8sv1.ReadOnlyMany || accessMode == k8sv1.ReadWriteMany {
			return false
		}
	}
	return true
}

func IsPreallocated(annotations map[string]string) bool {
	for a, value := range annotations {
		if strings.Contains(a, "/storage.preallocation") && value == "true" {
			return true
		}
		if strings.Contains(a, "/storage.thick-provisioned") && value == "true" {
			return true
		}
	}
	return false
}

func PVCNameFromVirtVolume(volume *virtv1.Volume) string {
	if volume.DataVolume != nil {
		// TODO, look up the correct PVC name based on the datavolume, right now they match, but that will not always be true.
		return volume.DataVolume.Name
	} else if volume.PersistentVolumeClaim != nil {
		return volume.PersistentVolumeClaim.ClaimName
	} else if volume.MemoryDump != nil {
		return volume.MemoryDump.ClaimName
	}

	return ""
}

func GetPVCsFromVolumes(volumes []virtv1.Volume) map[string]string {
	pvcs := map[string]string{}

	for _, volume := range volumes {
		pvcName := PVCNameFromVirtVolume(&volume)
		if pvcName == "" {
			continue
		}

		pvcs[volume.Name] = pvcName
	}

	return pvcs
}

func VirtVolumesToPVCMap(volumes []*virtv1.Volume, pvcStore cache.Store, namespace string) (map[string]*k8sv1.PersistentVolumeClaim, error) {
	volumeNamesPVCMap := make(map[string]*k8sv1.PersistentVolumeClaim)
	for _, volume := range volumes {
		claimName := PVCNameFromVirtVolume(volume)
		if claimName == "" {
			return nil, fmt.Errorf("volume %s is not a PVC or Datavolume", volume.Name)
		}
		pvc, exists, _, err := IsPVCBlockFromStore(pvcStore, namespace, claimName)
		if err != nil {
			return nil, fmt.Errorf("failed to get PVC: %v", err)
		}
		if !exists {
			return nil, fmt.Errorf("claim %s not found", claimName)
		}
		volumeNamesPVCMap[volume.Name] = pvc
	}
	return volumeNamesPVCMap, nil
}

func GetPersistentVolumeClaimFromCache(namespace, name string, pvcStore cache.Store) (*k8sv1.PersistentVolumeClaim, error) {
	key := controller.NamespacedKey(namespace, name)
	obj, exists, err := pvcStore.GetByKey(key)

	if err != nil {
		return nil, fmt.Errorf("error fetching PersistentVolumeClaim %s: %v", key, err)
	}
	if !exists {
		return nil, nil
	}

	pvc, ok := obj.(*k8sv1.PersistentVolumeClaim)
	if !ok {
		return nil, fmt.Errorf("error converting object to PersistentVolumeClaim: object is of type %T", obj)
	}

	return pvc.DeepCopy(), nil
}

func HasUnboundPVC(namespace string, volumes []virtv1.Volume, pvcStore cache.Store) bool {
	for _, volume := range volumes {
		claimName := PVCNameFromVirtVolume(&volume)
		if claimName == "" {
			continue
		}

		pvc, err := GetPersistentVolumeClaimFromCache(namespace, claimName, pvcStore)
		if err != nil {
			log.Log.Errorf("Error fetching PersistentVolumeClaim %s while determining virtual machine status: %v", claimName, err)
			continue
		}
		if pvc == nil {
			continue
		}

		if pvc.Status.Phase != k8sv1.ClaimBound {
			return true
		}
	}

	return false
}

func VolumeReadyToAttachToNode(namespace string, volume virtv1.Volume, dataVolumes []*cdiv1.DataVolume, dataVolumeStore, pvcStore cache.Store) (bool, bool, error) {
	name := PVCNameFromVirtVolume(&volume)

	dataVolumeFunc := DataVolumeByNameFunc(dataVolumeStore, dataVolumes)
	wffc := false
	ready := false
	// err is always nil
	pvcInterface, pvcExists, _ := pvcStore.GetByKey(fmt.Sprintf("%s/%s", namespace, name))
	if pvcExists {
		var err error
		pvc := pvcInterface.(*k8sv1.PersistentVolumeClaim)
		ready, err = cdiv1.IsSucceededOrPendingPopulation(pvc, dataVolumeFunc)
		if err != nil {
			return false, false, err
		}
		if !ready {
			waitsForFirstConsumer, err := cdiv1.IsWaitForFirstConsumerBeforePopulating(pvc, dataVolumeFunc)
			if err != nil {
				return false, false, err
			}
			if waitsForFirstConsumer {
				wffc = true
			}
		}
	} else {
		return false, false, PvcNotFoundError{Reason: fmt.Sprintf("didn't find PVC %v", name)}
	}
	return ready, wffc, nil
}

func RenderPVC(size *resource.Quantity, claimName, namespace, storageClass, accessMode string, blockVolume bool) *k8sv1.PersistentVolumeClaim {
	pvc := &k8sv1.PersistentVolumeClaim{
		ObjectMeta: metav1.ObjectMeta{
			Name:      claimName,
			Namespace: namespace,
		},
		Spec: k8sv1.PersistentVolumeClaimSpec{
			Resources: k8sv1.VolumeResourceRequirements{
				Requests: k8sv1.ResourceList{
					k8sv1.ResourceStorage: *size,
				},
			},
		},
	}

	if storageClass != "" {
		pvc.Spec.StorageClassName = &storageClass
	}

	if accessMode != "" {
		pvc.Spec.AccessModes = []k8sv1.PersistentVolumeAccessMode{k8sv1.PersistentVolumeAccessMode(accessMode)}
	} else {
		pvc.Spec.AccessModes = []k8sv1.PersistentVolumeAccessMode{k8sv1.ReadWriteOnce}
	}

	if blockVolume {
		volMode := k8sv1.PersistentVolumeBlock
		pvc.Spec.VolumeMode = &volMode
	}

	return pvc
}

func IsHotplugVolume(vol *virtv1.Volume) bool {
	if vol == nil {
		return false
	}
	volSrc := vol.VolumeSource
	if volSrc.PersistentVolumeClaim != nil && volSrc.PersistentVolumeClaim.Hotpluggable {
		return true
	}
	if volSrc.DataVolume != nil && volSrc.DataVolume.Hotpluggable {
		return true
	}
	if volSrc.MemoryDump != nil && volSrc.MemoryDump.PersistentVolumeClaimVolumeSource.Hotpluggable {
		return true
	}

	return false
}

func GetVolumesByName(vmiSpec *virtv1.VirtualMachineInstanceSpec) map[string]*virtv1.Volume {
	volumes := map[string]*virtv1.Volume{}
	for _, vol := range vmiSpec.Volumes {
		volumes[vol.Name] = vol.DeepCopy()
	}
	return volumes
}

func GetDisksByName(vmiSpec *virtv1.VirtualMachineInstanceSpec) map[string]*virtv1.Disk {
	disks := map[string]*virtv1.Disk{}
	for _, disk := range vmiSpec.Domain.Devices.Disks {
		disks[disk.Name] = disk.DeepCopy()
	}
	return disks
}

// Get expected disk capacity - a minimum between the request and the PVC capacity.
// Returns nil when we have insufficient data to calculate this minimum.
func GetDiskCapacity(pvcInfo *virtv1.PersistentVolumeClaimInfo) *int64 {
	logger := log.DefaultLogger()
	storageCapacityResource, ok := pvcInfo.Capacity[k8sv1.ResourceStorage]
	if !ok {
		return nil
	}
	storageCapacity, ok := storageCapacityResource.AsInt64()
	if !ok {
		logger.Infof("Failed to convert storage capacity %+v to int64", storageCapacityResource)
		return nil
	}
	storageRequestResource, ok := pvcInfo.Requests[k8sv1.ResourceStorage]
	if !ok {
		return nil
	}
	storageRequest, ok := storageRequestResource.AsInt64()
	if !ok {
		logger.Infof("Failed to convert storage request %+v to int64", storageRequestResource)
		return nil
	}
	preferredSize := min(storageRequest, storageCapacity)
	return &preferredSize
}

func GetFilesystemsFromVolumes(vmi *virtv1.VirtualMachineInstance) map[string]*virtv1.Filesystem {
	fs := map[string]*virtv1.Filesystem{}

	for _, f := range vmi.Spec.Domain.Devices.Filesystems {
		fs[f.Name] = f.DeepCopy()
	}

	return fs
}

func IsMigratedVolume(name string, vmi *virtv1.VirtualMachineInstance) bool {
	for _, v := range vmi.Status.MigratedVolumes {
		if v.VolumeName == name {
			return true
		}
	}
	return false
}

func GetTotalSizeMigratedVolumes(vmi *virtv1.VirtualMachineInstance) *resource.Quantity {
	size := int64(0)
	srcVols := make(map[string]bool)
	for _, v := range vmi.Status.MigratedVolumes {
		if v.SourcePVCInfo == nil {
			continue
		}
		srcVols[v.SourcePVCInfo.ClaimName] = true
	}
	for _, vstatus := range vmi.Status.VolumeStatus {
		if vstatus.PersistentVolumeClaimInfo == nil {
			continue
		}
		if _, ok := srcVols[vstatus.PersistentVolumeClaimInfo.ClaimName]; ok {
			if s := GetDiskCapacity(vstatus.PersistentVolumeClaimInfo); s != nil {
				size += *s
			}
		}
	}

	return resource.NewScaledQuantity(size, resource.Giga)
}
