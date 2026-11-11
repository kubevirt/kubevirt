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

	"k8s.io/apimachinery/pkg/api/errors"
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
	"kubevirt.io/kubevirt/pkg/virt-config/featuregate"
	"kubevirt.io/kubevirt/tests/console"
	cd "kubevirt.io/kubevirt/tests/containerdisk"
	"kubevirt.io/kubevirt/tests/decorators"
	"kubevirt.io/kubevirt/tests/flags"
	"kubevirt.io/kubevirt/tests/framework/checks"
	"kubevirt.io/kubevirt/tests/framework/kubevirt"
	"kubevirt.io/kubevirt/tests/framework/matcher"
	"kubevirt.io/kubevirt/tests/libkubevirt"
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

const (
	// Assuming ~500Mbps ≈ 17s/GiB one-way; 20s/GiB leaves margin for bulk copy to finish in pre-copy
	// before completion timeout, while 1ms max downtime blocks convergence into switchover.
	// Note: getVMIMigrationDataSize uses decimal GB (rounds 1Gi up to 2), so a 1Gi VMI times out at ~40s.
	postCopyCompletionTimeoutPerGiB int64 = 20
	// High enough that downtime tuning never ramps past InitialMs in these tests.
	postCopyDowntimeTuneNeverIterate int64 = 1 << 30
)

