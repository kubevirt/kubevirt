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

var _ = Describe("Migrate command", func() {
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
		cmd := clientcmd.NewRepeatableVirtctlCommand("migrate")
		err := cmd()
		Expect(err).To(HaveOccurred())
		Expect(err).Should(MatchError("argument validation failed"))
	})

	DescribeTable("should migrate a vm according to options", func(migrateOptions *v1.MigrateOptions) {
		vm := kubecli.NewMinimalVM(vmName)

		kubecli.MockKubevirtClientInstance.EXPECT().VirtualMachine(k8smetav1.NamespaceDefault).Return(vmInterface).Times(1)
		vmInterface.EXPECT().Migrate(context.Background(), vm.Name, migrateOptions).Return(nil).Times(1)

		var cmd func() error
		if len(migrateOptions.DryRun) == 0 {
			cmd = clientcmd.NewRepeatableVirtctlCommand("migrate", vmName)
		} else {
			cmd = clientcmd.NewRepeatableVirtctlCommand("migrate", "--dry-run", vmName)
		}

		Expect(cmd()).To(Succeed())
	},
		Entry("with default", &v1.MigrateOptions{}),
		Entry("with dry-run option", &v1.MigrateOptions{DryRun: []string{k8smetav1.DryRunAll}}),
	)
})
