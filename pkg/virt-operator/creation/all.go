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
	"kubevirt.io/kubevirt/pkg/virt-operator/install-strategy"
	"kubevirt.io/kubevirt/pkg/virt-operator/util"
)

func Create(kv *v1.KubeVirt, stores util.Stores, clientset kubecli.KubevirtClient, expectations *util.Expectations, strategy *installstrategy.InstallStrategy) (int, error) {

	objectsAdded, err := installstrategy.CreateAll(kv, strategy, stores, clientset, expectations)

	if err != nil {
		return objectsAdded, err
	}

	err = util.UpdateScc(clientset, stores.SCCCache, kv, true)
	if err != nil {
		return objectsAdded, err
	}

	log.Log.Object(kv).Infof("Created %d objects this round", objectsAdded)
	return objectsAdded, nil
}
