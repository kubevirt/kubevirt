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

	"kubevirt.io/kubevirt/tests/libmigration"

	migrationsv1 "kubevirt.io/api/migrations/v1alpha1"

	kvpointer "kubevirt.io/kubevirt/pkg/pointer"

	"kubevirt.io/kubevirt/tests/framework/kubevirt"
	"kubevirt.io/kubevirt/tests/framework/matcher"
	"kubevirt.io/kubevirt/tests/testsuite"

	"kubevirt.io/kubevirt/tests/framework/checks"
	"kubevirt.io/kubevirt/tests/libdv"

	expect "github.com/google/goexpect"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gstruct"
	gomegatypes "github.com/onsi/gomega/types"
	k8sv1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/rand"

	"kubevirt.io/kubevirt/tests/libvmi"

	. "kubevirt.io/kubevirt/tests/framework/matcher"

	v1 "kubevirt.io/api/core/v1"
	"kubevirt.io/client-go/kubecli"
	cdiv1 "kubevirt.io/containerized-data-importer-api/pkg/apis/core/v1beta1"

	"kubevirt.io/kubevirt/tests"
	"kubevirt.io/kubevirt/tests/console"
	cd "kubevirt.io/kubevirt/tests/containerdisk"
	"kubevirt.io/kubevirt/tests/flags"
	"kubevirt.io/kubevirt/tests/libnet"
	"kubevirt.io/kubevirt/tests/libpod"
	"kubevirt.io/kubevirt/tests/libstorage"
)

