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
	"context"
	"fmt"
	"time"

	expect "github.com/google/goexpect"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	k8sv1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/rand"

	v1 "kubevirt.io/api/core/v1"
	migrationsv1 "kubevirt.io/api/migrations/v1alpha1"
	"kubevirt.io/client-go/kubecli"
	cdiv1 "kubevirt.io/containerized-data-importer-api/pkg/apis/core/v1beta1"

	"kubevirt.io/kubevirt/pkg/libdv"
	"kubevirt.io/kubevirt/pkg/libvmi"
	libvmici "kubevirt.io/kubevirt/pkg/libvmi/cloudinit"
	kvpointer "kubevirt.io/kubevirt/pkg/pointer"
	"kubevirt.io/kubevirt/tests/console"
	cd "kubevirt.io/kubevirt/tests/containerdisk"
	"kubevirt.io/kubevirt/tests/decorators"
	"kubevirt.io/kubevirt/tests/flags"
	"kubevirt.io/kubevirt/tests/framework/kubevirt"
	"kubevirt.io/kubevirt/tests/framework/matcher"
	kvconfig "kubevirt.io/kubevirt/tests/libkubevirt/config"
	"kubevirt.io/kubevirt/tests/libmigration"
	"kubevirt.io/kubevirt/tests/libnet"
	"kubevirt.io/kubevirt/tests/libpod"
	"kubevirt.io/kubevirt/tests/libstorage"
	"kubevirt.io/kubevirt/tests/libvmifact"
	"kubevirt.io/kubevirt/tests/libvmops"
	"kubevirt.io/kubevirt/tests/libwait"
	"kubevirt.io/kubevirt/tests/testsuite"
)

