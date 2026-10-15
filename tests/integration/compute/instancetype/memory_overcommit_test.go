/*
This file is part of the KubeVirt project

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.

Copyright The KubeVirt Authors.
*/

package instancetype_test

import (
	"context"
	"fmt"
	"runtime"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	k8sv1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	virtv1 "kubevirt.io/api/core/v1"

	"kubevirt.io/kubevirt/pkg/hypervisor"
	"kubevirt.io/kubevirt/pkg/libvmi"
	"kubevirt.io/kubevirt/tests/integration/framework"
)

func (s *instancetypeSuite) describeMemoryOvercommit() {
	Context("Memory overcommit", func() {
		var vm *virtv1.VirtualMachine
		BeforeEach(func(ctx SpecContext) {
			vm = createOvercommitVM(ctx, s.f, s.namespace)
		})
		It("should apply the instancetype memory overcommit to VMI and pod requests", func(ctx SpecContext) {
			expectMemoryOvercommit(ctx, s.f, vm)
		})
	})
}

var _ = Describe("Instancetype with cluster memory overcommit", Ordered, ContinueOnFailure, func() {
	var f *framework.Framework
	var vm *virtv1.VirtualMachine

	BeforeAll(func() {
		f = framework.New(framework.WithKubeVirtConfig(&virtv1.KubeVirtConfiguration{
			DeveloperConfiguration: &virtv1.DeveloperConfiguration{MemoryOvercommit: 200},
		}))
		f.Start(context.Background())
	})
	BeforeEach(func(ctx SpecContext) {
		vm = createOvercommitVM(ctx, f, f.CreateNamespace(ctx))
	})
	It("should prefer instancetype memory overcommit over cluster overcommit", func(ctx SpecContext) {
		expectMemoryOvercommit(ctx, f, vm)
	})
})

func createOvercommitVM(ctx context.Context, f *framework.Framework, namespace string) *virtv1.VirtualMachine {
	GinkgoHelper()
	it := newInstancetype()
	it.Spec.Memory.OvercommitPercent = 15
	it, err := f.VirtClient().VirtualMachineInstancetype(namespace).Create(ctx, it, metav1.CreateOptions{})
	Expect(err).NotTo(HaveOccurred())
	vm := libvmi.NewVirtualMachine(libvmi.New(), libvmi.WithInstancetype(it.Name),
		libvmi.WithRunStrategy(virtv1.RunStrategyAlways))
	vm, err = f.VirtClient().VirtualMachine(namespace).Create(ctx, vm, metav1.CreateOptions{})
	Expect(err).NotTo(HaveOccurred())
	return vm
}

func expectMemoryOvercommit(ctx context.Context, f *framework.Framework, vm *virtv1.VirtualMachine) {
	GinkgoHelper()
	vmi := waitForScheduledVMI(ctx, f, vm)
	guest := resource.MustParse("128Mi")
	Expect(*vmi.Spec.Domain.Memory.Guest).To(Equal(guest))
	expectedRequest := int64(float32(guest.Value()) * (1 - float32(15)/100))
	vmiRequest := vmi.Spec.Domain.Resources.Requests[k8sv1.ResourceMemory]
	Expect(vmiRequest.Value()).To(Equal(expectedRequest))
	pods, err := f.K8sClient().CoreV1().Pods(vm.Namespace).List(ctx, metav1.ListOptions{
		LabelSelector: fmt.Sprintf("%s=%s", virtv1.CreatedByLabel, vmi.UID),
	})
	Expect(err).NotTo(HaveOccurred())
	Expect(pods.Items).To(HaveLen(1))
	compute := pods.Items[0].Spec.Containers[0]
	Expect(compute.Name).To(Equal("compute"))
	podRequest := compute.Resources.Requests[k8sv1.ResourceMemory]
	overhead := hypervisor.NewLauncherHypervisorResources("").GetMemoryOverhead(vmi, runtime.GOARCH, nil)
	Expect(podRequest.Value()).To(Equal(expectedRequest + overhead.Value()))
}
