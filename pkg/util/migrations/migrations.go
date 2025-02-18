package migrations

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	v12 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/tools/cache"
	v1 "kubevirt.io/api/core/v1"
	"kubevirt.io/client-go/log"

	virtconfig "kubevirt.io/kubevirt/pkg/virt-config"
)

const CancelMigrationFailedVmiNotMigratingErr = "failed to cancel migration - vmi is not migrating"

func ListUnfinishedMigrations(store cache.Store) []*v1.VirtualMachineInstanceMigration {
	objs := store.List()
	migrations := []*v1.VirtualMachineInstanceMigration{}
	for _, obj := range objs {
		migration := obj.(*v1.VirtualMachineInstanceMigration)
		if !migration.IsFinal() {
			migrations = append(migrations, migration)
		}
	}
	return migrations
}

func ListWorkloadUpdateMigrations(store cache.Store, vmiName, ns string) []v1.VirtualMachineInstanceMigration {
	objs := store.List()
	migrations := []v1.VirtualMachineInstanceMigration{}
	for _, obj := range objs {
		migration := obj.(*v1.VirtualMachineInstanceMigration)
		if migration.IsFinal() {
			continue
		}
		if migration.Namespace != ns {
			continue
		}
		if migration.Spec.VMIName != vmiName {
			continue
		}
		if !metav1.HasAnnotation(migration.ObjectMeta, v1.WorkloadUpdateMigrationAnnotation) {
			continue
		}
		migrations = append(migrations, *migration)
	}

	return migrations
}

func FilterRunningMigrations(migrations []*v1.VirtualMachineInstanceMigration) []*v1.VirtualMachineInstanceMigration {
	runningMigrations := []*v1.VirtualMachineInstanceMigration{}
	for _, migration := range migrations {
		if migration.IsRunning() {
			runningMigrations = append(runningMigrations, migration)
		}
	}
	return runningMigrations
}

// IsMigrating returns true if a given VMI is still migrating and false otherwise.
func IsMigrating(vmi *v1.VirtualMachineInstance) bool {
	if vmi == nil {
		log.Log.V(4).Infof("checking if VMI is migrating, but it is empty")
		return false
	}

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

func VMIEvictionStrategy(clusterConfig *virtconfig.ClusterConfig, vmi *v1.VirtualMachineInstance) *v1.EvictionStrategy {
	if vmi != nil && vmi.Spec.EvictionStrategy != nil {
		return vmi.Spec.EvictionStrategy
	}
	clusterStrategy := clusterConfig.GetConfig().EvictionStrategy
	return clusterStrategy
}

func VMIMigratableOnEviction(clusterConfig *virtconfig.ClusterConfig, vmi *v1.VirtualMachineInstance) bool {
	strategy := VMIEvictionStrategy(clusterConfig, vmi)
	if strategy == nil {
		return false
	}
	switch *strategy {
	case v1.EvictionStrategyLiveMigrate:
		return true
	case v1.EvictionStrategyLiveMigrateIfPossible:
		return vmi.IsMigratable()
	}
	return false
}

func ActiveMigrationExistsForVMI(migrationIndexer cache.Indexer, vmi *v1.VirtualMachineInstance) (bool, error) {
	objs, err := migrationIndexer.ByIndex(cache.NamespaceIndex, vmi.Namespace)
	if err != nil {
		return false, err
	}
	for _, obj := range objs {
		migration := obj.(*v1.VirtualMachineInstanceMigration)
		if migration.Spec.VMIName == vmi.Name && migration.IsRunning() {
			return true, nil
		}
	}

	return false, nil
}