var _ = Describe(SIG("VM Post Copy Live Migration", Serial, decorators.RequiresTwoSchedulableNodes, func() {
	var (
		virtClient                 kubecli.KubevirtClient
		err                        error
		migrationPolicy            *migrationsv1.MigrationPolicy
		defaultArch                string
		hadMigrationDowntimeTuning bool
	)

	BeforeEach(func() {
		virtClient = kubevirt.Client()
		var err error
		defaultArch, err = libkubevirt.GetDefaultArchitecture(virtClient)
		Expect(err).ToNot(HaveOccurred())

		hadMigrationDowntimeTuning = checks.HasFeature(featuregate.MigrationDowntimeTuning)
		if !hadMigrationDowntimeTuning {
			kvconfig.EnableFeatureGate(featuregate.MigrationDowntimeTuning)
		}

		By("Allowing post-copy with 1ms downtime to prevent precopy convergence")
		policyName := fmt.Sprintf("testpolicy-%s", rand.String(5))
		migrationPolicy = kubecli.NewMinimalMigrationPolicy(policyName)
		migrationPolicy.Spec.AllowPostCopy = kvpointer.P(true)
		migrationPolicy.Spec.CompletionTimeoutPerGiB = kvpointer.P(postCopyCompletionTimeoutPerGiB)
		migrationPolicy.Spec.MaxDowntimeMs = kvpointer.P(uint64(1))
		migrationPolicy.Spec.ExperimentalMigrationOptions = &v1.ExperimentalMigrationOptions{
			DowntimeTuning: &v1.DowntimeTuningOptions{
				InitialMs:           kvpointer.P(int64(1)),
				StartAfterIteration: kvpointer.P(postCopyDowntimeTuneNeverIterate),
			},
		}
	})

	AfterEach(func() {
		if hadMigrationDowntimeTuning {
			kvconfig.EnableFeatureGate(featuregate.MigrationDowntimeTuning)
		} else {
			kvconfig.DisableFeatureGate(featuregate.MigrationDowntimeTuning)
		}
	})

	Context("with datavolume", decorators.RequiresRWXFilesystemStorage, func() {
		var dv *cdiv1.DataVolume

		BeforeEach(func() {
			sc, foundSC := libstorage.GetRWXFileSystemStorageClass()
			if !foundSC {
				Fail("Failed test when RWX Filesystem storage is not present")
			}

			dv = libdv.NewDataVolume(
				libdv.WithRegistrySource(libdv.WithURL(cd.DataVolumeImportUrlForContainerDisk(cd.ContainerDiskFedoraTestTooling)), libdv.WithPullMethod(cdiv1.RegistryPullNode), libdv.WithPlatformArch(defaultArch)),
				libdv.WithStorage(
					libdv.StorageWithStorageClass(sc),
					libdv.StorageWithVolumeSize(cd.FedoraVolumeSize),
					libdv.StorageWithReadWriteManyAccessMode(),
				),
			)

			dv, err = virtClient.CdiClient().CdiV1beta1().DataVolumes(testsuite.GetTestNamespace(dv)).Create(context.Background(), dv, metav1.CreateOptions{})
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

		// Post-copy via CR; DowntimeTuning still needs applyDowntimeTuningOnlyPolicy (not on MigrationConfiguration).
		applyKubevirtCR := func() {
			config := getCurrentKvConfig(virtClient)
			config.MigrationConfiguration.AllowPostCopy = migrationPolicy.Spec.AllowPostCopy
			config.MigrationConfiguration.CompletionTimeoutPerGiB = migrationPolicy.Spec.CompletionTimeoutPerGiB
			config.MigrationConfiguration.MaxDowntimeMs = migrationPolicy.Spec.MaxDowntimeMs
			config.MigrationConfiguration.BandwidthPerMigration = nil
			kvconfig.UpdateKubeVirtConfigValueAndWait(config)
		}

		applyDowntimeTuningOnlyPolicy := func(vmi *v1.VirtualMachineInstance) *migrationsv1.MigrationPolicy {
			policyName := fmt.Sprintf("testpolicy-downtime-%s", rand.String(5))
			policy := kubecli.NewMinimalMigrationPolicy(policyName)
			policy.Spec.ExperimentalMigrationOptions = &v1.ExperimentalMigrationOptions{
				DowntimeTuning: &v1.DowntimeTuningOptions{
					InitialMs:           kvpointer.P(int64(1)),
					StartAfterIteration: kvpointer.P(postCopyDowntimeTuneNeverIterate),
				},
			}
			AlignPolicyAndVmi(vmi, policy)
			return CreateMigrationPolicy(virtClient, policy)
		}

		type applySettingsType string
		const (
			applyWithMigrationPolicy applySettingsType = "policy"
			applyWithKubevirtCR      applySettingsType = "kubevirt"
		)

		DescribeTable("[test_id:4747] using", func(settingsType applySettingsType) {
			vmi := libvmifact.NewFedora(libnet.WithMasqueradeNetworking())

			switch settingsType {
			case applyWithMigrationPolicy:
				applyMigrationPolicy(vmi)
			case applyWithKubevirtCR:
				applyKubevirtCR()
				Expect(applyDowntimeTuningOnlyPolicy(vmi).Spec.AllowPostCopy).To(BeNil(), "downtime-only policy must not set post-copy")
			}

			By("Starting the VirtualMachineInstance")
			vmi = libvmops.RunVMIAndExpectLaunch(vmi, flags.StartupTimeoutSecondsHuge())

			By("Checking that the VirtualMachineInstance console has expected output")
			Expect(console.LoginToFedora(vmi)).To(Succeed())

			// Need to wait for cloud init to finish and start the agent inside the vmi.
			Eventually(matcher.ThisVMI(vmi), 12*time.Minute, 2*time.Second).Should(matcher.HaveConditionTrue(v1.VirtualMachineInstanceAgentConnected))

			runStressTest(vmi)

			// execute a migration, wait for finalized state
			By("Starting the Migration")
			migration := libmigration.New(vmi.Name, vmi.Namespace)
			migration = libmigration.RunMigrationAndExpectToComplete(virtClient, migration, 150)

			// check VMI, confirm migration state
			libmigration.ConfirmVMIPostMigration(virtClient, vmi, migration)
			libmigration.ConfirmMigrationMode(virtClient, vmi, v1.MigrationPostCopy)
		},
			Entry("a migration policy", applyWithMigrationPolicy),
			Entry("the Kubevirt CR (plus a minimal downtime-tuning 1ms policy)", Serial, applyWithKubevirtCR),
		)

		Context("with a guest-agent-ping liveness probe", func() {
			// Regression test for probe suppression on the migration source pod
			// during post-copy. Once post-copy starts the source VM is handed off
			// to the target and enters a ghost/paused state, making it unreachable
			// via the QEMU guest agent. Because virt-launcher pods use
			// restartPolicy: Never, the failed probe kills the compute container
			// and the pod enters Failed phase, aborting the migration.
			It("[QUARANTINE] should complete post-copy migration without the liveness probe killing the source pod", decorators.Quarantine, func() {
				By("Creating a Fedora VMI with a GuestAgentPing liveness probe")
				vmi := libvmifact.NewFedora(
					libnet.WithMasqueradeNetworking(),
					libvmi.WithRng(),
					libvmi.WithGuestAgentPingLivenessProbe(120, 5, 2),
				)

				By("Applying the post-copy migration policy")
				AlignPolicyAndVmi(vmi, migrationPolicy)
				migrationPolicy = CreateMigrationPolicy(virtClient, migrationPolicy)

				By("Starting the VirtualMachineInstance")
				vmi = libvmops.RunVMIAndExpectLaunch(vmi, flags.StartupTimeoutSecondsHuge())

				By("Waiting for the guest agent to connect")
				Eventually(matcher.ThisVMI(vmi), 5*time.Minute, 2*time.Second).Should(matcher.HaveConditionTrue(v1.VirtualMachineInstanceAgentConnected))

				By("Logging into the VMI")
				Expect(console.LoginToFedora(vmi)).To(Succeed())

				By("Running a stress test to dirty pages and trigger post-copy")
				runStressTest(vmi)

				By("Starting the migration")
				migration := libmigration.New(vmi.Name, vmi.Namespace)
				migration = libmigration.RunMigrationAndExpectToCompleteWithDefaultTimeout(virtClient, migration)

				By("Confirming the VMI migrated successfully via post-copy and is still running")
				vmi = libmigration.ConfirmVMIPostMigration(virtClient, vmi, migration)
				libmigration.ConfirmMigrationMode(virtClient, vmi, v1.MigrationPostCopy)
			})
		})

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
					libvmi.WithMemoryRequest("3Gi"),
					libvmi.WithRng(),
					libvmi.WithNamespace(testsuite.NamespaceTestDefault),
				)

				By("applying post-copy settings (1ms downtime prevents precopy convergence)")
				applyKubevirtCR()
				downtimePolicy := applyDowntimeTuningOnlyPolicy(vmi)

				vm := libvmi.NewVirtualMachine(vmi, libvmi.WithRunStrategy(v1.RunStrategyRerunOnFailure))
				vm, err := virtClient.VirtualMachine(vm.Namespace).Create(context.Background(), vm, metav1.CreateOptions{})
				Expect(err).ToNot(HaveOccurred())

				By("Waiting for the VirtualMachine to be ready")
				vmi = libwait.WaitUntilVMIReady(vmi, console.LoginToFedora)

				// Need to wait for cloud init to finish and start the agent inside the vmi.
				Eventually(matcher.ThisVMI(vmi), 12*time.Minute, 2*time.Second).Should(matcher.HaveConditionTrue(v1.VirtualMachineInstanceAgentConnected))

				runStressTest(vmi)

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

				By("updating the migration configuration to default values")
				migrationPolicy.Spec.CompletionTimeoutPerGiB = nil
				migrationPolicy.Spec.MaxDowntimeMs = nil
				migrationPolicy.Spec.ExperimentalMigrationOptions = nil
				applyKubevirtCR()
				Expect(virtClient.MigrationPolicy().Delete(context.Background(), downtimePolicy.Name, metav1.DeleteOptions{})).To(Succeed())

				By("Ensuring the virtual machine is restarted")
				Eventually(matcher.ThisVMI(vmi), 5*time.Minute, 1*time.Second).Should(matcher.BeRestarted(vmi.UID))

				By("Ensuring the virtual machine is migratable")
				migration = libmigration.New(vmi.Name, vmi.Namespace)
				libmigration.RunMigrationAndExpectToComplete(virtClient, migration, 150)
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
		libvmi.WithInterface(libvmi.NewInterface(v1.DefaultPodNetwork().Name, libvmi.WithMasqueradeBinding())),
		libvmi.WithNetwork(v1.DefaultPodNetwork()),
		libvmi.WithPersistentVolumeClaim("disk0", pvName),
		libvmi.WithMemoryRequest(memoryRequestSize),
		libvmi.WithRng(),
		libvmi.WithCloudInitNoCloud(libvmici.WithNoCloudEncodedUserData(mountSvcAccCommands)),
		libvmi.WithServiceAccountDisk("default"),
	)

	mode := v1.MigrationPreCopy
	if migrationPolicy != nil && migrationPolicy.Spec.AllowPostCopy != nil && *migrationPolicy.Spec.AllowPostCopy {
		mode = v1.MigrationPostCopy
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
		runStressTest(vmi)
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
