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
 * Copyright 2017, 2018 Red Hat, Inc.
 *
 */

package snapshot

import (
	"fmt"
	"time"

	k8sv1 "k8s.io/api/core/v1"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/apimachinery/pkg/util/sets"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/util/workqueue"

	"kubevirt.io/client-go/log"
)

func processWorkItem(queue workqueue.RateLimitingInterface, handler func(string) (time.Duration, error)) bool {
	obj, shutdown := queue.Get()
	if shutdown {
		return false
	}

	err := func(obj interface{}) error {
		defer queue.Done(obj)
		key, ok := obj.(string)
		if !ok {
			queue.Forget(obj)
			utilruntime.HandleError(fmt.Errorf("expected string in workqueue but got %#v", obj))
			return nil
		}

		log.Log.V(3).Infof("processing key [%s]", key)

		if requeueAfter, err := handler(key); requeueAfter > 0 || err != nil {
			if requeueAfter > 0 {
				queue.AddAfter(key, requeueAfter)
			} else {
				queue.AddRateLimited(key)
			}

			return err
		}

		queue.Forget(obj)

		return nil

	}(obj)

	if err != nil {
		utilruntime.HandleError(err)
		return true
	}

	return true
}

func podsUsingPVCs(podInformer cache.SharedIndexInformer, namespace string, pvcNames sets.String) ([]k8sv1.Pod, error) {
	var pods []k8sv1.Pod

	if pvcNames.Len() < 1 {
		return pods, nil
	}

	objs, err := podInformer.GetIndexer().ByIndex(cache.NamespaceIndex, namespace)
	if err != nil {
		return nil, err
	}

	for _, obj := range objs {
		pod, ok := obj.(*k8sv1.Pod)
		if !ok {
			return nil, fmt.Errorf("expected Pod, got %T", obj)
		}

		for _, volume := range pod.Spec.Volumes {
			if volume.VolumeSource.PersistentVolumeClaim != nil &&
				pvcNames.Has(volume.PersistentVolumeClaim.ClaimName) {
				pods = append(pods, *pod)
			}
		}
	}

	return pods, nil
}
