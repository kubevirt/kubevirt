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
	"crypto/tls"
	"encoding/json"
	"strings"
	"sync"

	"fmt"
	"time"

	expect "github.com/google/goexpect"
	. "github.com/onsi/ginkgo"
	"github.com/onsi/ginkgo/extensions/table"
	. "github.com/onsi/gomega"
	k8sv1 "k8s.io/api/core/v1"
	"k8s.io/api/policy/v1beta1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/rand"

	v1 "kubevirt.io/client-go/api/v1"
	"kubevirt.io/client-go/kubecli"
	cdiv1 "kubevirt.io/containerized-data-importer/pkg/apis/core/v1alpha1"
	"kubevirt.io/kubevirt/pkg/certificates/triple"
	"kubevirt.io/kubevirt/pkg/certificates/triple/cert"
	migrations "kubevirt.io/kubevirt/pkg/util/migrations"
	"kubevirt.io/kubevirt/tests"
)

const (
	migrationWaitTime = 240
	fedoraVMSize      = "256M"
)

var _ = Describe("[rfe_id:393][crit:high[vendor:cnv-qe@redhat.com][level:system] VM Live Migration", func() {
	tests.FlagParse()

	virtClient, err := kubecli.GetKubevirtClient()
	tests.PanicOnError(err)

	var originalKubeVirtConfig *k8sv1.ConfigMap

	tests.BeforeAll(func() {

		originalKubeVirtConfig, err = virtClient.CoreV1().ConfigMaps(tests.KubeVirtInstallNamespace).Get("kubevirt-config", metav1.GetOptions{})
		if err != nil && !errors.IsNotFound(err) {
			Expect(err).ToNot(HaveOccurred())
		}

		if errors.IsNotFound(err) {
			// create an empty kubevirt-config configmap if none exists.
			cfgMap := &k8sv1.ConfigMap{
				ObjectMeta: metav1.ObjectMeta{Name: "kubevirt-config"},
				Data: map[string]string{
					"feature-gates": "",
				},
			}

			originalKubeVirtConfig, err = virtClient.CoreV1().ConfigMaps(tests.KubeVirtInstallNamespace).Create(cfgMap)
			if err != nil {
				Expect(err).ToNot(HaveOccurred())
			}

		}
	})

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
		curKubeVirtConfig, err := virtClient.CoreV1().ConfigMaps(tests.KubeVirtInstallNamespace).Get("kubevirt-config", metav1.GetOptions{})
		if err != nil {
			Expect(err).ToNot(HaveOccurred())
		}

		// if revision changed, patch data and reload everything
		if curKubeVirtConfig.ResourceVersion != originalKubeVirtConfig.ResourceVersion {
			// Add  Patch
			newData, err := json.Marshal(originalKubeVirtConfig.Data)
			Expect(err).ToNot(HaveOccurred())
			data := fmt.Sprintf(`[{ "op": "replace", "path": "/data", "value": %s }]`, string(newData))

			newConfig, err := virtClient.CoreV1().ConfigMaps(tests.KubeVirtInstallNamespace).Patch("kubevirt-config", types.JSONPatchType, []byte(data))
			Expect(err).ToNot(HaveOccurred())

			// update the restored originalKubeVirtConfig
			originalKubeVirtConfig = newConfig
		}

	})

	runVMIAndExpectLaunchWithIgnoreWarningArg := func(vmi *v1.VirtualMachineInstance, timeout int, ignoreWarnings bool) *v1.VirtualMachineInstance {
		By("Starting a VirtualMachineInstance")
		var obj *v1.VirtualMachineInstance
		var err error
		Eventually(func() error {
			obj, err = virtClient.VirtualMachineInstance(tests.NamespaceTestDefault).Create(vmi)
			return err
		}, timeout, 1*time.Second).ShouldNot(HaveOccurred())
		By("Waiting until the VirtualMachineInstance starts")
		if ignoreWarnings {
			tests.WaitForSuccessfulVMIStartWithTimeoutIgnoreWarnings(obj, timeout)
		} else {
			tests.WaitForSuccessfulVMIStartWithTimeout(obj, timeout)
		}
		vmi, err = virtClient.VirtualMachineInstance(tests.NamespaceTestDefault).Get(vmi.Name, &metav1.GetOptions{})
		Expect(err).ToNot(HaveOccurred())
		return vmi
	}

	runVMIAndExpectLaunch := func(vmi *v1.VirtualMachineInstance, timeout int) *v1.VirtualMachineInstance {
		return runVMIAndExpectLaunchWithIgnoreWarningArg(vmi, timeout, false)
	}

	runVMIAndExpectLaunchIgnoreWarnings := func(vmi *v1.VirtualMachineInstance, timeout int) *v1.VirtualMachineInstance {
		return runVMIAndExpectLaunchWithIgnoreWarningArg(vmi, timeout, true)
	}

	confirmVMIPostMigration := func(vmi *v1.VirtualMachineInstance, migrationUID string) {
		By("Retrieving the VMI post migration")
		vmi, err = virtClient.VirtualMachineInstance(vmi.Namespace).Get(vmi.Name, &metav1.GetOptions{})
		Expect(err).ToNot(HaveOccurred())

		By("Verifying the VMI's migration state")
		Expect(vmi.Status.MigrationState).ToNot(BeNil())
		Expect(vmi.Status.MigrationState.StartTimestamp).ToNot(BeNil())
		Expect(vmi.Status.MigrationState.EndTimestamp).ToNot(BeNil())
		Expect(vmi.Status.MigrationState.TargetNode).To(Equal(vmi.Status.NodeName))
		Expect(vmi.Status.MigrationState.TargetNode).ToNot(Equal(vmi.Status.MigrationState.SourceNode))
		Expect(vmi.Status.MigrationState.Completed).To(BeTrue())
		Expect(vmi.Status.MigrationState.Failed).To(BeFalse())
		Expect(vmi.Status.MigrationState.TargetNodeAddress).ToNot(Equal(""))
		Expect(string(vmi.Status.MigrationState.MigrationUID)).To(Equal(migrationUID))

		By("Verifying the VMI's is in the running state")
		Expect(vmi.Status.Phase).To(Equal(v1.Running))
	}

	confirmVMIPostMigrationFailed := func(vmi *v1.VirtualMachineInstance, migrationUID string) {
		By("Retrieving the VMI post migration")
		vmi, err = virtClient.VirtualMachineInstance(vmi.Namespace).Get(vmi.Name, &metav1.GetOptions{})
		Expect(err).ToNot(HaveOccurred())

		By("Verifying the VMI's migration state")
		Expect(vmi.Status.MigrationState).ToNot(BeNil())
		Expect(vmi.Status.MigrationState.StartTimestamp).ToNot(BeNil())
		Expect(vmi.Status.MigrationState.EndTimestamp).ToNot(BeNil())
		Expect(vmi.Status.MigrationState.SourceNode).To(Equal(vmi.Status.NodeName))
		Expect(vmi.Status.MigrationState.TargetNode).ToNot(Equal(vmi.Status.MigrationState.SourceNode))
		Expect(vmi.Status.MigrationState.Completed).To(BeTrue())
		Expect(vmi.Status.MigrationState.Failed).To(BeTrue())
		Expect(vmi.Status.MigrationState.TargetNodeAddress).ToNot(Equal(""))
		Expect(string(vmi.Status.MigrationState.MigrationUID)).To(Equal(migrationUID))

		By("Verifying the VMI's is in the running state")
		Expect(vmi.Status.Phase).To(Equal(v1.Running))
	}
	confirmVMIPostMigrationAborted := func(vmi *v1.VirtualMachineInstance, migrationUID string, timeout int) *v1.VirtualMachineInstance {
		By("Waiting until the migration is completed")
		Eventually(func() bool {
			vmi, err = virtClient.VirtualMachineInstance(vmi.Namespace).Get(vmi.Name, &metav1.GetOptions{})
			Expect(err).ToNot(HaveOccurred())

			if vmi.Status.MigrationState != nil && vmi.Status.MigrationState.Completed &&
				vmi.Status.MigrationState.AbortStatus == v1.MigrationAbortSucceeded {
				return true
			}
			return false

		}, timeout, 1*time.Second).Should(BeTrue())

		By("Retrieving the VMI post migration")
		vmi, err = virtClient.VirtualMachineInstance(vmi.Namespace).Get(vmi.Name, &metav1.GetOptions{})
		Expect(err).ToNot(HaveOccurred())

		By("Verifying the VMI's migration state")
		Expect(vmi.Status.MigrationState).ToNot(BeNil())
		Expect(vmi.Status.MigrationState.StartTimestamp).ToNot(BeNil())
		Expect(vmi.Status.MigrationState.EndTimestamp).ToNot(BeNil())
		Expect(vmi.Status.MigrationState.SourceNode).To(Equal(vmi.Status.NodeName))
		Expect(vmi.Status.MigrationState.TargetNode).ToNot(Equal(vmi.Status.MigrationState.SourceNode))
		Expect(vmi.Status.MigrationState.TargetNodeAddress).ToNot(Equal(""))
		Expect(string(vmi.Status.MigrationState.MigrationUID)).To(Equal(migrationUID))
		Expect(vmi.Status.MigrationState.Failed).To(BeTrue())
		Expect(vmi.Status.MigrationState.AbortRequested).To(BeTrue())

		By("Verifying the VMI's is in the running state")
		Expect(vmi.Status.Phase).To(Equal(v1.Running))
		return vmi
	}
	runMigrationAndExpectCompletion := func(migration *v1.VirtualMachineInstanceMigration, timeout int) string {
		By("Starting a Migration")
		Eventually(func() error {
			migration, err = virtClient.VirtualMachineInstanceMigration(migration.Namespace).Create(migration)
			return err
		}, timeout, 1*time.Second).ShouldNot(HaveOccurred())
		By("Waiting until the Migration Completes")

		uid := ""
		Eventually(func() error {
			migration, err := virtClient.VirtualMachineInstanceMigration(migration.Namespace).Get(migration.Name, &metav1.GetOptions{})
			if err != nil {
				return err
			}

			Expect(migration.Status.Phase).ToNot(Equal(v1.MigrationFailed))

			uid = string(migration.UID)
			if migration.Status.Phase == v1.MigrationSucceeded {
				return nil
			}
			return fmt.Errorf("Migration is in the phase: %s", migration.Status.Phase)

		}, timeout, 1*time.Second).ShouldNot(HaveOccurred(), fmt.Sprintf("migration should succeed after %d s", timeout))
		return uid
	}
	runAndCancelMigration := func(migration *v1.VirtualMachineInstanceMigration, vmi *v1.VirtualMachineInstance, timeout int) *v1.VirtualMachineInstanceMigration {
		By("Starting a Migration")
		Eventually(func() error {
			migration, err = virtClient.VirtualMachineInstanceMigration(migration.Namespace).Create(migration)
			return err
		}, timeout, 1*time.Second).ShouldNot(HaveOccurred())

		By("Waiting until the Migration is Running")

		Eventually(func() bool {
			migration, err := virtClient.VirtualMachineInstanceMigration(migration.Namespace).Get(migration.Name, &metav1.GetOptions{})
			Expect(err).ToNot(HaveOccurred())

			Expect(migration.Status.Phase).ToNot(Equal(v1.MigrationFailed))
			if migration.Status.Phase == v1.MigrationRunning {
				vmi, err = virtClient.VirtualMachineInstance(vmi.Namespace).Get(vmi.Name, &metav1.GetOptions{})
				Expect(err).ToNot(HaveOccurred())
				if vmi.Status.MigrationState.Completed != true {
					return true
				}
			}
			return false

		}, timeout, 1*time.Second).Should(BeTrue())

		By("Cancelling a Migration")
		Expect(virtClient.VirtualMachineInstanceMigration(migration.Namespace).Delete(migration.Name, &metav1.DeleteOptions{})).To(Succeed(), "Migration should be deleted successfully")
		return migration
	}
	runAndImmediatelyCancelMigration := func(migration *v1.VirtualMachineInstanceMigration, vmi *v1.VirtualMachineInstance, timeout int) *v1.VirtualMachineInstanceMigration {
		By("Starting a Migration")
		Eventually(func() error {
			migration, err = virtClient.VirtualMachineInstanceMigration(migration.Namespace).Create(migration)
			return err
		}, timeout, 1*time.Second).ShouldNot(HaveOccurred())

		By("Waiting until the Migration is Running")

		Eventually(func() bool {
			migration, err := virtClient.VirtualMachineInstanceMigration(migration.Namespace).Get(migration.Name, &metav1.GetOptions{})
			Expect(err).ToNot(HaveOccurred())
			return string(migration.UID) != ""

		}, timeout, 1*time.Second).Should(BeTrue())

		By("Cancelling a Migration")
		Expect(virtClient.VirtualMachineInstanceMigration(migration.Namespace).Delete(migration.Name, &metav1.DeleteOptions{})).To(Succeed(), "Migration should be deleted successfully")
		return migration
	}

	runMigrationAndExpectFailure := func(migration *v1.VirtualMachineInstanceMigration, timeout int) string {
		By("Starting a Migration")
		Eventually(func() error {
			migration, err = virtClient.VirtualMachineInstanceMigration(migration.Namespace).Create(migration)
			return err
		}, timeout, 1*time.Second).ShouldNot(HaveOccurred())
		By("Waiting until the Migration Completes")

		uid := ""
		Eventually(func() bool {
			migration, err := virtClient.VirtualMachineInstanceMigration(migration.Namespace).Get(migration.Name, &metav1.GetOptions{})
			Expect(err).ToNot(HaveOccurred())

			Expect(migration.Status.Phase).NotTo(Equal(v1.MigrationSucceeded))

			uid = string(migration.UID)
			if migration.Status.Phase == v1.MigrationFailed {
				return true
			}
			return false

		}, timeout, 1*time.Second).Should(BeTrue())
		return uid
	}

	runStressTest := func(expecter expect.Expecter) {
		By("Run a stress test to dirty some pages and slow down the migration")
		_, err = expecter.ExpectBatch([]expect.Batcher{
			&expect.BSnd{S: "stress --vm 1 --vm-bytes 600M --vm-keep --timeout 1600s&\n"},
		}, 15*time.Second)
		Expect(err).ToNot(HaveOccurred(), "should run a stress test")
	}

	deleteDataVolume := func(dv *cdiv1.DataVolume) {
		if dv != nil {
			By("Deleting the DataVolume")
			ExpectWithOffset(1, virtClient.CdiClient().CdiV1alpha1().DataVolumes(dv.Namespace).Delete(dv.Name, &metav1.DeleteOptions{})).To(Succeed())
		}
	}

	Describe("Starting a VirtualMachineInstance ", func() {
		Context("with a bridge network interface", func() {
			It("[test_id:3226]should reject a migration of a vmi with a bridge interface", func() {
				vmi := tests.NewRandomVMIWithEphemeralDisk(tests.ContainerDiskFor(tests.ContainerDiskAlpine))
				vmi.Spec.Domain.Devices.Interfaces = []v1.Interface{
					{
						Name: "default",
						InterfaceBindingMethod: v1.InterfaceBindingMethod{
							Bridge: &v1.InterfaceBridge{},
						},
					},
				}
				vmi = runVMIAndExpectLaunch(vmi, 240)

				// Verify console on last iteration to verify the VirtualMachineInstance is still booting properly
				// after being restarted multiple times
				By("Checking that the VirtualMachineInstance console has expected output")
				expecter, err := tests.LoggedInAlpineExpecter(vmi)
				Expect(err).ToNot(HaveOccurred())
				expecter.Close()

				for _, c := range vmi.Status.Conditions {
					if c.Type == v1.VirtualMachineInstanceIsMigratable {
						Expect(c.Status).To(Equal(k8sv1.ConditionFalse))
					}
				}

				// execute a migration, wait for finalized state
				migration := tests.NewRandomMigration(vmi.Name, vmi.Namespace)

				By("Starting a Migration")
				migration, err = virtClient.VirtualMachineInstanceMigration(migration.Namespace).Create(migration)
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("InterfaceNotLiveMigratable"))

				// delete VMI
				By("Deleting the VMI")
				Expect(virtClient.VirtualMachineInstance(vmi.Namespace).Delete(vmi.Name, &metav1.DeleteOptions{})).To(Succeed())

				By("Waiting for VMI to disappear")
				tests.WaitForVirtualMachineToDisappearWithTimeout(vmi, 120)
			})
		})
		Context("with a Cirros disk", func() {
			It("[test_id:1783]should be successfully migrated multiple times with cloud-init disk", func() {

				vmi := tests.NewRandomVMIWithEphemeralDisk(tests.ContainerDiskFor(tests.ContainerDiskCirros))
				tests.AddUserData(vmi, "cloud-init", "#!/bin/bash\necho 'hello'\n")

				By("Starting the VirtualMachineInstance")
				vmi = runVMIAndExpectLaunch(vmi, 240)

				By("Checking that the VirtualMachineInstance console has expected output")
				expecter, err := tests.LoggedInCirrosExpecter(vmi)
				Expect(err).ToNot(HaveOccurred())
				expecter.Close()

				num := 2

				for i := 0; i < num; i++ {
					// execute a migration, wait for finalized state
					By(fmt.Sprintf("Starting the Migration for iteration %d", i))
					migration := tests.NewRandomMigration(vmi.Name, vmi.Namespace)
					migrationUID := runMigrationAndExpectCompletion(migration, migrationWaitTime)

					// check VMI, confirm migration state
					confirmVMIPostMigration(vmi, migrationUID)
				}
				// delete VMI
				By("Deleting the VMI")
				Expect(virtClient.VirtualMachineInstance(vmi.Namespace).Delete(vmi.Name, &metav1.DeleteOptions{})).To(Succeed())

				By("Waiting for VMI to disappear")
				tests.WaitForVirtualMachineToDisappearWithTimeout(vmi, 240)

			})
		})
		Context("with auto converge enabled", func() {
			var options metav1.GetOptions
			var cfgMap *k8sv1.ConfigMap
			var originalMigrationConfig string
			var kubevirtConfig = "kubevirt-config"
			BeforeEach(func() {
				tests.BeforeTestCleanup()

				// set autoconverge flag
				options = metav1.GetOptions{}
				cfgMap, err = virtClient.CoreV1().ConfigMaps(tests.KubeVirtInstallNamespace).Get(kubevirtConfig, options)
				Expect(err).ToNot(HaveOccurred())
				originalMigrationConfig = cfgMap.Data["migrations"]

				tests.UpdateClusterConfigValueAndWait("migrations", `{"allowAutoConverge": true}`)
			})

			AfterEach(func() {
				tests.UpdateClusterConfigValueAndWait("migrations", originalMigrationConfig)
			})

			It("[test_id:3237]should complete a migration", func() {
				vmi := tests.NewRandomFedoraVMIWitGuestAgent()
				vmi.Spec.Domain.Resources.Requests[k8sv1.ResourceMemory] = resource.MustParse(fedoraVMSize)

				By("Starting the VirtualMachineInstance")
				vmi = runVMIAndExpectLaunch(vmi, 240)

				By("Checking that the VirtualMachineInstance console has expected output")
				expecter, expecterErr := tests.LoggedInFedoraExpecter(vmi)
				Expect(expecterErr).ToNot(HaveOccurred())
				defer expecter.Close()

				// Need to wait for cloud init to finnish and start the agent inside the vmi.
				tests.WaitAgentConnected(virtClient, vmi)

				runStressTest(expecter)

				// execute a migration, wait for finalized state
				By("Starting the Migration")
				migration := tests.NewRandomMigration(vmi.Name, vmi.Namespace)
				migrationUID := runMigrationAndExpectCompletion(migration, migrationWaitTime)

				// check VMI, confirm migration state
				confirmVMIPostMigration(vmi, migrationUID)

				// delete VMI
				By("Deleting the VMI")
				Expect(virtClient.VirtualMachineInstance(vmi.Namespace).Delete(vmi.Name, &metav1.DeleteOptions{})).To(Succeed())

				By("Waiting for VMI to disappear")
				tests.WaitForVirtualMachineToDisappearWithTimeout(vmi, 240)
			})
		})
		Context("with a shared ISCSI Filesystem PVC", func() {
			var options metav1.GetOptions
			var cfgMap *k8sv1.ConfigMap
			var originalMigrationConfig string
			var kubevirtConfig = "kubevirt-config"
			BeforeEach(func() {
				tests.BeforeTestCleanup()
				if !tests.HasCDI() {
					Skip("Skip DataVolume tests when CDI is not present")
				}

				tests.SkipIfVersionAboveOrEqual("re-enable this once https://github.com/kubevirt/kubevirt/issues/2272 is fixed", "1.13.3")

				// set unsafe migration flag
				options = metav1.GetOptions{}
				cfgMap, err = virtClient.CoreV1().ConfigMaps(tests.KubeVirtInstallNamespace).Get(kubevirtConfig, options)
				Expect(err).ToNot(HaveOccurred())
				originalMigrationConfig = cfgMap.Data["migrations"]
				tests.UpdateClusterConfigValueAndWait("migrations", `{"unsafeMigrationOverride": true}`)
			})

			AfterEach(func() {
				tests.UpdateClusterConfigValueAndWait("migrations", originalMigrationConfig)
			})

			It("[test_id:3238]should migrate a vmi with UNSAFE_MIGRATION flag set", func() {
				// Normally, live migration with a shared volume that contains
				// a non-clustered filesystem will be prevented for disk safety reasons.
				// This test sets a UNSAFE_MIGRATION flag and a migration with an ext4 filesystem
				// should succeed.

				pvName := "test-iscsi-dv" + rand.String(48)
				// Start a ISCSI POD and service
				By("Starting an iSCSI POD")
				iscsiIP := tests.CreateISCSITargetPOD(tests.ContainerDiskEmpty)
				_, err = virtClient.CoreV1().PersistentVolumes().Create(tests.NewISCSIPV(pvName, "2Gi", iscsiIP, k8sv1.ReadWriteMany, k8sv1.PersistentVolumeFilesystem))
				Expect(err).ToNot(HaveOccurred())
				dataVolume := tests.NewRandomDataVolumeWithHttpImport(tests.AlpineHttpUrl, tests.NamespaceTestDefault, k8sv1.ReadWriteMany)
				volMode := k8sv1.PersistentVolumeFilesystem
				dataVolume.Spec.PVC.VolumeMode = &volMode
				vmi := tests.NewRandomVMIWithDataVolume(dataVolume.Name)

				_, err := virtClient.CdiClient().CdiV1alpha1().DataVolumes(dataVolume.Namespace).Create(dataVolume)
				Expect(err).ToNot(HaveOccurred())

				tests.WaitForSuccessfulDataVolumeImportOfVMI(vmi, 340)

				vmi = runVMIAndExpectLaunch(vmi, 240)

				// Verify console on last iteration to verify the VirtualMachineInstance is still booting properly
				// after being restarted multiple times
				By("Checking that the VirtualMachineInstance console has expected output")
				expecter, err := tests.LoggedInAlpineExpecter(vmi)
				Expect(err).ToNot(HaveOccurred())
				expecter.Close()

				// execute a migration, wait for finalized state
				By("Starting the Migration")
				migration := tests.NewRandomMigration(vmi.Name, vmi.Namespace)
				migrationUID := runMigrationAndExpectCompletion(migration, migrationWaitTime)

				// check VMI, confirm migration state
				confirmVMIPostMigration(vmi, migrationUID)

				// delete VMI
				By("Deleting the VMI")
				Expect(virtClient.VirtualMachineInstance(vmi.Namespace).Delete(vmi.Name, &metav1.DeleteOptions{})).To(Succeed())

				By("Waiting for VMI to disappear")
				tests.WaitForVirtualMachineToDisappearWithTimeout(vmi, 120)

				Expect(virtClient.CdiClient().CdiV1alpha1().DataVolumes(dataVolume.Namespace).Delete(dataVolume.Name, &metav1.DeleteOptions{})).To(Succeed())
			})
		})
		Context("with an Alpine DataVolume", func() {
			BeforeEach(func() {
				tests.BeforeTestCleanup()
				if !tests.HasCDI() {
					Skip("Skip DataVolume tests when CDI is not present")
				}
			})
			It("[test_id:3239]should reject a migration of a vmi with a non-shared data volume", func() {
				dataVolume := tests.NewRandomDataVolumeWithHttpImport(tests.AlpineHttpUrl, tests.NamespaceTestDefault, k8sv1.ReadWriteOnce)
				vmi := tests.NewRandomVMIWithDataVolume(dataVolume.Name)

				_, err := virtClient.CdiClient().CdiV1alpha1().DataVolumes(dataVolume.Namespace).Create(dataVolume)
				Expect(err).ToNot(HaveOccurred())

				tests.WaitForSuccessfulDataVolumeImportOfVMI(vmi, 240)

				vmi = runVMIAndExpectLaunch(vmi, 240)

				// Verify console on last iteration to verify the VirtualMachineInstance is still booting properly
				// after being restarted multiple times
				By("Checking that the VirtualMachineInstance console has expected output")
				expecter, err := tests.LoggedInAlpineExpecter(vmi)
				Expect(err).ToNot(HaveOccurred())
				expecter.Close()

				for _, c := range vmi.Status.Conditions {
					if c.Type == v1.VirtualMachineInstanceIsMigratable {
						Expect(c.Status).To(Equal(k8sv1.ConditionFalse))
					}
				}

				// execute a migration, wait for finalized state
				migration := tests.NewRandomMigration(vmi.Name, vmi.Namespace)

				By("Starting a Migration")
				migration, err = virtClient.VirtualMachineInstanceMigration(migration.Namespace).Create(migration)
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("DisksNotLiveMigratable"))

				// delete VMI
				By("Deleting the VMI")
				Expect(virtClient.VirtualMachineInstance(vmi.Namespace).Delete(vmi.Name, &metav1.DeleteOptions{})).To(Succeed())

				By("Waiting for VMI to disappear")
				tests.WaitForVirtualMachineToDisappearWithTimeout(vmi, 120)

				Expect(virtClient.CdiClient().CdiV1alpha1().DataVolumes(dataVolume.Namespace).Delete(dataVolume.Name, &metav1.DeleteOptions{})).To(Succeed())
			})
			It("[test_id:1479] should migrate a vmi with a shared OCS disk", func() {
				vmi, dv := tests.NewRandomVirtualMachineInstanceWithOCSDisk(tests.AlpineHttpUrl, tests.NamespaceTestDefault, k8sv1.ReadWriteMany, k8sv1.PersistentVolumeBlock)
				defer deleteDataVolume(dv)

				By("Starting the VirtualMachineInstance")
				vmi = runVMIAndExpectLaunch(vmi, 300)

				By("Checking that the VirtualMachineInstance console has expected output")
				expecter, err := tests.LoggedInAlpineExpecter(vmi)
				Expect(err).ToNot(HaveOccurred())
				expecter.Close()

				By("Starting a Migration")
				migration := tests.NewRandomMigration(vmi.Name, vmi.Namespace)
				migrationUID := runMigrationAndExpectCompletion(migration, migrationWaitTime)

				// check VMI, confirm migration state
				confirmVMIPostMigration(vmi, migrationUID)

				// delete VMI
				By("Deleting the VMI")
				Expect(virtClient.VirtualMachineInstance(vmi.Namespace).Delete(vmi.Name, &metav1.DeleteOptions{})).To(Succeed())

				By("Waiting for VMI to disappear")
				tests.WaitForVirtualMachineToDisappearWithTimeout(vmi, 120)
			})
		})
		Context("with an Alpine shared ISCSI PVC", func() {
			var pvName string
			BeforeEach(func() {
				tests.SkipIfVersionAboveOrEqual("re-enable this once https://github.com/kubevirt/kubevirt/issues/2272 is fixed", "1.13.3")
				pvName = "test-iscsi-lun" + rand.String(48)
				// Start a ISCSI POD and service
				By("Starting an iSCSI POD")
				iscsiIP := tests.CreateISCSITargetPOD(tests.ContainerDiskAlpine)
				// create a new PV and PVC (PVs can't be reused)
				By("create a new iSCSI PV and PVC")
				tests.CreateISCSIPvAndPvc(pvName, "1Gi", iscsiIP, k8sv1.ReadWriteMany, k8sv1.PersistentVolumeBlock)
			})

			AfterEach(func() {
				// create a new PV and PVC (PVs can't be reused)
				tests.DeletePvAndPvc(pvName)
			})
			It("[test_id:1854]should migrate a VMI with shared and non-shared disks", func() {
				// Start the VirtualMachineInstance with PVC and Ephemeral Disks
				vmi := tests.NewRandomVMIWithPVC(pvName)
				image := tests.ContainerDiskFor(tests.ContainerDiskAlpine)
				tests.AddEphemeralDisk(vmi, "myephemeral", "virtio", image)

				By("Starting the VirtualMachineInstance")
				vmi = runVMIAndExpectLaunch(vmi, 180)

				By("Checking that the VirtualMachineInstance console has expected output")
				expecter, err := tests.LoggedInAlpineExpecter(vmi)
				Expect(err).ToNot(HaveOccurred())
				expecter.Close()

				By("Starting a Migration")
				migration := tests.NewRandomMigration(vmi.Name, vmi.Namespace)
				migrationUID := runMigrationAndExpectCompletion(migration, migrationWaitTime)

				// check VMI, confirm migration state
				confirmVMIPostMigration(vmi, migrationUID)

				// delete VMI
				By("Deleting the VMI")
				Expect(virtClient.VirtualMachineInstance(vmi.Namespace).Delete(vmi.Name, &metav1.DeleteOptions{})).To(Succeed())

				By("Waiting for VMI to disappear")
				tests.WaitForVirtualMachineToDisappearWithTimeout(vmi, 120)
			})
			It("[test_id:1377]should be successfully migrated multiple times", func() {
				// Start the VirtualMachineInstance with the PVC attached
				vmi := tests.NewRandomVMIWithPVC(pvName)
				vmi = runVMIAndExpectLaunch(vmi, 180)

				By("Checking that the VirtualMachineInstance console has expected output")
				expecter, err := tests.LoggedInAlpineExpecter(vmi)
				Expect(err).ToNot(HaveOccurred())
				expecter.Close()

				// execute a migration, wait for finalized state
				migration := tests.NewRandomMigration(vmi.Name, vmi.Namespace)
				migrationUID := runMigrationAndExpectCompletion(migration, 180)

				// check VMI, confirm migration state
				confirmVMIPostMigration(vmi, migrationUID)

				// delete VMI
				By("Deleting the VMI")
				Expect(virtClient.VirtualMachineInstance(vmi.Namespace).Delete(vmi.Name, &metav1.DeleteOptions{})).To(Succeed())

				By("Waiting for VMI to disappear")
				tests.WaitForVirtualMachineToDisappearWithTimeout(vmi, 120)

			})
		})
		Context("with an Cirros shared ISCSI PVC", func() {
			var pvName string
			BeforeEach(func() {
				tests.SkipIfVersionAboveOrEqual("re-enable this once https://github.com/kubevirt/kubevirt/issues/2272 is fixed", "1.13.3")
				pvName = "test-iscsi-lun" + rand.String(48)
				// Start a ISCSI POD and service
				By("Starting an iSCSI POD")
				iscsiIP := tests.CreateISCSITargetPOD(tests.ContainerDiskCirros)
				// create a new PV and PVC (PVs can't be reused)
				By("create a new iSCSI PV and PVC")
				tests.CreateISCSIPvAndPvc(pvName, "1Gi", iscsiIP, k8sv1.ReadWriteMany, k8sv1.PersistentVolumeBlock)
			})

			AfterEach(func() {
				// create a new PV and PVC (PVs can't be reused)
				tests.DeletePvAndPvc(pvName)
			})
			It("[test_id:3240]should be successfully with a cloud init", func() {
				// Start the VirtualMachineInstance with the PVC attached
				vmi := tests.NewRandomVMIWithPVC(pvName)
				tests.AddUserData(vmi, "cloud-init", "#!/bin/bash\necho 'hello'\n")
				vmi.Spec.Hostname = fmt.Sprintf("%s", tests.ContainerDiskCirros)
				vmi = runVMIAndExpectLaunch(vmi, 180)

				By("Checking that the VirtualMachineInstance console has expected output")
				expecter, err := tests.LoggedInCirrosExpecter(vmi)
				Expect(err).ToNot(HaveOccurred())
				expecter.Close()

				By("Checking that MigrationMethod is set to BlockMigration")
				Expect(vmi.Status.MigrationMethod).To(Equal(v1.BlockMigration))

				// execute a migration, wait for finalized state
				By("Starting the Migration for iteration")
				migration := tests.NewRandomMigration(vmi.Name, vmi.Namespace)
				migrationUID := runMigrationAndExpectCompletion(migration, migrationWaitTime)

				// check VMI, confirm migration state
				confirmVMIPostMigration(vmi, migrationUID)

				// delete VMI
				By("Deleting the VMI")
				Expect(virtClient.VirtualMachineInstance(vmi.Namespace).Delete(vmi.Name, &metav1.DeleteOptions{})).To(Succeed())

				By("Waiting for VMI to disappear")
				tests.WaitForVirtualMachineToDisappearWithTimeout(vmi, 120)
			})
		})
		Context("with an Fedora shared NFS PVC, cloud init and service account", func() {
			var pvName string
			var vmi *v1.VirtualMachineInstance
			BeforeEach(func() {
				pvName = "test-nfs" + rand.String(48)
				// Prepare a NFS backed PV
				By("Starting an NFS POD")
				os := string(tests.ContainerDiskFedora)
				nfsIP := tests.CreateNFSTargetPOD(os)
				// create a new PV and PVC (PVs can't be reused)
				By("create a new NFS PV and PVC")
				tests.CreateNFSPvAndPvc(pvName, "5Gi", nfsIP, os)
			})

			AfterEach(func() {
				// delete VMI
				By("Deleting the VMI")
				Expect(virtClient.VirtualMachineInstance(vmi.Namespace).Delete(vmi.Name, &metav1.DeleteOptions{})).To(Succeed())

				By("Waiting for VMI to disappear")
				tests.WaitForVirtualMachineToDisappearWithTimeout(vmi, 120)
				// create a new PV and PVC (PVs can't be reused)
				tests.DeletePvAndPvc(pvName)
			})
			It("[test_id:1785]  should be migrated successfully", func() {
				// Start the VirtualMachineInstance with the PVC attached
				By("Creating the  VMI")
				vmi = tests.NewRandomVMIWithPVC(pvName)
				vmi.Spec.Domain.Resources.Requests[k8sv1.ResourceMemory] = resource.MustParse(fedoraVMSize)
				vmi.Spec.Domain.Devices.Rng = &v1.Rng{}

				// add userdata for guest agent and service account mount
				secretDiskSerial := "D23YZ9W6WA5DJ487"
				mountSvcAccCommands := fmt.Sprintf(`
					mkdir /mnt/servacc
					mount /dev/$(lsblk --nodeps -no name,serial | grep %s | cut -f1 -d' ') /mnt/servacc
				`, secretDiskSerial)
				userData := fmt.Sprintf("%s\n%s", tests.GetGuestAgentUserData(), mountSvcAccCommands)
				tests.AddUserData(vmi, "cloud-init", userData)

				tests.AddServiceAccountDisk(vmi, "default")
				disks := vmi.Spec.Domain.Devices.Disks
				disks[len(disks)-1].Serial = secretDiskSerial

				vmi = runVMIAndExpectLaunchIgnoreWarnings(vmi, 180)

				// Wait for cloud init to finish and start the agent inside the vmi.
				tests.WaitAgentConnected(virtClient, vmi)

				By("Checking that the VirtualMachineInstance console has expected output")
				expecter, err := tests.LoggedInFedoraExpecter(vmi)
				Expect(err).ToNot(HaveOccurred())
				expecter.Close()

				// execute a migration, wait for finalized state
				By("Starting the Migration for iteration")
				migration := tests.NewRandomMigration(vmi.Name, vmi.Namespace)
				migrationUID := runMigrationAndExpectCompletion(migration, migrationWaitTime)

				// check VMI, confirm migration state
				confirmVMIPostMigration(vmi, migrationUID)

				// Is agent connected after migration
				tests.WaitAgentConnected(virtClient, vmi)

				By("Checking that the migrated VirtualMachineInstance console has expected output")
				expecter, err = tests.ReLoggedInFedoraExpecter(vmi, 60)
				defer expecter.Close()
				Expect(err).ToNot(HaveOccurred())

				By("Checking that the service account is mounted")
				_, err = expecter.ExpectBatch([]expect.Batcher{
					&expect.BSnd{S: "cat /mnt/servacc/namespace\n"},
					&expect.BExp{R: tests.NamespaceTestDefault},
				}, 30*time.Second)
				Expect(err).ToNot(HaveOccurred())

			})
		})

		Context("migration security", func() {
			var options metav1.GetOptions
			var cfgMap *k8sv1.ConfigMap
			var originalMigrationConfig string
			var kubevirtConfig = "kubevirt-config"

			BeforeEach(func() {
				// update migration timeouts
				options = metav1.GetOptions{}
				cfgMap, err = virtClient.CoreV1().ConfigMaps(tests.KubeVirtInstallNamespace).Get(kubevirtConfig, options)
				Expect(err).ToNot(HaveOccurred())
				originalMigrationConfig = cfgMap.Data["migrations"]
				tests.UpdateClusterConfigValueAndWait("migrations", `{"bandwidthPerMigration" : "1Mi"}`)
			})
			AfterEach(func() {
				tests.UpdateClusterConfigValueAndWait("migrations", originalMigrationConfig)
			})
			It("[test_id:2303][posneg:negative] should secure migrations with TLS", func() {
				vmi := tests.NewRandomFedoraVMIWitGuestAgent()
				vmi.Spec.Domain.Resources.Requests[k8sv1.ResourceMemory] = resource.MustParse(fedoraVMSize)

				By("Starting the VirtualMachineInstance")
				vmi = runVMIAndExpectLaunch(vmi, 240)

				// Need to wait for cloud init to finish and start the agent inside the vmi.
				tests.WaitAgentConnected(virtClient, vmi)

				// Run
				expecter, expecterErr := tests.LoggedInFedoraExpecter(vmi)
				Expect(expecterErr).ToNot(HaveOccurred())
				defer expecter.Close()

				runStressTest(expecter)

				// execute a migration, wait for finalized state
				By("Starting the Migration")
				migration := tests.NewRandomMigration(vmi.Name, vmi.Namespace)
				migration, err = virtClient.VirtualMachineInstanceMigration(migration.Namespace).Create(migration)

				By("Waiting for the proxy connection details to appear")
				Eventually(func() bool {
					migratingVMI, err := virtClient.VirtualMachineInstance(vmi.Namespace).Get(vmi.Name, &metav1.GetOptions{})
					Expect(err).ToNot(HaveOccurred())
					if migratingVMI.Status.MigrationState == nil {
						return false
					}

					if migratingVMI.Status.MigrationState.TargetNodeAddress == "" || len(migratingVMI.Status.MigrationState.TargetDirectMigrationNodePorts) == 0 {
						return false
					}
					vmi = migratingVMI
					return true
				}, 60*time.Second, 1*time.Second).Should(BeTrue())

				By("checking if we fail to connect with our own cert")
				// Generate new certs if secret doesn't already exist
				caKeyPair, _ := triple.NewCA("kubevirt.io")

				clientKeyPair, _ := triple.NewClientKeyPair(caKeyPair,
					"kubevirt.io:system:node:virt-handler",
					nil,
				)

				certPEM := cert.EncodeCertPEM(clientKeyPair.Cert)
				keyPEM := cert.EncodePrivateKeyPEM(clientKeyPair.Key)
				cert, err := tls.X509KeyPair(certPEM, keyPEM)
				Expect(err).ToNot(HaveOccurred())
				tlsConfig := &tls.Config{
					InsecureSkipVerify: true,
					GetClientCertificate: func(info *tls.CertificateRequestInfo) (certificate *tls.Certificate, e error) {
						return &cert, nil
					},
				}
				handler, err := kubecli.NewVirtHandlerClient(virtClient).Namespace(tests.KubeVirtInstallNamespace).ForNode(vmi.Status.MigrationState.TargetNode).Pod()
				Expect(err).ToNot(HaveOccurred())

				var wg sync.WaitGroup
				wg.Add(len(vmi.Status.MigrationState.TargetDirectMigrationNodePorts))

				i := 0
				errors := make(chan error, len(vmi.Status.MigrationState.TargetDirectMigrationNodePorts))
				for port, _ := range vmi.Status.MigrationState.TargetDirectMigrationNodePorts {
					go func(i int, port int) {
						defer GinkgoRecover()
						defer wg.Done()
						stopChan := make(chan struct{})
						defer close(stopChan)
						Expect(tests.ForwardPorts(handler, []string{fmt.Sprintf("4321%d:%d", i, port)}, stopChan, 10*time.Second)).To(Succeed())
						_, err = tls.Dial("tcp", fmt.Sprintf("localhost:4321%d", i), tlsConfig)
						Expect(err).To(HaveOccurred())
						errors <- err
					}(i, port)
					i++
				}
				wg.Wait()
				close(errors)

				By("checking that we were never able to connect")
				tlsErrorFound := false
				for err := range errors {
					if strings.Contains(err.Error(), "remote error: tls: bad certificate") {
						tlsErrorFound = true
					}
					Expect(err.Error()).To(Or(ContainSubstring("remote error: tls: bad certificate"), ContainSubstring("EOF")))
				}

				Expect(tlsErrorFound).To(BeTrue())
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
				cfgMap, err = virtClient.CoreV1().ConfigMaps(tests.KubeVirtInstallNamespace).Get(kubevirtConfig, options)
				Expect(err).ToNot(HaveOccurred())
				originalMigrationConfig = cfgMap.Data["migrations"]
				tests.UpdateClusterConfigValueAndWait("migrations", `{"progressTimeout" : 5, "completionTimeoutPerGiB": 5}`)
			})
			AfterEach(func() {
				tests.UpdateClusterConfigValueAndWait("migrations", originalMigrationConfig)
			})
			It("[test_id:2227]should abort a vmi migration without progress", func() {
				vmi := tests.NewRandomFedoraVMIWitGuestAgent()
				vmi.Spec.Domain.Resources.Requests[k8sv1.ResourceMemory] = resource.MustParse("1Gi")

				By("Starting the VirtualMachineInstance")
				vmi = runVMIAndExpectLaunch(vmi, 240)

				By("Checking that the VirtualMachineInstance console has expected output")
				expecter, expecterErr := tests.LoggedInFedoraExpecter(vmi)
				Expect(expecterErr).ToNot(HaveOccurred())
				defer expecter.Close()

				// Need to wait for cloud init to finish and start the agent inside the vmi.
				tests.WaitAgentConnected(virtClient, vmi)

				runStressTest(expecter)

				// execute a migration, wait for finalized state
				By("Starting the Migration")
				migration := tests.NewRandomMigration(vmi.Name, vmi.Namespace)
				migrationUID := runMigrationAndExpectFailure(migration, 180)

				// check VMI, confirm migration state
				confirmVMIPostMigrationFailed(vmi, migrationUID)

				// delete VMI
				By("Deleting the VMI")
				Expect(virtClient.VirtualMachineInstance(vmi.Namespace).Delete(vmi.Name, &metav1.DeleteOptions{})).To(Succeed())

				By("Waiting for VMI to disappear")
				tests.WaitForVirtualMachineToDisappearWithTimeout(vmi, 240)
			})
		})
		Context("with an Cirros non-shared ISCSI PVC", func() {
			var pvName string
			BeforeEach(func() {
				tests.SkipIfVersionAboveOrEqual("re-enable this once https://github.com/kubevirt/kubevirt/issues/2272 is fixed", "1.13.3")
				pvName = "test-iscsi-lun" + rand.String(48)
				// Start a ISCSI POD and service
				By("Starting an iSCSI POD")
				iscsiIP := tests.CreateISCSITargetPOD(tests.ContainerDiskCirros)
				// create a new PV and PVC (PVs can't be reused)
				By("create a new iSCSI PV and PVC")
				tests.CreateISCSIPvAndPvc(pvName, "1Gi", iscsiIP, k8sv1.ReadWriteOnce, k8sv1.PersistentVolumeBlock)
			})

			AfterEach(func() {
				// create a new PV and PVC (PVs can't be reused)
				tests.DeletePvAndPvc(pvName)
			})
			It("[test_id:1862][posneg:negative]should reject migrations for a non-migratable vmi", func() {
				// Start the VirtualMachineInstance with the PVC attached
				vmi := tests.NewRandomVMIWithPVC(pvName)
				tests.AddUserData(vmi, "cloud-init", "#!/bin/bash\necho 'hello'\n")
				vmi.Spec.Hostname = fmt.Sprintf("%s", tests.ContainerDiskCirros)
				vmi = runVMIAndExpectLaunch(vmi, 180)

				By("Checking that the VirtualMachineInstance console has expected output")
				expecter, err := tests.LoggedInCirrosExpecter(vmi)
				Expect(err).ToNot(HaveOccurred())
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
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("DisksNotLiveMigratable"))

				// delete VMI
				By("Deleting the VMI")
				Expect(virtClient.VirtualMachineInstance(vmi.Namespace).Delete(vmi.Name, &metav1.DeleteOptions{})).To(Succeed())

				By("Waiting for VMI to disappear")
				tests.WaitForVirtualMachineToDisappearWithTimeout(vmi, 120)
			})
		})
		Context("live migration cancelation", func() {
			type vmiBuilder func() (*v1.VirtualMachineInstance, *cdiv1.DataVolume)

			newVirtualMachineInstanceWithFedoraContainerDisk := func() (*v1.VirtualMachineInstance, *cdiv1.DataVolume) {
				return tests.NewRandomFedoraVMIWitGuestAgent(), nil
			}

			newVirtualMachineInstanceWithFedoraOCSDisk := func() (*v1.VirtualMachineInstance, *cdiv1.DataVolume) {
				// It could have been cleaner to import tests.ContainerDiskFedora from cdi-http-server but that does
				// not work so as a temporary workaround the following imports the image from an ISCSI target pod
				if !tests.HasCDI() {
					Skip("Skip DataVolume tests when CDI is not present")
				}
				sc, exists := tests.GetCephStorageClass()
				if !exists {
					Skip("Skip OCS tests when Ceph is not present")
				}

				By("Starting an iSCSI POD")
				iscsiIP := tests.CreateISCSITargetPOD(tests.ContainerDiskFedora)
				volMode := k8sv1.PersistentVolumeBlock
				// create a new PV and PVC (PVs can't be reused)
				pvName := "test-iscsi-lun" + rand.String(48)
				tests.CreateISCSIPvAndPvc(pvName, "5Gi", iscsiIP, k8sv1.ReadWriteMany, volMode)
				Expect(err).ToNot(HaveOccurred())
				defer tests.DeletePvAndPvc(pvName)

				dv := tests.NewRandomDataVolumeWithPVCSourceWithStorageClass(tests.NamespaceTestDefault, pvName, tests.NamespaceTestDefault, sc, "5Gi", k8sv1.ReadWriteMany)
				dv.Spec.PVC.VolumeMode = &volMode
				_, err := virtClient.CdiClient().CdiV1alpha1().DataVolumes(dv.Namespace).Create(dv)
				Expect(err).ToNot(HaveOccurred())
				tests.WaitForSuccessfulDataVolumeImport(dv, 600)
				vmi := tests.NewRandomVMIWithDataVolume(dv.Name)
				tests.AddUserData(vmi, "disk1", tests.GetGuestAgentUserData())
				return vmi, dv
			}

			table.DescribeTable("should be able to cancel a migration", func(createVMI vmiBuilder) {
				vmi, dv := createVMI()
				vmi.Spec.Domain.Resources.Requests[k8sv1.ResourceMemory] = resource.MustParse(fedoraVMSize)
				defer deleteDataVolume(dv)

				By("Starting the VirtualMachineInstance")
				vmi = runVMIAndExpectLaunch(vmi, 240)

				By("Checking that the VirtualMachineInstance console has expected output")
				expecter, expecterErr := tests.LoggedInFedoraExpecter(vmi)
				Expect(expecterErr).ToNot(HaveOccurred())
				defer expecter.Close()

				// Need to wait for cloud init to finish and start the agent inside the vmi.
				tests.WaitAgentConnected(virtClient, vmi)

				runStressTest(expecter)

				// execute a migration, wait for finalized state
				By("Starting the Migration")
				migration := tests.NewRandomMigration(vmi.Name, vmi.Namespace)

				migration = runAndCancelMigration(migration, vmi, 180)
				migrationUID := string(migration.UID)

				// check VMI, confirm migration state
				confirmVMIPostMigrationAborted(vmi, migrationUID, 180)

				By("Waiting for the migration object to disappear")
				tests.WaitForMigrationToDisappearWithTimeout(migration, 240)

				// delete VMI
				By("Deleting the VMI")
				Expect(virtClient.VirtualMachineInstance(vmi.Namespace).Delete(vmi.Name, &metav1.DeleteOptions{})).To(Succeed())

				By("Waiting for VMI to disappear")
				tests.WaitForVirtualMachineToDisappearWithTimeout(vmi, 240)
			},
				table.Entry("[test_id:2226]with ContainerDisk", newVirtualMachineInstanceWithFedoraContainerDisk),
				table.Entry("[test_id:2731]with OCS Disk", newVirtualMachineInstanceWithFedoraOCSDisk),
			)
			It("[test_id:3241]should be able to cancel a migration right after posting it", func() {
				vmi := tests.NewRandomFedoraVMIWitGuestAgent()
				vmi.Spec.Domain.Resources.Requests[k8sv1.ResourceMemory] = resource.MustParse(fedoraVMSize)

				By("Starting the VirtualMachineInstance")
				vmi = runVMIAndExpectLaunch(vmi, 240)

				By("Checking that the VirtualMachineInstance console has expected output")
				expecter, expecterErr := tests.LoggedInFedoraExpecter(vmi)
				Expect(expecterErr).ToNot(HaveOccurred())
				defer expecter.Close()

				// execute a migration, wait for finalized state
				By("Starting the Migration")
				migration := tests.NewRandomMigration(vmi.Name, vmi.Namespace)

				migration = runAndImmediatelyCancelMigration(migration, vmi, 180)

				// check VMI, confirm migration state
				confirmVMIPostMigrationAborted(vmi, string(migration.UID), 180)

				By("Waiting for the migration object to disappear")
				tests.WaitForMigrationToDisappearWithTimeout(migration, 240)

				// delete VMI
				By("Deleting the VMI")
				Expect(virtClient.VirtualMachineInstance(vmi.Namespace).Delete(vmi.Name, &metav1.DeleteOptions{})).To(Succeed())

				By("Waiting for VMI to disappear")
				tests.WaitForVirtualMachineToDisappearWithTimeout(vmi, 240)

			})
		})
	})

	Context("with sata disks", func() {

		It("[test_id:1853]VM with containerDisk + CloudInit + ServiceAccount + ConfigMap + Secret", func() {
			configMapName := "configmap-" + rand.String(5)
			secretName := "secret-" + rand.String(5)

			config_data := map[string]string{
				"config1": "value1",
				"config2": "value2",
			}

			secret_data := map[string]string{
				"user":     "admin",
				"password": "redhat",
			}

			tests.CreateConfigMap(configMapName, config_data)
			tests.CreateSecret(secretName, secret_data)

			vmi := tests.NewRandomVMIWithEphemeralDisk(tests.ContainerDiskFor(tests.ContainerDiskFedora))
			vmi.Spec.Domain.Resources.Requests[k8sv1.ResourceMemory] = resource.MustParse(fedoraVMSize)
			tests.AddUserData(vmi, "cloud-init", "#cloud-config\npassword: fedora\nchpasswd: { expire: False }\n")
			tests.AddConfigMapDisk(vmi, configMapName)
			tests.AddSecretDisk(vmi, secretName)
			tests.AddServiceAccountDisk(vmi, "default")

			vmi = runVMIAndExpectLaunch(vmi, 180)

			// execute a migration, wait for finalized state
			By("Starting the Migration")
			migration := tests.NewRandomMigration(vmi.Name, vmi.Namespace)
			migrationUID := runMigrationAndExpectCompletion(migration, migrationWaitTime)

			// check VMI, confirm migration state
			confirmVMIPostMigration(vmi, migrationUID)

			// delete VMI
			By("Deleting the VMI")
			Expect(virtClient.VirtualMachineInstance(vmi.Namespace).Delete(vmi.Name, &metav1.DeleteOptions{})).To(Succeed())

			By("Waiting for VMI to disappear")
			tests.WaitForVirtualMachineToDisappearWithTimeout(vmi, 120)
		})
	})

	Context("with a live-migrate eviction strategy set", func() {
		Context("[ref_id:2293] with a VMI running with an eviction strategy set", func() {

			var vmi *v1.VirtualMachineInstance

			BeforeEach(func() {
				vmi = cirrosVMIWithEvictionStrategy()
			})

			It("[test_id:3242]should block the eviction api", func() {
				vmi = runVMIAndExpectLaunch(vmi, 180)
				pod := tests.GetRunningPodByVirtualMachineInstance(vmi, vmi.Namespace)
				err := virtClient.CoreV1().Pods(vmi.Namespace).Evict(&v1beta1.Eviction{ObjectMeta: metav1.ObjectMeta{Name: pod.Name}})
				Expect(errors.IsTooManyRequests(err)).To(BeTrue())
			})

			It("[test_id:3243]should recreate the PDB if VMIs with similar names are recreated", func() {
				for x := 0; x < 3; x++ {
					By("creating the VMI")
					_, err := virtClient.VirtualMachineInstance(tests.NamespaceTestDefault).Create(vmi)
					Expect(err).ToNot(HaveOccurred())

					By("checking that the PDB appeared")
					Eventually(func() []v1beta1.PodDisruptionBudget {
						pdbs, err := virtClient.PolicyV1beta1().PodDisruptionBudgets(tests.NamespaceTestDefault).List(metav1.ListOptions{})
						Expect(err).ToNot(HaveOccurred())
						return pdbs.Items
					}, 3*time.Second, 500*time.Millisecond).Should(HaveLen(1))
					By("deleting the VMI")
					Expect(virtClient.VirtualMachineInstance(tests.NamespaceTestDefault).Delete(vmi.Name, &metav1.DeleteOptions{})).To(Succeed())
					By("checking that the PDB disappeared")
					Eventually(func() []v1beta1.PodDisruptionBudget {
						pdbs, err := virtClient.PolicyV1beta1().PodDisruptionBudgets(tests.NamespaceTestDefault).List(metav1.ListOptions{})
						Expect(err).ToNot(HaveOccurred())
						return pdbs.Items
					}, 3*time.Second, 500*time.Millisecond).Should(HaveLen(0))
					Eventually(func() bool {
						_, err := virtClient.VirtualMachineInstance(tests.NamespaceTestDefault).Get(vmi.Name, &metav1.GetOptions{})
						return errors.IsNotFound(err)
					}, 60*time.Second, 500*time.Millisecond).Should(BeTrue())
				}
			})

			It("[test_id:3244]should block the eviction api while a slow migration is in progress", func() {
				vmi = fedoraVMIWithEvictionStrategy()

				By("Starting the VirtualMachineInstance")
				vmi = runVMIAndExpectLaunch(vmi, 240)

				By("Checking that the VirtualMachineInstance console has expected output")
				expecter, expecterErr := tests.LoggedInFedoraExpecter(vmi)
				Expect(expecterErr).ToNot(HaveOccurred())
				defer expecter.Close()

				tests.WaitAgentConnected(virtClient, vmi)

				runStressTest(expecter)

				// execute a migration, wait for finalized state
				By("Starting the Migration")
				migration := tests.NewRandomMigration(vmi.Name, vmi.Namespace)
				migration, err := virtClient.VirtualMachineInstanceMigration(vmi.Namespace).Create(migration)
				Expect(err).ToNot(HaveOccurred())

				By("Waiting until we have two available pods")
				var pods *k8sv1.PodList
				Eventually(func() []k8sv1.Pod {
					labelSelector := fmt.Sprintf("%s=%s", v1.CreatedByLabel, vmi.GetUID())
					fieldSelector := fmt.Sprintf("status.phase==%s", k8sv1.PodRunning)
					pods, err = virtClient.CoreV1().Pods(vmi.Namespace).List(metav1.ListOptions{LabelSelector: labelSelector, FieldSelector: fieldSelector})
					Expect(err).ToNot(HaveOccurred())
					return pods.Items
				}, 90*time.Second, 500*time.Millisecond).Should(HaveLen(2))

				By("Verifying at least once that both pods are protected")
				for _, pod := range pods.Items {
					err := virtClient.CoreV1().Pods(vmi.Namespace).Evict(&v1beta1.Eviction{ObjectMeta: metav1.ObjectMeta{Name: pod.Name}})
					Expect(errors.IsTooManyRequests(err)).To(BeTrue())
				}
				By("Verifying that both pods are protected by the PodDisruptionBudget for the whole migration")
				getOptions := &metav1.GetOptions{}
				Eventually(func() v1.VirtualMachineInstanceMigrationPhase {
					currentMigration, err := virtClient.VirtualMachineInstanceMigration(vmi.Namespace).Get(migration.Name, getOptions)
					Expect(err).ToNot(HaveOccurred())
					Expect(currentMigration.Status.Phase).NotTo(Equal(v1.MigrationFailed))
					for _, pod := range pods.Items {
						err := virtClient.CoreV1().Pods(vmi.Namespace).Evict(&v1beta1.Eviction{ObjectMeta: metav1.ObjectMeta{Name: pod.Name}})
						if !errors.IsTooManyRequests(err) && currentMigration.Status.Phase != v1.MigrationRunning {
							// In case we get an unexpected error and the migration isn't running anymore, let's not fail
							continue
						}
						Expect(errors.IsTooManyRequests(err)).To(BeTrue())
					}
					return currentMigration.Status.Phase
				}, 180*time.Second, 500*time.Millisecond).Should(Equal(v1.MigrationSucceeded))
			})

			Context("with node tainted during node drain", func() {
				It("[test_id:2221] should migrate a VMI under load to another node", func() {
					tests.SkipIfVersionBelow("Eviction of completed pods requires v1.13 and above", "1.13")
					vmi = fedoraVMIWithEvictionStrategy()

					By("Starting the VirtualMachineInstance")
					vmi = runVMIAndExpectLaunch(vmi, 180)

					By("Checking that the VirtualMachineInstance console has expected output")
					expecter, expecterErr := tests.LoggedInFedoraExpecter(vmi)
					Expect(expecterErr).ToNot(HaveOccurred())
					defer expecter.Close()

					tests.WaitAgentConnected(virtClient, vmi)

					// Put VMI under load
					runStressTest(expecter)

					// Taint Node.
					By("Tainting node with kubevirt.io/drain=NoSchedule")
					node := vmi.Status.NodeName
					tests.Taint(node, "kubevirt.io/drain", k8sv1.TaintEffectNoSchedule)

					// Drain Node using cli client
					k8sClient := tests.GetK8sCmdClient()
					if k8sClient == "oc" {
						_, _, err = tests.RunCommandWithNS("", k8sClient, "adm", "drain", node, "--delete-local-data", "--ignore-daemonsets=true", "--force", "--timeout=180s")
						Expect(err).ToNot(HaveOccurred(), "Draining node")
					} else {
						_, _, err = tests.RunCommandWithNS("", k8sClient, "drain", node, "--delete-local-data", "--ignore-daemonsets=true", "--force", "--timeout=180s")
						Expect(err).ToNot(HaveOccurred(), "Draining node")
					}

					// verify VMI migrated and lives on another node now.
					Eventually(func() error {
						vmi, err := virtClient.VirtualMachineInstance(vmi.Namespace).Get(vmi.Name, &metav1.GetOptions{})
						if err != nil {
							return err
						} else if vmi.Status.NodeName == node {
							return fmt.Errorf("VMI still exist on the same node")
						} else if vmi.Status.MigrationState == nil || vmi.Status.MigrationState.SourceNode != node {
							return fmt.Errorf("VMI did not migrate yet")
						}

						// VMI should still be running at this point. If it
						// isn't, then there's nothing to be waiting on.
						Expect(vmi.Status.Phase).To(Equal(v1.Running))

						return nil
					}, 180*time.Second, 1*time.Second).ShouldNot(HaveOccurred())
				})

				It("[test_id:2222] should migrate a VMI when custom taint key is configured", func() {
					tests.SkipIfVersionBelow("Eviction of completed pods requires v1.13 and above", "1.13")
					vmi = cirrosVMIWithEvictionStrategy()

					By("Configuring a custom nodeDrainTaintKey in kubevirt-config")
					cfg, err := virtClient.CoreV1().ConfigMaps(tests.KubeVirtInstallNamespace).Get("kubevirt-config", metav1.GetOptions{})
					Expect(err).ToNot(HaveOccurred())

					// set a custom taint value
					cfg.Data["migrations"] = "nodeDrainTaintKey: kubevirt.io/alt-drain"

					newData, err := json.Marshal(cfg.Data)
					Expect(err).ToNot(HaveOccurred())
					data := fmt.Sprintf(`[{ "op": "replace", "path": "/data", "value": %s }]`, string(newData))

					_, err = virtClient.CoreV1().ConfigMaps(tests.KubeVirtInstallNamespace).Patch("kubevirt-config", types.JSONPatchType, []byte(data))
					Expect(err).ToNot(HaveOccurred())
					// this sleep is to allow the config to stick. The informers on virt-controller have to
					// be notified of the config change.
					time.Sleep(3)

					By("Starting the VirtualMachineInstance")
					vmi = runVMIAndExpectLaunch(vmi, 180)

					// Taint Node.
					By("Tainting node with kubevirt.io/alt-drain=NoSchedule")
					node := vmi.Status.NodeName
					tests.Taint(node, "kubevirt.io/alt-drain", k8sv1.TaintEffectNoSchedule)

					// Drain Node using cli client
					k8sClient := tests.GetK8sCmdClient()
					if k8sClient == "oc" {
						_, _, err = tests.RunCommandWithNS("", k8sClient, "adm", "drain", node, "--delete-local-data", "--ignore-daemonsets=true", "--force", "--timeout=180s")
						Expect(err).ToNot(HaveOccurred(), "Draining node")
					} else {
						_, _, err = tests.RunCommandWithNS("", k8sClient, "drain", node, "--delete-local-data", "--ignore-daemonsets=true", "--force", "--timeout=180s")
						Expect(err).ToNot(HaveOccurred(), "Draining node")
					}

					// verify VMI migrated and lives on another node now.
					Eventually(func() error {
						vmi, err := virtClient.VirtualMachineInstance(vmi.Namespace).Get(vmi.Name, &metav1.GetOptions{})
						if err != nil {
							return err
						} else if vmi.Status.NodeName == node {
							return fmt.Errorf("VMI still exist on the same node")
						} else if vmi.Status.MigrationState == nil || vmi.Status.MigrationState.SourceNode != node {
							return fmt.Errorf("VMI did not migrate yet")
						}
						return nil
					}, 180*time.Second, 1*time.Second).ShouldNot(HaveOccurred())
				}, 400)

				It("[test_id:2224] should handle mixture of VMs with different eviction strategies.", func() {
					tests.SkipIfVersionBelow("Eviction of completed pods requires v1.13 and above", "1.13")

					vmi_evict1 := cirrosVMIWithEvictionStrategy()
					vmi_evict2 := cirrosVMIWithEvictionStrategy()
					vmi_noevict := tests.NewRandomVMIWithEphemeralDiskAndUserdata(tests.ContainerDiskFor(tests.ContainerDiskCirros), "#!/bin/bash\necho 'hello'\n")

					labelKey := "testkey"
					labels := map[string]string{
						labelKey: "",
					}

					// give an affinity rule to ensure the vmi's get placed on the same node.
					affinityRule := &k8sv1.Affinity{
						PodAffinity: &k8sv1.PodAffinity{
							PreferredDuringSchedulingIgnoredDuringExecution: []k8sv1.WeightedPodAffinityTerm{
								{
									Weight: int32(1),
									PodAffinityTerm: k8sv1.PodAffinityTerm{
										LabelSelector: &metav1.LabelSelector{
											MatchExpressions: []metav1.LabelSelectorRequirement{
												{
													Key:      labelKey,
													Operator: metav1.LabelSelectorOpIn,
													Values:   []string{string("")}},
											},
										},
										TopologyKey: "kubernetes.io/hostname",
									},
								},
							},
						},
					}

					vmi_evict1.Labels = labels
					vmi_evict2.Labels = labels
					vmi_noevict.Labels = labels

					vmi_evict1.Spec.Affinity = affinityRule
					vmi_evict2.Spec.Affinity = affinityRule
					vmi_noevict.Spec.Affinity = affinityRule

					By("Starting the VirtualMachineInstance with eviction set to live migration")
					vm_evict1 := tests.NewRandomVirtualMachine(vmi_evict1, false)
					vm_evict2 := tests.NewRandomVirtualMachine(vmi_evict2, false)
					vm_noevict := tests.NewRandomVirtualMachine(vmi_noevict, false)

					// post VMs
					vm_evict1, err = virtClient.VirtualMachine(tests.NamespaceTestDefault).Create(vm_evict1)
					Expect(err).ToNot(HaveOccurred())
					vm_evict2, err = virtClient.VirtualMachine(tests.NamespaceTestDefault).Create(vm_evict2)
					Expect(err).ToNot(HaveOccurred())
					vm_noevict, err = virtClient.VirtualMachine(tests.NamespaceTestDefault).Create(vm_noevict)
					Expect(err).ToNot(HaveOccurred())

					// Start VMs
					vm_evict1 = tests.StartVirtualMachine(vm_evict1)
					vm_evict2 = tests.StartVirtualMachine(vm_evict2)
					vm_noevict = tests.StartVirtualMachine(vm_noevict)

					// Get VMIs
					vmi_evict1, err = virtClient.VirtualMachineInstance(vmi_evict1.Namespace).Get(vmi_evict1.Name, &metav1.GetOptions{})
					vmi_evict2, err = virtClient.VirtualMachineInstance(vmi_evict1.Namespace).Get(vmi_evict2.Name, &metav1.GetOptions{})
					vmi_noevict, err = virtClient.VirtualMachineInstance(vmi_evict1.Namespace).Get(vmi_noevict.Name, &metav1.GetOptions{})

					By("Verifying all VMIs are collcated on the same node")
					Expect(vmi_evict1.Status.NodeName).To(Equal(vmi_evict2.Status.NodeName))
					Expect(vmi_evict1.Status.NodeName).To(Equal(vmi_noevict.Status.NodeName))

					// Taint Node.
					By("Tainting node with kubevirt.io/drain=NoSchedule")
					node := vmi_evict1.Status.NodeName
					tests.Taint(node, "kubevirt.io/drain", k8sv1.TaintEffectNoSchedule)

					// Drain Node using cli client
					By("Draining using kubectl drain")
					k8sClient := tests.GetK8sCmdClient()
					if k8sClient == "oc" {
						_, _, err = tests.RunCommandWithNS("", k8sClient, "adm", "drain", node, "--delete-local-data", "--pod-selector=kubevirt.io/created-by", "--ignore-daemonsets=true", "--force", "--timeout=180s")
						Expect(err).ToNot(HaveOccurred(), "Draining node")
					} else {
						_, _, err = tests.RunCommandWithNS("", k8sClient, "drain", node, "--delete-local-data", "--pod-selector=kubevirt.io/created-by", "--ignore-daemonsets=true", "--force", "--timeout=180s")
						Expect(err).ToNot(HaveOccurred(), "Draining node")
					}

					By("Verify expected vmis migrated after node drain completes")
					// verify migrated where expected to migrate.
					Eventually(func() error {
						vmi, err := virtClient.VirtualMachineInstance(vmi_evict1.Namespace).Get(vmi_evict1.Name, &metav1.GetOptions{})
						if err != nil {
							return err
						} else if vmi.Status.NodeName == node {
							return fmt.Errorf("VMI still exist on the same node")
						} else if vmi.Status.MigrationState == nil || vmi.Status.MigrationState.SourceNode != node {
							return fmt.Errorf("VMI did not migrate yet")
						}

						vmi, err = virtClient.VirtualMachineInstance(vmi_evict2.Namespace).Get(vmi_evict2.Name, &metav1.GetOptions{})
						if err != nil {
							return err
						} else if vmi.Status.NodeName == node {
							return fmt.Errorf("VMI still exist on the same node")
						} else if vmi.Status.MigrationState == nil || vmi.Status.MigrationState.SourceNode != node {
							return fmt.Errorf("VMI did not migrate yet")
						}

						// This VMI should be terminated
						vmi, err = virtClient.VirtualMachineInstance(vmi_noevict.Namespace).Get(vmi_noevict.Name, &metav1.GetOptions{})
						if err != nil {
							return err
						} else if vmi.Status.NodeName == node {
							return fmt.Errorf("VMI still exist on the same node")
						}
						// this VM should not have migrated. Instead it should have been shutdown and started on the other node.
						Expect(vmi.Status.MigrationState).To(BeNil())
						return nil
					}, 180*time.Second, 1*time.Second).ShouldNot(HaveOccurred())

				})
			})
		})
		Context("with multiple VMIs with eviction policies set", func() {

			It("[test_id:3245]should not migrate more than two VMIs at the same time from a node", func() {
				var vmis []*v1.VirtualMachineInstance
				for i := 0; i < 4; i++ {
					vmi := cirrosVMIWithEvictionStrategy()
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

				By("waiting until migration kicks in")
				Eventually(func() int {
					migrationList, err := virtClient.VirtualMachineInstanceMigration(k8sv1.NamespaceAll).List(&metav1.ListOptions{})
					Expect(err).ToNot(HaveOccurred())

					runningMigrations := migrations.FilterRunningMigrations(migrationList.Items)

					return len(runningMigrations)
				}, 2*time.Minute, 1*time.Second).Should(BeNumerically(">", 0))

				By("checking that all VMIs were migrated, and we never see more than two running migrations in parallel")
				Eventually(func() []string {
					var nodes []string
					for _, vmi := range vmis {
						vmi, err = virtClient.VirtualMachineInstance(tests.NamespaceTestDefault).Get(vmi.Name, &metav1.GetOptions{})
						nodes = append(nodes, vmi.Status.NodeName)
					}

					migrationList, err := virtClient.VirtualMachineInstanceMigration(k8sv1.NamespaceAll).List(&metav1.ListOptions{})
					Expect(err).ToNot(HaveOccurred())

					runningMigrations := migrations.FilterRunningMigrations(migrationList.Items)
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

func fedoraVMIWithEvictionStrategy() *v1.VirtualMachineInstance {
	vmi := tests.NewRandomFedoraVMIWitGuestAgent()
	strategy := v1.EvictionStrategyLiveMigrate
	vmi.Spec.EvictionStrategy = &strategy
	vmi.Spec.Domain.Resources.Requests[k8sv1.ResourceMemory] = resource.MustParse(fedoraVMSize)
	return vmi
}

func cirrosVMIWithEvictionStrategy() *v1.VirtualMachineInstance {
	strategy := v1.EvictionStrategyLiveMigrate
	vmi := tests.NewRandomVMIWithEphemeralDiskAndUserdata(tests.ContainerDiskFor(tests.ContainerDiskCirros), "#!/bin/bash\necho 'hello'\n")
	vmi.Spec.EvictionStrategy = &strategy
	return vmi
}
