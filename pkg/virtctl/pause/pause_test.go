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
 */

package pause_test

import (
	"context"

	"github.com/golang/mock/gomock"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	k8smetav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	v1 "kubevirt.io/api/core/v1"
	"kubevirt.io/client-go/api"
	"kubevirt.io/client-go/kubecli"

	"kubevirt.io/kubevirt/pkg/virtctl/testing"
)

var _ = Describe("Pausing", func() {

	const (
		COMMAND_PAUSE = "pause"
		vmName        = "testvm"
	)
	var vmInterface *kubecli.MockVirtualMachineInterface
	var vmiInterface *kubecli.MockVirtualMachineInstanceInterface
	var ctrl *gomock.Controller

	BeforeEach(func() {
		ctrl = gomock.NewController(GinkgoT())
		kubecli.GetKubevirtClientFromClientConfig = kubecli.GetMockKubevirtClientFromClientConfig
		kubecli.MockKubevirtClientInstance = kubecli.NewMockKubevirtClient(ctrl)
		vmInterface = kubecli.NewMockVirtualMachineInterface(ctrl)
		vmiInterface = kubecli.NewMockVirtualMachineInstanceInterface(ctrl)
	})

	Context("With missing input parameters", func() {
		It("should fail a pause", func() {
			cmd := testing.NewRepeatableVirtctlCommand(COMMAND_PAUSE)
			err := cmd()
			Expect(err).To(HaveOccurred())
		})
	})

	DescribeTable("should pause VMI", func(pauseOptions *v1.PauseOptions) {
		vmi := api.NewMinimalVMI(vmName)

		kubecli.MockKubevirtClientInstance.EXPECT().VirtualMachineInstance(k8smetav1.NamespaceDefault).Return(vmiInterface).Times(1)
		vmiInterface.EXPECT().Pause(context.Background(), vmi.Name, pauseOptions).Return(nil).Times(1)

		args := []string{COMMAND_PAUSE, "vmi", vmName}
		if len(pauseOptions.DryRun) > 0 {
			args = append(args, "--dry-run")
		}
		Expect(testing.NewRepeatableVirtctlCommand(args...)()).To(Succeed())
	},
		Entry("", &v1.PauseOptions{}),
		Entry("with dry-run option", &v1.PauseOptions{DryRun: []string{k8smetav1.DryRunAll}}),
	)

	DescribeTable("should pause VM", func(pauseOptions *v1.PauseOptions) {
		vmi := api.NewMinimalVMI(vmName)
		vm := kubecli.NewMinimalVM(vmName)
		vm.Spec.Template = &v1.VirtualMachineInstanceTemplateSpec{
			Spec: vmi.Spec,
		}

		kubecli.MockKubevirtClientInstance.EXPECT().VirtualMachine(k8smetav1.NamespaceDefault).Return(vmInterface).Times(1)
		kubecli.MockKubevirtClientInstance.EXPECT().VirtualMachineInstance(k8smetav1.NamespaceDefault).Return(vmiInterface).Times(1)

		vmInterface.EXPECT().Get(context.Background(), vm.Name, k8smetav1.GetOptions{}).Return(vm, nil).Times(1)
		vmiInterface.EXPECT().Pause(context.Background(), vm.Name, pauseOptions).Return(nil).Times(1)

		args := []string{COMMAND_PAUSE, "vm", vmName}
		if len(pauseOptions.DryRun) > 0 {
			args = append(args, "--dry-run")
		}
		Expect(testing.NewRepeatableVirtctlCommand(args...)()).To(Succeed())
	},
		Entry("", &v1.PauseOptions{}),
		Entry("with dry-run option", &v1.PauseOptions{DryRun: []string{k8smetav1.DryRunAll}}),
	)
})
