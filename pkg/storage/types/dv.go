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
 * Copyright 2021 Red Hat, Inc.
 *
 */

package types

import (
	"context"
	"fmt"
	"maps"

	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/tools/cache"

	virtv1 "kubevirt.io/api/core/v1"
	"kubevirt.io/client-go/kubecli"
	"kubevirt.io/client-go/log"
	cdiv1 "kubevirt.io/containerized-data-importer-api/pkg/apis/core/v1beta1"

	storagev1 "k8s.io/api/storage/v1"

	"kubevirt.io/kubevirt/pkg/controller"
)

type CloneSource struct {
	Namespace string
	Name      string
}

func DataVolumeWFFC(dv *cdiv1.DataVolume) bool {
	return dv.Status.Phase == cdiv1.WaitForFirstConsumer
}

func HasWFFCDataVolumes(dvs []*cdiv1.DataVolume) bool {
	for _, dv := range dvs {
		if DataVolumeWFFC(dv) {
			return true
		}
	}
	return false
}

func DataVolumeFailed(dv *cdiv1.DataVolume) bool {
	return dv.Status.Phase == cdiv1.Failed
}

func HasFailedDataVolumes(dvs []*cdiv1.DataVolume) bool {
	for _, dv := range dvs {
		if DataVolumeFailed(dv) {
			return true
		}
	}
	return false
}

// GetResolvedCloneSource resolves the clone source of a datavolume with sourceRef
// This will be moved to the CDI API package
func GetResolvedCloneSource(ctx context.Context, client kubecli.KubevirtClient, namespace string, dvSpec *cdiv1.DataVolumeSpec) (*cdiv1.DataVolumeSource, error) {
	ns := namespace
	source := dvSpec.Source

	if dvSpec.SourceRef != nil && dvSpec.SourceRef.Kind == "DataSource" {
		if dvSpec.SourceRef.Namespace != nil {
			ns = *dvSpec.SourceRef.Namespace
		}

		ds, err := client.CdiClient().CdiV1beta1().DataSources(ns).Get(ctx, dvSpec.SourceRef.Name, metav1.GetOptions{})
		if err != nil {
			return nil, err
		}

		source = &cdiv1.DataVolumeSource{
			PVC:      ds.Spec.Source.PVC,
			Snapshot: ds.Spec.Source.Snapshot,
		}
	}

	if source == nil {
		return source, nil
	}
	switch {
	case source.PVC != nil:
		if source.PVC.Namespace == "" {
			source.PVC.Namespace = ns
		}
	case source.Snapshot != nil:
		if source.Snapshot.Namespace == "" {
			source.Snapshot.Namespace = ns
		}
	default:
		source = nil
	}

	return source, nil
}

func GenerateDataVolumeFromTemplate(clientset kubecli.KubevirtClient, dataVolumeTemplate virtv1.DataVolumeTemplateSpec, namespace, priorityClassName string) (*cdiv1.DataVolume, error) {
	newDataVolume := &cdiv1.DataVolume{}
	newDataVolume.Spec = *dataVolumeTemplate.Spec.DeepCopy()
	newDataVolume.ObjectMeta = *dataVolumeTemplate.ObjectMeta.DeepCopy()
	newDataVolume.ObjectMeta.Labels = maps.Clone(dataVolumeTemplate.Labels)
	if newDataVolume.ObjectMeta.Labels == nil {
		newDataVolume.ObjectMeta.Labels = make(map[string]string)
	}
	newDataVolume.ObjectMeta.Annotations = maps.Clone(dataVolumeTemplate.Annotations)
	if newDataVolume.ObjectMeta.Annotations == nil {
		newDataVolume.ObjectMeta.Annotations = make(map[string]string, 1)
	}
	newDataVolume.ObjectMeta.Annotations[allowClaimAdoptionAnnotation] = "true"

	if newDataVolume.Spec.PriorityClassName == "" && priorityClassName != "" {
		newDataVolume.Spec.PriorityClassName = priorityClassName
	}

	dvSource, err := GetResolvedCloneSource(context.TODO(), clientset, namespace, &newDataVolume.Spec)
	if err != nil {
		return nil, err
	}

	if dvSource != nil {
		// If SourceRef is set, populate spec.Source with data from the DataSource
		// If not, update the field anyway to account for possible namespace changes
		if newDataVolume.Spec.SourceRef != nil {
			newDataVolume.Spec.SourceRef = nil
		}
		newDataVolume.Spec.Source = dvSource
	}

	return newDataVolume, nil
}

func GetDataVolumeFromCache(namespace, name string, dataVolumeStore cache.Store) (*cdiv1.DataVolume, error) {
	key := controller.NamespacedKey(namespace, name)
	obj, exists, err := dataVolumeStore.GetByKey(key)

	if err != nil {
		return nil, fmt.Errorf("error fetching DataVolume %s: %v", key, err)
	}
	if !exists {
		return nil, nil
	}

	dv, ok := obj.(*cdiv1.DataVolume)
	if !ok {
		return nil, fmt.Errorf("error converting object to DataVolume: object is of type %T", obj)
	}

	return dv.DeepCopy(), nil
}

