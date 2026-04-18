/*
Copyright The KubeVirt Authors.
SPDX-License-Identifier: Apache-2.0
*/

package virtcontroller

import (
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	v1 "kubevirt.io/api/core/v1"
)

var _ = Describe("VMI migration phase transition time histogram", func() {
	Context("Transition Time calculations", func() {
		DescribeTable("time diff calculations", func(expectedVal float64, curPhase, oldPhase v1.VirtualMachineInstanceMigrationPhase) {
			var oldMigration *v1.VirtualMachineInstanceMigration

			migration := createVMIMigrationSForPhaseTransitionTime(curPhase, expectedVal*1000)

			oldMigration = migration.DeepCopy()
			oldMigration.Status.Phase = oldPhase

			diffSeconds, err := getVMIMigrationTransitionTimeSeconds(migration)
			Expect(err).ToNot(HaveOccurred())

			Expect(diffSeconds).To(Equal(expectedVal))
		},
			Entry("Time between succeeded and scheduled", 5.0, v1.MigrationSucceeded, v1.MigrationScheduled),
			Entry("Time between succeeded and scheduled using fraction of a second", .5, v1.MigrationSucceeded, v1.MigrationScheduled),
			Entry("Time between scheduled and scheduling", 2.0, v1.MigrationScheduled, v1.MigrationScheduling),
			Entry("Time between scheduling and pending", 1.0, v1.MigrationScheduling, v1.MigrationPending),
			Entry("Time between running and failed", 1.0, v1.MigrationRunning, v1.MigrationFailed),
		)
	})
})

func createVMIMigrationSForPhaseTransitionTime(
	phase v1.VirtualMachineInstanceMigrationPhase, offset float64,
) *v1.VirtualMachineInstanceMigration {
	now := metav1.NewTime(time.Now())
	old := metav1.NewTime(now.Time.Add(-time.Duration(int64(offset)) * time.Millisecond))
	creation := metav1.NewTime(old.Time.Add(-time.Duration(int64(offset)) * time.Millisecond))

	migration := &v1.VirtualMachineInstanceMigration{
		ObjectMeta: metav1.ObjectMeta{
			Namespace:         "test-ns",
			Name:              "testvmimigration",
			CreationTimestamp: creation,
		},
		Status: v1.VirtualMachineInstanceMigrationStatus{
			Phase: phase,
		},
	}

	migration.Status.PhaseTransitionTimestamps = append(
		migration.Status.PhaseTransitionTimestamps,
		v1.VirtualMachineInstanceMigrationPhaseTransitionTimestamp{
			Phase:                    phase,
			PhaseTransitionTimestamp: old,
		},
	)

	return migration
}