var _ = SIGMigrationDescribe("VM Post Copy Live Migration", func() {
	var (
		virtClient kubecli.KubevirtClient
		err        error
	)

	BeforeEach(func() {
		checks.SkipIfMigrationIsNotPossible()
		virtClient = kubevirt.Client()
	})

	waitUntilMigrationMode := func(vmi *v1.VirtualMachineInstance, expectedMode v1.MigrationMode, timeout int) *v1.VirtualMachineInstance {
		By("Waiting until migration status")
		EventuallyWithOffset(2, func() v1.MigrationMode {
			By("Retrieving the VMI post migration")
			vmi, err = virtClient.VirtualMachineInstance(vmi.Namespace).Get(context.Background(), vmi.Name, &metav1.GetOptions{})
			Expect(err).ToNot(HaveOccurred())
			if vmi.Status.MigrationState != nil {
				return vmi.Status.MigrationState.Mode
			}
			return v1.MigrationPreCopy
		}, timeout, 1*time.Second).Should(Equal(expectedMode), fmt.Sprintf("migration should be in %s after %d s", expectedMode, timeout))
		return vmi
	}

	Describe("Starting a VirtualMachineInstance ", func() {

		Context("migration postcopy", func() {

			var migrationPolicy *migrationsv1.MigrationPolicy

			BeforeEach(func() {
				By("Allowing post-copy and limit migration bandwidth")
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
						libdv.WithPVC(
							libdv.PVCWithStorageClass(sc),
							libdv.PVCWithVolumeSize(cd.FedoraVolumeSize),
							libdv.PVCWithReadWriteManyAccessMode(),
						),
					)

					dv, err = virtClient.CdiClient().CdiV1beta1().DataVolumes(testsuite.NamespacePrivileged).Create(context.Background(), dv, metav1.CreateOptions{})
					Expect(err).ToNot(HaveOccurred())
				})

				AfterEach(func() {
					libstorage.DeleteDataVolume(&dv)
				})

				It("[test_id:5004] should be migrated successfully, using guest agent on VM with postcopy", func() {
					VMIMigrationWithGuestaAgent(virtClient, dv.Name, "1Gi", migrationPolicy)
				})

			})

			Context("should migrate using for postcopy", func() {

				applyMigrationPolicy := func(vmi *v1.VirtualMachineInstance) {
					AlignPolicyAndVmi(vmi, migrationPolicy)
					migrationPolicy = CreateMigrationPolicy(virtClient, migrationPolicy)
				}

				applyKubevirtCR := func() {
					config := getCurrentKvConfig(virtClient)
					config.MigrationConfiguration.AllowPostCopy = migrationPolicy.Spec.AllowPostCopy
					config.MigrationConfiguration.CompletionTimeoutPerGiB = migrationPolicy.Spec.CompletionTimeoutPerGiB
					config.MigrationConfiguration.BandwidthPerMigration = migrationPolicy.Spec.BandwidthPerMigration
					tests.UpdateKubeVirtConfigValueAndWait(config)
				}

				type applySettingsType string
				const (
					applyWithMigrationPolicy applySettingsType = "policy"
					applyWithKubevirtCR      applySettingsType = "kubevirt"
				)

				DescribeTable("[test_id:4747] using", func(settingsType applySettingsType) {
					vmi := libvmi.NewFedora(libnet.WithMasqueradeNetworking()...)
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
					vmi = tests.RunVMIAndExpectLaunch(vmi, 240)

					By("Checking that the VirtualMachineInstance console has expected output")
					Expect(console.LoginToFedora(vmi)).To(Succeed())

					// Need to wait for cloud init to finish and start the agent inside the vmi.
					Eventually(matcher.ThisVMI(vmi), 12*time.Minute, 2*time.Second).Should(matcher.HaveConditionTrue(v1.VirtualMachineInstanceAgentConnected))

					runStressTest(vmi, "350M", stressDefaultSleepDuration)

					// execute a migration, wait for finalized state
					By("Starting the Migration")
					migration := libmigration.New(vmi.Name, vmi.Namespace)
					migration = libmigration.RunMigrationAndExpectToComplete(virtClient, migration, 150)

					// check VMI, confirm migration state
					libmigration.ConfirmVMIPostMigration(virtClient, vmi, migration)
					confirmMigrationMode(virtClient, vmi, v1.MigrationPostCopy)
				},
					Entry("a migration policy", applyWithMigrationPolicy),
					Entry("[Serial] Kubevirt CR", Serial, applyWithKubevirtCR),
				)

				Context("[Serial] and fail", Serial, func() {
					var createdPods []string
					BeforeEach(func() {
						createdPods = []string{}
					})

					createLargeVirtualMachine := func(namespace string) *v1.VirtualMachine {
						vmi := libvmi.NewFedora(libnet.WithMasqueradeNetworking()...)
						vmi.Namespace = namespace
						vmi.Spec.Domain.Resources.Requests[k8sv1.ResourceMemory] = resource.MustParse("3Gi")
						vmi.Spec.Domain.Devices.Rng = &v1.Rng{}
						vm := libvmi.NewVirtualMachine(vmi)

						vm, err := virtClient.VirtualMachine(testsuite.NamespacePrivileged).Create(context.Background(), vm)
						Expect(err).ToNot(HaveOccurred())
						return vm
					}

					runMigrationKillerPod := func(nodeName string) {
						podName := fmt.Sprintf("migration-killer-pod-%s", rand.String(5))

						// kill the handler
						pod := libpod.RenderPrivilegedPod(podName, []string{"/bin/bash", "-c"}, []string{fmt.Sprintf("while true; do pkill -9 virt-handler && sleep 5; done")})

						pod.Spec.NodeSelector = map[string]string{"kubernetes.io/hostname": nodeName}
						createdPod, err := virtClient.CoreV1().Pods(pod.Namespace).Create(context.Background(), pod, metav1.CreateOptions{})
						Expect(err).ToNot(HaveOccurred(), "Should create helper pod")
						createdPods = append(createdPods, createdPod.Name)
						Expect(createdPods).ToNot(BeEmpty(), "There is no node for migration")
					}

					removeMigrationKillerPod := func() {
						for _, podName := range createdPods {
							Eventually(func() error {
								err := virtClient.CoreV1().Pods(testsuite.NamespacePrivileged).Delete(context.Background(), podName, metav1.DeleteOptions{})
								return err
							}, 10*time.Second, 1*time.Second).Should(MatchError(errors.IsNotFound, "k8serrors.IsNotFound"), "Should delete helper pod")

							Eventually(func() error {
								_, err := virtClient.CoreV1().Pods(testsuite.NamespacePrivileged).Get(context.Background(), podName, metav1.GetOptions{})
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
					}
					It("should make sure that VM restarts after failure", func() {

						By("creating a large VM with RunStrategyRerunOnFailure")
						vm := createLargeVirtualMachine(testsuite.NamespacePrivileged)

						// update the migration policy to ensure slow pre-copy migration progress instead of an immidiate cancelation.
						migrationPolicy.Spec.CompletionTimeoutPerGiB = kvpointer.P(int64(20))
						migrationPolicy.Spec.BandwidthPerMigration = kvpointer.P(resource.MustParse("1Mi"))
						applyKubevirtCR()

						By("Starting the VirtualMachine")
						vm = tests.RunVMAndExpectLaunchWithRunStrategy(virtClient, vm, v1.RunStrategyRerunOnFailure)
						vmi, err := virtClient.VirtualMachineInstance(vm.Namespace).Get(context.Background(), vm.Name, &metav1.GetOptions{})
						Expect(err).ToNot(HaveOccurred())

						By("Checking that the VirtualMachineInstance console has expected output")
						Expect(console.LoginToFedora(vmi)).To(Succeed())

						// Need to wait for cloud init to finish and start the agent inside the vmi.
						Eventually(matcher.ThisVMI(vmi), 12*time.Minute, 2*time.Second).Should(matcher.HaveConditionTrue(v1.VirtualMachineInstanceAgentConnected))

						runStressTest(vmi, "350M", stressDefaultSleepDuration)

						By("Starting the Migration")
						migration := libmigration.New(vmi.Name, vmi.Namespace)
						migration = libmigration.RunMigration(virtClient, migration)

						// check VMI, confirm migration state
						waitUntilMigrationMode(vmi, v1.MigrationPostCopy, 300)

						// launch killer pod on every node that isn't the vmi's node
						By("Starting migration killer pods")
						runMigrationKillerPod(vmi.Status.NodeName)

						By("Making sure that post-copy migration failed")
                        Eventually(matcher.ThisMigration(migration), 150, 1*time.Second).Should(BeInPhase(v1.MigrationFailed))

						By("Removing migration killer pods")
						removeMigrationKillerPod()

						By("Ensuring the VirtualMachineInstance is restarted")
						Eventually(ThisVMI(vmi), 240*time.Second, 1*time.Second).Should(beRestarted(vmi.UID))
					})
				})
			})
		})
	})
})

