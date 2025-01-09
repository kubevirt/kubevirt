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

package migration

import (
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"golang.org/x/net/context"

	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/resource"
	k8smeta "k8s.io/apimachinery/pkg/apis/meta/v1"

	v1 "kubevirt.io/api/core/v1"
	"kubevirt.io/client-go/kubecli"

	"kubevirt.io/kubevirt/pkg/libvmi"
	"kubevirt.io/kubevirt/pkg/pointer"
	"kubevirt.io/kubevirt/tests/console"
	"kubevirt.io/kubevirt/tests/decorators"
	"kubevirt.io/kubevirt/tests/libkubevirt/config"
	"kubevirt.io/kubevirt/tests/libmigration"
	"kubevirt.io/kubevirt/tests/libpod"
	"kubevirt.io/kubevirt/tests/libstorage"
	"kubevirt.io/kubevirt/tests/libvmifact"
	"kubevirt.io/kubevirt/tests/libvmops"
	"kubevirt.io/kubevirt/tests/libwait"
	"kubevirt.io/kubevirt/tests/testsuite"
)

var _ = Describe("[sig-compute]Migration recovery", decorators.SigCompute, func() {
	DescribeTable("should successfully defer a migration", func(fakeSuccess, flakeCheck bool) {
		virtClient, err := kubecli.GetKubevirtClient()
		Expect(err).NotTo(HaveOccurred())

		if flakeCheck {
			kv := getCurrentKvConfig(virtClient)
			var exists bool
			kv.VMStateStorageClass, exists = libstorage.GetRWOFileSystemStorageClass()
			Expect(exists).To(BeTrue())
			config.UpdateKubeVirtConfigValueAndWait(kv)
		}

		By("Creating a VMI with RWO backend-storage")
		vmi := libvmifact.NewFedora(
			libvmi.WithInterface(libvmi.InterfaceDeviceWithMasqueradeBinding()),
			libvmi.WithNetwork(v1.DefaultPodNetwork()),
		)
		vmi.Spec.Domain.Devices.TPM = &v1.TPMDevice{
			Persistent: pointer.P(true),
		}
		vmi.Namespace = testsuite.GetTestNamespace(vmi)

		By("Creating a migration policy for that VMI")
		migrationPolicy := PreparePolicyAndVMIWithNSAndVMILabelsWithPreexistingPolicy(vmi, nil, 1, 0, nil)
		migrationPolicy.Spec.BandwidthPerMigration = pointer.P(resource.MustParse("1Mi"))
		CreateMigrationPolicy(virtClient, migrationPolicy)

		By("Starting the VMI as a VM")
		vm := libvmi.NewVirtualMachine(vmi, libvmi.WithRunStrategy(v1.RunStrategyManual))
		vm, err = virtClient.VirtualMachine(vmi.Namespace).Create(context.Background(), vm, k8smeta.CreateOptions{})
		Expect(err).NotTo(HaveOccurred())
		vm = libvmops.StartVirtualMachine(vm)
		vmi = libwait.WaitForVMIPhase(vmi, []v1.VirtualMachineInstancePhase{v1.Running})
		Expect(console.LoginToFedora(vmi)).To(Succeed())
		pod, err := libpod.GetPodByVirtualMachineInstance(vmi, vmi.Namespace)
		Expect(err).NotTo(HaveOccurred())

		By("Stressing")
		runStressTest(vmi, stressDefaultVMSize, 42)

		By("Starting a slow migration")
		migration := libmigration.New(vmi.Name, vmi.Namespace)
		migration = libmigration.RunMigration(virtClient, migration)
		Eventually(func() v1.VirtualMachineInstanceMigrationPhase {
			migration, err = virtClient.VirtualMachineInstanceMigration(vmi.Namespace).Get(context.Background(), migration.Name, k8smeta.GetOptions{})
			Expect(err).NotTo(HaveOccurred())
			return migration.Status.Phase
		}).WithTimeout(time.Minute).WithPolling(time.Second).Should(Equal(v1.MigrationRunning))

		By("Killing the source pod")
		err = virtClient.CoreV1().Pods(vmi.Namespace).Delete(context.Background(), pod.Name, k8smeta.DeleteOptions{
			GracePeriodSeconds: pointer.P(int64(0)),
		})
		Expect(err).NotTo(HaveOccurred())

		By("Waiting for the VMI to be gone or failed")
		Eventually(func() string {
			vmi, err = virtClient.VirtualMachineInstance(vmi.Namespace).Get(context.Background(), vmi.Name, k8smeta.GetOptions{})
			if err != nil {
				return err.Error()
			}
			return string(vmi.Status.Phase)
		}).WithTimeout(time.Minute).WithPolling(time.Second).Should(Or(
			ContainSubstring("the server could not find the requested resource"),
			Equal(string(v1.Failed)),
		))

		By("Expecting the migration object, source and target PVCs to still exist")
		migration, err = virtClient.VirtualMachineInstanceMigration(migration.Namespace).Get(context.Background(), migration.Name, k8smeta.GetOptions{})
		Expect(err).NotTo(HaveOccurred())
		Expect(migration.Status.MigrationState).NotTo(BeNil())
		Expect(migration.Status.Phase).To(Equal(v1.MigrationInterrupted))
		sourcePVC := migration.Status.MigrationState.SourcePersistentStatePVCName
		targetPVC := migration.Status.MigrationState.TargetPersistentStatePVCName
		Expect(sourcePVC).NotTo(BeEmpty())
		Expect(targetPVC).NotTo(BeEmpty())
		Expect(sourcePVC).NotTo(Equal(targetPVC), "This test can't run on RWX storage")
		_, err = virtClient.CoreV1().PersistentVolumeClaims(migration.Namespace).Get(context.Background(), sourcePVC, k8smeta.GetOptions{})
		Expect(err).NotTo(HaveOccurred())
		_, err = virtClient.CoreV1().PersistentVolumeClaims(migration.Namespace).Get(context.Background(), targetPVC, k8smeta.GetOptions{})
		Expect(err).NotTo(HaveOccurred())

		keptPVC := sourcePVC
		nukedPVC := targetPVC
		if fakeSuccess {
			By("Simulating a migration success by manually adding /meta/migrated to the source PVC")
			libmigration.FakeMigrationSuccessInPVC(virtClient, sourcePVC, migration.Namespace)

			keptPVC = targetPVC
			nukedPVC = sourcePVC
		}

		By("Starting the VM again")
		vm = libvmops.StartVirtualMachine(vm)

		By("Expecting the right PVC to be removed")
		Eventually(func() error {
			_, err = virtClient.CoreV1().PersistentVolumeClaims(migration.Namespace).Get(context.Background(), nukedPVC, k8smeta.GetOptions{})
			return err
		}).WithTimeout(time.Minute).WithPolling(time.Second).Should(MatchError(errors.IsNotFound, "k8serrors.IsNotFound"))

		By("Expecting the right PVC to be preserved")
		pvc, err := virtClient.CoreV1().PersistentVolumeClaims(migration.Namespace).Get(context.Background(), keptPVC, k8smeta.GetOptions{})
		Expect(err).NotTo(HaveOccurred())
		Expect(pvc.Labels).To(HaveKeyWithValue("persistent-state-for", vmi.Name))
	},
		Entry("failure", decorators.NoFlakeCheck, false, false),
		Entry("success", decorators.NoFlakeCheck, true, false),
		Entry("failure [Serial]", decorators.FlakeCheck, Serial, false, true),
		Entry("success [Serial]", decorators.FlakeCheck, Serial, true, true),
	)
})
