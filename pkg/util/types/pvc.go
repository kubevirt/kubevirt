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
	"context"
	"fmt"
	"strconv"
	"strings"

	k8sv1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/tools/cache"

	"kubevirt.io/client-go/kubecli"
	cdiv1beta1 "kubevirt.io/containerized-data-importer/pkg/apis/core/v1beta1"
)

func IsPVCBlockFromStore(store cache.Store, namespace string, claimName string) (pvc *k8sv1.PersistentVolumeClaim, exists bool, isBlockDevice bool, err error) {
	obj, exists, err := store.GetByKey(namespace + "/" + claimName)
	if err != nil || !exists {
		return nil, exists, false, err
	}
	if pvc, ok := obj.(*k8sv1.PersistentVolumeClaim); ok {
		return obj.(*k8sv1.PersistentVolumeClaim), true, isPVCBlock(pvc), nil
	}
	return nil, false, false, fmt.Errorf("this is not a PVC! %v", obj)
}

func IsPVCBlockFromClient(client kubecli.KubevirtClient, namespace string, claimName string) (pvc *k8sv1.PersistentVolumeClaim, exists bool, isBlockDevice bool, err error) {
	pvc, err = client.CoreV1().PersistentVolumeClaims(namespace).Get(context.Background(), claimName, v1.GetOptions{})
	if errors.IsNotFound(err) {
		return nil, false, false, nil
	} else if err != nil {
		return nil, false, false, err
	}
	return pvc, true, isPVCBlock(pvc), nil
}

func isPVCBlock(pvc *k8sv1.PersistentVolumeClaim) bool {
	// We do not need to consider the data in a PersistentVolume (as of Kubernetes 1.9)
	// If a PVC does not specify VolumeMode and the PV specifies VolumeMode = Block
	// the claim will not be bound. So for the sake of a boolean answer, if the PVC's
	// VolumeMode is Block, that unambiguously answers the question
	return pvc.Spec.VolumeMode != nil && *pvc.Spec.VolumeMode == k8sv1.PersistentVolumeBlock
}

func IsPVCShared(pvc *k8sv1.PersistentVolumeClaim) bool {
	for _, accessMode := range pvc.Spec.AccessModes {
		if accessMode == k8sv1.ReadWriteMany {
			return true
		}
	}
	return false
}

func IsSharedPVCFromClient(client kubecli.KubevirtClient, namespace string, claimName string) (pvc *k8sv1.PersistentVolumeClaim, isShared bool, err error) {
	pvc, err = client.CoreV1().PersistentVolumeClaims(namespace).Get(context.Background(), claimName, v1.GetOptions{})
	if err == nil {
		isShared = IsPVCShared(pvc)
	}
	return
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

func getStorageClassName(client kubecli.KubevirtClient, pvc *k8sv1.PersistentVolumeClaim) (string, error) {
	scName := pvc.Spec.StorageClassName
	if scName == nil {
		scList, err := client.StorageV1().StorageClasses().List(context.Background(), v1.ListOptions{})
		if err != nil {
			return "", err
		}
		for _, sc := range scList.Items {
			if sc.Annotations["storageclass.kubernetes.io/is-default-class"] == "true" {
				return sc.Name, nil
			}
		}
		return "", nil
	}
	return *scName, nil
}

func getFilesystemOverhead(client kubecli.KubevirtClient, pvc *k8sv1.PersistentVolumeClaim) (cdiv1beta1.Percent, error) {
	if isPVCBlock(pvc) {
		return "0", nil
	}
	cdiConfigs, err := client.CdiClient().CdiV1beta1().CDIConfigs().List(context.Background(), v1.ListOptions{})
	if err != nil || len(cdiConfigs.Items) == 0 {
		return "0", err
	}
	cdiConfig := cdiv1beta1.CDIConfig{}
	for _, cdiConfig = range cdiConfigs.Items {
		break
	}
	if cdiConfig.Status.FilesystemOverhead == nil {
		return "0", nil
	}
	storageClassName, err := getStorageClassName(client, pvc)
	if err != nil {
		return "0", err
	}
	fsOverhead, ok := cdiConfig.Status.FilesystemOverhead.StorageClass[storageClassName]
	if !ok {
		fsOverhead = cdiConfig.Status.FilesystemOverhead.Global
	}
	return fsOverhead, nil
}

func ExpectedDiskSize(client kubecli.KubevirtClient, pvc *k8sv1.PersistentVolumeClaim) (int64, bool) {
	capacityResource, ok := pvc.Status.Capacity[k8sv1.ResourceStorage]
	if !ok {
		return 0, false
	}
	capacity, ok := capacityResource.AsInt64()
	if !ok {
		return 0, false
	}
	filesystemOverheadString, err := getFilesystemOverhead(client, pvc)
	if err != nil {
		return 0, false
	}
	filesystemOverhead, err := strconv.ParseFloat(string(filesystemOverheadString), 64)
	if err != nil {
		return 0, false
	}

	return int64((1 - filesystemOverhead) * float64(capacity)), true
}
