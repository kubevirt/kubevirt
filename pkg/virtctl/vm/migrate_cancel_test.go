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
	"fmt"

	"github.com/golang/mock/gomock"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	k8smetav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	v1 "kubevirt.io/api/core/v1"
	"kubevirt.io/client-go/kubecli"

	"kubevirt.io/kubevirt/pkg/virtctl/testing"
)

var _ = Describe("Migrate cancel command", func() {
	var migrationInterface *kubecli.MockVirtualMachineInstanceMigrationInterface
	var ctrl *gomock.Controller

	var listoptions k8smetav1.ListOptions
	var vmiMigration *v1.VirtualMachineInstanceMigration
	var vm *v1.VirtualMachine
	const vmName = "testvm"

	BeforeEach(func() {
		ctrl = gomock.NewController(GinkgoT())
		kubecli.GetKubevirtClientFromClientConfig = kubecli.GetMockKubevirtClientFromClientConfig
		kubecli.MockKubevirtClientInstance = kubecli.NewMockKubevirtClient(ctrl)
		migrationInterface = kubecli.NewMockVirtualMachineInstanceMigrationInterface(ctrl)

		vm = kubecli.NewMinimalVM(vmName)
		vmiMigration = kubecli.NewMinimalMigration(fmt.Sprintf("%s-%s", vm.Name, "migration")) // "testvm-migration"
		listoptions = k8smetav1.ListOptions{LabelSelector: fmt.Sprintf("%s==%s", v1.MigrationSelectorLabel, vm.Name)}
	})

	It("should fail with missing input parameters", func() {
		cmd := testing.NewRepeatableVirtctlCommand("migrate-cancel")
		err := cmd()
		Expect(err).To(HaveOccurred())
		Expect(err).Should(MatchError("accepts 1 arg(s), received 0"))
	})

	It("should cancel the vm migration", func() {
		cmd := testing.NewRepeatableVirtctlCommand("migrate-cancel", vm.Name)

		vmiMigration.Status.Phase = v1.MigrationRunning
		migList := v1.VirtualMachineInstanceMigrationList{
			Items: []v1.VirtualMachineInstanceMigration{
				*vmiMigration,
			},
		}

		kubecli.MockKubevirtClientInstance.EXPECT().
			VirtualMachineInstanceMigration(k8smetav1.NamespaceDefault).
			Return(migrationInterface).Times(2)

		migrationInterface.EXPECT().List(context.Background(), listoptions).Return(&migList, nil).Times(1)
		migrationInterface.EXPECT().Delete(context.Background(), vmiMigration.Name, k8smetav1.DeleteOptions{}).Return(nil).Times(1)

		Expect(cmd()).To(Succeed())
	})

	It("Should fail if no active migration is found", func() {
		cmd := testing.NewRepeatableVirtctlCommand("migrate-cancel", vm.Name)

		vmiMigration.Status.Phase = v1.MigrationSucceeded
		migList := v1.VirtualMachineInstanceMigrationList{
			Items: []v1.VirtualMachineInstanceMigration{
				*vmiMigration,
			},
		}

		kubecli.MockKubevirtClientInstance.EXPECT().
			VirtualMachineInstanceMigration(k8smetav1.NamespaceDefault).
			Return(migrationInterface).Times(1)

		migrationInterface.EXPECT().List(context.Background(), listoptions).Return(&migList, nil).Times(1)

		err := cmd()
		Expect(err).To(HaveOccurred())
		errstr := fmt.Sprintf("Found no migration to cancel for %s", vm.Name)
		Expect(err.Error()).To(ContainSubstring(errstr))
	})

})
