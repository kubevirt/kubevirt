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
	"context"
	"encoding/json"

	jsonpatch "github.com/evanphx/json-patch"
	"github.com/golang/mock/gomock"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	v1 "kubevirt.io/api/core/v1"

	"kubevirt.io/client-go/kubecli"

	"kubevirt.io/kubevirt/pkg/controller"
	"kubevirt.io/kubevirt/pkg/virt-controller/liveupdate"

	k8score "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
)

var _ = Describe("MemoryHotplugHandler", func() {

	var updater *liveupdate.LiveUpdater
	var memoryHandler *liveupdate.MemoryHotplugHandler
	var vmiInterface *kubecli.MockVirtualMachineInstanceInterface

	BeforeEach(func() {
		ctrl := gomock.NewController(GinkgoT())
		virtClient := kubecli.NewMockKubevirtClient(ctrl)
		vmiInterface = kubecli.NewMockVirtualMachineInstanceInterface(ctrl)
		memoryHandler = liveupdate.NewMemoryHotplugHandler(virtClient)
		updater = &liveupdate.LiveUpdater{}

		virtClient.EXPECT().VirtualMachineInstance(gomock.Any()).Return(vmiInterface).AnyTimes()

		Expect(updater.RegisterHandlers(memoryHandler)).To(Succeed())
	})

	DescribeTable("should patch VMI when memory hotplug is requested", func(resources *v1.ResourceRequirements) {
		newMemory := resource.MustParse("128Mi")
		vm := fakeVM(&newMemory)
		vm.Spec.Template.Spec.Domain.Resources = *resources

		guestMemory := resource.MustParse("64Mi")
		oldVM := vm.DeepCopy()
		oldVM.Spec.Template.Spec.Domain.Memory.Guest = &guestMemory

		vmi := fakeVMI(&guestMemory)
		vmi.Spec.Domain.Resources = *resources

		vmiInterface.EXPECT().Patch(context.Background(), vmi.Name, types.JSONPatchType, gomock.Any(), metav1.PatchOptions{}).Do(
			func(ctx context.Context, name, patchType, patch, opts interface{}, subs ...interface{}) {
				originalVMIBytes, err := json.Marshal(vmi)
				Expect(err).ToNot(HaveOccurred())
				patchBytes := patch.([]byte)

				patchJSON, err := jsonpatch.DecodePatch(patchBytes)
				Expect(err).ToNot(HaveOccurred())
				newVMIBytes, err := patchJSON.Apply(originalVMIBytes)
				Expect(err).ToNot(HaveOccurred())

				var newVMI *v1.VirtualMachineInstance
				err = json.Unmarshal(newVMIBytes, &newVMI)
				Expect(err).ToNot(HaveOccurred())

				Expect(newVMI.Spec.Domain.Memory.Guest.Value()).To(Equal(vm.Spec.Template.Spec.Domain.Memory.Guest.Value()))

				if !resources.Requests.Memory().IsZero() {
					expectedMemReq := resources.Requests.Memory().Value() + newMemory.Value() - guestMemory.Value()
					Expect(newVMI.Spec.Domain.Resources.Requests.Memory().Value()).To(Equal(expectedMemReq))
				}

				if !resources.Limits.Memory().IsZero() {
					expectedMemLimit := resources.Limits.Memory().Value() + newMemory.Value() - guestMemory.Value()
					Expect(newVMI.Spec.Domain.Resources.Limits.Memory().Value()).To(Equal(expectedMemLimit))
				}
			},
		)

		err := updater.HandleLiveUpdates(vm, oldVM, vmi)
		Expect(err).ToNot(HaveOccurred())
	},
		Entry("with memory request set", &v1.ResourceRequirements{
			Requests: k8score.ResourceList{
				k8score.ResourceMemory: resource.MustParse("128Mi"),
			},
		}),
		Entry("with memory request and limits set", &v1.ResourceRequirements{
			Requests: k8score.ResourceList{
				k8score.ResourceMemory: resource.MustParse("128Mi"),
			},
			Limits: k8score.ResourceList{
				k8score.ResourceMemory: resource.MustParse("512Mi"),
			},
		}),
	)

	It("should not patch VMI if memory hotplug is already in progress", func() {
		newMemory := resource.MustParse("128Mi")
		vm := fakeVM(&newMemory)

		guestMemory := resource.MustParse("64Mi")
		oldVM := vm.DeepCopy()
		oldVM.Spec.Template.Spec.Domain.Memory.Guest = &guestMemory

		vmi := fakeVMI(&guestMemory)

		condition := v1.VirtualMachineInstanceCondition{
			Type:   v1.VirtualMachineInstanceMemoryChange,
			Status: k8score.ConditionTrue,
		}
		controller.NewVirtualMachineInstanceConditionManager().UpdateCondition(vmi, &condition)

		Expect(updater.HandleLiveUpdates(vm, oldVM, vmi)).ToNot(Succeed())
	})

	It("should set a restartRequired condition if the memory decreased from start", func() {
		guestMemory := resource.MustParse("64Mi")
		newMemory := resource.MustParse("32Mi")

		vm := fakeVM(&newMemory)
		vmi := fakeVMI(&guestMemory)
		oldVM := vm.DeepCopy()
		oldVM.Spec.Template.Spec.Domain.Memory.Guest = &guestMemory

		Expect(updater.HandleLiveUpdates(oldVM, vm, vmi)).To(Succeed())

		vmConditionController := controller.NewVirtualMachineConditionManager()
		Expect(vmConditionController.HasCondition(vm, v1.VirtualMachineRestartRequired)).To(BeTrue())
	})

})

func fakeVM(memory *resource.Quantity) *v1.VirtualMachine {
	vm := &v1.VirtualMachine{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "fake-vm",
			Namespace: "fake-ns",
		},
		Spec: v1.VirtualMachineSpec{
			Template: &v1.VirtualMachineInstanceTemplateSpec{
				Spec: v1.VirtualMachineInstanceSpec{
					Domain: v1.DomainSpec{
						Memory: &v1.Memory{
							Guest: memory,
						},
					},
				},
			},
		},
	}
	return vm
}

func fakeVMI(memory *resource.Quantity) *v1.VirtualMachineInstance {
	vmi := &v1.VirtualMachineInstance{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "fake-vm",
			Namespace: "fake-ns",
		},
		Spec: v1.VirtualMachineInstanceSpec{
			Domain: v1.DomainSpec{
				Memory: &v1.Memory{
					Guest: memory,
				},
				Resources: v1.ResourceRequirements{
					Requests: k8score.ResourceList{
						k8score.ResourceMemory: *memory,
					},
				},
			},
		},
	}
	vmi.Status.Memory = &v1.MemoryStatus{
		GuestAtBoot:    memory,
		GuestCurrent:   memory,
		GuestRequested: memory,
	}
	return vmi
}