func HasDataVolumeErrors(namespace string, volumes []virtv1.Volume, dataVolumeStore cache.Store) error {
	for _, volume := range volumes {
		if volume.DataVolume == nil {
			continue
		}

		dv, err := GetDataVolumeFromCache(namespace, volume.DataVolume.Name, dataVolumeStore)
		if err != nil {
			log.Log.Errorf("Error fetching DataVolume %s: %v", volume.DataVolume.Name, err)
			continue
		}
		if dv == nil {
			continue
		}

		if DataVolumeFailed(dv) {
			return fmt.Errorf("DataVolume %s is in Failed phase", volume.DataVolume.Name)
		}

		dvRunningCond := NewDataVolumeConditionManager().GetCondition(dv, cdiv1.DataVolumeRunning)
		if dvRunningCond != nil &&
			dvRunningCond.Status == v1.ConditionFalse &&
			(dvRunningCond.Reason == "Error" || dvRunningCond.Reason == "ImagePullFailed") {
			return fmt.Errorf("DataVolume %s importer has stopped running due to an error: %v",
				volume.DataVolume.Name, dvRunningCond.Message)
		}
	}

	return nil
}

// FIXME: Bound mistakenly reports ErrExceededQuota with ConditionUnknown status
func HasDataVolumeExceededQuotaError(dv *cdiv1.DataVolume) error {
	dvBoundCond := NewDataVolumeConditionManager().GetCondition(dv, cdiv1.DataVolumeBound)
	if dvBoundCond != nil && dvBoundCond.Status != v1.ConditionTrue && dvBoundCond.Reason == "ErrExceededQuota" {
		return fmt.Errorf("DataVolume %s importer is not running due to an error: %v", dv.Name, dvBoundCond.Message)
	}

	return nil
}

func HasDataVolumeProvisioning(namespace string, volumes []virtv1.Volume, dataVolumeStore cache.Store) bool {
	for _, volume := range volumes {
		if volume.DataVolume == nil {
			continue
		}

		dv, err := GetDataVolumeFromCache(namespace, volume.DataVolume.Name, dataVolumeStore)
		if err != nil {
			log.Log.Errorf("Error fetching DataVolume %s while determining virtual machine status: %v", volume.DataVolume.Name, err)
			continue
		}
		if dv == nil || dv.Status.Phase == cdiv1.Succeeded || dv.Status.Phase == cdiv1.PendingPopulation {
			continue
		}

		dvConditions := NewDataVolumeConditionManager()
		isBound := dvConditions.HasConditionWithStatus(dv, cdiv1.DataVolumeBound, v1.ConditionTrue)
		// WFFC + plus unbound is not provisioning
		if isBound || dv.Status.Phase != cdiv1.WaitForFirstConsumer {
			return true
		}
	}

	return false
}

func ListDataVolumesFromTemplates(namespace string, dvTemplates []virtv1.DataVolumeTemplateSpec, dataVolumeStore cache.Store) ([]*cdiv1.DataVolume, error) {
	dataVolumes := []*cdiv1.DataVolume{}

	for _, template := range dvTemplates {
		// get DataVolume from cache for each templated dataVolume
		dv, err := GetDataVolumeFromCache(namespace, template.Name, dataVolumeStore)
		if err != nil {
			return dataVolumes, err
		} else if dv == nil {
			continue
		}

		dataVolumes = append(dataVolumes, dv)
	}
	return dataVolumes, nil
}

func ListDataVolumesFromVolumes(namespace string, volumes []virtv1.Volume, dataVolumeStore cache.Store, pvcStore cache.Store) ([]*cdiv1.DataVolume, error) {
	dataVolumes := []*cdiv1.DataVolume{}

	for _, volume := range volumes {
		dataVolumeName := getDataVolumeName(namespace, volume, pvcStore)
		if dataVolumeName == nil {
			continue
		}

		dv, err := GetDataVolumeFromCache(namespace, *dataVolumeName, dataVolumeStore)
		if err != nil {
			return dataVolumes, err
		} else if dv == nil {
			continue
		}

		dataVolumes = append(dataVolumes, dv)
	}

	return dataVolumes, nil
}

