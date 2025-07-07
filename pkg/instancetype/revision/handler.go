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
 * Copyright The KubeVirt Authors
 *
 */
package revision

import (
	"k8s.io/client-go/tools/cache"

	"kubevirt.io/client-go/kubecli"
)

type revisionHandler struct {
	instancetypeStore        cache.Store
	clusterInstancetypeStore cache.Store
	preferenceStore          cache.Store
	clusterPreferenceStore   cache.Store
	virtClient               kubecli.KubevirtClient
}

func New(
	instancetypeStore,
	clusterInstancetypeStore,
	preferenceStore,
	clusterPreferenceStore cache.Store,
	virtClient kubecli.KubevirtClient,
) *revisionHandler {
	return &revisionHandler{
		instancetypeStore:        instancetypeStore,
		clusterInstancetypeStore: clusterInstancetypeStore,
		preferenceStore:          preferenceStore,
		clusterPreferenceStore:   clusterPreferenceStore,
		virtClient:               virtClient,
	}
}
