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

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/tools/cache"

	"kubevirt.io/kubevirt/pkg/api/v1"
	"kubevirt.io/kubevirt/pkg/virt-controller/services"

	cdiv1 "kubevirt.io/containerized-data-importer/pkg/apis/datavolumecontroller/v1alpha1"
)

// Reasons for DataVolume events
const (
	// FailedDataVolumeImportReason is added in an event when a dynamically generated
	// dataVolume reaches the failed status phase.
	FailedDataVolumeImportReason = "FailedDataVolumeImport"
	// FailedDataVolumeCreateReason is added in an event when posting a dynamically
	// generated dataVolume to the cluster fails.
	FailedDataVolumeCreateReason = "FailedDataVolumeCreate"
	// FailedDataVolumeDeleteReason is added in an event when deleting a dynamically
	// generated dataVolume in the cluster fails.
	FailedDataVolumeDeleteReason = "FailedDataVolumeDelete"
	// SuccessfulDataVolumeCreateReason is added in an event when a dynamically generated
	// dataVolume is successfully created
	SuccessfulDataVolumeCreateReason = "SuccessfulDataVolumeCreate"
	// SuccessfulDataVolumeImportReason is added in an event when a dynamically generated
	// dataVolume is successfully imports its data
	SuccessfulDataVolumeImportReason = "SuccessfulDataVolumeImport"
	// SuccessfulDataVolumeDeleteReason is added in an event when a dynamically generated
	// dataVolume is successfully deleted
	SuccessfulDataVolumeDeleteReason = "SuccessfulDataVolumeDelete"
)

func listDataVolumesFromNamespace(indexer cache.Indexer, namespace string) ([]*cdiv1.DataVolume, error) {
	objs, err := indexer.ByIndex(cache.NamespaceIndex, namespace)
	if err != nil {
		return nil, err
	}
	dataVolumes := []*cdiv1.DataVolume{}
	for _, obj := range objs {
		dataVolume := obj.(*cdiv1.DataVolume)
		dataVolumes = append(dataVolumes, dataVolume)
	}
	return dataVolumes, nil
}

func createDataVolumeManifest(volume *v1.Volume,
	ownerMeta *metav1.ObjectMeta,
	ownerAPIVersion string,
	ownerKind string) (*cdiv1.DataVolume, error) {

	if volume == nil || volume.VolumeSource.DataVolume == nil {
		return nil, fmt.Errorf("Unable to generate DataVolume spec, invalid volume type")
	}

	labels := map[string]string{}
	annotations := map[string]string{}

	annotations[v1.DataVolumeSourceName] = volume.Name
	annotations[v1.CreatedByAnnotation] = string(ownerMeta.UID)
	annotations[v1.OwnedByAnnotation] = "virt-controller"

	newDataVolume := &cdiv1.DataVolume{
		ObjectMeta: metav1.ObjectMeta{
			Name:        services.GetDataVolumeName(ownerMeta.Name, volume.Name),
			Labels:      labels,
			Annotations: annotations,
		},
		Spec: *volume.VolumeSource.DataVolume,
	}

	tr := true

	newDataVolume.ObjectMeta.OwnerReferences = []metav1.OwnerReference{{
		APIVersion:         ownerAPIVersion,
		Kind:               ownerKind,
		Name:               ownerMeta.Name,
		UID:                ownerMeta.UID,
		Controller:         &tr,
		BlockOwnerDeletion: &tr,
	}}

	return newDataVolume, nil
}
