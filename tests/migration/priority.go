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
	policyv1 "k8s.io/api/policy/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	v1 "kubevirt.io/api/core/v1"
	"kubevirt.io/client-go/kubecli"

	"kubevirt.io/kubevirt/pkg/libvmi"
	"kubevirt.io/kubevirt/pkg/pointer"
	"kubevirt.io/kubevirt/pkg/virt-config/featuregate"
	"kubevirt.io/kubevirt/tests/decorators"
	"kubevirt.io/kubevirt/tests/framework/k8s"
	"kubevirt.io/kubevirt/tests/framework/kubevirt"
	"kubevirt.io/kubevirt/tests/framework/matcher"
	"kubevirt.io/kubevirt/tests/libkubevirt/config"
	"kubevirt.io/kubevirt/tests/libmigration"
	"kubevirt.io/kubevirt/tests/libnet"
	"kubevirt.io/kubevirt/tests/libpod"
	"kubevirt.io/kubevirt/tests/libvmifact"
	"kubevirt.io/kubevirt/tests/libwait"
	"kubevirt.io/kubevirt/tests/testsuite"
)

var _ = Describe(SIG("Live Migrations with priority", decorators.RequiresTwoSchedulableNodes, Serial, func() {
	var virtClient kubecli.KubevirtClient

	BeforeEach(func() {
		virtClient = kubevirt.Client()

		cfg := getCurrentKvConfig(virtClient)
		cfg.MigrationConfiguration = &v1.MigrationConfiguration{
			ParallelMigrationsPerCluster: pointer.P(uint32(1)),
		}
		if cfg.DeveloperConfiguration == nil {
			cfg.DeveloperConfiguration = &v1.DeveloperConfiguration{
				FeatureGates: []string{},
			}
		}
		cfg.DeveloperConfiguration.FeatureGates = append(cfg.DeveloperConfiguration.FeatureGates, featuregate.MigrationPriorityQueue)
		cfg.DeveloperConfiguration.LogVerbosity = &v1.LogVerbosity{
			VirtController: 9,
		}
		config.UpdateKubeVirtConfigValueAndWait(cfg)
	})

	It("with a live-migrate eviction strategy set", Serial, func() {
		var vmis []*v1.VirtualMachineInstance
		for i := 0; i < 5; i++ {
			vmi := libvmifact.NewAlpine(
				libnet.WithMasqueradeNetworking(),
				libvmi.WithEvictionStrategy(v1.EvictionStrategyLiveMigrate),
				libvmi.WithNamespace(testsuite.GetTestNamespace(nil)),
			)
			vmis = append(vmis, vmi)
		}

		By("starting five VMIs")
		for _, vmi := range vmis {
			_, err := virtClient.VirtualMachineInstance(vmi.Namespace).Create(context.Background(), vmi, metav1.CreateOptions{})
			Expect(err).ToNot(HaveOccurred())
		}

		By("waiting until the VMIs are ready")
		for i, vmi := range vmis {
			vmis[i] = libwait.WaitForSuccessfulVMIStart(vmi,
				libwait.WithTimeout(180),
			)
		}

		// - Start a slow vmim for the vmis[0] (so that it blocks the other being scheduled)
		// - manual migrate the vmis[1]
		// - manual migrate the vmis[2]
		// - evict the launcher pod of the vmis[3]
		// - evict the launcher pod of the vmis[4]
		// - Expect vmis[3] and vmis[4] are the first 2 migrations after vmis[0]

		var podsToEvict []*k8sv1.Pod
		pod1, err := libpod.GetPodByVirtualMachineInstance(vmis[3], vmis[4].Namespace)
		Expect(err).NotTo(HaveOccurred())
		podsToEvict = append(podsToEvict, pod1)
		pod2, err := libpod.GetPodByVirtualMachineInstance(vmis[4], vmis[4].Namespace)
		Expect(err).NotTo(HaveOccurred())
		podsToEvict = append(podsToEvict, pod2)

		// - Start a vmim for the vmis[0]
		By("Creating a migration policy that overrides cluster policy")
		policy := GeneratePolicyAndAlignVMI(vmis[0])
		policy.Spec.BandwidthPerMigration = pointer.P(resource.MustParse("1Ki"))
		_, err = virtClient.MigrationPolicy().Create(context.Background(), policy, metav1.CreateOptions{})
		Expect(err).ToNot(HaveOccurred())

		By(fmt.Sprintf("Migrate the VirtualMachineInstance %s", vmis[0].Name))
		migration := libmigration.RunMigration(virtClient, libmigration.New(vmis[0].Name, vmis[0].Namespace))

		Eventually(matcher.ThisMigration(migration), 10*time.Second, 500*time.Millisecond).Should(matcher.BeInPhase(v1.Scheduling), fmt.Sprintf("migration be scheduled"))

		// - manual migrate the vmis[1]
		// - manual migrate the vmis[2]
		for i := 1; i < 3; i++ {
			By(fmt.Sprintf("Migrate the VirtualMachineInstance %s", vmis[i].Name))
			libmigration.RunMigration(virtClient, libmigration.New(vmis[i].Name, vmis[i].Namespace))
		}

		// - evict the launcher pod of the vmis[3]
		// - evict the launcher pod of the vmis[4]
		for _, pod := range podsToEvict {
			By("calling evict on VMI's pod")
			err = k8s.Client().CoreV1().Pods(pod.Namespace).EvictV1(context.Background(), &policyv1.Eviction{ObjectMeta: metav1.ObjectMeta{Name: pod.Name}})
			// The "too many requests" err is what get's returned when an
			// eviction would invalidate a pdb. This is what we want to see here.
			Expect(err).To(MatchError(errors.IsTooManyRequests, "too many requests should be returned as way of blocking eviction"))
			Expect(err).To(MatchError(ContainSubstring("Eviction triggered evacuation of VMI")))
		}

		Eventually(func() (int, error) {
			migList, err := virtClient.VirtualMachineInstanceMigration(testsuite.GetTestNamespace(nil)).List(context.Background(), metav1.ListOptions{})
			return len(migList.Items), err
		}, 30*time.Second, 1*time.Second).Should(BeNumerically(">=", 5))

		//Delete the slow blocking Migration
		err = virtClient.VirtualMachineInstanceMigration(migration.Namespace).Delete(context.Background(), migration.Name, metav1.DeleteOptions{})
		Expect(err).To(Or(
			Not(HaveOccurred()),
			MatchError(errors.IsNotFound, "errors.IsNotFound"),
		))

		migrationStarTimestamp := make(map[string]*metav1.Time)
		for i := 1; i < 5; i++ {
			Eventually(func() (bool, error) {
				vmi, err := virtClient.VirtualMachineInstance(vmis[i].Namespace).Get(context.Background(), vmis[i].Name, metav1.GetOptions{})
				return vmi.Status.MigrationState != nil && vmi.Status.MigrationState.StartTimestamp != nil && vmi.Status.MigrationState.EndTimestamp != nil, err
			}, libmigration.MigrationWaitTime, 1*time.Second).Should(BeTrue(), fmt.Sprintf("vmi %s should be migrated", vmis[i].Name))

			vmi, err := virtClient.VirtualMachineInstance(vmis[i].Namespace).Get(context.Background(), vmis[i].Name, metav1.GetOptions{})
			Expect(err).NotTo(HaveOccurred())
			migrationStarTimestamp[vmi.Name] = vmi.Status.MigrationState.StartTimestamp
		}

		// - Expect vmis[3] and vmis[4] are the first 2 migrations
		// vmis[3] and vmis[4] have the same priority and we cannot assert anything
		Expect(migrationStarTimestamp[vmis[3].Name].Before(migrationStarTimestamp[vmis[1].Name])).To(BeTrue())
		Expect(migrationStarTimestamp[vmis[3].Name].Before(migrationStarTimestamp[vmis[2].Name])).To(BeTrue())

		Expect(migrationStarTimestamp[vmis[4].Name].Before(migrationStarTimestamp[vmis[1].Name])).To(BeTrue())
		Expect(migrationStarTimestamp[vmis[4].Name].Before(migrationStarTimestamp[vmis[2].Name])).To(BeTrue())

	})
}))
