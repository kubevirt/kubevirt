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

package vm_test

import (
	"context"

	"github.com/golang/mock/gomock"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	k8smetav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	v1 "kubevirt.io/api/core/v1"
	"kubevirt.io/client-go/kubecli"

	"kubevirt.io/kubevirt/pkg/virtctl/testing"
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
		cmd := testing.NewRepeatableVirtctlCommand("migrate")
		err := cmd()
		Expect(err).To(HaveOccurred())
		Expect(err).Should(MatchError("accepts 1 arg(s), received 0"))
	})

	DescribeTable("should migrate a vm according to options", func(expectedMigrateOptions *v1.MigrateOptions, extraArgs ...string) {
		vm := kubecli.NewMinimalVM(vmName)

		kubecli.MockKubevirtClientInstance.EXPECT().VirtualMachine(k8smetav1.NamespaceDefault).Return(vmInterface).Times(1)
		vmInterface.EXPECT().Migrate(context.Background(), vm.Name, expectedMigrateOptions).Return(nil).Times(1)

		args := []string{"migrate", vmName}
		args = append(args, extraArgs...)
		Expect(testing.NewRepeatableVirtctlCommand(args...)()).To(Succeed())
	},
		Entry(
			"with default",
			&v1.MigrateOptions{}),
		Entry(
			"with dry-run option",
			&v1.MigrateOptions{
				DryRun: []string{k8smetav1.DryRunAll}},
			"--dry-run"),
		Entry(
			"with addedNodeSelector option",
			&v1.MigrateOptions{
				AddedNodeSelector: map[string]string{"key1": "value1", "key2": "value2"}},
			"--addedNodeSelector", "key1=value1,key2=value2"),
		Entry(
			"with dry-run and addedNodeSelector options",
			&v1.MigrateOptions{
				AddedNodeSelector: map[string]string{"key1": "value1", "key2": "value2"},
				DryRun:            []string{k8smetav1.DryRunAll}},
			"--dry-run", "--addedNodeSelector", "key1=value1,key2=value2"),
		Entry(
			"with repeated addedNodeSelector",
			&v1.MigrateOptions{
				AddedNodeSelector: map[string]string{"key1": "value1", "key2": "value2"}},
			"--addedNodeSelector", "key1=value1", "--addedNodeSelector", "key2=value2"),
	)

	DescribeTable("should fail with badly formatted addedNodeSelector", func(extraArgs ...string) {
		args := []string{"migrate", vmName}
		args = append(args, extraArgs...)
		err := testing.NewRepeatableVirtctlCommand(args...)()
		Expect(err).To(HaveOccurred())
		Expect(err).Should(MatchError(ContainSubstring("must be formatted as key=value")))
	},
		Entry(
			"with key only",
			"--addedNodeSelector", "key1"),
		Entry(
			"with multiple keys only",
			"--addedNodeSelector", "key1,key2"),
	)

})
