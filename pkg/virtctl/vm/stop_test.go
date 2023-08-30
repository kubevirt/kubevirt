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
* Copyright 2023 Red Hat, Inc.
*
 */

package vm_test

import (
	"context"

	"github.com/golang/mock/gomock"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	k8smetav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/pointer"
	v1 "kubevirt.io/api/core/v1"
	"kubevirt.io/client-go/kubecli"

	"kubevirt.io/kubevirt/tests/clientcmd"
)

var _ = Describe("Stop command", func() {
	var vmInterface *kubecli.MockVirtualMachineInterface
	var ctrl *gomock.Controller
	const vmName = "testvm"

	BeforeEach(func() {
		ctrl = gomock.NewController(GinkgoT())
		kubecli.GetKubevirtClientFromClientConfig = kubecli.GetMockKubevirtClientFromClientConfig
		kubecli.MockKubevirtClientInstance = kubecli.NewMockKubevirtClient(ctrl)
		vmInterface = kubecli.NewMockVirtualMachineInterface(ctrl)
	})

	It("should fail with missing input parameters", func() {
		cmd := clientcmd.NewRepeatableVirtctlCommand("stop")
		err := cmd()
		Expect(err).To(HaveOccurred())
		Expect(err).Should(MatchError("argument validation failed"))
	})

	It("with dry-run parameter should not stop VM", func() {
		vm := kubecli.NewMinimalVM(vmName)
		vm.Spec.Running = pointer.Bool(true)

		kubecli.MockKubevirtClientInstance.EXPECT().VirtualMachine(k8smetav1.NamespaceDefault).Return(vmInterface).Times(1)
		vmInterface.EXPECT().Stop(context.Background(), vm.Name, &v1.StopOptions{DryRun: []string{k8smetav1.DryRunAll}}).Return(nil).Times(1)

		cmd := clientcmd.NewRepeatableVirtctlCommand("stop", vmName, "--dry-run")
		Expect(cmd()).To(Succeed())
	})

	It("should force stop vm", func() {
		vm := kubecli.NewMinimalVM(vmName)

		kubecli.MockKubevirtClientInstance.EXPECT().VirtualMachine(k8smetav1.NamespaceDefault).Return(vmInterface).Times(1)
		gracePeriod := int64(0)
		stopOptions := v1.StopOptions{
			GracePeriod: &gracePeriod,
			DryRun:      nil,
		}
		vmInterface.EXPECT().ForceStop(context.Background(), vm.Name, &stopOptions).Return(nil).Times(1)

		cmd := clientcmd.NewRepeatableVirtctlCommand("stop", vmName, "--force", "--grace-period=0")
		Expect(cmd()).To(Succeed())
	})

	DescribeTable("should patch VM", func(modifyFn func(vm *v1.VirtualMachine), args ...string) {
		vm := kubecli.NewMinimalVM(vmName)
		modifyFn(vm)

		kubecli.MockKubevirtClientInstance.EXPECT().VirtualMachine(k8smetav1.NamespaceDefault).Return(vmInterface).Times(1)
		vmInterface.EXPECT().Stop(context.Background(), vm.Name, &v1.StopOptions{DryRun: nil}).Return(nil).Times(1)

		cmd := clientcmd.NewRepeatableVirtctlCommand(args...)
		Expect(cmd()).To(Succeed())
	},
		Entry("with spec:runStrategy:always",
			func(vm *v1.VirtualMachine) {
				runStrategy := v1.RunStrategyAlways
				vm.Spec.RunStrategy = &runStrategy
			},
			"stop", vmName),
		Entry("with spec:runStrategy:halted when it's false already",
			func(vm *v1.VirtualMachine) {
				runStrategy := v1.RunStrategyHalted
				vm.Spec.RunStrategy = &runStrategy
			},
			"stop", vmName),
		Entry("with spec:running:false",
			func(vm *v1.VirtualMachine) {
				vm.Spec.Running = pointer.Bool(true)
			},
			"stop", vmName),
		Entry("with spec:running:false when it's false already",
			func(vm *v1.VirtualMachine) {
				vm.Spec.Running = pointer.Bool(false)
			},
			"stop", vmName),
	)
})
