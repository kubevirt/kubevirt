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
	"context"
	"crypto/tls"
	"encoding/json"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"sync"

	virtconfig "kubevirt.io/kubevirt/pkg/virt-config"
	virthandler "kubevirt.io/kubevirt/pkg/virt-handler"
	"kubevirt.io/kubevirt/tests/framework/checks"
	"kubevirt.io/kubevirt/tests/util"
	"kubevirt.io/kubevirt/tools/vms-generator/utils"

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
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/apimachinery/pkg/util/rand"
	"k8s.io/utils/pointer"

	"kubevirt.io/kubevirt/tests/libvmi"

	storageframework "kubevirt.io/kubevirt/tests/framework/storage"

	"k8s.io/apimachinery/pkg/util/strategicpatch"

	. "kubevirt.io/kubevirt/tests/framework/matcher"

	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/api"

	v1 "kubevirt.io/api/core/v1"
	"kubevirt.io/client-go/kubecli"
	"kubevirt.io/client-go/log"
	cdiv1 "kubevirt.io/containerized-data-importer-api/pkg/apis/core/v1beta1"
	"kubevirt.io/kubevirt/pkg/certificates/triple"
	"kubevirt.io/kubevirt/pkg/certificates/triple/cert"
	"kubevirt.io/kubevirt/pkg/util/cluster"
	migrations "kubevirt.io/kubevirt/pkg/util/migrations"
	"kubevirt.io/kubevirt/tests"
	"kubevirt.io/kubevirt/tests/console"
	cd "kubevirt.io/kubevirt/tests/containerdisk"
	"kubevirt.io/kubevirt/tests/flags"
	"kubevirt.io/kubevirt/tests/libnet"
)

const (
	fedoraVMSize         = "256M"
	secretDiskSerial     = "D23YZ9W6WA5DJ487"
	stressDefaultVMSize  = "100"
	stressLargeVMSize    = "400"
	stressDefaultTimeout = 1600
)

