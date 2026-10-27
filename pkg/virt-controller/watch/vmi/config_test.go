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

	for field := range reflect.TypeFor[Informers]().Fields() {
		It("waits for the "+field.Name+" informer to sync", func() {
			shared := &syncStatusInformer{synced: true}
			value := reflect.ValueOf(&informers).Elem()
			for _, field := range value.Fields() {
				field.Set(reflect.ValueOf(shared))
			}
			pending := &syncStatusInformer{}
			value.FieldByName(field.Name).Set(reflect.ValueOf(pending))

			hasSynced, err := informers.syncFunc()
			Expect(err).NotTo(HaveOccurred())
			Expect(hasSynced()).To(BeFalse())

			pending.synced = true
			Expect(hasSynced()).To(BeTrue())
		})
	}
})
