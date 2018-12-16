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
 * Copyright 2018 Red Hat, Inc.
 *
 */

package tests_test

import (
	"flag"
	"fmt"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/rand"

	v1 "kubevirt.io/kubevirt/pkg/api/v1"
	"kubevirt.io/kubevirt/pkg/kubecli"
	"kubevirt.io/kubevirt/tests"
)

var _ = Describe("Migrations", func() {
	flag.Parse()

	virtClient, err := kubecli.GetKubevirtClient()
	tests.PanicOnError(err)

	BeforeEach(func() {
		tests.BeforeTestCleanup()

		nodes, err := virtClient.CoreV1().Nodes().List(metav1.ListOptions{LabelSelector: v1.NodeSchedulable + "=" + "true"})
		Expect(err).To(BeNil())

		if len(nodes.Items) < 2 {
			Skip("Migration tests require at least 2 nodes")
		}
	})

	AfterEach(func() {
	})

	runVMIAndExpectLaunch := func(vmi *v1.VirtualMachineInstance, timeout int) *v1.VirtualMachineInstance {
		By("Starting a VirtualMachineInstance")
		var obj *v1.VirtualMachineInstance
		var err error
		Eventually(func() error {
			obj, err = virtClient.VirtualMachineInstance(tests.NamespaceTestDefault).Create(vmi)
			return err
		}, timeout, 1*time.Second).ShouldNot(HaveOccurred())
		By("Waiting until the VirtualMachineInstance starts")
		tests.WaitForSuccessfulVMIStartWithTimeout(obj, timeout)
		return obj
	}

	confirmVMIPostMigration := func(vmi *v1.VirtualMachineInstance, migrationUID string) {
		By("Retrieving the VMI post migration")
		vmi, err = virtClient.VirtualMachineInstance(vmi.Namespace).Get(vmi.Name, &metav1.GetOptions{})
		Expect(err).To(BeNil())

		By("Verifying the VMI's migration state")
		Expect(vmi.Status.MigrationState).ToNot(BeNil())
		Expect(vmi.Status.MigrationState.StartTimestamp).ToNot(BeNil())
		Expect(vmi.Status.MigrationState.EndTimestamp).ToNot(BeNil())
		Expect(vmi.Status.MigrationState.TargetNode).To(Equal(vmi.Status.NodeName))
		Expect(vmi.Status.MigrationState.TargetNode).ToNot(Equal(vmi.Status.MigrationState.SourceNode))
		Expect(vmi.Status.MigrationState.Completed).To(Equal(true))
		Expect(vmi.Status.MigrationState.Failed).To(Equal(false))
		Expect(vmi.Status.MigrationState.TargetNodeAddress).ToNot(Equal(""))
		Expect(string(vmi.Status.MigrationState.MigrationUID)).To(Equal(migrationUID))

		By("Verifying the VMI's is in the running state")
		Expect(vmi.Status.Phase).To(Equal(v1.Running))
	}

	runMigrationAndExpectCompletion := func(migration *v1.VirtualMachineInstanceMigration, timeout int) string {
		By("Starting a Migration")
		Eventually(func() error {
			_, err := virtClient.VirtualMachineInstanceMigration(migration.Namespace).Create(migration)
			return err
		}, timeout, 1*time.Second).ShouldNot(HaveOccurred())
		By("Waiting until the Migration Completes")

		uid := ""
		Eventually(func() bool {
			migration, err := virtClient.VirtualMachineInstanceMigration(migration.Namespace).Get(migration.Name, &metav1.GetOptions{})
			Expect(err).To(BeNil())

			Expect(migration.Status.Phase).ToNot(Equal(v1.MigrationFailed))

			uid = string(migration.UID)
			if migration.Status.Phase == v1.MigrationSucceeded {
				return true
			}
			return false

		}, timeout, 1*time.Second).Should(Equal(true))
		return uid
	}

	Describe("Starting a VirtualMachineInstance ", func() {
		Context("with an Alpine read only disk", func() {
			It("should be successfully migrated multiple times", func() {

				vmi := tests.NewRandomVMIWithEphemeralDisk(tests.ContainerDiskFor(tests.ContainerDiskAlpine))
				vmi.Spec.Domain.Devices.Disks[0].DiskDevice.Disk.ReadOnly = true

				By("Starting the VirtualMachineInstance")
				vmi = runVMIAndExpectLaunch(vmi, 240)

				By("Checking that the VirtualMachineInstance console has expected output")
				expecter, err := tests.LoggedInAlpineExpecter(vmi)
				Expect(err).To(BeNil())
				expecter.Close()

				num := 2

				for i := 0; i < num; i++ {
					// execute a migration, wait for finalized state
					By(fmt.Sprintf("Starting the Migration for iteration %d", i))
					migration := tests.NewRandomMigration(vmi.Name, vmi.Namespace)
					migrationUID := runMigrationAndExpectCompletion(migration, 180)

					// check VMI, confirm migration state
					confirmVMIPostMigration(vmi, migrationUID)
				}
				// delete VMI
				By("Deleting the VMI")
				err = virtClient.VirtualMachineInstance(vmi.Namespace).Delete(vmi.Name, &metav1.DeleteOptions{})
				Expect(err).To(BeNil())

				By("Waiting for VMI to disappear")
				tests.WaitForVirtualMachineToDisappearWithTimeout(vmi, 240)

			})
		})
		Context("with an Alpine shared ISCSI PVC", func() {
			var pvName string
			BeforeEach(func() {
				pvName = "test-iscsi-lun" + rand.String(48)
				// Start a ISCSI POD and service
				By("Starting an iSCSI POD")
				iscsiIP := tests.CreateISCSITargetPOD()
				// create a new PV and PVC (PVs can't be reused)
				By("create a new iSCSI PV and PVC")
				tests.CreateISCSIPvAndPvc(pvName, "1Gi", iscsiIP)
			}, 60)

			AfterEach(func() {
				// create a new PV and PVC (PVs can't be reused)
				tests.DeletePvAndPvc(pvName)
			}, 60)
			It("should reject migration specs with shared and non-shared disks", func() {
				// Start the VirtualMachineInstance with PVC and Ephemeral Disks
				vmi := tests.NewRandomVMIWithPVC(pvName)
				image := tests.ContainerDiskFor(tests.ContainerDiskAlpine)
				tests.AddEphemeralDisk(vmi, "myephemeral", "virtio", image)

				By("Starting the VirtualMachineInstance")
				vmi = runVMIAndExpectLaunch(vmi, 120)

				By("Checking that the VirtualMachineInstance console has expected output")
				expecter, err := tests.LoggedInAlpineExpecter(vmi)
				Expect(err).To(BeNil())
				expecter.Close()

				By("Starting a Migration and expecting it to be rejected")
				migration := tests.NewRandomMigration(vmi.Name, vmi.Namespace)
				Eventually(func() error {
					_, err := virtClient.VirtualMachineInstanceMigration(migration.Namespace).Create(migration)
					return err
				}, 120, 1*time.Second).Should(HaveOccurred())
			})
			It("should be successfully migrated multiple times", func() {
				// Start the VirtualMachineInstance with the PVC attached
				vmi := tests.NewRandomVMIWithPVC(pvName)

				vmi = runVMIAndExpectLaunch(vmi, 180)

				By("Checking that the VirtualMachineInstance console has expected output")
				expecter, err := tests.LoggedInAlpineExpecter(vmi)
				Expect(err).To(BeNil())
				expecter.Close()

				num := 2

				for i := 0; i < num; i++ {
					// execute a migration, wait for finalized state
					By(fmt.Sprintf("Starting the Migration for iteration %d", i))
					migration := tests.NewRandomMigration(vmi.Name, vmi.Namespace)
					migrationUID := runMigrationAndExpectCompletion(migration, 180)

					// check VMI, confirm migration state
					confirmVMIPostMigration(vmi, migrationUID)
				}
				// delete VMI
				By("Deleting the VMI")
				err = virtClient.VirtualMachineInstance(vmi.Namespace).Delete(vmi.Name, &metav1.DeleteOptions{})
				Expect(err).To(BeNil())

				By("Waiting for VMI to disappear")
				tests.WaitForVirtualMachineToDisappearWithTimeout(vmi, 120)

			})
		})
	})

})
