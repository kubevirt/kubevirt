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
	v1 "kubevirt.io/api/core/v1"
	"kubevirt.io/client-go/kubecli"

	"kubevirt.io/kubevirt/tests/clientcmd"
)

var _ = Describe("Restart command", func() {
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
		cmd := clientcmd.NewRepeatableVirtctlCommand("restart")
		err := cmd()
		Expect(err).To(HaveOccurred())
		Expect(err).Should(MatchError("argument validation failed"))
	})

	DescribeTable("test", func(restartOptions v1.RestartOptions, runStrategy bool, running bool, args ...string) {
		vm := kubecli.NewMinimalVM(vmName)
		runStrategyManual := v1.RunStrategyManual

		if runStrategy {
			vm.Spec.RunStrategy = &runStrategyManual
		}

		if running {
			vm.Spec.Running = &running
		}

		kubecli.MockKubevirtClientInstance.EXPECT().VirtualMachine(k8smetav1.NamespaceDefault).Return(vmInterface).Times(1)
		vmInterface.EXPECT().Restart(context.Background(), vm.Name, &restartOptions).Return(nil).Times(1)

		cmd := clientcmd.NewRepeatableVirtctlCommand(args...)
		Expect(cmd()).To(Succeed())
	},
		Entry("should restart vm", v1.RestartOptions{DryRun: nil}, false, false, "restart", vmName),
		Entry("should restart a vm with runStrategy:manual", v1.RestartOptions{DryRun: nil}, true, false, "restart", vmName),
		Entry("with dry-run parameter should not restart VM", v1.RestartOptions{DryRun: []string{k8smetav1.DryRunAll}}, false, true, "restart", vmName, "--dry-run"),
	)

	It("should force restart vm", func() {
		vm := kubecli.NewMinimalVM(vmName)

		kubecli.MockKubevirtClientInstance.EXPECT().VirtualMachine(k8smetav1.NamespaceDefault).Return(vmInterface).Times(1)
		gracePeriod := int64(0)
		restartOptions := v1.RestartOptions{
			GracePeriodSeconds: &gracePeriod,
			DryRun:             nil,
		}
		vmInterface.EXPECT().ForceRestart(context.Background(), vm.Name, &restartOptions).Return(nil).Times(1)

		cmd := clientcmd.NewRepeatableVirtctlCommand("restart", vmName, "--force", "--grace-period=0")
		Expect(cmd()).To(Succeed())
	})
})
