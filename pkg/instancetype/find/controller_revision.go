/*
Copyright The KubeVirt Authors.
SPDX-License-Identifier: Apache-2.0
*/

package find

import (
	"context"
	"fmt"

	appsv1 "k8s.io/api/apps/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/cache"
	"kubevirt.io/client-go/kubecli"
)

type controllerRevisionFinder struct {
	store      cache.Store
	virtClient kubecli.KubevirtClient
}

func NewControllerRevisionFinder(store cache.Store, virtClient kubecli.KubevirtClient) *controllerRevisionFinder {
	return &controllerRevisionFinder{
		store:      store,
		virtClient: virtClient,
	}
}

func (f *controllerRevisionFinder) Find(namespacedName types.NamespacedName) (*appsv1.ControllerRevision, error) {
	if f.store == nil {
		return f.virtClient.AppsV1().ControllerRevisions(namespacedName.Namespace).Get(
			context.Background(), namespacedName.Name, metav1.GetOptions{})
	}

	obj, exists, err := f.store.GetByKey(namespacedName.String())
	if err != nil {
		return nil, err
	}
	if !exists {
		return f.virtClient.AppsV1().ControllerRevisions(namespacedName.Namespace).Get(
			context.Background(), namespacedName.Name, metav1.GetOptions{})
	}
	revision, ok := obj.(*appsv1.ControllerRevision)
	if !ok {
		return nil, fmt.Errorf("unknown object type found in ControllerRevision informer")
	}
	return revision, nil
}
