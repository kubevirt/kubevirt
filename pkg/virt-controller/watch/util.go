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

package watch

import (
	"fmt"
	"time"

	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
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
