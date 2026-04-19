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

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	k8sv1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	"k8s.io/apimachinery/pkg/util/rand"

	v1 "kubevirt.io/api/core/v1"
	"kubevirt.io/client-go/kubecli"

	"kubevirt.io/kubevirt/pkg/libvmi"
	"kubevirt.io/kubevirt/pkg/pointer"
	"kubevirt.io/kubevirt/pkg/virt-config/featuregate"
	"kubevirt.io/kubevirt/tests/console"
	"kubevirt.io/kubevirt/tests/decorators"
	"kubevirt.io/kubevirt/tests/framework/kubevirt"
	"kubevirt.io/kubevirt/tests/framework/matcher"
	kvconfig "kubevirt.io/kubevirt/tests/libkubevirt/config"
	"kubevirt.io/kubevirt/tests/libmigration"
	"kubevirt.io/kubevirt/tests/libnet"
	"kubevirt.io/kubevirt/tests/libvmifact"
	"kubevirt.io/kubevirt/tests/libvmops"
	"kubevirt.io/kubevirt/tests/testsuite"
)

// getSourceLauncherLogs fetches the "compute" container logs from the
// virt-launcher pod that was the migration source.
func getSourceLauncherLogs(virtClient kubecli.KubevirtClient, vmi *v1.VirtualMachineInstance) string {
	GinkgoHelper()
	Expect(vmi.Status.MigrationState).ToNot(BeNil())
	sourcePod := vmi.Status.MigrationState.SourcePod
	Expect(sourcePod).ToNot(BeEmpty(), "source pod name must be populated after migration")

	logsRaw, err := virtClient.CoreV1().
		Pods(vmi.Namespace).
		GetLogs(sourcePod, &k8sv1.PodLogOptions{Container: "compute"}).
		DoRaw(context.Background())
	Expect(err).ToNot(HaveOccurred(), "should get virt-launcher source pod logs")
	return string(logsRaw)
}

