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