var _ = Describe(SIG("VM Post Copy Live Migration", decorators.RequiresTwoSchedulableNodes, func() {
	var (
		virtClient      kubecli.KubevirtClient
		err             error
		migrationPolicy *migrationsv1.MigrationPolicy
	)

	BeforeEach(func() {
		virtClient = kubevirt.Client()

		By("Allowing post-copy and limiting migration bandwidth")
		policyName := fmt.Sprintf("testpolicy-%s", rand.String(5))
		migrationPolicy = kubecli.NewMinimalMigrationPolicy(policyName)
		migrationPolicy.Spec.AllowPostCopy = kvpointer.P(true)
		migrationPolicy.Spec.CompletionTimeoutPerGiB = kvpointer.P(int64(1))
		migrationPolicy.Spec.BandwidthPerMigration = kvpointer.P(resource.MustParse("5Mi"))
	})

	Context("with datavolume", func() {
		var dv *cdiv1.DataVolume

		BeforeEach(func() {
			sc, foundSC := libstorage.GetRWXFileSystemStorageClass()
			if !foundSC {
				Skip("Skip test when Filesystem storage is not present")
			}

			dv = libdv.NewDataVolume(
				libdv.WithRegistryURLSourceAndPullMethod(cd.DataVolumeImportUrlForContainerDisk(cd.ContainerDiskFedoraTestTooling), cdiv1.RegistryPullNode),
				libdv.WithStorage(
					libdv.StorageWithStorageClass(sc),
					libdv.StorageWithVolumeSize(cd.FedoraVolumeSize),
					libdv.StorageWithReadWriteManyAccessMode(),
				),
			)

			dv, err = virtClient.CdiClient().CdiV1beta1().DataVolumes(testsuite.NamespacePrivileged).Create(context.Background(), dv, metav1.CreateOptions{})
			Expect(err).ToNot(HaveOccurred())
		})

		It("[test_id:5004] should be migrated successfully, using guest agent on VM with post-copy", func() {
			VMIMigrationWithGuestAgent(virtClient, dv.Name, "1Gi", migrationPolicy)
		})
	})

	Context("should migrate using post-copy", func() {
		applyMigrationPolicy := func(vmi *v1.VirtualMachineInstance) {
			AlignPolicyAndVmi(vmi, migrationPolicy)
			migrationPolicy = CreateMigrationPolicy(virtClient, migrationPolicy)
		}

		applyKubevirtCR := func() {
			config := getCurrentKvConfig(virtClient)
			config.MigrationConfiguration.AllowPostCopy = migrationPolicy.Spec.AllowPostCopy
			config.MigrationConfiguration.CompletionTimeoutPerGiB = migrationPolicy.Spec.CompletionTimeoutPerGiB
			config.MigrationConfiguration.BandwidthPerMigration = migrationPolicy.Spec.BandwidthPerMigration
			kvconfig.UpdateKubeVirtConfigValueAndWait(config)
		}

		type applySettingsType string
		const (
			applyWithMigrationPolicy applySettingsType = "policy"
			applyWithKubevirtCR      applySettingsType = "kubevirt"
		)

		DescribeTable("[test_id:4747] using", func(settingsType applySettingsType) {
			vmi := libvmifact.NewFedora(libnet.WithMasqueradeNetworking())
			vmi.Spec.Domain.Resources.Requests[k8sv1.ResourceMemory] = resource.MustParse("512Mi")
			vmi.Spec.Domain.Devices.Rng = &v1.Rng{}
			vmi.Namespace = testsuite.NamespacePrivileged

			switch settingsType {
			case applyWithMigrationPolicy:
				applyMigrationPolicy(vmi)
			case applyWithKubevirtCR:
				applyKubevirtCR()
			}

			By("Starting the VirtualMachineInstance")
			vmi = libvmops.RunVMIAndExpectLaunch(vmi, 240)

			By("Checking that the VirtualMachineInstance console has expected output")
			Expect(console.LoginToFedora(vmi)).To(Succeed())

			// Need to wait for cloud init to finish and start the agent inside the vmi.
			Eventually(matcher.ThisVMI(vmi), 12*time.Minute, 2*time.Second).Should(matcher.HaveConditionTrue(v1.VirtualMachineInstanceAgentConnected))

			runStressTest(vmi, "350M")

			// execute a migration, wait for finalized state
			By("Starting the Migration")
			migration := libmigration.New(vmi.Name, vmi.Namespace)
			migration = libmigration.RunMigrationAndExpectToComplete(virtClient, migration, 150)

			// check VMI, confirm migration state
			libmigration.ConfirmVMIPostMigration(virtClient, vmi, migration)
			libmigration.ConfirmMigrationMode(virtClient, vmi, v1.MigrationPostCopy)
		},
			Entry("a migration policy", applyWithMigrationPolicy),
			Entry("the Kubevirt CR", Serial, applyWithKubevirtCR),
		)

		Context("and fail", Serial, func() {
			var killerPod string

			runVirtHandlerKillerPod := func(nodeName string) {
				podName := "migration-killer-pod-"

				// kill the handler
				pod := libpod.RenderPrivilegedPod(podName, []string{"/bin/bash", "-c"}, []string{"date; pkill -e -9 virt-handler || echo not found"})

				pod.Spec.NodeSelector = map[string]string{"kubernetes.io/hostname": nodeName}
				createdPod, err := virtClient.CoreV1().Pods(pod.Namespace).Create(context.Background(), pod, metav1.CreateOptions{})
				Expect(err).ToNot(HaveOccurred(), "Should create killer pod")
				killerPod = createdPod.Name
			}

			removeVirtHandlerKillerPod := func() {
				Expect(killerPod).NotTo(BeEmpty())
				Eventually(func() error {
					err := virtClient.CoreV1().Pods(testsuite.NamespacePrivileged).Delete(context.Background(), killerPod, metav1.DeleteOptions{})
					return err
				}, 1*time.Minute, 1*time.Second).Should(MatchError(errors.IsNotFound, "k8serrors.IsNotFound"), "Should delete helper pod")

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
				}, 2*time.Minute, 1*time.Second).Should(Succeed(), "Virt handler should come online")
			}

			AfterEach(func() {
				By("Ensuring the virt-handler killer pod is removed")
				removeVirtHandlerKillerPod()
			})

			It("and make sure VMs restart after failure", func() {
				By("creating a large VM with RunStrategyRerunOnFailure")
				vmi := libvmifact.NewFedora(
					libnet.WithMasqueradeNetworking(),
					libvmi.WithResourceMemory("3Gi"),
					libvmi.WithRng(),
					libvmi.WithNamespace(testsuite.NamespaceTestDefault),
				)
				vm := libvmi.NewVirtualMachine(vmi, libvmi.WithRunStrategy(v1.RunStrategyRerunOnFailure))

				vm, err := virtClient.VirtualMachine(vm.Namespace).Create(context.Background(), vm, metav1.CreateOptions{})
				Expect(err).ToNot(HaveOccurred())

				// update the migration policy to ensure slow pre-copy migration progress instead of an immediate cancellation.
				migrationPolicy.Spec.CompletionTimeoutPerGiB = kvpointer.P(int64(20))
				migrationPolicy.Spec.BandwidthPerMigration = kvpointer.P(resource.MustParse("1Mi"))
				applyKubevirtCR()

				By("Waiting for the VirtualMachine to be ready")
				vmi = libwait.WaitUntilVMIReady(vmi, console.LoginToFedora)

				// Need to wait for cloud init to finish and start the agent inside the vmi.
				Eventually(matcher.ThisVMI(vmi), 12*time.Minute, 2*time.Second).Should(matcher.HaveConditionTrue(v1.VirtualMachineInstanceAgentConnected))

				runStressTest(vmi, "350M")

				By("Starting the Migration")
				migration := libmigration.New(vmi.Name, vmi.Namespace)
				migration = libmigration.RunMigration(virtClient, migration)

				// check VMI, confirm migration state
				libmigration.WaitUntilMigrationMode(virtClient, vmi, v1.MigrationPostCopy, 5*time.Minute)

				By("Starting virt-handler killer pod")
				runVirtHandlerKillerPod(vmi.Status.NodeName)

				By("Making sure that post-copy migration failed")
				Eventually(matcher.ThisMigration(migration), 3*time.Minute, 1*time.Second).Should(matcher.BeInPhase(v1.MigrationFailed))

				By("Removing virt-handler killer pod")
				removeVirtHandlerKillerPod()

				By("Ensuring the VirtualMachineInstance is restarted")
				Eventually(matcher.ThisVMI(vmi), 5*time.Minute, 1*time.Second).Should(matcher.BeRestarted(vmi.UID))
			})
		})
	})
}))

