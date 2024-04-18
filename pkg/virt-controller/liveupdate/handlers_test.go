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
 * Copyright 2024 The KubeVirt Authors.
 *
 */

package liveupdate_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	types "github.com/onsi/gomega/types"

	v1 "kubevirt.io/api/core/v1"

	"kubevirt.io/kubevirt/pkg/pointer"
	"kubevirt.io/kubevirt/pkg/virt-controller/liveupdate"

	"k8s.io/apimachinery/pkg/api/resource"
)

type fakeLiveUpdateHandler struct {
	HandleLiveUpdateCounter int
	managedFields           []string
}

func newFakeHandler(fakeFields []string) *fakeLiveUpdateHandler {
	return &fakeLiveUpdateHandler{
		managedFields: fakeFields,
	}
}

func (fake *fakeLiveUpdateHandler) GetManagedFields() []string {
	return fake.managedFields
}

func (fake *fakeLiveUpdateHandler) HandleLiveUpdate(vm *v1.VirtualMachine, vmi *v1.VirtualMachineInstance) error {
	fake.HandleLiveUpdateCounter++
	return nil
}

var _ = Describe("LiveUpdater", func() {

	var updater *liveupdate.LiveUpdater

	BeforeEach(func() {
		updater = &liveupdate.LiveUpdater{}
	})

	DescribeTable("Registering a handler", func(fields []string, matcher types.GomegaMatcher) {
		fakeHandler := newFakeHandler(fields)
		Expect(updater.RegisterHandlers(fakeHandler)).To(matcher)
	},
		Entry("should fail if field does not exist", []string{"/Spec/Running", "/Spec/Foo/Bar"}, Not(Succeed())),
		Entry("should succeed", []string{"/Spec/Running", "/Spec/Template/Spec/Domain/Memory/Guest"}, Succeed()),
	)

	It("Fitering updatable fields should copy over only managed fields", func() {
		oldVM := v1.VirtualMachine{
			Spec: v1.VirtualMachineSpec{
				Running: pointer.P(true),
				Template: &v1.VirtualMachineInstanceTemplateSpec{
					Spec: v1.VirtualMachineInstanceSpec{
						Domain: v1.DomainSpec{
							Memory: &v1.Memory{
								Guest: pointer.P(resource.MustParse("128Mi")),
							},
						},
					},
				},
			},
		}

		newVM := oldVM.DeepCopy()
		newVM.Spec.Template.Spec.Domain.Memory.Guest = pointer.P(resource.MustParse("256Mi"))
		newVM.Spec.Running = pointer.P(false)

		fakeHandler := newFakeHandler([]string{"/Spec/Template/Spec/Domain/Memory/Guest"})
		Expect(updater.RegisterHandlers(fakeHandler)).To(Succeed())
		Expect(updater.FilterUpdatableFields(newVM, &oldVM)).To(Succeed())

		Expect(newVM.Spec.Template.Spec.Domain.Memory.Guest.Value()).To(Equal(oldVM.Spec.Template.Spec.Domain.Memory.Guest.Value()))
		Expect(newVM.Spec.Running).ToNot(Equal(oldVM.Spec.Running))
	})

	DescribeTable("LiveUpdate handlers", func(handlers []liveupdate.LiveUpdateHandler, oldVM, newVM *v1.VirtualMachine, updateCounter int) {
		Expect(updater.RegisterHandlers(handlers...)).To(Succeed())
		Expect(updater.HandleLiveUpdates(newVM, oldVM, nil)).To(Succeed())

		// how many times handlers have been called
		sumCounter := 0
		for _, handler := range handlers {
			fakeHandler := handler.(*fakeLiveUpdateHandler)
			sumCounter = sumCounter + fakeHandler.HandleLiveUpdateCounter
		}
		Expect(sumCounter).To(BeNumerically("==", updateCounter))
	},
		Entry("should not be called if nothing changed",
			[]liveupdate.LiveUpdateHandler{newFakeHandler([]string{"/Spec/Running"})},
			&v1.VirtualMachine{Spec: v1.VirtualMachineSpec{Running: pointer.P(true)}},
			&v1.VirtualMachine{Spec: v1.VirtualMachineSpec{Running: pointer.P(true)}},
			0,
		),
		Entry("should be called when managed values change",
			[]liveupdate.LiveUpdateHandler{newFakeHandler([]string{"/Spec/Running"})},
			&v1.VirtualMachine{Spec: v1.VirtualMachineSpec{Running: pointer.P(true)}},
			&v1.VirtualMachine{Spec: v1.VirtualMachineSpec{Running: pointer.P(false)}},
			1,
		),
		Entry("should not be called repeatedly when managed values change (only the first handler should)",
			[]liveupdate.LiveUpdateHandler{
				newFakeHandler([]string{"/Spec/Running"}),
				newFakeHandler([]string{"/Spec/Running"}),
			},
			&v1.VirtualMachine{Spec: v1.VirtualMachineSpec{Running: pointer.P(true)}},
			&v1.VirtualMachine{Spec: v1.VirtualMachineSpec{Running: pointer.P(false)}},
			1,
		),
	)

})
