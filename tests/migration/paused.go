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
	"fmt"
	"time"

	"kubevirt.io/kubevirt/tests/decorators"
	kvconfig "kubevirt.io/kubevirt/tests/libkubevirt/config"
	"kubevirt.io/kubevirt/tests/libmigration"

	migrationsv1 "kubevirt.io/api/migrations/v1alpha1"

	"kubevirt.io/kubevirt/pkg/libvmi"

	kvpointer "kubevirt.io/kubevirt/pkg/pointer"

	"kubevirt.io/kubevirt/tests/framework/kubevirt"

	"kubevirt.io/kubevirt/tests/framework/matcher"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"k8s.io/apimachinery/pkg/api/resource"
	"k8s.io/apimachinery/pkg/util/rand"

	"kubevirt.io/kubevirt/tests/libvmifact"

	v1 "kubevirt.io/api/core/v1"
	"kubevirt.io/client-go/kubecli"

	"kubevirt.io/kubevirt/tests/console"
	"kubevirt.io/kubevirt/tests/libnet"
	"kubevirt.io/kubevirt/tests/libvmops"
)

var _ = Describe(SIG("Live Migrate A Paused VMI", decorators.RequiresTwoSchedulableNodes, func() {
	var (
		virtClient kubecli.KubevirtClient
	)

	BeforeEach(func() {
		virtClient = kubevirt.Client()
	})

	Describe("Starting a VirtualMachineInstance ", func() {

		Context("paused vmi during migration", func() {

			var migrationPolicy *migrationsv1.MigrationPolicy

			BeforeEach(func() {
				By("enable AllowWorkloadDisruption and limit migration bandwidth")
				policyName := fmt.Sprintf("testpolicy-%s", rand.String(5))
				migrationPolicy = kubecli.NewMinimalMigrationPolicy(policyName)
				migrationPolicy.Spec.AllowWorkloadDisruption = kvpointer.P(true)
				migrationPolicy.Spec.AllowPostCopy = kvpointer.P(false)
				migrationPolicy.Spec.CompletionTimeoutPerGiB = kvpointer.P(int64(20))
				migrationPolicy.Spec.BandwidthPerMigration = kvpointer.P(resource.MustParse("5Mi"))
			})

			Context("should migrate paused", func() {

				applyMigrationPolicy := func(vmi *v1.VirtualMachineInstance) {
					AlignPolicyAndVmi(vmi, migrationPolicy)
					migrationPolicy = CreateMigrationPolicy(virtClient, migrationPolicy)
				}

				applyKubevirtCR := func() {
					config := getCurrentKvConfig(virtClient)
					config.MigrationConfiguration.AllowPostCopy = migrationPolicy.Spec.AllowPostCopy
					config.MigrationConfiguration.AllowWorkloadDisruption = migrationPolicy.Spec.AllowWorkloadDisruption
					config.MigrationConfiguration.CompletionTimeoutPerGiB = migrationPolicy.Spec.CompletionTimeoutPerGiB
					config.MigrationConfiguration.BandwidthPerMigration = migrationPolicy.Spec.BandwidthPerMigration
					kvconfig.UpdateKubeVirtConfigValueAndWait(config)
				}

				type applySettingsType string
				const (
					applyWithMigrationPolicy applySettingsType = "policy"
					applyWithKubevirtCR      applySettingsType = "kubevirt"
					expectSuccess                              = true
					expectFailure                              = false
				)

				Context("when acceptable time exceeded and post-copy is forbidden", func() {
					DescribeTable("should pause the VMI and ", func(expectSuccess bool, bandwidth string, settingsType applySettingsType) {

						By("creating a large Virtual Machine Instance")
						vmi := libvmifact.NewFedora(
							libnet.WithMasqueradeNetworking(),
							libvmi.WithMemoryRequest("512Mi"),
							libvmi.WithRng())

						// update the migration policy to ensure slow pre-copy migration progress instead of an immediate cancellation.
						migrationPolicy.Spec.BandwidthPerMigration = kvpointer.P(resource.MustParse(bandwidth))
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

						By("Starting the Migration")
						migration := libmigration.New(vmi.Name, vmi.Namespace)
						migration = libmigration.RunMigration(virtClient, migration)

						// check VMI, confirm migration state
						libmigration.WaitUntilMigrationMode(virtClient, vmi, v1.MigrationPaused, 5*time.Minute)

						if expectSuccess {
							libmigration.ExpectMigrationToSucceed(virtClient, migration, 100)
						} else {
							Eventually(matcher.ThisMigration(migration), 150*time.Second, 1*time.Second).Should(matcher.BeInPhase(v1.MigrationFailed))
						}

						By("Making sure that the VMI is unpaused")
						Eventually(matcher.ThisVMI(vmi), 30*time.Second, time.Second).Should(matcher.HaveConditionMissingOrFalse(v1.VirtualMachineInstancePaused))

					},
						Entry("migrate successfully (migration policy)", expectSuccess, "10Mi", applyWithMigrationPolicy),
						Entry("migrate successfully (CR change)", Serial, expectSuccess, "10Mi", applyWithKubevirtCR),
					)
				})
			})
		})
	})
}))
