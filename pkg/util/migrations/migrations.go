package migrations

import (
	"fmt"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/tools/cache"
	v1 "kubevirt.io/api/core/v1"
	"kubevirt.io/client-go/log"

	"kubevirt.io/kubevirt/pkg/controller"
	"kubevirt.io/kubevirt/pkg/pointer"
	virtconfig "kubevirt.io/kubevirt/pkg/virt-config"
)

const CancelMigrationFailedVmiNotMigratingErr = "failed to cancel migration - vmi is not migrating"

const (
	QueuePriorityRunning           int = 1000
	QueuePrioritySystemCritical    int = 100
	QueuePriorityUserTriggered     int = 50
	QueuePrioritySystemMaintenance int = 20
	QueuePriorityDefault           int = 0
	QueuePriorityPending           int = -100
)

func ListUnfinishedMigrations(indexer cache.Indexer) []*v1.VirtualMachineInstanceMigration {
	objs, err := indexer.ByIndex(controller.UnfinishedIndex, controller.UnfinishedIndex)
	if err != nil {
		log.Log.Reason(err).Errorf("Failed to use unfinished index")
		return nil
	}

	var migrations []*v1.VirtualMachineInstanceMigration
	for _, obj := range objs {
		migration := obj.(*v1.VirtualMachineInstanceMigration)
		if !migration.IsFinal() {
			migrations = append(migrations, migration)
		}
	}
	return migrations
}

func ListWorkloadUpdateMigrations(indexer cache.Indexer, vmiName, ns string) []v1.VirtualMachineInstanceMigration {
	objs, err := indexer.ByIndex(controller.ByVMINameIndex, fmt.Sprintf("%s/%s", ns, vmiName))
	if err != nil {
		log.Log.Reason(err).Errorf("Failed to use byVMIName index for workload migrations")
		return nil
	}

	migrations := []v1.VirtualMachineInstanceMigration{}
	for _, obj := range objs {
		migration := obj.(*v1.VirtualMachineInstanceMigration)
		if migration.IsFinal() {
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

	now := metav1.Now()

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

func IsBackupInProgress(vmi *v1.VirtualMachineInstance) bool {
	if vmi == nil || vmi.Status.ChangedBlockTracking == nil || vmi.Status.ChangedBlockTracking.BackupStatus == nil {
		return false
	}
	backupStatus := vmi.Status.ChangedBlockTracking.BackupStatus
	return backupStatus.BackupName != "" && !backupStatus.Completed
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
	objs, err := migrationIndexer.ByIndex(controller.ByVMINameIndex, fmt.Sprintf("%s/%s", vmi.Namespace, vmi.Name))
	if err != nil {
		return false, err
	}
	for _, obj := range objs {
		migration := obj.(*v1.VirtualMachineInstanceMigration)
		if migration.IsRunning() {
			return true, nil
		}
	}
	return false, nil
}

func PriorityFromMigration(migration *v1.VirtualMachineInstanceMigration) *int {
	if migration.Spec.Priority == nil {
		return pointer.P(QueuePriorityDefault)
	}
	switch *migration.Spec.Priority {
	case v1.PrioritySystemCritical:
		return pointer.P(QueuePrioritySystemCritical)
	case v1.PriorityUserTriggered:
		return pointer.P(QueuePriorityUserTriggered)
	case v1.PrioritySystemMaintenance:
		return pointer.P(QueuePrioritySystemMaintenance)
	default:
		return pointer.P(QueuePriorityDefault)
	}
}
