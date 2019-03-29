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

func ListRunningMigrations(informer cache.SharedIndexInformer) ([]*v1.VirtualMachineInstanceMigration, error) {
	objs := informer.GetStore().List()
	migrations := []*v1.VirtualMachineInstanceMigration{}
	for _, obj := range objs {
		migration := obj.(*v1.VirtualMachineInstanceMigration)
		switch migration.Status.Phase {
		case v1.MigrationFailed, v1.MigrationPending, v1.MigrationPhaseUnset, v1.MigrationSucceeded:
			continue
		default:
			migrations = append(migrations, migration)
		}
	}
	return migrations, nil
}
