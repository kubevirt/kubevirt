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
 * Copyright 2017, 2018 Red Hat, Inc.
 *
 */

package watch

import (
	"fmt"

	k8sv1 "k8s.io/api/core/v1"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/tools/record"

	v1 "kubevirt.io/client-go/api/v1"
	"kubevirt.io/client-go/log"
)

func handlePVCMisuseInVM(pvcInformer cache.SharedIndexInformer, recorder record.EventRecorder, vm *v1.VirtualMachine) error {
	logger := log.Log.Object(vm)

	volumeName, err := handlePVCMisuse(pvcInformer, recorder, vm.Namespace, vm.Spec.Template.Spec.Volumes, logger)
	if err != nil && volumeName != nil {
		recorder.Eventf(vm,
			k8sv1.EventTypeWarning,
			FailedPVCVolumeSourceMisusedReason,
			"PVC '%s' used as volume source where Data Volume should be used",
			*volumeName)
	}

	return err
}

func handlePVCMisuseInVMI(pvcInformer cache.SharedIndexInformer, recorder record.EventRecorder, vmi *v1.VirtualMachineInstance) error {
	logger := log.Log.Object(vmi)

	volumeName, err := handlePVCMisuse(pvcInformer, recorder, vmi.Namespace, vmi.Spec.Volumes, logger)
	if err != nil && volumeName != nil {
		recorder.Eventf(vmi,
			k8sv1.EventTypeWarning,
			FailedPVCVolumeSourceMisusedReason,
			"PVC '%s' used as volume source where Data Volume should be used",
			*volumeName)
	}

	return err
}

// Checks if PVC used in VolumeSource is owned by a DataVolume.
// If so, the DV should be used instead and this is an error
func handlePVCMisuse(pvcInformer cache.SharedIndexInformer, recorder record.EventRecorder, namespace string, volumes []v1.Volume, logger *log.FilteredLogger) (*string, error) {

	for _, volume := range volumes {
		if volume.VolumeSource.PersistentVolumeClaim == nil {
			continue
		}

		key := fmt.Sprintf("%s/%s", namespace, volume.VolumeSource.PersistentVolumeClaim.ClaimName)
		obj, exists, err := pvcInformer.GetStore().GetByKey(key)
		if err != nil {
			logger.Reason(err).Warning("failed to fetch PVC for namespace from cache")
			return nil, err
		} else if !exists {
			continue
		}

		pvc := obj.(*k8sv1.PersistentVolumeClaim)
		for _, or := range pvc.ObjectMeta.OwnerReferences {
			if or.Kind != "DataVolume" {
				continue
			}

			err := fmt.Errorf("PVC %v owned by DataVolume %v cannot be used as a volume source. Use DataVolume instead",
				key, or.Name)
			logger.Reason(err).Error("Invalid VM spec")
			return &volume.VolumeSource.PersistentVolumeClaim.ClaimName, err
		}
	}

	return nil, nil
}
