/*
Copyright The KubeVirt Authors.
SPDX-License-Identifier: Apache-2.0
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
