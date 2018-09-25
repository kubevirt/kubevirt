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
	"k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/tools/cache"

	"kubevirt.io/kubevirt/pkg/kubecli"
)

func IsPVCBlockFromStore(store cache.Store, namespace string, claimName string) (bool, error) {
	pvc, err := getPersistentVolumeClaim(store, namespace, claimName)
	if err != nil {
		return false, err
	}
	if pvc == nil {
		return false, fmt.Errorf("unknown persistentvolumeclaim: %v/%v", namespace, claimName)
	}
	return isPVCBlock(pvc), nil
}

func IsPVCBlockFromClient(client kubecli.KubevirtClient, namespace string, claimName string) (bool, error) {
	pvc, err := client.CoreV1().PersistentVolumeClaims(namespace).Get(claimName, v1.GetOptions{})
	if err != nil {
		return false, err
	}
	if pvc == nil {
		return false, fmt.Errorf("unknown persistentvolumeclaim: %v/%v", namespace, claimName)
	}
	return isPVCBlock(pvc), nil
}

func getPersistentVolumeClaim(store cache.Store, namespace string, name string) (*k8sv1.PersistentVolumeClaim, error) {
	if obj, exists, err := store.GetByKey(namespace + "/" + name); err != nil {
		return nil, err
	} else if !exists {
		return nil, nil
	} else {
		if pvc, ok := obj.(*k8sv1.PersistentVolumeClaim); ok {
			return pvc, nil
		}
		return nil, fmt.Errorf("this is not a PVC! %v", obj)
	}
}

func isPVCBlock(pvc *k8sv1.PersistentVolumeClaim) bool {
	// We do not need to consider the data in a PersistentVolume (as of Kubernetes 1.9)
	// If a PVC does not specify VolumeMode and the PV specifies VolumeMode = Block
	// the claim will not be bound. So for the sake of a boolean answer, if the PVC's
	// VolumeMode is Block, that unambiguously answers the question
	return pvc.Spec.VolumeMode != nil && *pvc.Spec.VolumeMode == k8sv1.PersistentVolumeBlock
}
