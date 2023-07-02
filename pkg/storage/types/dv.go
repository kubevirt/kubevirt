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

	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/tools/cache"

	virtv1 "kubevirt.io/api/core/v1"
	"kubevirt.io/client-go/kubecli"
	"kubevirt.io/client-go/log"
	cdiv1 "kubevirt.io/containerized-data-importer-api/pkg/apis/core/v1beta1"

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

	labels := map[string]string{}
	for k, v := range dataVolumeTemplate.Labels {
		labels[k] = v
	}
	newDataVolume.ObjectMeta.Labels = labels

	annotations := map[string]string{}
	for k, v := range dataVolumeTemplate.Annotations {
		annotations[k] = v
	}
	newDataVolume.ObjectMeta.Annotations = annotations

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

func GetDataVolumeFromCache(namespace, name string, dataVolumeInformer cache.SharedInformer) (*cdiv1.DataVolume, error) {
	key := controller.NamespacedKey(namespace, name)
	obj, exists, err := dataVolumeInformer.GetStore().GetByKey(key)

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

	return dv, nil
}

func HasDataVolumeErrors(namespace string, volumes []virtv1.Volume, dataVolumeInformer cache.SharedInformer) error {
	for _, volume := range volumes {
		if volume.DataVolume == nil {
			continue
		}

		dv, err := GetDataVolumeFromCache(namespace, volume.DataVolume.Name, dataVolumeInformer)
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
			dvRunningCond.Reason == "Error" {
			return fmt.Errorf("DataVolume %s importer has stopped running due to an error: %v",
				volume.DataVolume.Name, dvRunningCond.Message)
		}
	}

	return nil
}

func HasDataVolumeProvisioning(namespace string, volumes []virtv1.Volume, dataVolumeInformer cache.SharedInformer) bool {
	for _, volume := range volumes {
		if volume.DataVolume == nil {
			continue
		}

		dv, err := GetDataVolumeFromCache(namespace, volume.DataVolume.Name, dataVolumeInformer)
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

func ListDataVolumesFromTemplates(namespace string, dvTemplates []virtv1.DataVolumeTemplateSpec, dataVolumeInformer cache.SharedInformer) ([]*cdiv1.DataVolume, error) {
	dataVolumes := []*cdiv1.DataVolume{}

	for _, template := range dvTemplates {
		// get DataVolume from cache for each templated dataVolume
		dv, err := GetDataVolumeFromCache(namespace, template.Name, dataVolumeInformer)
		if err != nil {
			return dataVolumes, err
		} else if dv == nil {
			continue
		}

		dataVolumes = append(dataVolumes, dv)
	}
	return dataVolumes, nil
}

func ListDataVolumesFromVolumes(namespace string, volumes []virtv1.Volume, dataVolumeInformer cache.SharedInformer, pvcInformer cache.SharedInformer) ([]*cdiv1.DataVolume, error) {
	dataVolumes := []*cdiv1.DataVolume{}

	for _, volume := range volumes {
		dataVolumeName := getDataVolumeName(namespace, volume, pvcInformer)
		if dataVolumeName == nil {
			continue
		}

		dv, err := GetDataVolumeFromCache(namespace, *dataVolumeName, dataVolumeInformer)
		if err != nil {
			return dataVolumes, err
		} else if dv == nil {
			continue
		}

		dataVolumes = append(dataVolumes, dv)
	}

	return dataVolumes, nil
}

func getDataVolumeName(namespace string, volume virtv1.Volume, pvcInformer cache.SharedInformer) *string {
	if volume.VolumeSource.PersistentVolumeClaim != nil {
		pvcInterface, pvcExists, _ := pvcInformer.GetStore().
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

func DataVolumeByNameFunc(dataVolumeInformer cache.SharedInformer, dataVolumes []*cdiv1.DataVolume) func(name string, namespace string) (*cdiv1.DataVolume, error) {
	return func(name, namespace string) (*cdiv1.DataVolume, error) {
		for _, dataVolume := range dataVolumes {
			if dataVolume.Name == name && dataVolume.Namespace == namespace {
				return dataVolume, nil
			}
		}
		dv, exists, _ := dataVolumeInformer.GetStore().GetByKey(fmt.Sprintf("%s/%s", namespace, name))
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