var _ = Describe("[Serial][rfe_id:393][crit:high][vendor:cnv-qe@redhat.com][level:system][sig-compute] VM Live Migration", func() {
	var virtClient kubecli.KubevirtClient
	var err error

	BeforeEach(func() {
		virtClient, err = kubecli.GetKubevirtClient()
		Expect(err).ToNot(HaveOccurred())
	})

	setMastersUnschedulable := func(mode bool) {
		masters, err := virtClient.CoreV1().Nodes().List(context.Background(), metav1.ListOptions{LabelSelector: `node-role.kubernetes.io/master`})
		Expect(err).ShouldNot(HaveOccurred(), "could not list master nodes")
		Expect(len(masters.Items)).Should(BeNumerically(">=", 1))

		for _, node := range masters.Items {
			nodeCopy := node.DeepCopy()
			nodeCopy.Spec.Unschedulable = mode

			oldData, err := json.Marshal(node)
			Expect(err).ShouldNot(HaveOccurred())

			newData, err := json.Marshal(nodeCopy)
			Expect(err).ShouldNot(HaveOccurred())

			patch, err := strategicpatch.CreateTwoWayMergePatch(oldData, newData, node)
			Expect(err).ShouldNot(HaveOccurred())

			_, err = virtClient.CoreV1().Nodes().Patch(context.Background(), node.Name, types.StrategicMergePatchType, patch, metav1.PatchOptions{})
			Expect(err).ShouldNot(HaveOccurred())
		}
	}

	drainNode := func(node string) {
		By(fmt.Sprintf("Draining node %s", node))
		// we can't really expect an error during node drain because vms with eviction strategy can be migrated by the
		// time that we call it.
		k8sClient := tests.GetK8sCmdClient()
		if k8sClient == "oc" {
			tests.RunCommandWithNS("", k8sClient, "adm", "drain", node, "--delete-local-data",
				"--ignore-daemonsets=true", "--force", "--timeout=180s")
		} else {
			tests.RunCommandWithNS("", k8sClient, "drain", node, "--delete-local-data",
				"--ignore-daemonsets=true", "--force", "--timeout=180s")
		}
	}

	confirmMigrationMode := func(vmi *v1.VirtualMachineInstance, expectedMode v1.MigrationMode) {
		By("Retrieving the VMI post migration")
		vmi, err = virtClient.VirtualMachineInstance(vmi.Namespace).Get(vmi.Name, &metav1.GetOptions{})
		Expect(err).ToNot(HaveOccurred())

		By("Verifying the VMI's migration mode")
		Expect(vmi.Status.MigrationState.Mode).To(Equal(expectedMode))
	}

	getCurrentKv := func() v1.KubeVirtConfiguration {
		kvc := util.GetCurrentKv(virtClient)

		if kvc.Spec.Configuration.MigrationConfiguration == nil {
			kvc.Spec.Configuration.MigrationConfiguration = &v1.MigrationConfiguration{}
		}

		if kvc.Spec.Configuration.DeveloperConfiguration == nil {
			kvc.Spec.Configuration.DeveloperConfiguration = &v1.DeveloperConfiguration{}
		}

		if kvc.Spec.Configuration.NetworkConfiguration == nil {
			kvc.Spec.Configuration.NetworkConfiguration = &v1.NetworkConfiguration{}
		}

		return kvc.Spec.Configuration
	}

	BeforeEach(func() {
		tests.BeforeTestCleanup()

		tests.SkipIfMigrationIsNotPossible()

	})

	runVMIAndExpectLaunch := func(vmi *v1.VirtualMachineInstance, timeout int) *v1.VirtualMachineInstance {
		return tests.RunVMIAndExpectLaunchWithIgnoreWarningArg(vmi, timeout, false)
	}

	runVMIAndExpectLaunchIgnoreWarnings := func(vmi *v1.VirtualMachineInstance, timeout int) *v1.VirtualMachineInstance {
		return tests.RunVMIAndExpectLaunchWithIgnoreWarningArg(vmi, timeout, true)
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
		Eventually(func() v1.VirtualMachineInstanceMigrationPhase {
			migration, err := virtClient.VirtualMachineInstanceMigration(migration.Namespace).Get(migration.Name, &metav1.GetOptions{})
			Expect(err).ToNot(HaveOccurred())

			phase := migration.Status.Phase
			Expect(phase).NotTo(Equal(v1.MigrationSucceeded))

			uid = string(migration.UID)
			return phase

		}, timeout, 1*time.Second).Should(Equal(v1.MigrationFailed))
		return uid
	}

	runStressTest := func(vmi *v1.VirtualMachineInstance, vmsize string, stressTimeoutSeconds int) {
		By("Run a stress test to dirty some pages and slow down the migration")
		stressCmd := fmt.Sprintf("stress-ng --vm 1 --vm-bytes %sM --vm-keep --timeout %ds&\n", vmsize, stressTimeoutSeconds)
		Expect(console.SafeExpectBatch(vmi, []expect.Batcher{
			&expect.BSnd{S: "\n"},
			&expect.BExp{R: console.PromptExpression},
			&expect.BSnd{S: stressCmd},
			&expect.BExp{R: console.PromptExpression},
		}, 15)).To(Succeed(), "should run a stress test")

		// give stress tool some time to trash more memory pages before returning control to next steps
		if stressTimeoutSeconds < 15 {
			time.Sleep(time.Duration(stressTimeoutSeconds) * time.Second)
		} else {
			time.Sleep(15 * time.Second)
		}
	}

	getLibvirtdPid := func(pod *k8sv1.Pod) string {
		stdout, stderr, err := tests.ExecuteCommandOnPodV2(virtClient, pod, "compute",
			[]string{
				"pidof",
				"libvirtd",
			})
		errorMassageFormat := "faild after running `pidof libvirtd`  with stdout:\n %v \n stderr:\n %v \n err: \n %v \n"
		Expect(err).ToNot(HaveOccurred(), fmt.Sprintf(errorMassageFormat, stdout, stderr, err))
		pid := strings.TrimSuffix(stdout, "\n")
		return pid
	}

	deleteDataVolume := func(dv *cdiv1.DataVolume) {
		if dv != nil {
			By("Deleting the DataVolume")
			ExpectWithOffset(1, virtClient.CdiClient().CdiV1beta1().DataVolumes(dv.Namespace).Delete(context.Background(), dv.Name, metav1.DeleteOptions{})).To(Succeed(), metav1.DeleteOptions{})
		}
	}

	setMigrationBandwidthLimitation := func(migrationBandwidth resource.Quantity) {
		cfg := getCurrentKv()
		cfg.MigrationConfiguration.BandwidthPerMigration = &migrationBandwidth
		tests.UpdateKubeVirtConfigValueAndWait(cfg)
	}

	Describe("Starting a VirtualMachineInstance ", func() {

		var pvName string
		var memoryRequestSize resource.Quantity

		BeforeEach(func() {
			memoryRequestSize = resource.MustParse(fedoraVMSize)
			pvName = "test-nfs" + rand.String(48)
		})

		guestAgentMigrationTestFunc := func(mode v1.MigrationMode) {
			By("Creating the  VMI")
			vmi := tests.NewRandomVMIWithPVC(pvName)
			vmi.Spec.Domain.Resources.Requests[k8sv1.ResourceMemory] = memoryRequestSize
			vmi.Spec.Domain.Devices.Rng = &v1.Rng{}

			// add userdata for guest agent and service account mount
			mountSvcAccCommands := fmt.Sprintf(`#!/bin/bash
					mkdir /mnt/servacc
					mount /dev/$(lsblk --nodeps -no name,serial | grep %s | cut -f1 -d' ') /mnt/servacc
				`, secretDiskSerial)
			tests.AddUserData(vmi, "cloud-init", mountSvcAccCommands)

			tests.AddServiceAccountDisk(vmi, "default")
			disks := vmi.Spec.Domain.Devices.Disks
			disks[len(disks)-1].Serial = secretDiskSerial

			vmi = runVMIAndExpectLaunchIgnoreWarnings(vmi, 180)

			// Wait for cloud init to finish and start the agent inside the vmi.
			tests.WaitAgentConnected(virtClient, vmi)

			By("Checking that the VirtualMachineInstance console has expected output")
			Expect(libnet.WithIPv6(console.LoginToFedora)(vmi)).To(Succeed(), "Should be able to login to the Fedora VM")

			if mode == v1.MigrationPostCopy {
				By("Running stress test to allow transition to post-copy")
				runStressTest(vmi, stressLargeVMSize, stressDefaultTimeout)
			}

			// execute a migration, wait for finalized state
			By("Starting the Migration for iteration")
			migration := tests.NewRandomMigration(vmi.Name, vmi.Namespace)
			migrationUID := tests.RunMigrationAndExpectCompletion(virtClient, migration, tests.MigrationWaitTime)

			By("Checking VMI, confirm migration state")
			tests.ConfirmVMIPostMigration(virtClient, vmi, migrationUID)
			confirmMigrationMode(vmi, mode)

			By("Is agent connected after migration")
			tests.WaitAgentConnected(virtClient, vmi)

			By("Checking that the migrated VirtualMachineInstance console has expected output")
			Expect(console.OnPrivilegedPrompt(vmi, 60)).To(BeTrue(), "Should stay logged in to the migrated VM")

			By("Checking that the service account is mounted")
			Expect(console.SafeExpectBatch(vmi, []expect.Batcher{
				&expect.BSnd{S: "cat /mnt/servacc/namespace\n"},
				&expect.BExp{R: util.NamespaceTestDefault},
			}, 30)).To(Succeed(), "Should be able to access the mounted service account file")

			By("Deleting the VMI")
			Expect(virtClient.VirtualMachineInstance(vmi.Namespace).Delete(vmi.Name, &metav1.DeleteOptions{})).To(Succeed())

			By("Waiting for VMI to disappear")
			tests.WaitForVirtualMachineToDisappearWithTimeout(vmi, 120)
		}

		Context("with a bridge network interface", func() {
			It("[test_id:3226]should reject a migration of a vmi with a bridge interface", func() {
				vmi := tests.NewRandomVMIWithEphemeralDisk(cd.ContainerDiskFor(cd.ContainerDiskAlpine))
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
				Expect(console.LoginToAlpine(vmi)).To(Succeed())

				gotExpectedCondition := false
				for _, c := range vmi.Status.Conditions {
					if c.Type == v1.VirtualMachineInstanceIsMigratable {
						Expect(c.Status).To(Equal(k8sv1.ConditionFalse))
						gotExpectedCondition = true
					}
				}

				Expect(gotExpectedCondition).Should(BeTrue())

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
		Context("with bandwidth limitations", func() {

			var repeatedlyMigrateWithBandwidthLimitation = func(vmi *v1.VirtualMachineInstance, bandwidth string, repeat int) time.Duration {
				var migrationDurationTotal time.Duration
				config := getCurrentKv()
				limit := resource.MustParse(bandwidth)
				config.MigrationConfiguration.BandwidthPerMigration = &limit
				tests.UpdateKubeVirtConfigValueAndWait(config)

				for x := 0; x < repeat; x++ {
					By("Checking that the VirtualMachineInstance console has expected output")
					Expect(libnet.WithIPv6(console.LoginToCirros)(vmi)).To(Succeed())

					By("starting the migration")
					migration := tests.NewRandomMigration(vmi.Name, vmi.Namespace)
					migrationUID := tests.RunMigrationAndExpectCompletion(virtClient, migration, tests.MigrationWaitTime)

					// check VMI, confirm migration state
					tests.ConfirmVMIPostMigration(virtClient, vmi, migrationUID)

					vmi, err := virtClient.VirtualMachineInstance(vmi.Namespace).Get(vmi.Name, &metav1.GetOptions{})
					Expect(err).ToNot(HaveOccurred())
					migrationDuration := vmi.Status.MigrationState.EndTimestamp.Sub(vmi.Status.MigrationState.StartTimestamp.Time)
					log.DefaultLogger().Infof("Migration with bandwidth %v took: %v", bandwidth, migrationDuration)
					migrationDurationTotal += migrationDuration
				}
				return migrationDurationTotal
			}

			It("[test_id:6968]should apply them and result in different migration durations", func() {
				vmi := tests.NewRandomVMIWithEphemeralDisk(cd.ContainerDiskFor(cd.ContainerDiskCirros))
				tests.AddUserData(vmi, "cloud-init", "#!/bin/bash\necho 'hello'\n")
				By("Starting the VirtualMachineInstance")
				vmi = runVMIAndExpectLaunch(vmi, 240)

				durationLowBandwidth := repeatedlyMigrateWithBandwidthLimitation(vmi, "10Mi", 3)
				durationHighBandwidth := repeatedlyMigrateWithBandwidthLimitation(vmi, "128Mi", 3)
				Expect(durationHighBandwidth.Seconds() * 2).To(BeNumerically("<", durationLowBandwidth.Seconds()))
			})
		})
		Context("with a Cirros disk", func() {
			It("[test_id:6969]should be successfully migrate with a tablet device", func() {
				vmi := tests.NewRandomVMIWithEphemeralDisk(cd.ContainerDiskFor(cd.ContainerDiskCirros))
				tests.AddUserData(vmi, "cloud-init", "#!/bin/bash\necho 'hello'\n")
				vmi.Spec.Domain.Devices.Inputs = []v1.Input{
					{
						Name: "tablet0",
						Type: "tablet",
						Bus:  "usb",
					},
				}

				By("Starting the VirtualMachineInstance")
				vmi = runVMIAndExpectLaunch(vmi, 240)

				By("Checking that the VirtualMachineInstance console has expected output")
				Expect(libnet.WithIPv6(console.LoginToCirros)(vmi)).To(Succeed())

				By("starting the migration")
				migration := tests.NewRandomMigration(vmi.Name, vmi.Namespace)
				migrationUID := tests.RunMigrationAndExpectCompletion(virtClient, migration, tests.MigrationWaitTime)

				// check VMI, confirm migration state
				tests.ConfirmVMIPostMigration(virtClient, vmi, migrationUID)

				// delete VMI
				By("Deleting the VMI")
				Expect(virtClient.VirtualMachineInstance(vmi.Namespace).Delete(vmi.Name, &metav1.DeleteOptions{})).To(Succeed())

				By("Waiting for VMI to disappear")
				tests.WaitForVirtualMachineToDisappearWithTimeout(vmi, 240)
			})

			It("[test_id:6970]should migrate vmi with cdroms on various bus types", func() {
				vmi := tests.NewRandomVMIWithEphemeralDisk(cd.ContainerDiskFor(cd.ContainerDiskCirros))
				tests.AddEphemeralCdrom(vmi, "cdrom-0", "sata", cd.ContainerDiskFor(cd.ContainerDiskCirros))
				tests.AddEphemeralCdrom(vmi, "cdrom-1", "scsi", cd.ContainerDiskFor(cd.ContainerDiskCirros))
				tests.AddUserData(vmi, "cloud-init", "#!/bin/bash\necho 'hello'\n")

				By("Starting the VirtualMachineInstance")
				vmi = runVMIAndExpectLaunch(vmi, 240)

				By("Checking that the VirtualMachineInstance console has expected output")
				Expect(libnet.WithIPv6(console.LoginToCirros)(vmi)).To(Succeed())

				// execute a migration, wait for finalized state
				By("starting the migration")
				migration := tests.NewRandomMigration(vmi.Name, vmi.Namespace)
				migrationUID := tests.RunMigrationAndExpectCompletion(virtClient, migration, tests.MigrationWaitTime)

				// check VMI, confirm migration state
				tests.ConfirmVMIPostMigration(virtClient, vmi, migrationUID)
			})

			It("[test_id:6971]should migrate with a downwardMetrics disk", func() {
				vmi := libvmi.NewTestToolingFedora(
					libvmi.WithInterface(libvmi.InterfaceDeviceWithMasqueradeBinding()),
					libvmi.WithNetwork(v1.DefaultPodNetwork()),
				)
				tests.AddDownwardMetricsVolume(vmi, "vhostmd")
				vmi = tests.RunVMIAndExpectLaunch(vmi, 180)
				Expect(console.LoginToFedora(vmi)).To(Succeed())

				By("starting the migration")
				migration := tests.NewRandomMigration(vmi.Name, vmi.Namespace)
				migrationUID := tests.RunMigrationAndExpectCompletion(virtClient, migration, tests.MigrationWaitTime)

				tests.ConfirmVMIPostMigration(virtClient, vmi, migrationUID)

				By("checking if the metrics are still updated after the migration")
				Eventually(func() error {
					_, err := getDownwardMetrics(vmi)
					return err
				}, 20*time.Second, 1*time.Second).ShouldNot(HaveOccurred())
				metrics, err := getDownwardMetrics(vmi)
				Expect(err).ToNot(HaveOccurred())
				timestamp := getTimeFromMetrics(metrics)
				Eventually(func() int {
					metrics, err := getDownwardMetrics(vmi)
					Expect(err).ToNot(HaveOccurred())
					return getTimeFromMetrics(metrics)
				}, 10*time.Second, 1*time.Second).ShouldNot(Equal(timestamp))

				By("checking that the new nodename is reflected in the downward metrics")
				vmi, err = virtClient.VirtualMachineInstance(vmi.Namespace).Get(vmi.Name, &metav1.GetOptions{})
				Expect(err).ToNot(HaveOccurred())
				Expect(getHostnameFromMetrics(metrics)).To(Equal(vmi.Status.NodeName))
			})

			It("[test_id:6842]should migrate with TSC frequency set", func() {
				vmi := tests.NewRandomVMIWithEphemeralDisk(cd.ContainerDiskFor(cd.ContainerDiskCirros))
				tests.AddUserData(vmi, "cloud-init", "#!/bin/bash\necho 'hello'\n")
				vmi.Spec.Domain.CPU = &v1.CPU{
					Features: []v1.CPUFeature{
						{
							Name:   "invtsc",
							Policy: "require",
						},
					},
				}
				// only with this strategy will the frequency be set
				strategy := v1.EvictionStrategyLiveMigrate
				vmi.Spec.EvictionStrategy = &strategy

				vmi = tests.RunVMIAndExpectLaunch(vmi, 180)
				Expect(console.LoginToCirros(vmi)).To(Succeed())

				By("Checking the TSC frequency on the Domain XML")
				domainSpec, err := tests.GetRunningVMIDomainSpec(vmi)
				Expect(err).ToNot(HaveOccurred())
				timerFrequency := ""
				for _, timer := range domainSpec.Clock.Timer {
					if timer.Name == "tsc" {
						timerFrequency = timer.Frequency
					}
				}
				Expect(timerFrequency).ToNot(BeEmpty())

				By("starting the migration")
				migration := tests.NewRandomMigration(vmi.Name, vmi.Namespace)
				migrationUID := tests.RunMigrationAndExpectCompletion(virtClient, migration, tests.MigrationWaitTime)

				tests.ConfirmVMIPostMigration(virtClient, vmi, migrationUID)

				By("Checking the TSC frequency on the Domain XML on the new node")
				domainSpec, err = tests.GetRunningVMIDomainSpec(vmi)
				Expect(err).ToNot(HaveOccurred())
				timerFrequency = ""
				for _, timer := range domainSpec.Clock.Timer {
					if timer.Name == "tsc" {
						timerFrequency = timer.Frequency
					}
				}
				Expect(timerFrequency).ToNot(BeEmpty())
			})

			It("[test_id:4113]should be successfully migrate with cloud-init disk with devices on the root bus", func() {
				vmi := tests.NewRandomVMIWithEphemeralDisk(cd.ContainerDiskFor(cd.ContainerDiskCirros))
				vmi.Annotations = map[string]string{
					v1.PlacePCIDevicesOnRootComplex: "true",
				}
				tests.AddUserData(vmi, "cloud-init", "#!/bin/bash\necho 'hello'\n")

				By("Starting the VirtualMachineInstance")
				vmi = runVMIAndExpectLaunch(vmi, 240)

				By("Checking that the VirtualMachineInstance console has expected output")
				Expect(libnet.WithIPv6(console.LoginToCirros)(vmi)).To(Succeed())

				// execute a migration, wait for finalized state
				By("starting the migration")
				migration := tests.NewRandomMigration(vmi.Name, vmi.Namespace)
				migrationUID := tests.RunMigrationAndExpectCompletion(virtClient, migration, tests.MigrationWaitTime)

				// check VMI, confirm migration state
				tests.ConfirmVMIPostMigration(virtClient, vmi, migrationUID)

				By("checking that we really migrated a VMI with only the root bus")
				domSpec, err := tests.GetRunningVMIDomainSpec(vmi)
				Expect(err).ToNot(HaveOccurred())
				rootPortController := []api.Controller{}
				for _, c := range domSpec.Devices.Controllers {
					if c.Model == "pcie-root-port" {
						rootPortController = append(rootPortController, c)
					}
				}
				Expect(rootPortController).To(HaveLen(0), "libvirt should not add additional buses to the root one")

				// delete VMI
				By("Deleting the VMI")
				Expect(virtClient.VirtualMachineInstance(vmi.Namespace).Delete(vmi.Name, &metav1.DeleteOptions{})).To(Succeed())

				By("Waiting for VMI to disappear")
				tests.WaitForVirtualMachineToDisappearWithTimeout(vmi, 240)
			})

			It("[test_id:1783]should be successfully migrated multiple times with cloud-init disk", func() {
				vmi := tests.NewRandomVMIWithEphemeralDisk(cd.ContainerDiskFor(cd.ContainerDiskCirros))
				tests.AddUserData(vmi, "cloud-init", "#!/bin/bash\necho 'hello'\n")

				By("Starting the VirtualMachineInstance")
				vmi = runVMIAndExpectLaunch(vmi, 240)

				By("Checking that the VirtualMachineInstance console has expected output")
				Expect(libnet.WithIPv6(console.LoginToCirros)(vmi)).To(Succeed())

				num := 4

				for i := 0; i < num; i++ {
					// execute a migration, wait for finalized state
					By(fmt.Sprintf("Starting the Migration for iteration %d", i))
					migration := tests.NewRandomMigration(vmi.Name, vmi.Namespace)
					migrationUID := tests.RunMigrationAndExpectCompletion(virtClient, migration, tests.MigrationWaitTime)

					// check VMI, confirm migration state
					tests.ConfirmVMIPostMigration(virtClient, vmi, migrationUID)

					By("Check if Migrated VMI has updated IP and IPs fields")
					Eventually(func() error {
						newvmi, err := virtClient.VirtualMachineInstance(util.NamespaceTestDefault).Get(vmi.Name, &metav1.GetOptions{})
						Expect(err).ToNot(HaveOccurred(), "Should successfully get new VMI")
						vmiPod := tests.GetRunningPodByVirtualMachineInstance(newvmi, newvmi.Namespace)
						return libnet.ValidateVMIandPodIPMatch(newvmi, vmiPod)
					}, 180*time.Second, time.Second).Should(Succeed(), "Should have updated IP and IPs fields")
				}
				// delete VMI
				By("Deleting the VMI")
				Expect(virtClient.VirtualMachineInstance(vmi.Namespace).Delete(vmi.Name, &metav1.DeleteOptions{})).To(Succeed())

				By("Waiting for VMI to disappear")
				tests.WaitForVirtualMachineToDisappearWithTimeout(vmi, 240)

			})

			// We had a bug that prevent migrations and graceful shutdown when the libvirt connection
			// is reset. This can occur for many reasons, one easy way to trigger it is to
			// force libvirtd down, which will result in virt-launcher respawning it.
			// Previously, we'd stop getting events after libvirt reconnect, which
			// prevented things like migration. This test verifies we can migrate after
			// resetting libvirt
			It("[test_id:4746][QUARANTINE]should migrate even if libvirt has restarted at some point.", func() {
				vmi := tests.NewRandomVMIWithEphemeralDisk(cd.ContainerDiskFor(cd.ContainerDiskCirros))
				tests.AddUserData(vmi, "cloud-init", "#!/bin/bash\necho 'hello'\n")

				By("Starting the VirtualMachineInstance")
				vmi = runVMIAndExpectLaunch(vmi, 240)

				By("Checking that the VirtualMachineInstance console has expected output")
				Expect(libnet.WithIPv6(console.LoginToCirros)(vmi)).To(Succeed())

				pods, err := virtClient.CoreV1().Pods(vmi.Namespace).List(context.Background(), metav1.ListOptions{
					LabelSelector: v1.CreatedByLabel + "=" + string(vmi.GetUID()),
				})
				Expect(err).ToNot(HaveOccurred(), "Should list pods successfully")
				Expect(pods.Items).To(HaveLen(1), "There should be only one VMI pod")

				// find libvirtd pid
				pid := getLibvirtdPid(&pods.Items[0])

				// kill libvirtd
				By(fmt.Sprintf("Killing libvirtd with pid %s", pid))
				stdout, stderr, err := tests.ExecuteCommandOnPodV2(virtClient, &pods.Items[0], "compute",
					[]string{
						"kill",
						"-9",
						pid,
					})
				errorMassageFormat := "faild after running `kill -9 %v`  with stdout:\n %v \n stderr:\n %v \n err: \n %v \n"
				Expect(err).ToNot(HaveOccurred(), fmt.Sprintf(errorMassageFormat, pid, stdout, stderr, err))

				// wait for both libvirt to respawn and all connections to re-establish
				time.Sleep(30 * time.Second)

				// ensure new pid comes online
				newPid := getLibvirtdPid(&pods.Items[0])
				Expect(pid).ToNot(Equal(newPid), fmt.Sprintf("expected libvirtd to be cycled. original pid %s new pid %s", pid, newPid))

				// execute a migration, wait for finalized state
				By(fmt.Sprintf("Starting the Migration"))
				migration := tests.NewRandomMigration(vmi.Name, vmi.Namespace)
				migrationUID := tests.RunMigrationAndExpectCompletion(virtClient, migration, tests.MigrationWaitTime)

				// check VMI, confirm migration state
				tests.ConfirmVMIPostMigration(virtClient, vmi, migrationUID)

				// delete VMI
				By("Deleting the VMI")
				Expect(virtClient.VirtualMachineInstance(vmi.Namespace).Delete(vmi.Name, &metav1.DeleteOptions{})).To(Succeed())

				By("Waiting for VMI to disappear")
				tests.WaitForVirtualMachineToDisappearWithTimeout(vmi, 240)

			})

			It("[test_id:6972]should migrate to a persistent (non-transient) libvirt domain.", func() {
				vmi := tests.NewRandomVMIWithEphemeralDisk(cd.ContainerDiskFor(cd.ContainerDiskCirros))
				tests.AddUserData(vmi, "cloud-init", "#!/bin/bash\necho 'hello'\n")

				By("Starting the VirtualMachineInstance")
				vmi = runVMIAndExpectLaunch(vmi, 240)

				By("Checking that the VirtualMachineInstance console has expected output")
				Expect(libnet.WithIPv6(console.LoginToCirros)(vmi)).To(Succeed())

				// execute a migration, wait for finalized state
				By(fmt.Sprintf("Starting the Migration"))
				migration := tests.NewRandomMigration(vmi.Name, vmi.Namespace)
				migrationUID := tests.RunMigrationAndExpectCompletion(virtClient, migration, tests.MigrationWaitTime)

				// check VMI, confirm migration state
				tests.ConfirmVMIPostMigration(virtClient, vmi, migrationUID)

				// ensure the libvirt domain is persistent
				persistent, err := tests.LibvirtDomainIsPersistent(virtClient, vmi)
				Expect(err).ToNot(HaveOccurred(), "Should list libvirt domains successfully")
				Expect(persistent).To(BeTrue(), "The VMI was not found in the list of libvirt persistent domains")
				tests.EnsureNoMigrationMetadataInPersistentXML(vmi)

				// delete VMI
				By("Deleting the VMI")
				Expect(virtClient.VirtualMachineInstance(vmi.Namespace).Delete(vmi.Name, &metav1.DeleteOptions{})).To(Succeed())

				By("Waiting for VMI to disappear")
				tests.WaitForVirtualMachineToDisappearWithTimeout(vmi, 240)

			})
			It("[test_id:6973]should be able to successfully migrate with a paused vmi", func() {
				vmi := tests.NewRandomVMIWithEphemeralDisk(cd.ContainerDiskFor(cd.ContainerDiskCirros))
				tests.AddUserData(vmi, "cloud-init", "#!/bin/bash\necho 'hello'\n")

				By("Starting the VirtualMachineInstance")
				vmi = runVMIAndExpectLaunch(vmi, 240)

				By("Checking that the VirtualMachineInstance console has expected output")
				Expect(libnet.WithIPv6(console.LoginToCirros)(vmi)).To(Succeed())

				By("Pausing the VirtualMachineInstance")
				virtClient.VirtualMachineInstance(vmi.Namespace).Pause(vmi.Name, &v1.PauseOptions{})
				tests.WaitForVMICondition(virtClient, vmi, v1.VirtualMachineInstancePaused, 30)

				By("verifying that the vmi is still paused before migration")
				isPausedb, err := tests.LibvirtDomainIsPaused(virtClient, vmi)
				Expect(err).ToNot(HaveOccurred(), "Should get domain state successfully")
				Expect(isPausedb).To(BeTrue(), "The VMI should be paused before migration, but it is not.")

				By("starting the migration")
				migration := tests.NewRandomMigration(vmi.Name, vmi.Namespace)
				migrationUID := tests.RunMigrationAndExpectCompletion(virtClient, migration, tests.MigrationWaitTime)

				// check VMI, confirm migration state
				tests.ConfirmVMIPostMigration(virtClient, vmi, migrationUID)

				By("verifying that the vmi is still paused after migration")
				isPaused, err := tests.LibvirtDomainIsPaused(virtClient, vmi)
				Expect(err).ToNot(HaveOccurred(), "Should get domain state successfully")
				Expect(isPaused).To(BeTrue(), "The VMI should be paused after migration, but it is not.")

				By("verify that VMI can be unpaused after migration")
				command := tests.NewRepeatableVirtctlCommand("unpause", "vmi", "--namespace", util.NamespaceTestDefault, vmi.Name)
				Expect(command()).To(Succeed(), "should successfully unpause tthe vmi")
				tests.WaitForVMIConditionRemovedOrFalse(virtClient, vmi, v1.VirtualMachineInstancePaused, 30)

				By("verifying that the vmi is running")
				isPaused, err = tests.LibvirtDomainIsPaused(virtClient, vmi)
				Expect(err).ToNot(HaveOccurred(), "Should get domain state successfully")
				Expect(isPaused).To(BeFalse(), "The VMI should be running, but it is not.")
			})
		})

		Context("with an pending target pod", func() {
			var nodes *k8sv1.NodeList
			BeforeEach(func() {
				tests.BeforeTestCleanup()
				Eventually(func() []k8sv1.Node {
					nodes = util.GetAllSchedulableNodes(virtClient)
					return nodes.Items
				}, 60*time.Second, 1*time.Second).ShouldNot(BeEmpty(), "There should be some compute node")
			})

			It("should automatically cancel unschedulable migration after a timeout period", func() {
				vmi := tests.NewRandomFedoraVMIWithGuestAgent()
				vmi.Spec.Domain.Resources.Requests[k8sv1.ResourceMemory] = resource.MustParse(fedoraVMSize)

				// Add node affinity to ensure VMI affinity rules block target pod from being created
				addNodeAffinityToVMI(vmi, nodes.Items[0].Name)

				By("Starting the VirtualMachineInstance")
				vmi = runVMIAndExpectLaunch(vmi, 240)

				// execute a migration that is expected to fail
				By("Starting the Migration")
				migration := tests.NewRandomMigration(vmi.Name, vmi.Namespace)
				migration.Annotations = map[string]string{v1.MigrationUnschedulablePodTimeoutSecondsAnnotation: "130"}

				By("Starting a Migration")
				var err error
				Eventually(func() error {
					migration, err = virtClient.VirtualMachineInstanceMigration(migration.Namespace).Create(migration)
					return err
				}, 5, 1*time.Second).Should(Succeed(), "migration creation should succeed")

				By("Migration should observe a timeout period before canceling unschedulable target pod")
				Consistently(func() error {

					migration, err := virtClient.VirtualMachineInstanceMigration(migration.Namespace).Get(migration.Name, &metav1.GetOptions{})
					if err != nil {
						return err
					}

					if migration.Status.Phase == v1.MigrationFailed {
						return fmt.Errorf("Migration should observe timeout period before transitioning to failed state")
					}
					return nil

				}, 1*time.Minute, 10*time.Second).Should(Succeed())

				By("Migration should fail eventually due to pending target pod timeout")
				Eventually(func() error {
					migration, err := virtClient.VirtualMachineInstanceMigration(migration.Namespace).Get(migration.Name, &metav1.GetOptions{})
					if err != nil {
						return err
					}

					if migration.Status.Phase != v1.MigrationFailed {
						return fmt.Errorf("Waiting on migration with phase %s to reach phase Failed", migration.Status.Phase)
					}
					return nil
				}, 2*time.Minute, 5*time.Second).Should(Succeed(), "migration creation should fail")

				// delete VMI
				By("Deleting the VMI")
				Expect(virtClient.VirtualMachineInstance(vmi.Namespace).Delete(vmi.Name, &metav1.DeleteOptions{})).To(Succeed())

				By("Waiting for VMI to disappear")
				tests.WaitForVirtualMachineToDisappearWithTimeout(vmi, 240)
			})

			It("should automatically cancel pending target pod after a catch all timeout period", func() {
				vmi := tests.NewRandomFedoraVMIWithGuestAgent()
				vmi.Spec.Domain.Resources.Requests[k8sv1.ResourceMemory] = resource.MustParse(fedoraVMSize)

				By("Starting the VirtualMachineInstance")
				vmi = runVMIAndExpectLaunch(vmi, 240)

				// execute a migration that is expected to fail
				By("Starting the Migration")
				migration := tests.NewRandomMigration(vmi.Name, vmi.Namespace)
				migration.Annotations = map[string]string{v1.MigrationPendingPodTimeoutSecondsAnnotation: "130"}

				// Add a fake continer image to the target pod to force a image pull failure which
				// keeps the target pod in pending state
				// Make sure to actually use an image repository we own here so no one
				// can somehow figure out a way to execute custom logic in our func tests.
				migration.Annotations[v1.FuncTestMigrationTargetImageOverrideAnnotation] = "quay.io/kubevirtci/some-fake-image:" + rand.String(12)

				By("Starting a Migration")
				var err error
				Eventually(func() error {
					migration, err = virtClient.VirtualMachineInstanceMigration(migration.Namespace).Create(migration)
					return err
				}, 5, 1*time.Second).Should(Succeed(), "migration creation should succeed")

				By("Migration should observe a timeout period before canceling pending target pod")
				Consistently(func() error {

					migration, err := virtClient.VirtualMachineInstanceMigration(migration.Namespace).Get(migration.Name, &metav1.GetOptions{})
					if err != nil {
						return err
					}

					if migration.Status.Phase == v1.MigrationFailed {
						return fmt.Errorf("Migration should observe timeout period before transitioning to failed state")
					}
					return nil

				}, 1*time.Minute, 10*time.Second).Should(Succeed())

				By("Migration should fail eventually due to pending target pod timeout")
				Eventually(func() error {
					migration, err := virtClient.VirtualMachineInstanceMigration(migration.Namespace).Get(migration.Name, &metav1.GetOptions{})
					if err != nil {
						return err
					}

					if migration.Status.Phase != v1.MigrationFailed {
						return fmt.Errorf("Waiting on migration with phase %s to reach phase Failed", migration.Status.Phase)
					}
					return nil
				}, 2*time.Minute, 5*time.Second).Should(Succeed(), "migration creation should fail")

				// delete VMI
				By("Deleting the VMI")
				Expect(virtClient.VirtualMachineInstance(vmi.Namespace).Delete(vmi.Name, &metav1.DeleteOptions{})).To(Succeed())

				By("Waiting for VMI to disappear")
				tests.WaitForVirtualMachineToDisappearWithTimeout(vmi, 240)
			})
		})
		Context("with auto converge enabled", func() {
			BeforeEach(func() {
				tests.BeforeTestCleanup()

				// set autoconverge flag
				config := getCurrentKv()
				allowAutoConverage := true
				config.MigrationConfiguration.AllowAutoConverge = &allowAutoConverage
				tests.UpdateKubeVirtConfigValueAndWait(config)
			})

			It("[test_id:3237]should complete a migration", func() {
				vmi := tests.NewRandomFedoraVMIWithGuestAgent()
				vmi.Spec.Domain.Resources.Requests[k8sv1.ResourceMemory] = resource.MustParse(fedoraVMSize)

				By("Starting the VirtualMachineInstance")
				vmi = runVMIAndExpectLaunch(vmi, 240)

				// Need to wait for cloud init to finnish and start the agent inside the vmi.
				tests.WaitAgentConnected(virtClient, vmi)

				By("Checking that the VirtualMachineInstance console has expected output")
				Expect(console.LoginToFedora(vmi)).To(Succeed())

				runStressTest(vmi, stressDefaultVMSize, stressDefaultTimeout)

				// execute a migration, wait for finalized state
				By("Starting the Migration")
				migration := tests.NewRandomMigration(vmi.Name, vmi.Namespace)
				migrationUID := tests.RunMigrationAndExpectCompletion(virtClient, migration, tests.MigrationWaitTime)

				// check VMI, confirm migration state
				tests.ConfirmVMIPostMigration(virtClient, vmi, migrationUID)

				// delete VMI
				By("Deleting the VMI")
				Expect(virtClient.VirtualMachineInstance(vmi.Namespace).Delete(vmi.Name, &metav1.DeleteOptions{})).To(Succeed())

				By("Waiting for VMI to disappear")
				tests.WaitForVirtualMachineToDisappearWithTimeout(vmi, 240)
			})
		})
		Context("with setting guest time", func() {
			It("[test_id:4114]should set an updated time after a migration", func() {
				vmi := tests.NewRandomFedoraVMIWithGuestAgent()
				vmi.Spec.Domain.Resources.Requests[k8sv1.ResourceMemory] = resource.MustParse(fedoraVMSize)
				vmi.Spec.Domain.Devices.Rng = &v1.Rng{}

				By("Starting the VirtualMachineInstance")
				vmi = runVMIAndExpectLaunch(vmi, 240)

				By("Checking that the VirtualMachineInstance console has expected output")
				Expect(console.LoginToFedora(vmi)).To(Succeed())

				// Need to wait for cloud init to finnish and start the agent inside the vmi.
				tests.WaitAgentConnected(virtClient, vmi)

				By("Set wrong time on the guest")
				Expect(console.SafeExpectBatch(vmi, []expect.Batcher{
					&expect.BSnd{S: "date +%T -s 23:26:00\n"},
					&expect.BExp{R: console.PromptExpression},
				}, 15)).To(Succeed(), "should set guest time")

				// execute a migration, wait for finalized state
				By("Starting the Migration")
				migration := tests.NewRandomMigration(vmi.Name, vmi.Namespace)
				migrationUID := tests.RunMigrationAndExpectCompletion(virtClient, migration, tests.MigrationWaitTime)

				// check VMI, confirm migration state
				tests.ConfirmVMIPostMigration(virtClient, vmi, migrationUID)
				tests.WaitAgentConnected(virtClient, vmi)

				By("Checking that the migrated VirtualMachineInstance has an updated time")
				if !console.OnPrivilegedPrompt(vmi, 60) {
					Expect(console.LoginToFedora(vmi)).To(Succeed())
				}

				By("Waiting for the agent to set the right time")
				Eventually(func() error {
					// get current time on the node
					output := tests.RunCommandOnVmiPod(vmi, []string{"date", "+%H:%M"})
					expectedTime := strings.TrimSpace(output)
					log.DefaultLogger().Infof("expoected time: %v", expectedTime)

					By("Checking that the guest has an updated time")
					return console.SafeExpectBatch(vmi, []expect.Batcher{
						&expect.BSnd{S: "date +%H:%M\n"},
						&expect.BExp{R: expectedTime},
					}, 30)
				}, 240*time.Second, 1*time.Second).Should(Succeed())
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
				dataVolume := tests.NewRandomDataVolumeWithRegistryImport(cd.DataVolumeImportUrlForContainerDisk(cd.ContainerDiskAlpine), util.NamespaceTestDefault, k8sv1.ReadWriteOnce)
				vmi := tests.NewRandomVMIWithDataVolume(dataVolume.Name)

				_, err := virtClient.CdiClient().CdiV1beta1().DataVolumes(dataVolume.Namespace).Create(context.Background(), dataVolume, metav1.CreateOptions{})
				Expect(err).ToNot(HaveOccurred())

				Eventually(ThisDV(dataVolume), 240).Should(Or(HaveSucceeded(), BeInPhase(cdiv1.WaitForFirstConsumer)))

				vmi = runVMIAndExpectLaunch(vmi, 240)

				// Verify console on last iteration to verify the VirtualMachineInstance is still booting properly
				// after being restarted multiple times
				By("Checking that the VirtualMachineInstance console has expected output")
				Expect(console.LoginToAlpine(vmi)).To(Succeed())

				gotExpectedCondition := false
				for _, c := range vmi.Status.Conditions {
					if c.Type == v1.VirtualMachineInstanceIsMigratable {
						Expect(c.Status).To(Equal(k8sv1.ConditionFalse))
						gotExpectedCondition = true
					}
				}
				Expect(gotExpectedCondition).Should(BeTrue())

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

				Expect(virtClient.CdiClient().CdiV1beta1().DataVolumes(dataVolume.Namespace).Delete(context.Background(), dataVolume.Name, metav1.DeleteOptions{})).To(Succeed(), metav1.DeleteOptions{})
			})
			It("[test_id:1479][rook-ceph] should migrate a vmi with a shared OCS disk", func() {
				vmi, dv := tests.NewRandomVirtualMachineInstanceWithOCSDisk(cd.DataVolumeImportUrlForContainerDisk(cd.ContainerDiskAlpine), util.NamespaceTestDefault, k8sv1.ReadWriteMany, k8sv1.PersistentVolumeBlock)
				defer deleteDataVolume(dv)

				By("Starting the VirtualMachineInstance")
				vmi = runVMIAndExpectLaunch(vmi, 300)

				By("Checking that the VirtualMachineInstance console has expected output")
				Expect(console.LoginToAlpine(vmi)).To(Succeed())

				By("Starting a Migration")
				migration := tests.NewRandomMigration(vmi.Name, vmi.Namespace)
				migrationUID := tests.RunMigrationAndExpectCompletion(virtClient, migration, tests.MigrationWaitTime)

				// check VMI, confirm migration state
				tests.ConfirmVMIPostMigration(virtClient, vmi, migrationUID)

				// delete VMI
				By("Deleting the VMI")
				Expect(virtClient.VirtualMachineInstance(vmi.Namespace).Delete(vmi.Name, &metav1.DeleteOptions{})).To(Succeed())

				By("Waiting for VMI to disappear")
				tests.WaitForVirtualMachineToDisappearWithTimeout(vmi, 120)
			})
			It("[test_id:6974]should reject additional migrations on the same VMI if the first one is not finished", func() {
				vmi := tests.NewRandomFedoraVMIWithGuestAgent()
				vmi.Spec.Domain.Resources.Requests[k8sv1.ResourceMemory] = resource.MustParse(fedoraVMSize)

				By("Starting the VirtualMachineInstance")
				vmi = runVMIAndExpectLaunch(vmi, 240)

				// Need to wait for cloud init to finish and start the agent inside the vmi.
				tests.WaitAgentConnected(virtClient, vmi)

				By("Checking that the VirtualMachineInstance console has expected output")
				Expect(console.LoginToFedora(vmi)).To(Succeed())

				// Only stressing the VMI for 60 seconds to ensure the first migration eventually succeeds
				By("Stressing the VMI")
				runStressTest(vmi, stressDefaultVMSize, 60)

				By("Starting a first migration")
				migration1 := tests.NewRandomMigration(vmi.Name, vmi.Namespace)
				migration1, err = virtClient.VirtualMachineInstanceMigration(migration1.Namespace).Create(migration1)
				Expect(err).To(BeNil())

				// Successfully tested with 40, but requests start getting throttled above 10, which is better to avoid to prevent flakyness
				By("Starting 10 more migrations expecting all to fail to create")
				var wg sync.WaitGroup
				for n := 0; n < 10; n++ {
					wg.Add(1)
					go func(n int) {
						defer GinkgoRecover()
						defer wg.Done()
						migration := tests.NewRandomMigration(vmi.Name, vmi.Namespace)
						_, err = virtClient.VirtualMachineInstanceMigration(migration.Namespace).Create(migration)
						Expect(err).To(HaveOccurred(), fmt.Sprintf("Extra migration %d should have failed to create", n))
						Expect(err.Error()).To(ContainSubstring(`admission webhook "migration-create-validator.kubevirt.io" denied the request: in-flight migration detected.`))
					}(n)
				}
				wg.Wait()

				tests.ExpectMigrationSuccess(virtClient, migration1, tests.MigrationWaitTime)

				// delete VMI
				By("Deleting the VMI")
				Expect(virtClient.VirtualMachineInstance(vmi.Namespace).Delete(vmi.Name, &metav1.DeleteOptions{})).To(Succeed())

				By("Waiting for VMI to disappear")
				tests.WaitForVirtualMachineToDisappearWithTimeout(vmi, 120)
			})
		})
		Context("[rook-ceph]with an Alpine shared block volume PVC", func() {

			It("[test_id:1854]should migrate a VMI with shared and non-shared disks", func() {
				// Start the VirtualMachineInstance with PVC and Ephemeral Disks
				vmi, _ := tests.NewRandomVirtualMachineInstanceWithOCSDisk(cd.DataVolumeImportUrlForContainerDisk(cd.ContainerDiskAlpine), util.NamespaceTestDefault, k8sv1.ReadWriteMany, k8sv1.PersistentVolumeBlock)
				image := cd.ContainerDiskFor(cd.ContainerDiskAlpine)
				tests.AddEphemeralDisk(vmi, "myephemeral", "virtio", image)

				By("Starting the VirtualMachineInstance")
				vmi = runVMIAndExpectLaunch(vmi, 180)

				By("Checking that the VirtualMachineInstance console has expected output")
				Expect(console.LoginToAlpine(vmi)).To(Succeed())

				By("Starting a Migration")
				migration := tests.NewRandomMigration(vmi.Name, vmi.Namespace)
				migrationUID := tests.RunMigrationAndExpectCompletion(virtClient, migration, tests.MigrationWaitTime)

				// check VMI, confirm migration state
				tests.ConfirmVMIPostMigration(virtClient, vmi, migrationUID)

				// delete VMI
				By("Deleting the VMI")
				Expect(virtClient.VirtualMachineInstance(vmi.Namespace).Delete(vmi.Name, &metav1.DeleteOptions{})).To(Succeed())

				By("Waiting for VMI to disappear")
				tests.WaitForVirtualMachineToDisappearWithTimeout(vmi, 120)
			})
			It("[release-blocker][test_id:1377]should be successfully migrated multiple times", func() {
				// Start the VirtualMachineInstance with the PVC attached
				vmi, _ := tests.NewRandomVirtualMachineInstanceWithOCSDisk(cd.DataVolumeImportUrlForContainerDisk(cd.ContainerDiskAlpine), util.NamespaceTestDefault, k8sv1.ReadWriteMany, k8sv1.PersistentVolumeBlock)
				vmi = runVMIAndExpectLaunch(vmi, 180)

				By("Checking that the VirtualMachineInstance console has expected output")
				Expect(console.LoginToAlpine(vmi)).To(Succeed())

				// execute a migration, wait for finalized state
				migration := tests.NewRandomMigration(vmi.Name, vmi.Namespace)
				migrationUID := tests.RunMigrationAndExpectCompletion(virtClient, migration, 180)

				// check VMI, confirm migration state
				tests.ConfirmVMIPostMigration(virtClient, vmi, migrationUID)

				// delete VMI
				By("Deleting the VMI")
				Expect(virtClient.VirtualMachineInstance(vmi.Namespace).Delete(vmi.Name, &metav1.DeleteOptions{})).To(Succeed())

				By("Waiting for VMI to disappear")
				tests.WaitForVirtualMachineToDisappearWithTimeout(vmi, 120)
			})
		})
		Context("[rook-ceph]with an Cirros shared block volume PVC", func() {

			It("[test_id:3240]should be successfully with a cloud init", func() {
				// Start the VirtualMachineInstance with the PVC attached

				vmi, _ := tests.NewRandomVirtualMachineInstanceWithOCSDisk(cd.DataVolumeImportUrlForContainerDisk(cd.ContainerDiskCirros), util.NamespaceTestDefault, k8sv1.ReadWriteMany, k8sv1.PersistentVolumeBlock)
				tests.AddUserData(vmi, "cloud-init", "#!/bin/bash\necho 'hello'\n")
				vmi.Spec.Hostname = fmt.Sprintf("%s", cd.ContainerDiskCirros)
				vmi = runVMIAndExpectLaunch(vmi, 180)

				By("Checking that the VirtualMachineInstance console has expected output")
				Expect(libnet.WithIPv6(console.LoginToCirros)(vmi)).To(Succeed())

				By("Checking that MigrationMethod is set to BlockMigration")
				Expect(vmi.Status.MigrationMethod).To(Equal(v1.BlockMigration))

				// execute a migration, wait for finalized state
				By("Starting the Migration for iteration")
				migration := tests.NewRandomMigration(vmi.Name, vmi.Namespace)
				migrationUID := tests.RunMigrationAndExpectCompletion(virtClient, migration, tests.MigrationWaitTime)

				// check VMI, confirm migration state
				tests.ConfirmVMIPostMigration(virtClient, vmi, migrationUID)

				// delete VMI
				By("Deleting the VMI")
				Expect(virtClient.VirtualMachineInstance(vmi.Namespace).Delete(vmi.Name, &metav1.DeleteOptions{})).To(Succeed())

				By("Waiting for VMI to disappear")
				tests.WaitForVirtualMachineToDisappearWithTimeout(vmi, 120)
			})
		})
		Context("with a Fedora shared NFS PVC (using nfs ipv4 address), cloud init and service account", func() {
			var vmi *v1.VirtualMachineInstance
			var dv *cdiv1.DataVolume
			var wffcPod *k8sv1.Pod

			BeforeEach(func() {
				quantity, err := resource.ParseQuantity("5Gi")
				Expect(err).ToNot(HaveOccurred())
				url := "docker://" + cd.ContainerDiskFor(cd.ContainerDiskFedoraTestTooling)
				dv = tests.NewRandomDataVolumeWithRegistryImport(url, util.NamespaceTestDefault, k8sv1.ReadWriteOnce)
				dv.Spec.PVC.Resources.Requests["storage"] = quantity
				_, err = virtClient.CdiClient().CdiV1beta1().DataVolumes(dv.Namespace).Create(context.Background(), dv, metav1.CreateOptions{})
				Expect(err).ToNot(HaveOccurred())

				wffcPod = tests.RenderPod("wffc-temp-pod", []string{"echo"}, []string{"done"})
				wffcPod.Spec.Containers[0].VolumeMounts = []k8sv1.VolumeMount{

					{
						Name:      "tmp-data",
						MountPath: "/data/tmp-data",
					},
				}
				wffcPod.Spec.Volumes = []k8sv1.Volume{
					{
						Name: "tmp-data",
						VolumeSource: k8sv1.VolumeSource{
							PersistentVolumeClaim: &k8sv1.PersistentVolumeClaimVolumeSource{
								ClaimName: dv.Name,
							},
						},
					},
				}

				By("pinning the wffc dv")
				wffcPod, err = virtClient.CoreV1().Pods(util.NamespaceTestDefault).Create(context.Background(), wffcPod, metav1.CreateOptions{})
				Expect(err).ToNot(HaveOccurred())
				Eventually(ThisPod(wffcPod), 120).Should(BeInPhase(k8sv1.PodSucceeded))

				By("waiting for the dv import to pvc to finish")
				Eventually(ThisDV(dv), 600).Should(HaveSucceeded())

				// Prepare a NFS backed PV
				By("Starting an NFS POD to serve the PVC contents")
				nfsPod := storageframework.RenderNFSServerWithPVC("nfsserver", dv.Name)
				nfsPod, err = virtClient.CoreV1().Pods(util.NamespaceTestDefault).Create(context.Background(), nfsPod, metav1.CreateOptions{})
				Expect(err).ToNot(HaveOccurred())
				Eventually(ThisPod(nfsPod), 120).Should(BeInPhase(k8sv1.PodRunning))
				nfsPod, err = ThisPod(nfsPod)()
				Expect(err).ToNot(HaveOccurred())
				nfsIP := libnet.GetPodIpByFamily(nfsPod, k8sv1.IPv4Protocol)
				Expect(nfsIP).NotTo(BeEmpty())
				// create a new PV and PVC (PVs can't be reused)
				By("create a new NFS PV and PVC")
				os := string(cd.ContainerDiskFedoraTestTooling)
				tests.CreateNFSPvAndPvc(pvName, util.NamespaceTestDefault, "5Gi", nfsIP, os)
			})

			AfterEach(func() {
				By("Deleting NFS pod")
				// PVs can't be reused
				tests.DeletePvAndPvc(pvName)

				if dv != nil {
					By("Deleting the DataVolume")
					Expect(virtClient.CdiClient().CdiV1beta1().DataVolumes(dv.Namespace).Delete(context.Background(), dv.Name, metav1.DeleteOptions{})).To(Succeed())
					dv = nil
				}
				if wffcPod != nil {
					By("Deleting the wffc pod")
					err = virtClient.CoreV1().Pods(util.NamespaceTestDefault).Delete(context.Background(), wffcPod.Name, metav1.DeleteOptions{})
					Expect(err).ToNot(HaveOccurred())
					wffcPod = nil
				}
			})

			It("[test_id:2653] should be migrated successfully, using guest agent on VM with default migration configuration", func() {
				guestAgentMigrationTestFunc(v1.MigrationPreCopy)
			})

			It("[test_id:6975] should have guest agent functional after migration", func() {
				By("Creating the  VMI")
				vmi = tests.NewRandomVMIWithPVC(pvName)
				vmi.Spec.Domain.Resources.Requests[k8sv1.ResourceMemory] = resource.MustParse(fedoraVMSize)
				vmi.Spec.Domain.Devices.Rng = &v1.Rng{}

				tests.AddUserData(vmi, "cloud-init", "#!/bin/bash\n echo hello\n")
				vmi = runVMIAndExpectLaunchIgnoreWarnings(vmi, 180)

				By("Checking guest agent")
				tests.WaitAgentConnected(virtClient, vmi)

				By("Starting the Migration for iteration")
				migration := tests.NewRandomMigration(vmi.Name, vmi.Namespace)
				_ = tests.RunMigrationAndExpectCompletion(virtClient, migration, tests.MigrationWaitTime)

				By("Agent stays connected")
				Consistently(func() error {
					updatedVmi, err := virtClient.VirtualMachineInstance(util.NamespaceTestDefault).Get(vmi.Name, &metav1.GetOptions{})
					Expect(err).ToNot(HaveOccurred())
					for _, condition := range updatedVmi.Status.Conditions {
						if condition.Type == v1.VirtualMachineInstanceAgentConnected && condition.Status == k8sv1.ConditionTrue {
							return nil
						}
					}
					return fmt.Errorf("Guest Agent Disconnected")
				}, 5*time.Minute, 10*time.Second).Should(Succeed())
			})
		})

		Context("migration security", func() {
			Context("with TLS disabled", func() {
				It("[test_id:6976] should be successfully migrated", func() {
					cfg := getCurrentKv()
					cfg.MigrationConfiguration.DisableTLS = pointer.BoolPtr(true)
					tests.UpdateKubeVirtConfigValueAndWait(cfg)

					vmi := tests.NewRandomVMIWithEphemeralDisk(cd.ContainerDiskFor(cd.ContainerDiskCirros))

					tests.AddUserData(vmi, "cloud-init", "#!/bin/bash\necho 'hello'\n")

					By("Starting the VirtualMachineInstance")
					vmi = runVMIAndExpectLaunch(vmi, 240)

					By("Checking that the VirtualMachineInstance console has expected output")
					Expect(libnet.WithIPv6(console.LoginToCirros)(vmi)).To(Succeed())

					By("starting the migration")
					migration := tests.NewRandomMigration(vmi.Name, vmi.Namespace)
					migrationUID := tests.RunMigrationAndExpectCompletion(virtClient, migration, tests.MigrationWaitTime)

					// check VMI, confirm migration state
					tests.ConfirmVMIPostMigration(virtClient, vmi, migrationUID)

					// delete VMI
					By("Deleting the VMI")
					Expect(virtClient.VirtualMachineInstance(vmi.Namespace).Delete(vmi.Name, &metav1.DeleteOptions{})).To(Succeed())

					By("Waiting for VMI to disappear")
					tests.WaitForVirtualMachineToDisappearWithTimeout(vmi, 240)
				})

				It("[test_id:6977]should not secure migrations with TLS", func() {
					cfg := getCurrentKv()
					cfg.MigrationConfiguration.BandwidthPerMigration = resource.NewMilliQuantity(1, resource.BinarySI)
					cfg.MigrationConfiguration.DisableTLS = pointer.BoolPtr(true)
					tests.UpdateKubeVirtConfigValueAndWait(cfg)
					vmi := tests.NewRandomFedoraVMIWithGuestAgent()
					vmi.Spec.Domain.Resources.Requests[k8sv1.ResourceMemory] = resource.MustParse(fedoraVMSize)

					By("Starting the VirtualMachineInstance")
					vmi = runVMIAndExpectLaunch(vmi, 240)

					// Need to wait for cloud init to finish and start the agent inside the vmi.
					tests.WaitAgentConnected(virtClient, vmi)

					// Run
					Expect(console.LoginToFedora(vmi)).To(Succeed())

					runStressTest(vmi, stressDefaultVMSize, stressDefaultTimeout)

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
					tlsConfig := temporaryTLSConfig()

					handler, err := kubecli.NewVirtHandlerClient(virtClient).Namespace(flags.KubeVirtInstallNamespace).ForNode(vmi.Status.MigrationState.TargetNode).Pod()
					Expect(err).ToNot(HaveOccurred())

					var wg sync.WaitGroup
					wg.Add(len(vmi.Status.MigrationState.TargetDirectMigrationNodePorts))

					i := 0
					errors := make(chan error, len(vmi.Status.MigrationState.TargetDirectMigrationNodePorts))
					for port := range vmi.Status.MigrationState.TargetDirectMigrationNodePorts {
						portI, _ := strconv.Atoi(port)
						go func(i int, port int) {
							defer GinkgoRecover()
							defer wg.Done()
							stopChan := make(chan struct{})
							defer close(stopChan)
							Expect(tests.ForwardPorts(handler, []string{fmt.Sprintf("4321%d:%d", i, port)}, stopChan, 10*time.Second)).To(Succeed())
							_, err := tls.Dial("tcp", fmt.Sprintf("localhost:4321%d", i), tlsConfig)
							Expect(err).To(HaveOccurred())
							errors <- err
						}(i, portI)
						i++
					}
					wg.Wait()
					close(errors)

					By("checking that we were never able to connect")
					for err := range errors {
						Expect(err.Error()).To(Or(ContainSubstring("EOF"), ContainSubstring("first record does not look like a TLS handshake")))
					}
				})
			})
			Context("with TLS enabled", func() {
				BeforeEach(func() {
					cfg := getCurrentKv()
					cfg.MigrationConfiguration.BandwidthPerMigration = resource.NewMilliQuantity(1, resource.BinarySI)
					tests.UpdateKubeVirtConfigValueAndWait(cfg)
				})

				It("[test_id:2303][posneg:negative] should secure migrations with TLS", func() {
					vmi := tests.NewRandomFedoraVMIWithGuestAgent()
					vmi.Spec.Domain.Resources.Requests[k8sv1.ResourceMemory] = resource.MustParse(fedoraVMSize)

					By("Starting the VirtualMachineInstance")
					vmi = runVMIAndExpectLaunch(vmi, 240)

					// Need to wait for cloud init to finish and start the agent inside the vmi.
					tests.WaitAgentConnected(virtClient, vmi)

					// Run
					Expect(console.LoginToFedora(vmi)).To(Succeed())

					runStressTest(vmi, stressDefaultVMSize, stressDefaultTimeout)

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
					tlsConfig := temporaryTLSConfig()

					handler, err := kubecli.NewVirtHandlerClient(virtClient).Namespace(flags.KubeVirtInstallNamespace).ForNode(vmi.Status.MigrationState.TargetNode).Pod()
					Expect(err).ToNot(HaveOccurred())

					var wg sync.WaitGroup
					wg.Add(len(vmi.Status.MigrationState.TargetDirectMigrationNodePorts))

					i := 0
					errors := make(chan error, len(vmi.Status.MigrationState.TargetDirectMigrationNodePorts))
					for port := range vmi.Status.MigrationState.TargetDirectMigrationNodePorts {
						portI, _ := strconv.Atoi(port)
						go func(i int, port int) {
							defer GinkgoRecover()
							defer wg.Done()
							stopChan := make(chan struct{})
							defer close(stopChan)
							Expect(tests.ForwardPorts(handler, []string{fmt.Sprintf("4321%d:%d", i, port)}, stopChan, 10*time.Second)).To(Succeed())
							conn, err := tls.Dial("tcp", fmt.Sprintf("localhost:4321%d", i), tlsConfig)
							if conn != nil {
								b := make([]byte, 1)
								_, err = conn.Read(b)
							}
							Expect(err).To(HaveOccurred())
							errors <- err
						}(i, portI)
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
		})

		Context("migration postcopy", func() {

			var dv *cdiv1.DataVolume
			var wffcPod *k8sv1.Pod

			BeforeEach(func() {
				By("Limit migration bandwidth")
				setMigrationBandwidthLimitation(resource.MustParse("40Mi"))

				By("Allowing post-copy")
				config := getCurrentKv()
				config.MigrationConfiguration.AllowPostCopy = pointer.BoolPtr(true)
				config.MigrationConfiguration.CompletionTimeoutPerGiB = pointer.Int64Ptr(1)
				tests.UpdateKubeVirtConfigValueAndWait(config)
				memoryRequestSize = resource.MustParse("1Gi")

				quantity, err := resource.ParseQuantity("5Gi")
				Expect(err).ToNot(HaveOccurred())
				url := "docker://" + cd.ContainerDiskFor(cd.ContainerDiskFedoraTestTooling)
				dv := tests.NewRandomDataVolumeWithRegistryImport(url, util.NamespaceTestDefault, k8sv1.ReadWriteOnce)
				dv.Spec.PVC.Resources.Requests["storage"] = quantity
				_, err = virtClient.CdiClient().CdiV1beta1().DataVolumes(dv.Namespace).Create(context.Background(), dv, metav1.CreateOptions{})
				Expect(err).ToNot(HaveOccurred())

				wffcPod = tests.RenderPod("wffc-temp-pod", []string{"echo"}, []string{"done"})
				wffcPod.Spec.Containers[0].VolumeMounts = []k8sv1.VolumeMount{

					{
						Name:      "tmp-data",
						MountPath: "/data/tmp-data",
					},
				}
				wffcPod.Spec.Volumes = []k8sv1.Volume{
					{
						Name: "tmp-data",
						VolumeSource: k8sv1.VolumeSource{
							PersistentVolumeClaim: &k8sv1.PersistentVolumeClaimVolumeSource{
								ClaimName: dv.Name,
							},
						},
					},
				}

				By("pinning the wffc dv")
				wffcPod, err = virtClient.CoreV1().Pods(util.NamespaceTestDefault).Create(context.Background(), wffcPod, metav1.CreateOptions{})
				Expect(err).ToNot(HaveOccurred())
				Eventually(ThisPod(wffcPod), 120).Should(BeInPhase(k8sv1.PodSucceeded))

				By("waiting for the dv import to pvc to finish")
				Eventually(ThisDV(dv), 600).Should(HaveSucceeded())

				// Prepare a NFS backed PV
				By("Starting an NFS POD to serve the PVC contents")
				nfsPod := storageframework.RenderNFSServerWithPVC("nfsserver", dv.Name)
				nfsPod, err = virtClient.CoreV1().Pods(util.NamespaceTestDefault).Create(context.Background(), nfsPod, metav1.CreateOptions{})
				Expect(err).ToNot(HaveOccurred())
				Eventually(ThisPod(nfsPod), 120).Should(BeInPhase(k8sv1.PodRunning))
				nfsPod, err = ThisPod(nfsPod)()
				Expect(err).ToNot(HaveOccurred())
				nfsIP := libnet.GetPodIpByFamily(nfsPod, k8sv1.IPv4Protocol)
				Expect(nfsIP).NotTo(BeEmpty())
				// create a new PV and PVC (PVs can't be reused)
				By("create a new NFS PV and PVC")
				os := string(cd.ContainerDiskFedoraTestTooling)
				tests.CreateNFSPvAndPvc(pvName, util.NamespaceTestDefault, "5Gi", nfsIP, os)
			})

			AfterEach(func() {
				By("Deleting NFS pod")
				// PVs can't be reused
				tests.DeletePvAndPvc(pvName)

				if dv != nil {
					By("Deleting the DataVolume")
					Expect(virtClient.CdiClient().CdiV1beta1().DataVolumes(dv.Namespace).Delete(context.Background(), dv.Name, metav1.DeleteOptions{})).To(Succeed())
					dv = nil
				}
				if wffcPod != nil {
					By("Deleting the wffc pod")
					err = virtClient.CoreV1().Pods(util.NamespaceTestDefault).Delete(context.Background(), wffcPod.Name, metav1.DeleteOptions{})
					Expect(err).ToNot(HaveOccurred())
					wffcPod = nil
				}
			})

			It("[QUARANTINE][test_id:5004] should be migrated successfully, using guest agent on VM with postcopy", func() {
				guestAgentMigrationTestFunc(v1.MigrationPostCopy)
			})

			It("[QUARANTINE][test_id:4747] should migrate using cluster level config for postcopy", func() {
				config := getCurrentKv()
				config.MigrationConfiguration.AllowPostCopy = pointer.BoolPtr(true)
				config.MigrationConfiguration.CompletionTimeoutPerGiB = pointer.Int64Ptr(1)
				bandwidth := resource.MustParse("256Mi")
				config.MigrationConfiguration.BandwidthPerMigration = &bandwidth
				tests.UpdateKubeVirtConfigValueAndWait(config)

				vmi := tests.NewRandomFedoraVMIWithGuestAgent()
				vmi.Spec.Domain.Resources.Requests[k8sv1.ResourceMemory] = resource.MustParse("1Gi")

				By("Starting the VirtualMachineInstance")
				vmi = runVMIAndExpectLaunch(vmi, 240)

				By("Checking that the VirtualMachineInstance console has expected output")
				Expect(console.LoginToFedora(vmi)).To(Succeed())

				// Need to wait for cloud init to finish and start the agent inside the vmi.
				tests.WaitAgentConnected(virtClient, vmi)

				runStressTest(vmi, stressLargeVMSize, stressDefaultTimeout)

				// execute a migration, wait for finalized state
				By("Starting the Migration")
				migration := tests.NewRandomMigration(vmi.Name, vmi.Namespace)
				migrationUID := tests.RunMigrationAndExpectCompletion(virtClient, migration, 180)

				// check VMI, confirm migration state
				tests.ConfirmVMIPostMigration(virtClient, vmi, migrationUID)
				confirmMigrationMode(vmi, v1.MigrationPostCopy)

				// delete VMI
				By("Deleting the VMI")
				Expect(virtClient.VirtualMachineInstance(vmi.Namespace).Delete(vmi.Name, &metav1.DeleteOptions{})).To(Succeed())

				By("Waiting for VMI to disappear")
				tests.WaitForVirtualMachineToDisappearWithTimeout(vmi, 240)
			})
		})

		Context("migration monitor", func() {
			var createdPods []string
			AfterEach(func() {
				for _, podName := range createdPods {
					Eventually(func() error {
						err := virtClient.CoreV1().Pods(util.NamespaceTestDefault).Delete(context.Background(), podName, metav1.DeleteOptions{})

						if err != nil && errors.IsNotFound(err) {
							return nil
						}
						return err
					}, 10*time.Second, 1*time.Second).Should(Succeed(), "Should delete helper pod")
				}
			})
			BeforeEach(func() {
				createdPods = []string{}
				cfg := getCurrentKv()
				var timeout int64 = 5
				cfg.MigrationConfiguration = &v1.MigrationConfiguration{
					ProgressTimeout:         &timeout,
					CompletionTimeoutPerGiB: &timeout,
					BandwidthPerMigration:   resource.NewMilliQuantity(1, resource.BinarySI),
				}
				tests.UpdateKubeVirtConfigValueAndWait(cfg)
			})
			PIt("[test_id:2227] should abort a vmi migration without progress", func() {
				vmi := tests.NewRandomFedoraVMIWithGuestAgent()
				vmi.Spec.Domain.Resources.Requests[k8sv1.ResourceMemory] = resource.MustParse("1Gi")

				By("Starting the VirtualMachineInstance")
				vmi = runVMIAndExpectLaunch(vmi, 240)

				By("Checking that the VirtualMachineInstance console has expected output")
				Expect(console.LoginToFedora(vmi)).To(Succeed())

				// Need to wait for cloud init to finish and start the agent inside the vmi.
				tests.WaitAgentConnected(virtClient, vmi)

				runStressTest(vmi, stressLargeVMSize, stressDefaultTimeout)

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

			It("[test_id:6978][QUARANTINE] Should detect a failed migration", func() {
				vmi := tests.NewRandomFedoraVMIWithGuestAgent()
				vmi.Spec.Domain.Resources.Requests[k8sv1.ResourceMemory] = resource.MustParse("1Gi")

				By("Starting the VirtualMachineInstance")
				vmi = runVMIAndExpectLaunch(vmi, 240)

				By("Checking that the VirtualMachineInstance console has expected output")
				Expect(console.LoginToFedora(vmi)).To(Succeed())

				domSpec, err := tests.GetRunningVMIDomainSpec(vmi)
				Expect(err).ToNot(HaveOccurred())
				emulator := filepath.Base(strings.TrimPrefix(domSpec.Devices.Emulator, "/"))
				// ensure that we only match the process
				emulator = "[" + emulator[0:1] + "]" + emulator[1:]

				// launch killer pod on every node that isn't the vmi's node
				By("Starting our migration killer pods")
				nodes := util.GetAllSchedulableNodes(virtClient)
				Expect(nodes.Items).ToNot(BeEmpty(), "There should be some compute node")
				for idx, entry := range nodes.Items {
					if entry.Name == vmi.Status.NodeName {
						continue
					}

					podName := fmt.Sprintf("migration-killer-pod-%d", idx)

					// kill the handler right as we detect the qemu target process come online
					pod := tests.RenderPrivilegedPod(podName, []string{"/bin/bash", "-c"}, []string{fmt.Sprintf("while true; do ps aux | grep \"%s\" && pkill -9 virt-handler && sleep 5; done", emulator)})

					pod.Spec.NodeName = entry.Name
					createdPod, err := virtClient.CoreV1().Pods(util.NamespaceTestDefault).Create(context.Background(), pod, metav1.CreateOptions{})
					Expect(err).ToNot(HaveOccurred(), "Should create helper pod")
					createdPods = append(createdPods, createdPod.Name)
				}
				Expect(len(createdPods)).To(BeNumerically(">=", 1), "There is no node for migration")

				// execute a migration, wait for finalized state
				By("Starting the Migration")
				migration := tests.NewRandomMigration(vmi.Name, vmi.Namespace)
				migrationUID := runMigrationAndExpectFailure(migration, 180)

				// check VMI, confirm migration state
				confirmVMIPostMigrationFailed(vmi, migrationUID)

				By("Removing our migration killer pods")
				for _, podName := range createdPods {
					Eventually(func() error {
						err := virtClient.CoreV1().Pods(util.NamespaceTestDefault).Delete(context.Background(), podName, metav1.DeleteOptions{})

						if err != nil && errors.IsNotFound(err) {
							return nil
						}
						return err
					}, 10*time.Second, 1*time.Second).Should(Succeed(), "Should delete helper pod")

					Eventually(func() error {
						_, err := virtClient.CoreV1().Pods(util.NamespaceTestDefault).Get(context.Background(), podName, metav1.GetOptions{})
						return err
					}, 300*time.Second, 1*time.Second).Should(
						SatisfyAll(HaveOccurred(), WithTransform(errors.IsNotFound, BeTrue())),
						"The killer pod should be gone within the given timeout",
					)
				}

				By("Waiting for virt-handler to come back online")
				Eventually(func() error {
					handler, err := virtClient.AppsV1().DaemonSets(flags.KubeVirtInstallNamespace).Get(context.Background(), "virt-handler", metav1.GetOptions{})
					if err != nil {
						return err
					}

					if handler.Status.DesiredNumberScheduled == handler.Status.NumberAvailable {
						return nil
					}
					return fmt.Errorf("waiting for virt-handler pod to come back online")
				}, 120*time.Second, 1*time.Second).Should(Succeed(), "Virt handler should come online")

				By("Starting new migration and waiting for it to succeed")
				migration = tests.NewRandomMigration(vmi.Name, vmi.Namespace)
				migrationUID = tests.RunMigrationAndExpectCompletion(virtClient, migration, 340)

				By("Verifying Second Migration Succeeeds")
				tests.ConfirmVMIPostMigration(virtClient, vmi, migrationUID)

				// delete VMI
				By("Deleting the VMI")
				Expect(virtClient.VirtualMachineInstance(vmi.Namespace).Delete(vmi.Name, &metav1.DeleteOptions{})).To(Succeed())

				By("Waiting for VMI to disappear")
				tests.WaitForVirtualMachineToDisappearWithTimeout(vmi, 240)
			})

			It("[test_id:6979]Target pod should exit after failed migration", func() {
				vmi := tests.NewRandomFedoraVMIWithGuestAgent()
				vmi.Spec.Domain.Resources.Requests[k8sv1.ResourceMemory] = resource.MustParse("1Gi")

				// this annotation causes virt launcher to immediately fail a migration
				vmi.Annotations = map[string]string{v1.FuncTestForceLauncherMigrationFailureAnnotation: ""}

				By("Starting the VirtualMachineInstance")
				vmi = runVMIAndExpectLaunch(vmi, 240)

				// execute a migration, wait for finalized state
				By("Starting the Migration")
				migration := tests.NewRandomMigration(vmi.Name, vmi.Namespace)
				migrationUID := runMigrationAndExpectFailure(migration, 180)

				// check VMI, confirm migration state
				confirmVMIPostMigrationFailed(vmi, migrationUID)

				Eventually(func() error {
					vmi, err = virtClient.VirtualMachineInstance(vmi.Namespace).Get(vmi.Name, &metav1.GetOptions{})
					Expect(err).ToNot(HaveOccurred())

					pod, err := virtClient.CoreV1().Pods(vmi.Namespace).Get(context.Background(), vmi.Status.MigrationState.TargetPod, metav1.GetOptions{})
					Expect(err).ToNot(HaveOccurred())

					if pod.Status.Phase == k8sv1.PodFailed || pod.Status.Phase == k8sv1.PodSucceeded {
						return nil
					}

					return fmt.Errorf("still waiting on target pod to complete, current phase is %s", pod.Status.Phase)
				}, 10*time.Second, time.Second).Should(Succeed(), "Target pod should exit quickly after migration fails.")

				// delete VMI
				By("Deleting the VMI")
				Expect(virtClient.VirtualMachineInstance(vmi.Namespace).Delete(vmi.Name, &metav1.DeleteOptions{})).To(Succeed())

				By("Waiting for VMI to disappear")
				tests.WaitForVirtualMachineToDisappearWithTimeout(vmi, 240)
			})

			It("[test_id:6980]Migration should fail if target pod fails during target preparation", func() {
				vmi := tests.NewRandomFedoraVMIWithGuestAgent()
				vmi.Spec.Domain.Resources.Requests[k8sv1.ResourceMemory] = resource.MustParse("1Gi")

				// this annotation causes virt launcher to immediately fail a migration
				vmi.Annotations = map[string]string{v1.FuncTestBlockLauncherPrepareMigrationTargetAnnotation: ""}

				By("Starting the VirtualMachineInstance")
				vmi = runVMIAndExpectLaunch(vmi, 240)

				// execute a migration
				By("Starting the Migration")
				migration := tests.NewRandomMigration(vmi.Name, vmi.Namespace)
				migration, err = virtClient.VirtualMachineInstanceMigration(migration.Namespace).Create(migration)
				Expect(err).ToNot(HaveOccurred())

				By("Waiting for Migration to reach Preparing Target Phase")
				Eventually(func() v1.VirtualMachineInstanceMigrationPhase {
					migration, err = virtClient.VirtualMachineInstanceMigration(migration.Namespace).Get(migration.Name, &metav1.GetOptions{})
					Expect(err).ToNot(HaveOccurred())

					phase := migration.Status.Phase
					Expect(phase).NotTo(Equal(v1.MigrationSucceeded))
					return phase
				}, 120, 1*time.Second).Should(Equal(v1.MigrationPreparingTarget))

				By("Killing the target pod and expecting failure")
				vmi, err = virtClient.VirtualMachineInstance(vmi.Namespace).Get(vmi.Name, &metav1.GetOptions{})
				Expect(err).ToNot(HaveOccurred())
				Expect(vmi.Status.MigrationState).ToNot(BeNil())
				Expect(vmi.Status.MigrationState.TargetPod).ToNot(Equal(""))

				err = virtClient.CoreV1().Pods(vmi.Namespace).Delete(context.Background(), vmi.Status.MigrationState.TargetPod, metav1.DeleteOptions{})
				Expect(err).ToNot(HaveOccurred())

				By("Expecting VMI migration failure")
				Eventually(func() error {
					vmi, err = virtClient.VirtualMachineInstance(vmi.Namespace).Get(vmi.Name, &metav1.GetOptions{})
					Expect(err).ToNot(HaveOccurred())
					vmi, err = virtClient.VirtualMachineInstance(vmi.Namespace).Get(vmi.Name, &metav1.GetOptions{})
					Expect(err).ToNot(HaveOccurred())
					Expect(vmi.Status.MigrationState).ToNot(BeNil())

					if !vmi.Status.MigrationState.Failed {
						return fmt.Errorf("Waiting on vmi's migration state to be marked as failed")
					}

					// once set to failed, we expect start and end times and completion to be set as well.
					Expect(vmi.Status.MigrationState.StartTimestamp).ToNot(BeNil())
					Expect(vmi.Status.MigrationState.EndTimestamp).ToNot(BeNil())
					Expect(vmi.Status.MigrationState.Completed).To(BeTrue())

					return nil
				}, 120*time.Second, time.Second).Should(Succeed(), "vmi's migration state should be finalized as failed after target pod exits")

				// delete VMI
				By("Deleting the VMI")
				Expect(virtClient.VirtualMachineInstance(vmi.Namespace).Delete(vmi.Name, &metav1.DeleteOptions{})).To(Succeed())

				By("Waiting for VMI to disappear")
				tests.WaitForVirtualMachineToDisappearWithTimeout(vmi, 240)
			})
			It("Migration should generate empty isos of the right size on the target", func() {
				By("Creating a VMI with cloud-init and config maps")
				vmi := tests.NewRandomVMIWithEphemeralDisk(cd.ContainerDiskFor(cd.ContainerDiskCirros))
				configMapName := "configmap-" + rand.String(5)
				secretName := "secret-" + rand.String(5)
				downwardAPIName := "downwardapi-" + rand.String(5)
				config_data := map[string]string{
					"config1": "value1",
					"config2": "value2",
				}
				secret_data := map[string]string{
					"user":     "admin",
					"password": "community",
				}
				tests.CreateConfigMap(configMapName, config_data)
				tests.CreateSecret(secretName, secret_data)
				tests.AddUserData(vmi, "cloud-init", "#!/bin/bash\necho 'hello'\n")
				tests.AddConfigMapDisk(vmi, configMapName, configMapName)
				tests.AddSecretDisk(vmi, secretName, secretName)
				tests.AddServiceAccountDisk(vmi, "default")
				// In case there are no existing labels add labels to add some data to the downwardAPI disk
				if vmi.ObjectMeta.Labels == nil {
					vmi.ObjectMeta.Labels = map[string]string{"downwardTestLabelKey": "downwardTestLabelVal"}
				}
				tests.AddLabelDownwardAPIVolume(vmi, downwardAPIName)

				// this annotation causes virt launcher to immediately fail a migration
				vmi.Annotations = map[string]string{v1.FuncTestBlockLauncherPrepareMigrationTargetAnnotation: ""}

				By("Starting the VirtualMachineInstance")
				vmi = runVMIAndExpectLaunch(vmi, 240)

				// execute a migration
				By("Starting the Migration")
				migration := tests.NewRandomMigration(vmi.Name, vmi.Namespace)
				migration, err = virtClient.VirtualMachineInstanceMigration(migration.Namespace).Create(migration)
				Expect(err).ToNot(HaveOccurred())

				By("Waiting for Migration to reach Preparing Target Phase")
				Eventually(func() v1.VirtualMachineInstanceMigrationPhase {
					migration, err = virtClient.VirtualMachineInstanceMigration(migration.Namespace).Get(migration.Name, &metav1.GetOptions{})
					Expect(err).ToNot(HaveOccurred())

					phase := migration.Status.Phase
					Expect(phase).NotTo(Equal(v1.MigrationSucceeded))
					return phase
				}, 120, 1*time.Second).Should(Equal(v1.MigrationPreparingTarget))

				vmi, err = virtClient.VirtualMachineInstance(vmi.Namespace).Get(vmi.Name, &metav1.GetOptions{})
				Expect(err).ToNot(HaveOccurred())
				Expect(vmi.Status.MigrationState).ToNot(BeNil())
				Expect(vmi.Status.MigrationState.TargetPod).ToNot(Equal(""))

				By("Sanity checking the volume status size and the actual virt-launcher file")
				for _, volume := range vmi.Spec.Volumes {
					for _, volType := range []string{"cloud-init", "configmap-", "default-", "downwardapi-", "secret-"} {
						if strings.HasPrefix(volume.Name, volType) {
							for _, volStatus := range vmi.Status.VolumeStatus {
								if volStatus.Name == volume.Name {
									Expect(volStatus.Size).To(BeNumerically(">", 0), "Size of volume %s is 0", volume.Name)
									volPath, found := virthandler.IsoGuestVolumePath(vmi, &volume)
									if !found {
										continue
									}
									// Wait for the iso to be created
									Eventually(func() string {
										output, err := tests.RunCommandOnVmiTargetPod(vmi, []string{"/bin/bash", "-c", "[[ -f " + volPath + " ]] && echo found || true"})
										Expect(err).ToNot(HaveOccurred())
										return output
									}, 30*time.Second, time.Second).Should(ContainSubstring("found"), volPath+" never appeared")
									output, err := tests.RunCommandOnVmiTargetPod(vmi, []string{"/bin/bash", "-c", "/usr/bin/stat --printf=%s " + volPath})
									Expect(err).ToNot(HaveOccurred())
									Expect(strconv.Atoi(output)).To(Equal(int(volStatus.Size)), "ISO file for volume %s is not the right size", volume.Name)
									output, err = tests.RunCommandOnVmiTargetPod(vmi, []string{"/bin/bash", "-c", fmt.Sprintf(`/usr/bin/cmp -n %d %s /dev/zero || true`, volStatus.Size, volPath)})
									Expect(err).ToNot(HaveOccurred())
									Expect(output).ToNot(ContainSubstring("differ"), "ISO file for volume %s is not empty", volume.Name)
								}
							}
						}
					}
				}

				By("Deleting the VMI")
				Expect(virtClient.VirtualMachineInstance(vmi.Namespace).Delete(vmi.Name, &metav1.DeleteOptions{})).To(Succeed())

				By("Waiting for VMI to disappear")
				tests.WaitForVirtualMachineToDisappearWithTimeout(vmi, 240)
			})
		})
		Context("[rook-ceph]with an Cirros non-shared block volume PVC", func() {

			It("[test_id:1862][posneg:negative]should reject migrations for a non-migratable vmi", func() {
				// Start the VirtualMachineInstance with the PVC attached

				vmi, _ := tests.NewRandomVirtualMachineInstanceWithOCSDisk(cd.DataVolumeImportUrlForContainerDisk(cd.ContainerDiskCirros), util.NamespaceTestDefault, k8sv1.ReadWriteOnce, k8sv1.PersistentVolumeBlock)
				tests.AddUserData(vmi, "cloud-init", "#!/bin/bash\necho 'hello'\n")
				vmi.Spec.Hostname = string(cd.ContainerDiskCirros)
				vmi = runVMIAndExpectLaunch(vmi, 180)

				By("Checking that the VirtualMachineInstance console has expected output")
				Expect(libnet.WithIPv6(console.LoginToCirros)(vmi)).To(Succeed())

				gotExpectedCondition := false
				for _, c := range vmi.Status.Conditions {
					if c.Type == v1.VirtualMachineInstanceIsMigratable {
						Expect(c.Status).To(Equal(k8sv1.ConditionFalse))
						gotExpectedCondition = true
					}
				}
				Expect(gotExpectedCondition).Should(BeTrue())

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
				return tests.NewRandomFedoraVMIWithGuestAgent(), nil
			}

			newVirtualMachineInstanceWithFedoraOCSDisk := func() (*v1.VirtualMachineInstance, *cdiv1.DataVolume) {
				if !tests.HasCDI() {
					Skip("Skip DataVolume tests when CDI is not present")
				}
				sc, exists := tests.GetCephStorageClass()
				if !exists {
					Skip("Skip OCS tests when Ceph is not present")
				}

				quantity, err := resource.ParseQuantity("5Gi")
				Expect(err).ToNot(HaveOccurred())

				volMode := k8sv1.PersistentVolumeBlock
				url := "docker://" + cd.ContainerDiskFor(cd.ContainerDiskFedoraTestTooling)
				dv := tests.NewRandomDataVolumeWithRegistryImport(url, util.NamespaceTestDefault, k8sv1.ReadWriteMany)
				dv.Spec.PVC.StorageClassName = &sc
				dv.Spec.PVC.Resources.Requests["storage"] = quantity
				dv.Spec.PVC.VolumeMode = &volMode

				_, err = virtClient.CdiClient().CdiV1beta1().DataVolumes(dv.Namespace).Create(context.Background(), dv, metav1.CreateOptions{})
				Expect(err).ToNot(HaveOccurred())
				Eventually(ThisDV(dv), 600).Should(HaveSucceeded())
				vmi := tests.NewRandomVMIWithDataVolume(dv.Name)
				tests.AddUserData(vmi, "disk1", "#!/bin/bash\n echo hello\n")
				return vmi, dv
			}

			table.DescribeTable("should be able to cancel a migration", func(createVMI vmiBuilder) {
				vmi, dv := createVMI()
				vmi.Spec.Domain.Resources.Requests[k8sv1.ResourceMemory] = resource.MustParse(fedoraVMSize)
				defer deleteDataVolume(dv)

				By("Starting the VirtualMachineInstance")
				vmi = runVMIAndExpectLaunch(vmi, 240)

				By("Checking that the VirtualMachineInstance console has expected output")
				Expect(libnet.WithIPv6(console.LoginToFedora)(vmi)).To(Succeed())

				// Need to wait for cloud init to finish and start the agent inside the vmi.
				tests.WaitAgentConnected(virtClient, vmi)

				runStressTest(vmi, stressDefaultVMSize, stressDefaultTimeout)

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
				table.Entry("[sig-storage][test_id:2226] with ContainerDisk", newVirtualMachineInstanceWithFedoraContainerDisk),
				table.Entry("[sig-storage][rook-ceph][test_id:2731] with OCS disk from block volume PVC", newVirtualMachineInstanceWithFedoraOCSDisk),
			)
			It("[sig-compute][test_id:3241]should be able to cancel a migration right after posting it", func() {
				vmi := tests.NewRandomFedoraVMIWithGuestAgent()
				vmi.Spec.Domain.Resources.Requests[k8sv1.ResourceMemory] = resource.MustParse(fedoraVMSize)

				By("Starting the VirtualMachineInstance")
				vmi = runVMIAndExpectLaunch(vmi, 240)

				By("Checking that the VirtualMachineInstance console has expected output")
				Expect(console.LoginToFedora(vmi)).To(Succeed())

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

		Context("with a host-model cpu", func() {
			getNodeHostModel := func(node *k8sv1.Node) (hostModel string) {
				for key, _ := range node.Labels {
					if strings.HasPrefix(key, v1.HostModelCPULabel) {
						hostModel = strings.TrimPrefix(key, v1.HostModelCPULabel)
						break
					}
				}
				Expect(hostModel).ToNot(BeEmpty(), "must find node's host model")
				return hostModel
			}
			getNodeHostRequiredFeatures := func(node *k8sv1.Node) (features []string) {
				for key, _ := range node.Labels {
					if strings.HasPrefix(key, v1.HostModelRequiredFeaturesLabel) {
						features = append(features, strings.TrimPrefix(key, v1.HostModelRequiredFeaturesLabel))
					}
				}
				return features
			}
			isModelSupportedOnNode := func(node *k8sv1.Node, model string) bool {
				for key, _ := range node.Labels {
					if strings.HasPrefix(key, v1.HostModelCPULabel) && strings.Contains(key, model) {
						return true
					}
				}
				return false
			}
			expectFeatureToBeSupportedOnNode := func(node *k8sv1.Node, features []string) {
				isFeatureSupported := func(feature string) bool {
					for key, _ := range node.Labels {
						if strings.HasPrefix(key, v1.CPUFeatureLabel) && strings.Contains(key, feature) {
							return true
						}
					}
					return false
				}

				supportedFeatures := make(map[string]bool)
				for _, feature := range features {
					supportedFeatures[feature] = isFeatureSupported(feature)
				}

				Expect(supportedFeatures).Should(Not(ContainElement(false)),
					"copy features must be supported on node")
			}
			getOtherNodes := func(nodeList *k8sv1.NodeList, node *k8sv1.Node) (others []*k8sv1.Node) {
				for _, curNode := range nodeList.Items {
					if curNode.Name != node.Name {
						others = append(others, &curNode)
					}
				}
				return others
			}
			isHeterogeneousCluster := func() bool {
				nodes := util.GetAllSchedulableNodes(virtClient)
				for _, node := range nodes.Items {
					hostModel := getNodeHostModel(&node)
					otherNodes := getOtherNodes(nodes, &node)

					foundSupportedNode := false
					foundUnsupportedNode := false
					for _, otherNode := range otherNodes {
						if isModelSupportedOnNode(otherNode, hostModel) {
							foundSupportedNode = true
						} else {
							foundUnsupportedNode = true
						}

						if foundSupportedNode && foundUnsupportedNode {
							return true
						}
					}
				}

				return false
			}

			It("[test_id:6981]should migrate only to nodes supporting right cpu model", func() {
				if !isHeterogeneousCluster() {
					log.Log.Warning("all nodes have the same CPU model. Therefore the test is a happy-path since " +
						"VMIs with default CPU can be migrated to every other node")
				}

				By("Creating a VMI with default CPU mode")
				vmi := cirrosVMIWithEvictionStrategy()

				if cpu := vmi.Spec.Domain.CPU; cpu != nil && cpu.Model != v1.CPUModeHostModel {
					log.Log.Warning("test is not expected to pass with CPU model other than host-model")
				}

				By("Starting the VirtualMachineInstance")
				vmi = runVMIAndExpectLaunch(vmi, 240)

				By("Fetching original host CPU model & supported CPU features")
				originalNode, err := virtClient.CoreV1().Nodes().Get(context.Background(), vmi.Status.NodeName, metav1.GetOptions{})
				Expect(err).ToNot(HaveOccurred())

				hostModel := getNodeHostModel(originalNode)
				requiredFeatures := getNodeHostRequiredFeatures(originalNode)

				By("Starting the migration and expecting it to end successfully")
				migration := tests.NewRandomMigration(vmi.Name, vmi.Namespace)
				_ = tests.RunMigrationAndExpectCompletion(virtClient, migration, tests.MigrationWaitTime)

				By("Ensuring that target pod has correct nodeSelector label")
				vmiPod := tests.GetRunningPodByVirtualMachineInstance(vmi, vmi.Namespace)
				Expect(vmiPod.Spec.NodeSelector).To(HaveKey(v1.HostModelCPULabel+hostModel),
					"target pod is expected to have correct nodeSelector label defined")

				By("Ensuring that target node has correct CPU mode & features")
				newNode, err := virtClient.CoreV1().Nodes().Get(context.Background(), vmi.Status.NodeName, metav1.GetOptions{})
				Expect(err).ToNot(HaveOccurred())
				Expect(isModelSupportedOnNode(newNode, hostModel)).To(BeTrue(), "original host model should be supported on new node")
				expectFeatureToBeSupportedOnNode(newNode, requiredFeatures)
			})
		})
	})

	Context("with sata disks", func() {

		addKernelBootContainer := func(vmi *v1.VirtualMachineInstance) {
			kernelBootFirmware := utils.GetVMIKernelBoot().Spec.Domain.Firmware
			if vmiFirmware := vmi.Spec.Domain.Firmware; vmiFirmware == nil {
				vmiFirmware = kernelBootFirmware
			} else {
				vmiFirmware.KernelBoot = kernelBootFirmware.KernelBoot
			}
		}

		It("[test_id:1853]VM with containerDisk + CloudInit + ServiceAccount + ConfigMap + Secret + DownwardAPI + External Kernel Boot", func() {
			configMapName := "configmap-" + rand.String(5)
			secretName := "secret-" + rand.String(5)
			downwardAPIName := "downwardapi-" + rand.String(5)

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

			vmi := libvmi.NewTestToolingFedora(
				libvmi.WithNetwork(v1.DefaultPodNetwork()),
				libvmi.WithInterface(libvmi.InterfaceDeviceWithMasqueradeBinding()),
			)
			tests.AddUserData(vmi, "cloud-init", "#!/bin/bash\necho 'hello'\n")
			tests.AddConfigMapDisk(vmi, configMapName, configMapName)
			tests.AddSecretDisk(vmi, secretName, secretName)
			tests.AddServiceAccountDisk(vmi, "default")
			addKernelBootContainer(vmi)

			// In case there are no existing labels add labels to add some data to the downwardAPI disk
			if vmi.ObjectMeta.Labels == nil {
				vmi.ObjectMeta.Labels = map[string]string{"downwardTestLabelKey": "downwardTestLabelVal"}
			}
			tests.AddLabelDownwardAPIVolume(vmi, downwardAPIName)

			Expect(len(vmi.Spec.Domain.Devices.Disks)).To(Equal(6))
			Expect(len(vmi.Spec.Domain.Devices.Interfaces)).To(Equal(1))

			vmi = runVMIAndExpectLaunch(vmi, 180)

			// execute a migration, wait for finalized state
			By("Starting the Migration")
			migration := tests.NewRandomMigration(vmi.Name, vmi.Namespace)
			migrationUID := tests.RunMigrationAndExpectCompletion(virtClient, migration, tests.MigrationWaitTime)

			// check VMI, confirm migration state
			tests.ConfirmVMIPostMigration(virtClient, vmi, migrationUID)

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

			It("[test_id:3242]should block the eviction api and migrate", func() {
				vmi = runVMIAndExpectLaunch(vmi, 180)
				vmiNodeOrig := vmi.Status.NodeName
				pod := tests.GetRunningPodByVirtualMachineInstance(vmi, vmi.Namespace)
				err := virtClient.CoreV1().Pods(vmi.Namespace).Evict(context.Background(), &v1beta1.Eviction{ObjectMeta: metav1.ObjectMeta{Name: pod.Name}})
				Expect(errors.IsTooManyRequests(err)).To(BeTrue())

				By("Ensuring the VMI has migrated and lives on another node")
				Eventually(func() error {
					vmi, err := virtClient.VirtualMachineInstance(vmi.Namespace).Get(vmi.Name, &metav1.GetOptions{})
					if err != nil {
						return err
					}

					if vmi.Status.NodeName == vmiNodeOrig {
						return fmt.Errorf("VMI is still on the same node")
					}

					if vmi.Status.MigrationState == nil || vmi.Status.MigrationState.SourceNode != vmiNodeOrig {
						return fmt.Errorf("VMI did not migrate yet")
					}

					if vmi.Status.EvacuationNodeName != "" {
						return fmt.Errorf("VMI is still evacuating: %v", vmi.Status.EvacuationNodeName)
					}

					return nil
				}, 360*time.Second, 1*time.Second).ShouldNot(HaveOccurred())
				resVMI, err := virtClient.VirtualMachineInstance(vmi.Namespace).Get(vmi.Name, &metav1.GetOptions{})
				Expect(err).ShouldNot(HaveOccurred())
				Expect(resVMI.Status.EvacuationNodeName).To(Equal(""), "vmi evacuation state should be clean")
			})

			It("[sig-compute][test_id:3243]should recreate the PDB if VMIs with similar names are recreated", func() {
				for x := 0; x < 3; x++ {
					By("creating the VMI")
					_, err := virtClient.VirtualMachineInstance(util.NamespaceTestDefault).Create(vmi)
					Expect(err).ToNot(HaveOccurred())

					By("checking that the PDB appeared")
					Eventually(func() []v1beta1.PodDisruptionBudget {
						pdbs, err := virtClient.PolicyV1beta1().PodDisruptionBudgets(util.NamespaceTestDefault).List(context.Background(), metav1.ListOptions{})
						Expect(err).ToNot(HaveOccurred())
						return pdbs.Items
					}, 3*time.Second, 500*time.Millisecond).Should(HaveLen(1))
					By("waiting for VMI")
					tests.WaitForSuccessfulVMIStartWithTimeout(vmi, 60)

					By("deleting the VMI")
					Expect(virtClient.VirtualMachineInstance(util.NamespaceTestDefault).Delete(vmi.Name, &metav1.DeleteOptions{})).To(Succeed())
					By("checking that the PDB disappeared")
					Eventually(func() []v1beta1.PodDisruptionBudget {
						pdbs, err := virtClient.PolicyV1beta1().PodDisruptionBudgets(util.NamespaceTestDefault).List(context.Background(), metav1.ListOptions{})
						Expect(err).ToNot(HaveOccurred())
						return pdbs.Items
					}, 3*time.Second, 500*time.Millisecond).Should(HaveLen(0))
					Eventually(func() bool {
						_, err := virtClient.VirtualMachineInstance(util.NamespaceTestDefault).Get(vmi.Name, &metav1.GetOptions{})
						return errors.IsNotFound(err)
					}, 60*time.Second, 500*time.Millisecond).Should(BeTrue())
				}
			})

			It("[sig-compute]should delete PDBs created by an old virt-controller", func() {
				By("creating the VMI")
				createdVMI, err := virtClient.VirtualMachineInstance(util.NamespaceTestDefault).Create(vmi)
				Expect(err).ToNot(HaveOccurred())
				By("waiting for VMI")
				tests.WaitForSuccessfulVMIStartWithTimeout(createdVMI, 60)

				By("Adding a fake old virt-controller PDB")
				two := intstr.FromInt(2)
				pdb, err := virtClient.PolicyV1beta1().PodDisruptionBudgets(createdVMI.Namespace).Create(context.Background(), &v1beta1.PodDisruptionBudget{
					ObjectMeta: metav1.ObjectMeta{
						OwnerReferences: []metav1.OwnerReference{
							*metav1.NewControllerRef(createdVMI, v1.VirtualMachineInstanceGroupVersionKind),
						},
						GenerateName: "kubevirt-disruption-budget-",
					},
					Spec: v1beta1.PodDisruptionBudgetSpec{
						MinAvailable: &two,
						Selector: &metav1.LabelSelector{
							MatchLabels: map[string]string{
								v1.CreatedByLabel: string(createdVMI.UID),
							},
						},
					},
				}, metav1.CreateOptions{})
				Expect(err).ToNot(HaveOccurred())

				By("checking that the PDB disappeared")
				Eventually(func() bool {
					_, err := virtClient.PolicyV1beta1().PodDisruptionBudgets(util.NamespaceTestDefault).Get(context.Background(), pdb.Name, metav1.GetOptions{})
					return errors.IsNotFound(err)
				}, 60*time.Second, 1*time.Second).Should(BeTrue())
			})

			It("[test_id:3244]should block the eviction api while a slow migration is in progress", func() {
				vmi = fedoraVMIWithEvictionStrategy()

				By("Starting the VirtualMachineInstance")
				vmi = runVMIAndExpectLaunch(vmi, 240)

				By("Checking that the VirtualMachineInstance console has expected output")
				Expect(console.LoginToFedora(vmi)).To(Succeed())

				tests.WaitAgentConnected(virtClient, vmi)

				runStressTest(vmi, stressDefaultVMSize, stressDefaultTimeout)

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
					pods, err = virtClient.CoreV1().Pods(vmi.Namespace).List(context.Background(), metav1.ListOptions{LabelSelector: labelSelector, FieldSelector: fieldSelector})
					Expect(err).ToNot(HaveOccurred())
					return pods.Items
				}, 90*time.Second, 500*time.Millisecond).Should(HaveLen(2))

				By("Verifying at least once that both pods are protected")
				for _, pod := range pods.Items {
					err := virtClient.CoreV1().Pods(vmi.Namespace).Evict(context.Background(), &v1beta1.Eviction{ObjectMeta: metav1.ObjectMeta{Name: pod.Name}})
					Expect(errors.IsTooManyRequests(err)).To(BeTrue())
				}
				By("Verifying that both pods are protected by the PodDisruptionBudget for the whole migration")
				getOptions := metav1.GetOptions{}
				Eventually(func() v1.VirtualMachineInstanceMigrationPhase {
					currentMigration, err := virtClient.VirtualMachineInstanceMigration(vmi.Namespace).Get(migration.Name, &getOptions)
					Expect(err).ToNot(HaveOccurred())
					Expect(currentMigration.Status.Phase).NotTo(Equal(v1.MigrationFailed))
					for _, p := range pods.Items {
						pod, err := virtClient.CoreV1().Pods(vmi.Namespace).Get(context.Background(), p.Name, getOptions)
						if err != nil || pod.Status.Phase != k8sv1.PodRunning {
							continue
						}

						deleteOptions := &metav1.DeleteOptions{Preconditions: &metav1.Preconditions{ResourceVersion: &pod.ResourceVersion}}
						eviction := &v1beta1.Eviction{ObjectMeta: metav1.ObjectMeta{Name: pod.Name}, DeleteOptions: deleteOptions}
						err = virtClient.CoreV1().Pods(vmi.Namespace).Evict(context.Background(), eviction)
						Expect(errors.IsTooManyRequests(err)).To(BeTrue())

					}
					return currentMigration.Status.Phase
				}, 180*time.Second, 500*time.Millisecond).Should(Equal(v1.MigrationSucceeded))
			})

			Context("with node tainted during node drain", func() {

				BeforeEach(func() {
					// Taints defined by k8s are special and can't be applied manually.
					// Temporarily configure KubeVirt to use something else for the duration of these tests.
					if tests.IsUsingBuiltinNodeDrainKey() {
						drain := "kubevirt.io/drain"
						cfg := getCurrentKv()
						cfg.MigrationConfiguration.NodeDrainTaintKey = &drain
						tests.UpdateKubeVirtConfigValueAndWait(cfg)
					}
					setMastersUnschedulable(true)
				})

				AfterEach(func() {
					tests.CleanNodes()
				})

				It("[test_id:6982]should migrate a VMI only one time", func() {
					tests.SkipIfVersionBelow("Eviction of completed pods requires v1.13 and above", "1.13")

					vmi = fedoraVMIWithEvictionStrategy()

					By("Starting the VirtualMachineInstance")
					vmi = runVMIAndExpectLaunch(vmi, 180)

					tests.WaitAgentConnected(virtClient, vmi)

					// Mark the masters as schedulable so we can migrate there
					setMastersUnschedulable(false)

					// Drain node.
					node := vmi.Status.NodeName
					drainNode(node)

					// verify VMI migrated and lives on another node now.
					Eventually(func() error {
						vmi, err := virtClient.VirtualMachineInstance(vmi.Namespace).Get(vmi.Name, &metav1.GetOptions{})
						if err != nil {
							return err
						} else if vmi.Status.NodeName == node {
							return fmt.Errorf("VMI still exist on the same node")
						} else if vmi.Status.MigrationState == nil || vmi.Status.MigrationState.SourceNode != node {
							return fmt.Errorf("VMI did not migrate yet")
						} else if vmi.Status.EvacuationNodeName != "" {
							return fmt.Errorf("evacuation node name is still set on the VMI")
						}

						// VMI should still be running at this point. If it
						// isn't, then there's nothing to be waiting on.
						Expect(vmi.Status.Phase).To(Equal(v1.Running))

						return nil
					}, 180*time.Second, 1*time.Second).ShouldNot(HaveOccurred())

					Consistently(func() error {
						migrations, err := virtClient.VirtualMachineInstanceMigration(vmi.Namespace).List(&metav1.ListOptions{})
						if err != nil {
							return err
						}
						if len(migrations.Items) > 1 {
							return fmt.Errorf("should have only 1 migration issued for evacuation of 1 VM")
						}
						return nil
					}, 20*time.Second, 1*time.Second).ShouldNot(HaveOccurred())

				})

				It("[test_id:2221] should migrate a VMI under load to another node", func() {
					tests.SkipIfVersionBelow("Eviction of completed pods requires v1.13 and above", "1.13")

					vmi = fedoraVMIWithEvictionStrategy()

					By("Starting the VirtualMachineInstance")
					vmi = runVMIAndExpectLaunch(vmi, 180)

					By("Checking that the VirtualMachineInstance console has expected output")
					Expect(console.LoginToFedora(vmi)).To(Succeed())

					tests.WaitAgentConnected(virtClient, vmi)

					// Put VMI under load
					runStressTest(vmi, stressDefaultVMSize, stressDefaultTimeout)

					// Mark the masters as schedulable so we can migrate there
					setMastersUnschedulable(false)

					// Taint Node.
					By("Tainting node with node drain key")
					node := vmi.Status.NodeName
					tests.Taint(node, tests.GetNodeDrainKey(), k8sv1.TaintEffectNoSchedule)

					drainNode(node)

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

					By("Configuring a custom nodeDrainTaintKey in kubevirt configuration")
					cfg := getCurrentKv()
					drainKey := "kubevirt.io/alt-drain"
					cfg.MigrationConfiguration.NodeDrainTaintKey = &drainKey
					tests.UpdateKubeVirtConfigValueAndWait(cfg)

					By("Starting the VirtualMachineInstance")
					vmi = runVMIAndExpectLaunch(vmi, 180)

					// Mark the masters as schedulable so we can migrate there
					setMastersUnschedulable(false)

					// Taint Node.
					By("Tainting node with kubevirt.io/alt-drain=NoSchedule")
					node := vmi.Status.NodeName
					tests.Taint(node, "kubevirt.io/alt-drain", k8sv1.TaintEffectNoSchedule)

					drainNode(node)

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
					vmi_noevict := tests.NewRandomVMIWithEphemeralDiskAndUserdata(cd.ContainerDiskFor(cd.ContainerDiskCirros), "#!/bin/bash\necho 'hello'\n")

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
					vm_evict1, err = virtClient.VirtualMachine(util.NamespaceTestDefault).Create(vm_evict1)
					Expect(err).ToNot(HaveOccurred())
					vm_evict2, err = virtClient.VirtualMachine(util.NamespaceTestDefault).Create(vm_evict2)
					Expect(err).ToNot(HaveOccurred())
					vm_noevict, err = virtClient.VirtualMachine(util.NamespaceTestDefault).Create(vm_noevict)
					Expect(err).ToNot(HaveOccurred())

					// Start VMs
					tests.StartVirtualMachine(vm_evict1)
					tests.StartVirtualMachine(vm_evict2)
					tests.StartVirtualMachine(vm_noevict)

					// Get VMIs
					vmi_evict1, err = virtClient.VirtualMachineInstance(vmi_evict1.Namespace).Get(vmi_evict1.Name, &metav1.GetOptions{})
					vmi_evict2, err = virtClient.VirtualMachineInstance(vmi_evict1.Namespace).Get(vmi_evict2.Name, &metav1.GetOptions{})
					vmi_noevict, err = virtClient.VirtualMachineInstance(vmi_evict1.Namespace).Get(vmi_noevict.Name, &metav1.GetOptions{})

					By("Verifying all VMIs are collcated on the same node")
					Expect(vmi_evict1.Status.NodeName).To(Equal(vmi_evict2.Status.NodeName))
					Expect(vmi_evict1.Status.NodeName).To(Equal(vmi_noevict.Status.NodeName))

					// Mark the masters as schedulable so we can migrate there
					setMastersUnschedulable(false)

					// Taint Node.
					By("Tainting node with the node drain key")
					node := vmi_evict1.Status.NodeName
					tests.Taint(node, tests.GetNodeDrainKey(), k8sv1.TaintEffectNoSchedule)

					// Drain Node using cli client
					By("Draining using kubectl drain")
					drainNode(node)

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

			It("[release-blocker][test_id:3245]should not migrate more than two VMIs at the same time from a node", func() {
				var vmis []*v1.VirtualMachineInstance
				for i := 0; i < 4; i++ {
					vmi := cirrosVMIWithEvictionStrategy()
					vmi.Spec.NodeSelector = map[string]string{"tests.kubevirt.io": "target"}
					vmis = append(vmis, vmi)
				}

				By("selecting a node as the source")
				sourceNode := util.GetAllSchedulableNodes(virtClient).Items[0]
				tests.AddLabelToNode(sourceNode.Name, "tests.kubevirt.io", "target")

				By("starting four VMIs on that node")
				for _, vmi := range vmis {
					_, err := virtClient.VirtualMachineInstance(util.NamespaceTestDefault).Create(vmi)
					Expect(err).ToNot(HaveOccurred())
				}

				By("waiting until the VMIs are ready")
				for _, vmi := range vmis {
					tests.WaitForSuccessfulVMIStartWithTimeout(vmi, 180)
				}

				By("selecting a node as the target")
				targetNode := util.GetAllSchedulableNodes(virtClient).Items[1]
				tests.AddLabelToNode(targetNode.Name, "tests.kubevirt.io", "target")

				By("tainting the source node as non-schedulabele")
				tests.Taint(sourceNode.Name, tests.GetNodeDrainKey(), k8sv1.TaintEffectNoSchedule)

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
						vmi, err = virtClient.VirtualMachineInstance(util.NamespaceTestDefault).Get(vmi.Name, &metav1.GetOptions{})
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

				By("Checking that all migrated VMIs have the new pod IP address on VMI status")
				for _, vmi := range vmis {
					Eventually(func() error {
						newvmi, err := virtClient.VirtualMachineInstance(util.NamespaceTestDefault).Get(vmi.Name, &metav1.GetOptions{})
						Expect(err).ToNot(HaveOccurred(), "Should successfully get new VMI")
						vmiPod := tests.GetRunningPodByVirtualMachineInstance(newvmi, newvmi.Namespace)
						return libnet.ValidateVMIandPodIPMatch(newvmi, vmiPod)
					}, time.Minute, time.Second).Should(Succeed(), "Should match PodIP with latest VMI Status after migration")
				}
			})
		})

	})

	Context("With Huge Pages", func() {
		var hugepagesVmi *v1.VirtualMachineInstance

		BeforeEach(func() {
			hugepagesVmi = tests.NewRandomVMIWithEphemeralDiskAndUserdata(cd.ContainerDiskFor(cd.ContainerDiskCirros), "#!/bin/bash\necho 'hello'\n")
		})

		table.DescribeTable("should consume hugepages ", func(hugepageSize string, memory string) {
			hugepageType := k8sv1.ResourceName(k8sv1.ResourceHugePagesPrefix + hugepageSize)
			v, err := cluster.GetKubernetesVersion()
			Expect(err).ShouldNot(HaveOccurred())
			if strings.Contains(v, "1.16") {
				hugepagesVmi.Annotations = map[string]string{
					v1.MemfdMemoryBackend: "false",
				}
				log.DefaultLogger().Object(hugepagesVmi).Infof("Fall back to use hugepages source file. Libvirt in the 1.16 provider version doesn't support memfd as memory backend")
			}

			count := 0
			nodes, err := virtClient.CoreV1().Nodes().List(context.Background(), metav1.ListOptions{})
			ExpectWithOffset(1, err).ToNot(HaveOccurred())

			requestedMemory := resource.MustParse(memory)
			hugepagesVmi.Spec.Domain.Resources.Requests[k8sv1.ResourceMemory] = requestedMemory

			for _, node := range nodes.Items {
				// Cmp returns -1, 0, or 1 for less than, equal to, or greater than
				if v, ok := node.Status.Capacity[hugepageType]; ok && v.Cmp(requestedMemory) == 1 {
					count += 1
				}
			}

			if count < 2 {
				Skip(fmt.Sprintf("Not enough nodes with hugepages %s capacity. Need 2, found %d.", hugepageType, count))
			}

			hugepagesVmi.Spec.Domain.Memory = &v1.Memory{
				Hugepages: &v1.Hugepages{PageSize: hugepageSize},
			}

			By("Starting hugepages VMI")
			_, err = virtClient.VirtualMachineInstance(util.NamespaceTestDefault).Create(hugepagesVmi)
			Expect(err).ToNot(HaveOccurred())
			tests.WaitForSuccessfulVMIStart(hugepagesVmi)

			By("starting the migration")
			migration := tests.NewRandomMigration(hugepagesVmi.Name, hugepagesVmi.Namespace)
			migrationUID := tests.RunMigrationAndExpectCompletion(virtClient, migration, tests.MigrationWaitTime)

			// check VMI, confirm migration state
			tests.ConfirmVMIPostMigration(virtClient, hugepagesVmi, migrationUID)

			// delete VMI
			By("Deleting the VMI")
			Expect(virtClient.VirtualMachineInstance(hugepagesVmi.Namespace).Delete(hugepagesVmi.Name, &metav1.DeleteOptions{})).To(Succeed())

			By("Waiting for VMI to disappear")
			tests.WaitForVirtualMachineToDisappearWithTimeout(hugepagesVmi, 240)
		},
			table.Entry("[test_id:6983]hugepages-2Mi", "2Mi", "64Mi"),
			table.Entry("[test_id:6984]hugepages-1Gi", "1Gi", "1Gi"),
		)
	})

	Context("with CPU pinning and huge pages", func() {
		It("should not make migrations fail", func() {
			checks.SkipTestIfNotEnoughNodesWithCPUManagerWith2MiHugepages(2)
			var err error
			cpuVMI := tests.NewRandomVMIWithEphemeralDiskAndUserdata(cd.ContainerDiskFor(cd.ContainerDiskCirros), "#!/bin/bash\necho 'hello'\n")
			cpuVMI.Spec.Domain.Resources.Requests[k8sv1.ResourceMemory] = resource.MustParse("128Mi")
			cpuVMI.Spec.Domain.CPU = &v1.CPU{
				Cores:                 3,
				DedicatedCPUPlacement: true,
			}
			cpuVMI.Spec.Domain.Memory = &v1.Memory{
				Hugepages: &v1.Hugepages{PageSize: "2Mi"},
			}

			By("Starting a VirtualMachineInstance")
			cpuVMI, err = virtClient.VirtualMachineInstance(util.NamespaceTestDefault).Create(cpuVMI)
			Expect(err).ToNot(HaveOccurred())
			tests.WaitForSuccessfulVMIStart(cpuVMI)

			By("Performing a migration")
			migration := tests.NewRandomMigration(cpuVMI.Name, cpuVMI.Namespace)
			tests.RunMigrationAndExpectCompletion(virtClient, migration, tests.MigrationWaitTime)
		})
		Context("and NUMA passthrough", func() {
			It("should not make migrations fail", func() {
				checks.SkipTestIfNoFeatureGate(virtconfig.NUMAFeatureGate)
				checks.SkipTestIfNotEnoughNodesWithCPUManagerWith2MiHugepages(2)
				var err error
				cpuVMI := tests.NewRandomVMIWithEphemeralDiskAndUserdata(cd.ContainerDiskFor(cd.ContainerDiskCirros), "#!/bin/bash\necho 'hello'\n")
				cpuVMI.Spec.Domain.Resources.Requests[k8sv1.ResourceMemory] = resource.MustParse("128Mi")
				cpuVMI.Spec.Domain.CPU = &v1.CPU{
					Cores:                 3,
					DedicatedCPUPlacement: true,
					NUMA:                  &v1.NUMA{GuestMappingPassthrough: &v1.NUMAGuestMappingPassthrough{}},
				}
				cpuVMI.Spec.Domain.Memory = &v1.Memory{
					Hugepages: &v1.Hugepages{PageSize: "2Mi"},
				}

				By("Starting a VirtualMachineInstance")
				cpuVMI, err = virtClient.VirtualMachineInstance(util.NamespaceTestDefault).Create(cpuVMI)
				Expect(err).ToNot(HaveOccurred())
				tests.WaitForSuccessfulVMIStart(cpuVMI)

				By("Performing a migration")
				migration := tests.NewRandomMigration(cpuVMI.Name, cpuVMI.Namespace)
				tests.RunMigrationAndExpectCompletion(virtClient, migration, tests.MigrationWaitTime)
			})
		})
	})

	It("should replace containerdisk and kernel boot images with their reproducible digest during migration", func() {

		vmi := tests.NewRandomVMIWithEphemeralDiskAndUserdata(cd.ContainerDiskFor(cd.ContainerDiskCirros), "#!/bin/bash\necho 'hello'\n")
		vmi.Spec.Domain.Firmware = utils.GetVMIKernelBoot().Spec.Domain.Firmware

		By("Starting a VirtualMachineInstance")
		vmi, err = virtClient.VirtualMachineInstance(util.NamespaceTestDefault).Create(vmi)
		Expect(err).ToNot(HaveOccurred())
		tests.WaitForSuccessfulVMIStart(vmi)

		pod := tests.GetRunningPodByVirtualMachineInstance(vmi, vmi.Namespace)
		By("Verifying that all relevant images are without the digest on the source")
		for _, container := range append(pod.Spec.Containers, pod.Spec.InitContainers...) {
			if container.Name == "container-disk-binary" || container.Name == "compute" {
				continue
			}
			Expect(container.Image).ToNot(ContainSubstring("@sha256:"), "image:%s should not contain the container digest for container %s", container.Image, container.Name)
		}

		digestRegex := regexp.MustCompile(`sha256:[a-zA-Z0-9]+`)

		By("Collecting digest information from the container statuses")
		imageIDs := map[string]string{}
		for _, status := range append(pod.Status.ContainerStatuses, pod.Status.InitContainerStatuses...) {
			if status.Name == "container-disk-binary" || status.Name == "compute" {
				continue
			}
			digest := digestRegex.FindString(status.ImageID)
			Expect(digest).ToNot(BeEmpty())
			imageIDs[status.Name] = digest
		}

		By("Performing a migration")
		migration := tests.NewRandomMigration(vmi.Name, vmi.Namespace)
		tests.RunMigrationAndExpectCompletion(virtClient, migration, tests.MigrationWaitTime)

		By("Verifying that all imageIDs are in a reproducible form on the target")
		pod = tests.GetRunningPodByVirtualMachineInstance(vmi, vmi.Namespace)

		for _, container := range append(pod.Spec.Containers, pod.Spec.InitContainers...) {
			if container.Name == "container-disk-binary" || container.Name == "compute" {
				continue
			}
			digest := digestRegex.FindString(container.Image)
			Expect(container.Image).To(ContainSubstring(digest), "image:%s should contain the container digest for container %s", container.Image, container.Name)
			Expect(digest).ToNot(BeEmpty())
			Expect(imageIDs).To(HaveKeyWithValue(container.Name, digest), "expected image:%s for container %s to be the same like on the source pod but got %s", container.Image, container.Name, imageIDs[container.Name])
		}
	})
})

func fedoraVMIWithEvictionStrategy() *v1.VirtualMachineInstance {
	vmi := tests.NewRandomFedoraVMIWithGuestAgent()
	strategy := v1.EvictionStrategyLiveMigrate
	vmi.Spec.EvictionStrategy = &strategy
	vmi.Spec.Domain.Resources.Requests[k8sv1.ResourceMemory] = resource.MustParse(fedoraVMSize)
	return vmi
}

func cirrosVMIWithEvictionStrategy() *v1.VirtualMachineInstance {
	strategy := v1.EvictionStrategyLiveMigrate
	vmi := tests.NewRandomVMIWithEphemeralDiskAndUserdata(cd.ContainerDiskFor(cd.ContainerDiskCirros), "#!/bin/bash\necho 'hello'\n")
	vmi.Spec.EvictionStrategy = &strategy
	return vmi
}

func temporaryTLSConfig() *tls.Config {
	// Generate new certs if secret doesn't already exist
	caKeyPair, _ := triple.NewCA("kubevirt.io", time.Hour)

	clientKeyPair, _ := triple.NewClientKeyPair(caKeyPair,
		"kubevirt.io:system:node:virt-handler",
		nil,
		time.Hour,
	)

	certPEM := cert.EncodeCertPEM(clientKeyPair.Cert)
	keyPEM := cert.EncodePrivateKeyPEM(clientKeyPair.Key)
	cert, err := tls.X509KeyPair(certPEM, keyPEM)
	Expect(err).ToNot(HaveOccurred())
	return &tls.Config{
		InsecureSkipVerify: true,
		GetClientCertificate: func(info *tls.CertificateRequestInfo) (certificate *tls.Certificate, e error) {
			return &cert, nil
		},
	}
}
