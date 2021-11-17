package migrations

import (
	v12 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/tools/cache"

	v1 "kubevirt.io/api/core/v1"
)

func ListUnfinishedMigrations(informer cache.SharedIndexInformer) []*v1.VirtualMachineInstanceMigration {
	objs := informer.GetStore().List()
	migrations := []*v1.VirtualMachineInstanceMigration{}
	for _, obj := range objs {
		migration := obj.(*v1.VirtualMachineInstanceMigration)
		if !migration.IsFinal() {
			migrations = append(migrations, migration)
		}
	}
	return migrations
}

func FilterRunningMigrations(migrations []v1.VirtualMachineInstanceMigration) []v1.VirtualMachineInstanceMigration {
	runningMigrations := []v1.VirtualMachineInstanceMigration{}
	for _, migration := range migrations {
		if migration.IsRunning() {
			runningMigrations = append(runningMigrations, migration)
		}
	}
	return runningMigrations
}

// IsMigrating returns true if a given VMI is still migrating and false otherwise.
func IsMigrating(vmi *v1.VirtualMachineInstance) bool {

	now := v12.Now()

	running := false
	if vmi.Status.MigrationState != nil {
		start := vmi.Status.MigrationState.StartTimestamp
		stop := vmi.Status.MigrationState.EndTimestamp
		if start != nil && (now.After(start.Time) || now.Equal(start)) {
			running = true
		}

		if stop != nil && (now.After(stop.Time) || now.Equal(stop)) {
			running = false
		}
	}
	return running
}

func MigrationFailed(vmi *v1.VirtualMachineInstance) bool {

	if vmi.Status.MigrationState != nil && vmi.Status.MigrationState.Failed {
		return true
	}

	return false
}

func MigrationNeedsProtection(vmi *v1.VirtualMachineInstance) bool {
	return vmi.Spec.EvictionStrategy != nil &&
		*vmi.Spec.EvictionStrategy == v1.EvictionStrategyLiveMigrate
}
