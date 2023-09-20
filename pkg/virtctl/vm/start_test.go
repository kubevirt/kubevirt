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

var _ = Describe("Start command", func() {
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
		cmd := clientcmd.NewRepeatableVirtctlCommand("start")
		err := cmd()
		Expect(err).To(HaveOccurred())
		Expect(err).Should(MatchError("argument validation failed"))
	})

	It("with dry-run parameter should not start VM", func() {
		vm := kubecli.NewMinimalVM(vmName)
		vm.Spec.Running = pointer.Bool(false)

		kubecli.MockKubevirtClientInstance.EXPECT().VirtualMachine(k8smetav1.NamespaceDefault).Return(vmInterface).Times(1)
		vmInterface.EXPECT().Start(context.Background(), vm.Name, &v1.StartOptions{DryRun: []string{k8smetav1.DryRunAll}}).Return(nil).Times(1)

		cmd := clientcmd.NewRepeatableVirtctlCommand("start", vmName, "--dry-run")
		Expect(cmd()).To(Succeed())
	})

	Context("should patch VM", func() {
		It("with spec:runStrategy:running", func() {
			vm := kubecli.NewMinimalVM(vmName)
			runStrategyHalted := v1.RunStrategyHalted

			vm.Spec.RunStrategy = &runStrategyHalted

			kubecli.MockKubevirtClientInstance.EXPECT().VirtualMachine(k8smetav1.NamespaceDefault).Return(vmInterface).Times(1)
			vmInterface.EXPECT().Start(context.Background(), vm.Name, &v1.StartOptions{Paused: false, DryRun: nil}).Return(nil).Times(1)

			cmd := clientcmd.NewRepeatableVirtctlCommand("start", vmName)
			Expect(cmd()).To(Succeed())
		})

		It("with spec:running:true", func() {
			vm := kubecli.NewMinimalVM(vmName)
			vm.Spec.Running = pointer.Bool(false)

			kubecli.MockKubevirtClientInstance.EXPECT().VirtualMachine(k8smetav1.NamespaceDefault).Return(vmInterface).Times(1)
			vmInterface.EXPECT().Start(context.Background(), vm.Name, &v1.StartOptions{Paused: false, DryRun: nil}).Return(nil).Times(1)

			cmd := clientcmd.NewRepeatableVirtctlCommand("start", vmName)
			Expect(cmd()).To(Succeed())
		})
	})

	Context("With --paused flag", func() {
		It("should start paused if --paused true", func() {
			vm := kubecli.NewMinimalVM(vmName)

			kubecli.MockKubevirtClientInstance.EXPECT().VirtualMachine(k8smetav1.NamespaceDefault).Return(vmInterface).Times(1)
			vmInterface.EXPECT().Start(context.Background(), vm.Name, &v1.StartOptions{Paused: true, DryRun: nil}).Return(nil).Times(1)

			cmd := clientcmd.NewRepeatableVirtctlCommand("start", vmName, "--paused")
			Expect(cmd()).To(Succeed())
		})

		It("should start if --paused false", func() {
			vm := kubecli.NewMinimalVM(vmName)

			kubecli.MockKubevirtClientInstance.EXPECT().VirtualMachine(k8smetav1.NamespaceDefault).Return(vmInterface).Times(1)
			vmInterface.EXPECT().Start(context.Background(), vm.Name, &v1.StartOptions{Paused: false, DryRun: nil}).Return(nil).Times(1)

			cmd := clientcmd.NewRepeatableVirtctlCommand("start", vmName, "--paused=false")
			Expect(cmd()).To(Succeed())
		})
	})

})
