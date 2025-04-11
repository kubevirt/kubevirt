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
 * Copyright The KubeVirt Authors.
 *
 */
package testing

import (
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/tools/cache"

	"github.com/onsi/gomega"
)

type interf interface {
	Execute() bool
}

func deepCopyList(objects []interface{}) []interface{} {
	for i := range objects {
		objects[i] = objects[i].(runtime.Object).DeepCopyObject()
	}
	return objects
}

func SanityExecute(c interf, stores []cache.Store, g gomega.Gomega) {
	listOfObjects := [][]interface{}{}

	for _, store := range stores {
		listOfObjects = append(listOfObjects, deepCopyList(store.List()))
	}

	c.Execute()

	for i, objects := range listOfObjects {
		g.ExpectWithOffset(1, stores[i].List()).To(gomega.ConsistOf(objects...))
	}
}
