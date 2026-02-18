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
 * Copyright The KubeVirt Authors
 *
 */

package subresources

import (
	"context"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	v1 "kubevirt.io/api/core/v1"
	"kubevirt.io/client-go/kubecli"

	"kubevirt.io/kubevirt/pkg/libvmi"
	"kubevirt.io/kubevirt/tests/compute"
	"kubevirt.io/kubevirt/tests/framework/kubevirt"
	"kubevirt.io/kubevirt/tests/framework/matcher"
	"kubevirt.io/kubevirt/tests/libnode"
	"kubevirt.io/kubevirt/tests/libvmifact"
	"kubevirt.io/kubevirt/tests/testsuite"
)

var _ = Describe(compute.SIG("Migrate subresource", func() {
	var virtClient kubecli.KubevirtClient

	BeforeEach(func() {
		virtClient = kubevirt.Client()
	})

	It("[test_id:4119]should migrate a running VM", func() {
		nodes := libnode.GetAllSchedulableNodes(virtClient)

		Expect(len(nodes.Items)).To(BeNumerically(">", 1), "Migration tests require at least 2 nodes")

		By("Creating a VM with RunStrategyAlways")
		vm := libvmi.NewVirtualMachine(libvmifact.NewAlpine(
			libvmi.WithInterface(libvmi.InterfaceDeviceWithMasqueradeBinding()),
			libvmi.WithNetwork(v1.DefaultPodNetwork()),
		), libvmi.WithRunStrategy(v1.RunStrategyAlways))
		vm, err := virtClient.VirtualMachine(testsuite.GetTestNamespace(vm)).Create(context.Background(), vm, metav1.CreateOptions{})
		Expect(err).ToNot(HaveOccurred())

		By("Waiting for VM to be ready")
		Eventually(matcher.ThisVM(vm), 360*time.Second, 1*time.Second).Should(matcher.BeReady())

		By("Migrating the VM")
		err = virtClient.VirtualMachine(vm.Namespace).Migrate(context.Background(), vm.Name, &v1.MigrateOptions{})
		Expect(err).ToNot(HaveOccurred())

		By("Ensuring the VirtualMachineInstance is migrated")
		Eventually(func(g Gomega) {
			vmi, err := virtClient.VirtualMachineInstance(vm.Namespace).Get(context.Background(), vm.Name, metav1.GetOptions{})
			g.Expect(err).ToNot(HaveOccurred())
			g.Expect(vmi.Status.MigrationState).ToNot(BeNil(), "vmi.Status.MigrationState should not be nil")
			g.Expect(vmi.Status.MigrationState.Completed).To(BeTrueBecause("migration should be completed"))
		}, 240*time.Second, 1*time.Second).Should(Succeed())
	})

	It("[test_id:7743]should not migrate a running vm if dry-run option is passed", func() {
		nodes := libnode.GetAllSchedulableNodes(virtClient)
		if len(nodes.Items) < 2 {
			Fail("Migration tests require at least 2 nodes")
		}
		By("Creating a VM with RunStrategyAlways")
		vm := libvmi.NewVirtualMachine(libvmifact.NewAlpine(
			libvmi.WithInterface(libvmi.InterfaceDeviceWithMasqueradeBinding()),
			libvmi.WithNetwork(v1.DefaultPodNetwork()),
		), libvmi.WithRunStrategy(v1.RunStrategyAlways))
		vm, err := virtClient.VirtualMachine(testsuite.GetTestNamespace(vm)).Create(context.Background(), vm, metav1.CreateOptions{})
		Expect(err).ToNot(HaveOccurred())

		By("Waiting for VM to be ready")
		Eventually(matcher.ThisVM(vm), 360*time.Second, 1*time.Second).Should(matcher.BeReady())

		By("Migrating the VM with dry-run option")
		err = virtClient.VirtualMachine(vm.Namespace).Migrate(context.Background(), vm.Name, &v1.MigrateOptions{DryRun: []string{metav1.DryRunAll}})
		Expect(err).ToNot(HaveOccurred())

		By("Check that no migration was actually created")
		Consistently(func() error {
			_, err = virtClient.VirtualMachineInstanceMigration(vm.Namespace).Get(context.Background(), vm.Name, metav1.GetOptions{})
			return err
		}, 60*time.Second, 1*time.Second).Should(MatchError(errors.IsNotFound, "k8serrors.IsNotFound"), "migration should not be created in a dry run mode")
	})
}))
