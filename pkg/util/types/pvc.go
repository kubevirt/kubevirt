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

	k8sv1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/tools/cache"

	"kubevirt.io/kubevirt/pkg/kubecli"
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
	pvc, err = client.CoreV1().PersistentVolumeClaims(namespace).Get(claimName, v1.GetOptions{})
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

func IsPVCShared(pvc *k8sv1.PersistentVolumeClaim) (isShared bool) {
	for _, accessMode := range pvc.Spec.AccessModes {
		if accessMode == k8sv1.ReadWriteMany {
			isShared = true
			break
		}
	}
	return
}

func IsSharedPVCFromClient(client kubecli.KubevirtClient, namespace string, claimName string) (pvc *k8sv1.PersistentVolumeClaim, isShared bool, err error) {
	pvc, err = client.CoreV1().PersistentVolumeClaims(namespace).Get(claimName, v1.GetOptions{})
	if err != nil {
		return nil, false, err
	}

	isShared = IsPVCShared(pvc)
	return pvc, isShared, nil
}
