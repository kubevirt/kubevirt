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

package creation

import (
	"kubevirt.io/kubevirt/pkg/api/v1"
	"kubevirt.io/kubevirt/pkg/kubecli"
	"kubevirt.io/kubevirt/pkg/log"
	"kubevirt.io/kubevirt/pkg/virt-operator/creation/components"
	"kubevirt.io/kubevirt/pkg/virt-operator/creation/rbac"
	"kubevirt.io/kubevirt/pkg/virt-operator/util"
)

func Create(kv *v1.KubeVirt, config util.KubeVirtDeploymentConfig, stores util.Stores, clientset kubecli.KubevirtClient, expectations *util.Expectations) (int, error) {

	objectsAdded := 0

	added, err := rbac.CreateClusterRBAC(clientset, kv, stores, expectations)
	objectsAdded = objectsAdded + added
	if err != nil {
		log.Log.Errorf("Failed to create cluster RBAC: %v", err)
		return objectsAdded, err
	}
	added, err = rbac.CreateApiServerRBAC(clientset, kv, stores, expectations)
	objectsAdded = objectsAdded + added
	if err != nil {
		log.Log.Errorf("Failed to create apiserver RBAC: %v", err)
		return objectsAdded, err
	}

	added, err = rbac.CreateControllerRBAC(clientset, kv, stores, expectations)
	objectsAdded = objectsAdded + added
	if err != nil {
		log.Log.Errorf("Failed to create controller RBAC: %v", err)
		return objectsAdded, err
	}

	added, err = rbac.CreateHandlerRBAC(clientset, kv, stores, expectations)
	objectsAdded = objectsAdded + added
	if err != nil {
		log.Log.Errorf("Failed to create handler RBAC: %v", err)
		return objectsAdded, err
	}

	err = util.UpdateScc(clientset, stores.SCCCache, kv, true)

	if err != nil {
		log.Log.Errorf("Failed to update SCC: %v", err)
		return objectsAdded, err
	}

	added, err = components.CreateCRDs(clientset, kv, stores, expectations)
	objectsAdded = objectsAdded + added
	if err != nil {
		log.Log.Errorf("Failed to create crds: %v", err)
		return objectsAdded, err
	}

	added, err = components.CreateControllers(clientset, kv, config, stores, expectations)
	objectsAdded = objectsAdded + added
	if err != nil {
		log.Log.Errorf("Failed to create controllers: %v", err)
		return objectsAdded, err
	}

	return objectsAdded, nil
}
