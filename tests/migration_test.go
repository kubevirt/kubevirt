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

	expect "github.com/google/goexpect"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	k8sv1 "k8s.io/api/core/v1"
	"k8s.io/api/policy/v1beta1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/rand"

	v1 "kubevirt.io/kubevirt/pkg/api/v1"
	"kubevirt.io/kubevirt/pkg/kubecli"
	migrations2 "kubevirt.io/kubevirt/pkg/util/migrations"
	"kubevirt.io/kubevirt/tests"
)

var _ = Describe("Migrations", func() {
	flag.Parse()

	virtClient, err := kubecli.GetKubevirtClient()
	tests.PanicOnError(err)

	BeforeEach(func() {
		tests.BeforeTestCleanup()
		if !tests.HasLiveMigration() {
			Skip("LiveMigration feature gate is not enabled in kubevirt-config")
		}

		nodes := tests.GetAllSchedulableNodes(virtClient)
		Expect(nodes.Items).ToNot(BeEmpty(), "There should be some compute node")

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
		vmi, err = virtClient.VirtualMachineInstance(tests.NamespaceTestDefault).Get(vmi.Name, &metav1.GetOptions{})
		Expect(err).ToNot(HaveOccurred())
		return vmi
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

	confirmVMIPostMigrationFailed := func(vmi *v1.VirtualMachineInstance, migrationUID string) {
		By("Retrieving the VMI post migration")
		vmi, err = virtClient.VirtualMachineInstance(vmi.Namespace).Get(vmi.Name, &metav1.GetOptions{})
		Expect(err).To(BeNil())

		By("Verifying the VMI's migration state")
		Expect(vmi.Status.MigrationState).ToNot(BeNil())
		Expect(vmi.Status.MigrationState.StartTimestamp).ToNot(BeNil())
		Expect(vmi.Status.MigrationState.EndTimestamp).ToNot(BeNil())
		Expect(vmi.Status.MigrationState.SourceNode).To(Equal(vmi.Status.NodeName))
		Expect(vmi.Status.MigrationState.TargetNode).ToNot(Equal(vmi.Status.MigrationState.SourceNode))
		Expect(vmi.Status.MigrationState.Completed).To(Equal(true))
		Expect(vmi.Status.MigrationState.Failed).To(Equal(true))
		Expect(vmi.Status.MigrationState.TargetNodeAddress).ToNot(Equal(""))
		Expect(string(vmi.Status.MigrationState.MigrationUID)).To(Equal(migrationUID))

		By("Verifying the VMI's is in the running state")
		Expect(vmi.Status.Phase).To(Equal(v1.Running))
	}
	confirmVMIPostMigrationAborted := func(vmi *v1.VirtualMachineInstance, migrationUID string, timeout int) *v1.VirtualMachineInstance {
		By("Waiting until the migration is completed")
		Eventually(func() bool {
			vmi, err = virtClient.VirtualMachineInstance(vmi.Namespace).Get(vmi.Name, &metav1.GetOptions{})
			Expect(err).To(BeNil())

			Expect(vmi.Status.MigrationState).ToNot(BeNil())

			if vmi.Status.MigrationState.Completed &&
				vmi.Status.MigrationState.AbortStatus == v1.MigrationAbortSucceeded {
				return true
			}
			return false

		}, timeout, 1*time.Second).Should(Equal(true))

		By("Retrieving the VMI post migration")
		vmi, err = virtClient.VirtualMachineInstance(vmi.Namespace).Get(vmi.Name, &metav1.GetOptions{})
		Expect(err).To(BeNil())

		By("Verifying the VMI's migration state")
		Expect(vmi.Status.MigrationState).ToNot(BeNil())
		Expect(vmi.Status.MigrationState.StartTimestamp).ToNot(BeNil())
		Expect(vmi.Status.MigrationState.EndTimestamp).ToNot(BeNil())
		Expect(vmi.Status.MigrationState.SourceNode).To(Equal(vmi.Status.NodeName))
		Expect(vmi.Status.MigrationState.TargetNode).ToNot(Equal(vmi.Status.MigrationState.SourceNode))
		Expect(vmi.Status.MigrationState.TargetNodeAddress).ToNot(Equal(""))
		Expect(string(vmi.Status.MigrationState.MigrationUID)).To(Equal(migrationUID))
		Expect(vmi.Status.MigrationState.Failed).To(Equal(true))
		Expect(vmi.Status.MigrationState.AbortRequested).To(Equal(true))

		By("Verifying the VMI's is in the running state")
		Expect(vmi.Status.Phase).To(Equal(v1.Running))
		return vmi
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
	runAndCancelMigration := func(migration *v1.VirtualMachineInstanceMigration, vmi *v1.VirtualMachineInstance, timeout int) string {
		By("Starting a Migration")
		Eventually(func() error {
			_, err := virtClient.VirtualMachineInstanceMigration(migration.Namespace).Create(migration)
			return err
		}, timeout, 1*time.Second).ShouldNot(HaveOccurred())

		By("Waiting until the Migration is Running")

		uid := ""
		Eventually(func() bool {
			migration, err := virtClient.VirtualMachineInstanceMigration(migration.Namespace).Get(migration.Name, &metav1.GetOptions{})
			Expect(err).To(BeNil())

			Expect(migration.Status.Phase).ToNot(Equal(v1.MigrationFailed))
			uid = string(migration.UID)
			if migration.Status.Phase == v1.MigrationRunning {
				vmi, err = virtClient.VirtualMachineInstance(vmi.Namespace).Get(vmi.Name, &metav1.GetOptions{})
				Expect(err).To(BeNil())
				if vmi.Status.MigrationState.Completed != true {
					return true
				}
			}
			return false

		}, timeout, 1*time.Second).Should(Equal(true))

		By("Cancelling a Migration")
		Expect(virtClient.VirtualMachineInstanceMigration(migration.Namespace).Delete(migration.Name, &metav1.DeleteOptions{})).To(Succeed(), "Migration should be deleted successfully")

		return uid
	}

	runMigrationAndExpectFailure := func(migration *v1.VirtualMachineInstanceMigration, timeout int) string {
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

			Expect(migration.Status.Phase).NotTo(Equal(v1.MigrationSucceeded))

			uid = string(migration.UID)
			if migration.Status.Phase == v1.MigrationFailed {
				return true
			}
			return false

		}, timeout, 1*time.Second).Should(Equal(true))
		return uid
	}

	Describe("Starting a VirtualMachineInstance ", func() {
		Context("with a Cirros disk", func() {
			It("should be successfully migrated multiple times with cloud-init disk", func() {

				vmi := tests.NewRandomVMIWithEphemeralDisk(tests.ContainerDiskFor(tests.ContainerDiskCirros))
				tests.AddUserData(vmi, "cloud-init", "#!/bin/bash\necho 'hello'\n")

				By("Starting the VirtualMachineInstance")
				vmi = runVMIAndExpectLaunch(vmi, 240)

				By("Checking that the VirtualMachineInstance console has expected output")
				expecter, err := tests.LoggedInCirrosExpecter(vmi)
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
				iscsiIP := tests.CreateISCSITargetPOD(tests.ContainerDiskAlpine)
				// create a new PV and PVC (PVs can't be reused)
				By("create a new iSCSI PV and PVC")
				tests.CreateISCSIPvAndPvc(pvName, "1Gi", iscsiIP)
			}, 60)

			AfterEach(func() {
				// create a new PV and PVC (PVs can't be reused)
				tests.DeletePvAndPvc(pvName)
			}, 60)
			It("should migrate a VMI with shared and non-shared disks", func() {
				// Start the VirtualMachineInstance with PVC and Ephemeral Disks
				vmi := tests.NewRandomVMIWithPVC(pvName)
				image := tests.ContainerDiskFor(tests.ContainerDiskAlpine)
				tests.AddEphemeralDisk(vmi, "myephemeral", "virtio", image)

				By("Starting the VirtualMachineInstance")
				vmi = runVMIAndExpectLaunch(vmi, 180)

				By("Checking that the VirtualMachineInstance console has expected output")
				expecter, err := tests.LoggedInAlpineExpecter(vmi)
				Expect(err).To(BeNil())
				expecter.Close()

				By("Starting a Migration")
				migration := tests.NewRandomMigration(vmi.Name, vmi.Namespace)
				migrationUID := runMigrationAndExpectCompletion(migration, 180)

				// check VMI, confirm migration state
				confirmVMIPostMigration(vmi, migrationUID)

				// delete VMI
				By("Deleting the VMI")
				err = virtClient.VirtualMachineInstance(vmi.Namespace).Delete(vmi.Name, &metav1.DeleteOptions{})
				Expect(err).To(BeNil())

				By("Waiting for VMI to disappear")
				tests.WaitForVirtualMachineToDisappearWithTimeout(vmi, 120)
			})
			It("should be successfully migrated multiple times", func() {
				// Start the VirtualMachineInstance with the PVC attached
				vmi := tests.NewRandomVMIWithPVC(pvName)
				vmi = runVMIAndExpectLaunch(vmi, 180)

				By("Checking that the VirtualMachineInstance console has expected output")
				expecter, err := tests.LoggedInAlpineExpecter(vmi)
				Expect(err).To(BeNil())
				expecter.Close()

				// execute a migration, wait for finalized state
				migration := tests.NewRandomMigration(vmi.Name, vmi.Namespace)
				migrationUID := runMigrationAndExpectCompletion(migration, 180)

				// check VMI, confirm migration state
				confirmVMIPostMigration(vmi, migrationUID)

				// delete VMI
				By("Deleting the VMI")
				err = virtClient.VirtualMachineInstance(vmi.Namespace).Delete(vmi.Name, &metav1.DeleteOptions{})
				Expect(err).To(BeNil())

				By("Waiting for VMI to disappear")
				tests.WaitForVirtualMachineToDisappearWithTimeout(vmi, 120)

			})
		})
		Context("with an Cirros shared ISCSI PVC", func() {
			var pvName string
			BeforeEach(func() {
				pvName = "test-iscsi-lun" + rand.String(48)
				// Start a ISCSI POD and service
				By("Starting an iSCSI POD")
				iscsiIP := tests.CreateISCSITargetPOD(tests.ContainerDiskCirros)
				// create a new PV and PVC (PVs can't be reused)
				By("create a new iSCSI PV and PVC")
				tests.CreateISCSIPvAndPvc(pvName, "1Gi", iscsiIP)
			}, 60)

			AfterEach(func() {
				// create a new PV and PVC (PVs can't be reused)
				tests.DeletePvAndPvc(pvName)
			}, 60)
			It("should be successfully with a cloud init", func() {
				// Start the VirtualMachineInstance with the PVC attached
				vmi := tests.NewRandomVMIWithPVC(pvName)
				tests.AddUserData(vmi, "cloud-init", "#!/bin/bash\necho 'hello'\n")
				vmi.Spec.Hostname = fmt.Sprintf("%s", tests.ContainerDiskCirros)
				vmi = runVMIAndExpectLaunch(vmi, 180)

				By("Checking that the VirtualMachineInstance console has expected output")
				expecter, err := tests.LoggedInCirrosExpecter(vmi)
				Expect(err).To(BeNil())
				expecter.Close()

				// execute a migration, wait for finalized state
				By("Starting the Migration for iteration")
				migration := tests.NewRandomMigration(vmi.Name, vmi.Namespace)
				migrationUID := runMigrationAndExpectCompletion(migration, 180)

				// check VMI, confirm migration state
				confirmVMIPostMigration(vmi, migrationUID)

				// delete VMI
				By("Deleting the VMI")
				err = virtClient.VirtualMachineInstance(vmi.Namespace).Delete(vmi.Name, &metav1.DeleteOptions{})
				Expect(err).To(BeNil())

				By("Waiting for VMI to disappear")
				tests.WaitForVirtualMachineToDisappearWithTimeout(vmi, 120)
			})
		})
		Context("migration monitor", func() {
			var options metav1.GetOptions
			var cfgMap *k8sv1.ConfigMap
			var originalMigrationConfig string
			var kubevirtConfig = "kubevirt-config"

			BeforeEach(func() {
				// update migration timeouts
				options = metav1.GetOptions{}
				cfgMap, err = virtClient.CoreV1().ConfigMaps(namespaceKubevirt).Get(kubevirtConfig, options)
				Expect(err).ToNot(HaveOccurred())
				originalMigrationConfig = cfgMap.Data["migrations"]
				cfgMap.Data["migrations"] = `{"progressTimeout" : 5, "completionTimeoutPerGiB": 5}`

				_, err = virtClient.CoreV1().ConfigMaps(namespaceKubevirt).Update(cfgMap)
				Expect(err).ToNot(HaveOccurred())
				time.Sleep(5 * time.Second)
			})
			AfterEach(func() {
				cfgMap, err = virtClient.CoreV1().ConfigMaps(namespaceKubevirt).Get(kubevirtConfig, options)
				Expect(err).ToNot(HaveOccurred())
				cfgMap.Data["migrations"] = originalMigrationConfig
				_, err = virtClient.CoreV1().ConfigMaps(namespaceKubevirt).Update(cfgMap)
				Expect(err).ToNot(HaveOccurred())
			})
			It("should abort a vmi migration without progress", func() {

				vmi := tests.NewRandomVMIWithEphemeralDisk(tests.ContainerDiskFor(tests.ContainerDiskFedora))
				vmi.Spec.Domain.Resources.Requests[k8sv1.ResourceMemory] = resource.MustParse("1Gi")
				tests.AddUserData(vmi, "cloud-init", fmt.Sprintf(`#!/bin/bash
					echo "fedora" |passwd fedora --stdin
					yum install -y stress qemu-guest-agent
                    systemctl start  qemu-guest-agent`))

				By("Starting the VirtualMachineInstance")
				vmi = runVMIAndExpectLaunch(vmi, 240)

				getOptions := &metav1.GetOptions{}
				var updatedVmi *v1.VirtualMachineInstance
				By("Checking that the VirtualMachineInstance console has expected output")
				expecter, expecterErr := tests.LoggedInFedoraExpecter(vmi)
				Expect(expecterErr).To(BeNil())
				defer expecter.Close()

				// Need to wait for cloud init to finnish and start the agent inside the vmi.
				Eventually(func() bool {
					updatedVmi, err = virtClient.VirtualMachineInstance(tests.NamespaceTestDefault).Get(vmi.Name, getOptions)
					Expect(err).ToNot(HaveOccurred())
					for _, condition := range updatedVmi.Status.Conditions {
						if condition.Type == "AgentConnected" && condition.Status == "True" {
							return true
						}
					}
					return false
				}, 420*time.Second, 2).Should(BeTrue(), "Should have agent connected condition")

				By("Run a stress test")
				_, err = expecter.ExpectBatch([]expect.Batcher{
					&expect.BSnd{S: "stress --vm 1 --vm-bytes 600M --vm-keep --timeout 1600s&\n"},
				}, 15*time.Second)
				Expect(err).ToNot(HaveOccurred(), "should run a stress test")

				// execute a migration, wait for finalized state
				By("Starting the Migration")
				migration := tests.NewRandomMigration(vmi.Name, vmi.Namespace)
				migrationUID := runMigrationAndExpectFailure(migration, 180)

				// check VMI, confirm migration state
				confirmVMIPostMigrationFailed(vmi, migrationUID)

				// delete VMI
				By("Deleting the VMI")
				err = virtClient.VirtualMachineInstance(vmi.Namespace).Delete(vmi.Name, &metav1.DeleteOptions{})
				Expect(err).To(BeNil())

				By("Waiting for VMI to disappear")
				tests.WaitForVirtualMachineToDisappearWithTimeout(vmi, 240)
			})
		})
		Context("with an Cirros non-shared ISCSI PVC", func() {
			var pvName string
			BeforeEach(func() {
				pvName = "test-iscsi-lun" + rand.String(48)
				// Start a ISCSI POD and service
				By("Starting an iSCSI POD")
				iscsiIP := tests.CreateISCSITargetPOD(tests.ContainerDiskCirros)
				// create a new PV and PVC (PVs can't be reused)
				By("create a new iSCSI PV and PVC")
				tests.NewISCSIPvAndPvc(pvName, "1Gi", iscsiIP, k8sv1.ReadWriteOnce)
			}, 60)

			AfterEach(func() {
				// create a new PV and PVC (PVs can't be reused)
				tests.DeletePvAndPvc(pvName)
			}, 60)
			It("should reject migrations for a non-migratable vmi", func() {
				// Start the VirtualMachineInstance with the PVC attached
				vmi := tests.NewRandomVMIWithPVC(pvName)
				tests.AddUserData(vmi, "cloud-init", "#!/bin/bash\necho 'hello'\n")
				vmi.Spec.Hostname = fmt.Sprintf("%s", tests.ContainerDiskCirros)
				vmi = runVMIAndExpectLaunch(vmi, 180)

				By("Checking that the VirtualMachineInstance console has expected output")
				expecter, err := tests.LoggedInCirrosExpecter(vmi)
				Expect(err).To(BeNil())
				expecter.Close()

				for _, c := range vmi.Status.Conditions {
					if c.Type == v1.VirtualMachineInstanceIsMigratable {
						Expect(c.Status).To(Equal(k8sv1.ConditionFalse))
					}
				}

				// execute a migration, wait for finalized state
				migration := tests.NewRandomMigration(vmi.Name, vmi.Namespace)

				By("Starting a Migration")
				_, err = virtClient.VirtualMachineInstanceMigration(migration.Namespace).Create(migration)
				Expect(err).ToNot(BeNil())
				Expect(err.Error()).To(ContainSubstring("DisksNotLiveMigratable"))

				// delete VMI
				By("Deleting the VMI")
				err = virtClient.VirtualMachineInstance(vmi.Namespace).Delete(vmi.Name, &metav1.DeleteOptions{})
				Expect(err).To(BeNil())

				By("Waiting for VMI to disappear")
				tests.WaitForVirtualMachineToDisappearWithTimeout(vmi, 120)
			})
			It("should be able successfully cancel a migration", func() {

				vmi := tests.NewRandomVMIWithEphemeralDisk(tests.ContainerDiskFor(tests.ContainerDiskFedora))
				vmi.Spec.Domain.Resources.Requests[k8sv1.ResourceMemory] = resource.MustParse("1Gi")
				tests.AddUserData(vmi, "cloud-init", fmt.Sprintf(`#!/bin/bash
					echo "fedora" |passwd fedora --stdin
					yum install -y stress qemu-guest-agent
                    systemctl start  qemu-guest-agent`))

				By("Starting the VirtualMachineInstance")
				vmi = runVMIAndExpectLaunch(vmi, 240)

				getOptions := &metav1.GetOptions{}
				var updatedVmi *v1.VirtualMachineInstance
				By("Checking that the VirtualMachineInstance console has expected output")
				expecter, expecterErr := tests.LoggedInFedoraExpecter(vmi)
				Expect(expecterErr).To(BeNil())
				defer expecter.Close()

				// Need to wait for cloud init to finnish and start the agent inside the vmi.
				Eventually(func() bool {
					updatedVmi, err = virtClient.VirtualMachineInstance(tests.NamespaceTestDefault).Get(vmi.Name, getOptions)
					Expect(err).ToNot(HaveOccurred())
					for _, condition := range updatedVmi.Status.Conditions {
						if condition.Type == "AgentConnected" && condition.Status == "True" {
							return true
						}
					}
					return false
				}, 420*time.Second, 2).Should(BeTrue(), "Should have agent connected condition")

				By("Run a stress test")
				_, err = expecter.ExpectBatch([]expect.Batcher{
					&expect.BSnd{S: "stress --vm 1 --vm-bytes 600M --vm-keep --timeout 1600s&\n"},
				}, 15*time.Second)
				Expect(err).ToNot(HaveOccurred(), "should run a stress test")

				// execute a migration, wait for finalized state
				By("Starting the Migration")
				migration := tests.NewRandomMigration(vmi.Name, vmi.Namespace)

				migrationUID := runAndCancelMigration(migration, vmi, 180)

				// check VMI, confirm migration state
				confirmVMIPostMigrationAborted(vmi, migrationUID, 180)

				By("Waiting for the migration object to disappear")
				tests.WaitForMigrationToDisappearWithTimeout(migration, 240)

				// delete VMI
				By("Deleting the VMI")
				err = virtClient.VirtualMachineInstance(vmi.Namespace).Delete(vmi.Name, &metav1.DeleteOptions{})
				Expect(err).To(BeNil())

				By("Waiting for VMI to disappear")
				tests.WaitForVirtualMachineToDisappearWithTimeout(vmi, 240)

			})
		})
	})

	Context("with a live-migrate eviction strategy set", func() {

		AfterEach(func() {
			tests.CleanNodes()
		})

		Context("with a VMI running with an eviction strategy set", func() {

			var vmi *v1.VirtualMachineInstance

			BeforeEach(func() {
				vmi = vmiWithEvictionStrategy()
			})

			It("should block the eviction api", func() {
				vmi = runVMIAndExpectLaunch(vmi, 180)
				pod := tests.GetRunningPodByVirtualMachineInstance(vmi, vmi.Namespace)
				err := virtClient.CoreV1().Pods(vmi.Namespace).Evict(&v1beta1.Eviction{ObjectMeta: metav1.ObjectMeta{Name: pod.Name}})
				Expect(errors.IsTooManyRequests(err)).To(BeTrue())
			})

			Context("with node tainted", func() {

				It("should migrate the VMI to another node", func() {
					vmi = runVMIAndExpectLaunch(vmi, 180)
					node := vmi.Status.NodeName
					tests.Taint(node, "kubevirt.io/drain", k8sv1.TaintEffectNoSchedule)
					Eventually(func() string {
						vmi, err := virtClient.VirtualMachineInstance(vmi.Namespace).Get(vmi.Name, &metav1.GetOptions{})
						Expect(err).ToNot(HaveOccurred())
						return vmi.Status.NodeName
					}, 60*time.Second, 1*time.Second).ShouldNot(Equal(node))
				})

			})

		})
		Context("with multiple VMIs with eviction policies set", func() {

			It("should not migrate more than two VMIs at the same time from a node", func() {
				var vmis []*v1.VirtualMachineInstance
				for i := 0; i < 4; i++ {
					vmi := vmiWithEvictionStrategy()
					vmi.Spec.NodeSelector = map[string]string{"tests.kubevirt.io": "target"}
					vmis = append(vmis, vmi)
				}

				By("selecting a node as the source")
				sourceNode := tests.GetAllSchedulableNodes(virtClient).Items[0]
				tests.AddLabelToNode(sourceNode.Name, "tests.kubevirt.io", "target")

				By("starting four VMIs on that node")
				for _, vmi := range vmis {
					_, err := virtClient.VirtualMachineInstance(tests.NamespaceTestDefault).Create(vmi)
					Expect(err).ToNot(HaveOccurred())
				}

				By("waiting until the VMIs are ready")
				for _, vmi := range vmis {
					tests.WaitForSuccessfulVMIStartWithTimeout(vmi, 180)
				}

				By("selecting a  node as the target")
				targetNode := tests.GetAllSchedulableNodes(virtClient).Items[1]
				tests.AddLabelToNode(targetNode.Name, "tests.kubevirt.io", "target")

				By("tainting the source node as non-schedulabele")
				tests.Taint(sourceNode.Name, "kubevirt.io/drain", k8sv1.TaintEffectNoSchedule)

				By("checking that all VMIs were migrated, and we never see more than two running migrations in parallel")
				Eventually(func() []string {
					var nodes []string
					for _, vmi := range vmis {
						vmi, err = virtClient.VirtualMachineInstance(tests.NamespaceTestDefault).Get(vmi.Name, &metav1.GetOptions{})
						nodes = append(nodes, vmi.Status.NodeName)
					}
					migrations, err := virtClient.VirtualMachineInstanceMigration(k8sv1.NamespaceAll).List(&metav1.ListOptions{})
					Expect(err).ToNot(HaveOccurred())
					runningMigrations := migrations2.FilterRunningMigrations(migrations.Items)
					Expect(len(runningMigrations)).To(BeNumerically("<=", 2))
					return nodes
				}, 4*time.Minute, 1*time.Second).Should(ConsistOf(
					targetNode.Name,
					targetNode.Name,
					targetNode.Name,
					targetNode.Name,
				))
			})
		})

	})
})

func vmiWithEvictionStrategy() *v1.VirtualMachineInstance {
	strategy := v1.EvictionStrategyLiveMigrate
	vmi := tests.NewRandomVMIWithEphemeralDiskAndUserdata(tests.ContainerDiskFor(tests.ContainerDiskCirros), "#!/bin/bash\necho 'hello'\n")
	vmi.Spec.EvictionStrategy = &strategy
	return vmi
}