func VMIMigrationWithGuestAgent(virtClient kubecli.KubevirtClient, pvName string, memoryRequestSize string, migrationPolicy *migrationsv1.MigrationPolicy) {
	By("Creating the VMI")

	// add userdata for guest agent and service account mount
	mountSvcAccCommands := fmt.Sprintf(`#!/bin/bash
            mkdir /mnt/servacc
            mount /dev/$(lsblk --nodeps -no name,serial | grep %s | cut -f1 -d' ') /mnt/servacc
        `, secretDiskSerial)
	vmi := libvmi.New(
		libvmi.WithInterface(libvmi.InterfaceDeviceWithMasqueradeBinding()),
		libvmi.WithNetwork(v1.DefaultPodNetwork()),
		libvmi.WithPersistentVolumeClaim("disk0", pvName),
		libvmi.WithResourceMemory(memoryRequestSize),
		libvmi.WithRng(),
		libvmi.WithCloudInitNoCloud(libvmici.WithNoCloudEncodedUserData(mountSvcAccCommands)),
		libvmi.WithServiceAccountDisk("default"),
	)

	mode := v1.MigrationPreCopy
	if migrationPolicy != nil && migrationPolicy.Spec.AllowPostCopy != nil && *migrationPolicy.Spec.AllowPostCopy {
		mode = v1.MigrationPostCopy
	}

	// postcopy needs a privileged namespace
	if mode == v1.MigrationPostCopy {
		vmi.Namespace = testsuite.NamespacePrivileged
	}

	disks := vmi.Spec.Domain.Devices.Disks
	disks[len(disks)-1].Serial = secretDiskSerial

	if migrationPolicy != nil {
		AlignPolicyAndVmi(vmi, migrationPolicy)
		migrationPolicy = CreateMigrationPolicy(virtClient, migrationPolicy)
	}
	vmi = libvmops.RunVMIAndExpectLaunchIgnoreWarnings(vmi, 180)

	// Wait for cloud init to finish and start the agent inside the vmi.
	Eventually(matcher.ThisVMI(vmi), 12*time.Minute, 2*time.Second).Should(matcher.HaveConditionTrue(v1.VirtualMachineInstanceAgentConnected))

	By("Checking that the VirtualMachineInstance console has expected output")
	Expect(console.LoginToFedora(vmi)).To(Succeed(), "Should be able to login to the Fedora VM")

	if mode == v1.MigrationPostCopy {
		By("Running stress test to allow transition to post-copy")
		runStressTest(vmi, stressLargeVMSize)
	}

	// execute a migration, wait for finalized state
	By("Starting the Migration for iteration")
	sourcePod, err := libpod.GetPodByVirtualMachineInstance(vmi, vmi.Namespace)
	Expect(err).NotTo(HaveOccurred())

	migration := libmigration.New(vmi.Name, vmi.Namespace)
	migration = libmigration.RunMigrationAndExpectToCompleteWithDefaultTimeout(virtClient, migration)
	By("Checking VMI, confirm migration state")
	vmi = libmigration.ConfirmVMIPostMigration(virtClient, vmi, migration)
	Expect(vmi.Status.MigrationState.SourcePod).To(Equal(sourcePod.Name))
	libmigration.ConfirmMigrationMode(virtClient, vmi, mode)

	By("Is agent connected after migration")
	Eventually(matcher.ThisVMI(vmi), 12*time.Minute, 2*time.Second).Should(matcher.HaveConditionTrue(v1.VirtualMachineInstanceAgentConnected))

	By("Checking that the migrated VirtualMachineInstance console has expected output")
	Expect(console.OnPrivilegedPrompt(vmi, 60)).To(BeTrue(), "Should stay logged in to the migrated VM")

	By("Checking that the service account is mounted")
	Expect(console.SafeExpectBatch(vmi, []expect.Batcher{
		&expect.BSnd{S: "cat /mnt/servacc/namespace\n"},
		&expect.BExp{R: vmi.Namespace},
	}, 30)).To(Succeed(), "Should be able to access the mounted service account file")
}
