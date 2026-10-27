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

package vmi

import (
	"reflect"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"k8s.io/client-go/tools/cache"

	virtv1 "kubevirt.io/api/core/v1"

	"kubevirt.io/kubevirt/pkg/testutils"
)

var _ = Describe("Informer synchronization", func() {
	var informers Informers

	BeforeEach(func() {
		shared, _ := testutils.NewFakeInformerFor(&virtv1.VirtualMachineInstance{})
		informers = Informers{
			VMI: shared, VM: shared, Pod: shared, PVC: shared,
			Migration: shared, StorageClass: shared, DataVolume: shared,
			StorageProfile: shared, CDI: shared, CDIConfig: shared, KubeVirt: shared,
		}
	})

	for field := range reflect.TypeFor[Informers]().Fields() {
		It("rejects a missing "+field.Name+" informer before constructing the controller", func() {
			reflect.ValueOf(&informers).Elem().FieldByName(field.Name).SetZero()

			controller, err := NewController(ControllerConfig{Informers: informers})
			Expect(err).To(MatchError("informer " + field.Name + " must not be nil"))
			Expect(controller).To(BeNil())
		})
	}

	It("rejects a typed nil informer before constructing the controller", func() {
		var migration *nilInformer
		informers.Migration = migration

		controller, err := NewController(ControllerConfig{Informers: informers})
		Expect(err).To(MatchError("informer Migration must not be nil"))
		Expect(controller).To(BeNil())
	})
})

type nilInformer struct {
	cache.SharedIndexInformer
}