func beRestarted(oldUID types.UID) gomegatypes.GomegaMatcher {
	return gstruct.PointTo(gstruct.MatchFields(gstruct.IgnoreExtras, gstruct.Fields{
		"ObjectMeta": gstruct.MatchFields(gstruct.IgnoreExtras, gstruct.Fields{
			"UID": Not(Equal(oldUID)),
		}),
		"Status": gstruct.MatchFields(gstruct.IgnoreExtras, gstruct.Fields{
			"Phase": Equal(v1.Running),
		}),
	}))
}

func VMIMigrationWithGuestaAgent(virtClient kubecli.KubevirtClient, pvName string, memoryRequestSize string, migrationPolicy *migrationsv1.MigrationPolicy) {
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
		libvmi.WithCloudInitNoCloudEncodedUserData(mountSvcAccCommands),
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
	vmi = tests.RunVMIAndExpectLaunchIgnoreWarnings(vmi, 180)

	// Wait for cloud init to finish and start the agent inside the vmi.
	Eventually(matcher.ThisVMI(vmi), 12*time.Minute, 2*time.Second).Should(matcher.HaveConditionTrue(v1.VirtualMachineInstanceAgentConnected))

	By("Checking that the VirtualMachineInstance console has expected output")
	Expect(console.LoginToFedora(vmi)).To(Succeed(), "Should be able to login to the Fedora VM")

	if mode == v1.MigrationPostCopy {
		By("Running stress test to allow transition to post-copy")
		runStressTest(vmi, stressLargeVMSize, stressDefaultSleepDuration)
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
	confirmMigrationMode(virtClient, vmi, mode)

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

func confirmMigrationMode(virtClient kubecli.KubevirtClient, vmi *v1.VirtualMachineInstance, expectedMode v1.MigrationMode) {
	By("Retrieving the VMI post migration")
	vmi, err := virtClient.VirtualMachineInstance(vmi.Namespace).Get(context.Background(), vmi.Name, &metav1.GetOptions{})
	Expect(err).ToNot(HaveOccurred())

	By("Verifying the VMI's migration mode")
	Expect(vmi.Status.MigrationState.Mode).To(Equal(expectedMode))
}
