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
package revision_test

import (
	"context"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"go.uber.org/mock/gomock"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	virtv1 "kubevirt.io/api/core/v1"
	"kubevirt.io/client-go/kubecli"
	"kubevirt.io/client-go/kubevirt/fake"

	"kubevirt.io/kubevirt/pkg/instancetype/revision"
	"kubevirt.io/kubevirt/pkg/libvmi"
)

type clearHandler interface {
	Clear(*virtv1.VirtualMachine) error
}

var _ = Describe("Instancetype and Preferences revision handler", func() {
	var (
		handler    clearHandler
		virtClient *kubecli.MockKubevirtClient
	)

	BeforeEach(func() {
		ctrl := gomock.NewController(GinkgoT())
		virtClient = kubecli.NewMockKubevirtClient(ctrl)
		fakeVMClientset := fake.NewSimpleClientset().KubevirtV1().VirtualMachines(metav1.NamespaceDefault)
		virtClient.EXPECT().VirtualMachine(metav1.NamespaceDefault).Return(fakeVMClientset).AnyTimes()
		handler = revision.New(nil, nil, nil, nil, virtClient)
	})

	instancetypeStatusRef := &virtv1.InstancetypeStatusRef{
		ControllerRevisionRef: &virtv1.ControllerRevisionRef{
			Name: "bar",
		},
		Name: "foo",
	}
	withInstancetypeStatusRef := func() libvmi.VMOption {
		return func(vm *virtv1.VirtualMachine) {
			vm.Status.InstancetypeRef = instancetypeStatusRef
		}
	}
	withPreferenceStatusRef := func() libvmi.VMOption {
		return func(vm *virtv1.VirtualMachine) {
			vm.Status.PreferenceRef = instancetypeStatusRef
		}
	}

	Context("Clear", func() {
		DescribeTable("should", func(vm *virtv1.VirtualMachine, assertVM func(vm *virtv1.VirtualMachine)) {
			vm, err := virtClient.VirtualMachine(vm.Namespace).Create(context.Background(), vm, metav1.CreateOptions{})
			Expect(err).ToNot(HaveOccurred())

			Expect(handler.Clear(vm)).To(Succeed())
			assertVM(vm)

			vm, err = virtClient.VirtualMachine(vm.Namespace).Get(context.Background(), vm.Name, metav1.GetOptions{})
			Expect(err).ToNot(HaveOccurred())
			assertVM(vm)
		},
			Entry(
				"nil vm.status.instancetypeRef when vm.spec.instancetype nil",
				libvmi.NewVirtualMachine(
					libvmi.New(
						libvmi.WithNamespace(metav1.NamespaceDefault),
					),
					withInstancetypeStatusRef(),
				),
				func(vm *virtv1.VirtualMachine) {
					Expect(vm.Spec.Instancetype).To(BeNil())
					Expect(vm.Status.InstancetypeRef).To(BeNil())
				},
			),
			Entry(
				"not nil vm.status.instancetypeRef when vm.spec.instancetype not nil",
				libvmi.NewVirtualMachine(
					libvmi.New(
						libvmi.WithNamespace(metav1.NamespaceDefault),
					),
					libvmi.WithInstancetype("foo"),
					withInstancetypeStatusRef(),
				),
				func(vm *virtv1.VirtualMachine) {
					Expect(vm.Spec.Instancetype).ToNot(BeNil())
					Expect(vm.Status.InstancetypeRef).ToNot(BeNil())
				},
			),
			Entry(
				"nil vm.status.preferenceRef when vm.spec.preference nil",
				libvmi.NewVirtualMachine(
					libvmi.New(
						libvmi.WithNamespace(metav1.NamespaceDefault),
					),
					withPreferenceStatusRef(),
				),
				func(vm *virtv1.VirtualMachine) {
					Expect(vm.Spec.Preference).To(BeNil())
					Expect(vm.Status.PreferenceRef).To(BeNil())
				},
			),
			Entry(
				"not nil vm.status.preferenceRef when vm.spec.preference not nil",
				libvmi.NewVirtualMachine(
					libvmi.New(
						libvmi.WithNamespace(metav1.NamespaceDefault),
					),
					libvmi.WithPreference("foo"),
					withPreferenceStatusRef(),
				),
				func(vm *virtv1.VirtualMachine) {
					Expect(vm.Spec.Preference).ToNot(BeNil())
					Expect(vm.Status.PreferenceRef).ToNot(BeNil())
				},
			),
		)
	})
})
