package apply

import (
	"kubevirt.io/kubevirt/pkg/virt-config/featuregate"
)

func (r *Reconciler) updateKubeVirtSystem(controllerDeploymentsRolledOver bool) (bool, error) {
	// UPDATE PATH IS
	// 1. daemonsets - ensures all compute nodes are updated to handle new features
	// 2. wait for daemonsets to roll over
	// 3. controllers - ensures control plane is ready for new features
	// 4. wait for controllers to roll over
	// 5. apiserver - toggles on new features.

	// create/update Daemonsets
	for _, daemonSet := range r.targetStrategy.DaemonSets() {
		finished, err := r.syncDaemonSet(daemonSet)
		if !finished || err != nil {
			return false, err
		}
	}

	// create/update Controller Deployments
	for _, deployment := range r.targetStrategy.ControllerDeployments() {
		syncedDeployment, syncErr := r.syncDeployment(deployment)
		if syncErr != nil {
			return false, syncErr
		}
		if syncErr = r.syncPodDisruptionBudgetForDeployment(syncedDeployment); syncErr != nil {
			return false, syncErr
		}
	}

	// wait for controllers
	if !controllerDeploymentsRolledOver {
		// not rolled out yet
		return false, nil
	}

	// create/update ExportProxy Deployments
	for _, deployment := range r.targetStrategy.ExportProxyDeployments() {
		syncedDeployment, syncErr := r.syncDeployment(deployment)
		if syncErr != nil {
			return false, syncErr
		}
		if syncErr = r.syncPodDisruptionBudgetForDeployment(syncedDeployment); syncErr != nil {
			return false, syncErr
		}
	}

	// create/update Synchronization controller Deployments
	for _, deployment := range r.targetStrategy.SynchronizationControllerDeployments() {
		if r.isFeatureGateEnabled(featuregate.DecentralizedLiveMigration) {
			syncedDeployment, syncErr := r.syncDeployment(deployment)
			if syncErr != nil {
				return false, syncErr
			}
			if syncErr = r.syncPodDisruptionBudgetForDeployment(syncedDeployment); syncErr != nil {
				return false, syncErr
			}
		} else if deleteErr := r.deleteDeployment(deployment); deleteErr != nil {
			return false, deleteErr
		}
	}

	// create/update virt-template Deployments
	for _, deployment := range r.targetStrategy.VirtTemplateDeployments() {
		if r.virtTemplateDeploymentEnabled() {
			syncedDeployment, syncErr := r.syncDeployment(deployment)
			if syncErr != nil {
				return false, syncErr
			}
			if syncErr = r.syncPodDisruptionBudgetForDeployment(syncedDeployment); syncErr != nil {
				return false, syncErr
			}
		} else if deleteErr := r.deleteDeployment(deployment); deleteErr != nil {
			return false, deleteErr
		}
	}

	// create/update API Deployments
	for _, deployment := range r.targetStrategy.APIDeployments() {
		syncedDeployment, syncErr := r.syncDeployment(deployment)
		if syncErr != nil {
			return false, syncErr
		}
		if syncErr = r.syncPodDisruptionBudgetForDeployment(syncedDeployment); syncErr != nil {
			return false, syncErr
		}
	}

	return true, nil
}