func getDataVolumeName(namespace string, volume virtv1.Volume, pvcStore cache.Store) *string {
	if volume.VolumeSource.PersistentVolumeClaim != nil {
		pvcInterface, pvcExists, _ := pvcStore.
			GetByKey(fmt.Sprintf("%s/%s", namespace, volume.VolumeSource.PersistentVolumeClaim.ClaimName))
		if pvcExists {
			pvc := pvcInterface.(*v1.PersistentVolumeClaim)
			pvcOwner := metav1.GetControllerOf(pvc)
			if pvcOwner != nil && pvcOwner.Kind == "DataVolume" {
				return &pvcOwner.Name
			}
		}
	} else if volume.VolumeSource.DataVolume != nil {
		return &volume.VolumeSource.DataVolume.Name
	}
	return nil
}

func DataVolumeByNameFunc(dataVolumeStore cache.Store, dataVolumes []*cdiv1.DataVolume) func(name string, namespace string) (*cdiv1.DataVolume, error) {
	return func(name, namespace string) (*cdiv1.DataVolume, error) {
		for _, dataVolume := range dataVolumes {
			if dataVolume.Name == name && dataVolume.Namespace == namespace {
				return dataVolume, nil
			}
		}
		dv, exists, _ := dataVolumeStore.GetByKey(fmt.Sprintf("%s/%s", namespace, name))
		if !exists {
			return nil, fmt.Errorf("unable to find datavolume %s/%s", namespace, name)
		}
		return dv.(*cdiv1.DataVolume), nil
	}
}

type DataVolumeConditionManager struct {
}

func NewDataVolumeConditionManager() *DataVolumeConditionManager {
	return &DataVolumeConditionManager{}
}

func (d *DataVolumeConditionManager) GetCondition(dv *cdiv1.DataVolume, cond cdiv1.DataVolumeConditionType) *cdiv1.DataVolumeCondition {
	if dv == nil {
		return nil
	}
	for _, c := range dv.Status.Conditions {
		if c.Type == cond {
			return &c
		}
	}
	return nil
}

func (d *DataVolumeConditionManager) HasCondition(dv *cdiv1.DataVolume, cond cdiv1.DataVolumeConditionType) bool {
	return d.GetCondition(dv, cond) != nil
}

func (d *DataVolumeConditionManager) HasConditionWithStatus(dv *cdiv1.DataVolume, cond cdiv1.DataVolumeConditionType, status v1.ConditionStatus) bool {
	c := d.GetCondition(dv, cond)
	return c != nil && c.Status == status
}

func (d *DataVolumeConditionManager) HasConditionWithStatusAndReason(dv *cdiv1.DataVolume, cond cdiv1.DataVolumeConditionType, status v1.ConditionStatus, reason string) bool {
	c := d.GetCondition(dv, cond)
	return c != nil && c.Status == status && c.Reason == reason
}

func GetStorageClassFromCache(scName string, scStore cache.Store) (*storagev1.StorageClass, error) {
	obj, exists, err := scStore.GetByKey(scName)
	if err != nil {
		return nil, fmt.Errorf("error fetching the storageclass %s: %v", scName, err)
	}
	if !exists {
		return nil, nil
	}

	sc, ok := obj.(*storagev1.StorageClass)
	if !ok {
		return nil, fmt.Errorf("error converting object to StorageClass: object is of type %T", obj)
	}

	return sc.DeepCopy(), nil
}

func GetCSIDriverFromCache(driver string, csiDriverStore cache.Store) (*storagev1.CSIDriver, error) {
	obj, exists, err := csiDriverStore.GetByKey(driver)
	if err != nil {
		return nil, fmt.Errorf("error fetching the csi driver %s: %v", driver, err)
	}
	if !exists {
		return nil, nil
	}

	d, ok := obj.(*storagev1.CSIDriver)
	if !ok {
		return nil, fmt.Errorf("error converting object to storagev1.CSIDriver: object is of type %T", obj)
	}

	return d.DeepCopy(), nil
}

func IsStorageClassCSI(namespace, name string, dataVolumeStore, scStore, csiDriverStore cache.Store) (bool, error) {
	dv, err := GetDataVolumeFromCache(namespace, name, dataVolumeStore)
	if err != nil {
		return false, err
	}
	if dv == nil {
		return false, fmt.Errorf("datavolume %s/%s doesn't exist", namespace, name)
	}
	scName := ""
	if dv.Spec.Storage != nil && dv.Spec.Storage.StorageClassName != nil {
		scName = *dv.Spec.Storage.StorageClassName
	} else if dv.Spec.PVC != nil && dv.Spec.PVC.StorageClassName != nil {
		scName = *dv.Spec.PVC.StorageClassName
	} else {
		return false, fmt.Errorf("storage class for datavolume %s/%s is empty", namespace, name)
	}
	sc, err := GetStorageClassFromCache(scName, scStore)
	if err != nil {
		return false, err
	}
	if sc == nil {
		return false, fmt.Errorf("storage class %s for datavolume %s/%s doesn't exist", scName, namespace, name)
	}
	driver, err := GetCSIDriverFromCache(sc.Provisioner, csiDriverStore)
	if err != nil {
		return false, err
	}

	return driver != nil, nil
}
