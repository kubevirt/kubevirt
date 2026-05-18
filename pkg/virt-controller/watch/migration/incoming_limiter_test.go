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

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"go.uber.org/mock/gomock"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	k8sfake "k8s.io/client-go/kubernetes/fake"

	virtv1 "kubevirt.io/api/core/v1"
	"kubevirt.io/client-go/kubecli"
	kubevirtfake "kubevirt.io/client-go/kubevirt/fake"
)

var _ = Describe("Incoming migration limiter", func() {
	const namespace = "default"
	const targetNode = "node-a"

	var (
		ctx           context.Context
		kubeClient    *k8sfake.Clientset
		virtClientset *kubevirtfake.Clientset
		virtClient    *kubecli.MockKubevirtClient
		limiter       *LeaseIncomingMigrationLimiter
	)

	newMigration := func(name string, uid types.UID, phase virtv1.VirtualMachineInstanceMigrationPhase) *virtv1.VirtualMachineInstanceMigration {
		return &virtv1.VirtualMachineInstanceMigration{
			ObjectMeta: metav1.ObjectMeta{
				Name:      name,
				Namespace: namespace,
				UID:       uid,
				Labels: map[string]string{
					virtv1.MigrationSelectorLabel: name,
				},
			},
			Spec: virtv1.VirtualMachineInstanceMigrationSpec{VMIName: name},
			Status: virtv1.VirtualMachineInstanceMigrationStatus{
				Phase: phase,
			},
		}
	}

	storeMigration := func(migration *virtv1.VirtualMachineInstanceMigration) {
		_, err := virtClientset.KubevirtV1().VirtualMachineInstanceMigrations(migration.Namespace).Create(ctx, migration, metav1.CreateOptions{})
		Expect(err).ToNot(HaveOccurred())
	}

	listLeases := func() []string {
		leases, err := kubeClient.CoordinationV1().Leases(incomingMigrationLeaseNamespace).List(ctx, metav1.ListOptions{})
		Expect(err).ToNot(HaveOccurred())
		names := make([]string, 0, len(leases.Items))
		for _, lease := range leases.Items {
			names = append(names, lease.Name)
		}
		return names
	}

	BeforeEach(func() {
		ctx = context.Background()
		kubeClient = k8sfake.NewSimpleClientset()
		virtClientset = kubevirtfake.NewSimpleClientset()
		virtClient = kubecli.NewMockKubevirtClient(gomock.NewController(GinkgoT()))
		virtClient.EXPECT().CoordinationV1().Return(kubeClient.CoordinationV1()).AnyTimes()
		virtClient.EXPECT().VirtualMachineInstanceMigration(namespace).Return(virtClientset.KubevirtV1().VirtualMachineInstanceMigrations(namespace)).AnyTimes()
		limiter = NewLeaseIncomingMigrationLimiter(virtClient)
	})

	It("allows only one migration for limit one", func() {
		first := newMigration("migration-a", "uid-a", virtv1.MigrationRunning)
		second := newMigration("migration-b", "uid-b", virtv1.MigrationRunning)
		storeMigration(first)
		storeMigration(second)

		acquired, err := limiter.TryAcquire(ctx, first, targetNode, 1)
		Expect(err).ToNot(HaveOccurred())
		Expect(acquired).To(BeTrue())

		acquired, err = limiter.TryAcquire(ctx, second, targetNode, 1)
		Expect(err).ToNot(HaveOccurred())
		Expect(acquired).To(BeFalse())
		Expect(listLeases()).To(HaveLen(1))
	})

	It("assigns different slots up to limit", func() {
		for i := 0; i < 5; i++ {
			migration := newMigration(string(rune('a'+i)), types.UID(string(rune('a'+i))), virtv1.MigrationRunning)
			storeMigration(migration)
			acquired, err := limiter.TryAcquire(ctx, migration, targetNode, 5)
			Expect(err).ToNot(HaveOccurred())
			Expect(acquired).To(BeTrue())
		}

		sixth := newMigration("migration-f", "uid-f", virtv1.MigrationRunning)
		storeMigration(sixth)
		acquired, err := limiter.TryAcquire(ctx, sixth, targetNode, 5)
		Expect(err).ToNot(HaveOccurred())
		Expect(acquired).To(BeFalse())
		Expect(listLeases()).To(HaveLen(5))
	})

	It("reuses a slot owned by the same migration", func() {
		migration := newMigration("migration-a", "uid-a", virtv1.MigrationRunning)
		storeMigration(migration)

		acquired, err := limiter.TryAcquire(ctx, migration, targetNode, 1)
		Expect(err).ToNot(HaveOccurred())
		Expect(acquired).To(BeTrue())
		acquired, err = limiter.TryAcquire(ctx, migration, targetNode, 1)
		Expect(err).ToNot(HaveOccurred())
		Expect(acquired).To(BeTrue())
		Expect(listLeases()).To(HaveLen(1))
	})

	It("steals stale lease from a terminal migration", func() {
		stale := newMigration("migration-a", "uid-a", virtv1.MigrationSucceeded)
		current := newMigration("migration-b", "uid-b", virtv1.MigrationRunning)
		storeMigration(stale)
		storeMigration(current)

		acquired, err := limiter.TryAcquire(ctx, stale, targetNode, 1)
		Expect(err).ToNot(HaveOccurred())
		Expect(acquired).To(BeTrue())

		acquired, err = limiter.TryAcquire(ctx, current, targetNode, 1)
		Expect(err).ToNot(HaveOccurred())
		Expect(acquired).To(BeTrue())

		leases, err := kubeClient.CoordinationV1().Leases(incomingMigrationLeaseNamespace).List(ctx, metav1.ListOptions{})
		Expect(err).ToNot(HaveOccurred())
		Expect(leases.Items).To(HaveLen(1))
		Expect(leases.Items[0].Annotations[incomingMigrationNameAnnotation]).To(Equal(current.Name))
	})

	It("releases a slot outside the current limit", func() {
		for i := 0; i < 4; i++ {
			migration := newMigration(string(rune('a'+i)), types.UID(string(rune('a'+i))), virtv1.MigrationRunning)
			storeMigration(migration)
			acquired, err := limiter.TryAcquire(ctx, migration, targetNode, 5)
			Expect(err).ToNot(HaveOccurred())
			Expect(acquired).To(BeTrue())
		}

		migration := newMigration("migration-e", "uid-e", virtv1.MigrationRunning)
		storeMigration(migration)
		acquired, err := limiter.TryAcquire(ctx, migration, targetNode, 5)
		Expect(err).ToNot(HaveOccurred())
		Expect(acquired).To(BeTrue())
		Expect(listLeases()).To(HaveLen(5))

		Expect(limiter.Release(ctx, migration, targetNode, 1)).To(Succeed())
		Expect(listLeases()).To(HaveLen(4))
	})
})