var _ = Describe(SIG("Advanced Live Migration", decorators.RequiresTwoSchedulableNodes, Serial, func() {
	var virtClient kubecli.KubevirtClient

	BeforeEach(func() {
		virtClient = kubevirt.Client()
	})

	Context("with AdvancedLiveMigration feature gate enabled", func() {
		BeforeEach(func() {
			kvconfig.EnableFeatureGate(featuregate.AdvancedLiveMigration)
		})

		AfterEach(func() {
			kvconfig.DisableFeatureGate(featuregate.AdvancedLiveMigration)
		})

		It("should migrate with zstd compression and confirm via API and logs", func() {
			vmi := libvmifact.NewAlpineWithTestTooling(libnet.WithMasqueradeNetworking())

			By("Creating a migration policy with compression enabled")
			policy := GeneratePolicyAndAlignVMI(vmi)
			policy.Spec.Experimental = &v1.ExperimentalMigrationConfiguration{
				Compression: pointer.P(v1.MigrationCompressionZstd),
			}
			policy = CreateMigrationPolicy(virtClient, policy)

			By("Starting the VirtualMachineInstance")
			vmi = libvmops.RunVMIAndExpectLaunch(vmi, libvmops.StartupTimeoutSecondsHuge)

			By("Starting the Migration")
			migration := libmigration.New(vmi.Name, vmi.Namespace)
			migration = libmigration.RunMigrationAndExpectToCompleteWithDefaultTimeout(virtClient, migration)

			By("Confirming migration completed successfully")
			vmi = libmigration.ConfirmVMIPostMigration(virtClient, vmi, migration)

			By("Verifying the migration policy was applied")
			Expect(vmi.Status.MigrationState.MigrationPolicyName).ToNot(BeNil())
			Expect(*vmi.Status.MigrationState.MigrationPolicyName).To(Equal(policy.Name))

			By("Verifying compression is set in the migration configuration")
			Expect(vmi.Status.MigrationState.MigrationConfiguration).ToNot(BeNil())
			Expect(vmi.Status.MigrationState.MigrationConfiguration.Experimental).ToNot(BeNil())
			Expect(vmi.Status.MigrationState.MigrationConfiguration.Experimental.Compression).ToNot(BeNil())
			Expect(*vmi.Status.MigrationState.MigrationConfiguration.Experimental.Compression).To(Equal(v1.MigrationCompressionZstd))

			By("Verifying compression and multifd were applied via launcher logs")
			logs := getSourceLauncherLogs(virtClient, vmi)
			Expect(logs).To(ContainSubstring("Migration compression enabled: method=zstd"))
			Expect(logs).To(ContainSubstring("CompressionSet:true"))
			Expect(logs).To(ContainSubstring("ParallelConnectionsSet:true"))
		})

		It("should migrate with compression, autoconverge, and postcopy combined", func() {
			vmi := libvmifact.NewFedora(libnet.WithMasqueradeNetworking(), libvmi.WithMemoryRequest(fedoraVMSize))
			vmi.Namespace = testsuite.NamespacePrivileged

			By("Creating a migration policy with compression + postcopy + autoconverge")
			policyName := fmt.Sprintf("testpolicy-combo-%s", rand.String(5))
			policy := kubecli.NewMinimalMigrationPolicy(policyName)
			policy.Spec.AllowAutoConverge = pointer.P(true)
			policy.Spec.AllowPostCopy = pointer.P(true)
			policy.Spec.CompletionTimeoutPerGiB = pointer.P(int64(1))
			policy.Spec.BandwidthPerMigration = pointer.P(resource.MustParse("5Mi"))
			policy.Spec.Experimental = &v1.ExperimentalMigrationConfiguration{
				Compression: pointer.P(v1.MigrationCompressionZstd),
			}
			AlignPolicyAndVmi(vmi, policy)
			policy = CreateMigrationPolicy(virtClient, policy)

			By("Starting the VirtualMachineInstance")
			vmi = libvmops.RunVMIAndExpectLaunch(vmi, libvmops.StartupTimeoutSecondsHuge)

			Eventually(matcher.ThisVMI(vmi), 12*time.Minute, 2*time.Second).Should(matcher.HaveConditionTrue(v1.VirtualMachineInstanceAgentConnected))

			By("Checking that the VirtualMachineInstance console has expected output")
			Expect(console.LoginToFedora(vmi)).To(Succeed())

			By("Running stress test to generate memory pressure for postcopy")
			runStressTest(vmi, stressLargeVMSize)

			By("Starting the Migration")
			migration := libmigration.New(vmi.Name, vmi.Namespace)
			migration = libmigration.RunMigrationAndExpectToComplete(virtClient, migration, 180)

			By("Confirming migration completed successfully")
			vmi = libmigration.ConfirmVMIPostMigration(virtClient, vmi, migration)
			libmigration.ConfirmMigrationMode(virtClient, vmi, v1.MigrationPostCopy)

			By("Verifying the migration policy was applied")
			Expect(vmi.Status.MigrationState.MigrationPolicyName).ToNot(BeNil())
			Expect(*vmi.Status.MigrationState.MigrationPolicyName).To(Equal(policy.Name))

			By("Verifying the migration configuration contains all settings")
			mc := vmi.Status.MigrationState.MigrationConfiguration
			Expect(mc).ToNot(BeNil())
			Expect(mc.AllowAutoConverge).ToNot(BeNil())
			Expect(*mc.AllowAutoConverge).To(BeTrue())
			Expect(mc.AllowPostCopy).ToNot(BeNil())
			Expect(*mc.AllowPostCopy).To(BeTrue())
			Expect(mc.Experimental).ToNot(BeNil())
			Expect(mc.Experimental.Compression).ToNot(BeNil())
			Expect(*mc.Experimental.Compression).To(Equal(v1.MigrationCompressionZstd))

			By("Verifying compression, autoconverge, postcopy, and multifd via launcher logs")
			logs := getSourceLauncherLogs(virtClient, vmi)
			Expect(logs).To(ContainSubstring("Migration compression enabled: method=zstd"))
			Expect(logs).To(ContainSubstring("CompressionSet:true"))
			Expect(logs).To(ContainSubstring("ParallelConnectionsSet:true"))
		})
	})

	Context("without AdvancedLiveMigration feature gate", func() {
		BeforeEach(func() {
			kvconfig.DisableFeatureGate(featuregate.AdvancedLiveMigration)
		})

		It("should ignore experimental compression when the feature gate is disabled", func() {
			vmi := libvmifact.NewAlpineWithTestTooling(libnet.WithMasqueradeNetworking())

			By("Creating a migration policy with compression (but gate is off)")
			policy := GeneratePolicyAndAlignVMI(vmi)
			policy.Spec.Experimental = &v1.ExperimentalMigrationConfiguration{
				Compression: pointer.P(v1.MigrationCompressionZstd),
			}
			CreateMigrationPolicy(virtClient, policy)

			By("Starting the VirtualMachineInstance")
			vmi = libvmops.RunVMIAndExpectLaunch(vmi, libvmops.StartupTimeoutSecondsHuge)

			By("Starting the Migration")
			migration := libmigration.New(vmi.Name, vmi.Namespace)
			migration = libmigration.RunMigrationAndExpectToCompleteWithDefaultTimeout(virtClient, migration)

			By("Confirming migration completed successfully")
			vmi = libmigration.ConfirmVMIPostMigration(virtClient, vmi, migration)

			By("Verifying the experimental section is absent from the migration configuration")
			if vmi.Status.MigrationState.MigrationConfiguration != nil {
				Expect(vmi.Status.MigrationState.MigrationConfiguration.Experimental).To(BeNil(),
					"experimental config should not be propagated when AdvancedLiveMigration gate is disabled")
			}

			By("Verifying compression was not applied via launcher logs")
			logs := getSourceLauncherLogs(virtClient, vmi)
			Expect(logs).ToNot(ContainSubstring("Migration compression enabled"))
			Expect(logs).ToNot(ContainSubstring("CompressionSet:true"))
		})
	})
}))
