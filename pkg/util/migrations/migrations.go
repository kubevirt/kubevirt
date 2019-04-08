package migrations

import (
	"k8s.io/client-go/tools/cache"

	v1 "kubevirt.io/kubevirt/pkg/api/v1"
)

func ListUnfinishedMigrations(informer cache.SharedIndexInformer) ([]*v1.VirtualMachineInstanceMigration, error) {
	objs := informer.GetStore().List()
	migrations := []*v1.VirtualMachineInstanceMigration{}
	for _, obj := range objs {
		migration := obj.(*v1.VirtualMachineInstanceMigration)
		if !migration.IsFinal() {
			migrations = append(migrations, migration)
		}
	}
	return migrations, nil
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
