package migrations

import (
	"k8s.io/client-go/tools/cache"

	v1 "kubevirt.io/kubevirt/pkg/api/v1"
)

func ListNotFinishedMigrations(informer cache.SharedIndexInformer) ([]*v1.VirtualMachineInstanceMigration, error) {
	objs := informer.GetStore().List()
	migrations := []*v1.VirtualMachineInstanceMigration{}
	for _, obj := range objs {
		migration := obj.(*v1.VirtualMachineInstanceMigration)
		if migration.Status.Phase != v1.MigrationFailed && migration.Status.Phase != v1.MigrationSucceeded {
			migrations = append(migrations, migration)
		}
	}
	return migrations, nil
}

func IsMigrationRunning(migration *v1.VirtualMachineInstanceMigration) bool {
	switch migration.Status.Phase {
	case v1.MigrationFailed, v1.MigrationPending, v1.MigrationPhaseUnset, v1.MigrationSucceeded:
		return false
	}
	return true
}

func FilterRunningMigrations(migrations []v1.VirtualMachineInstanceMigration) []v1.VirtualMachineInstanceMigration {
	runningMigrations := []v1.VirtualMachineInstanceMigration{}
	for _, migration := range migrations {
		if IsMigrationRunning(&migration) {
			runningMigrations = append(runningMigrations, migration)
		}
	}
	return runningMigrations
}
