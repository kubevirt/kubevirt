package apply

import (
	virtconfig "kubevirt.io/kubevirt/pkg/virt-config"
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
		deployment, err := r.syncDeployment(deployment)
		if err != nil {
			return false, err
		}
		err = r.syncPodDisruptionBudgetForDeployment(deployment)
		if err != nil {
			return false, err
		}
	}

	// wait for controllers
	if !controllerDeploymentsRolledOver {
		// not rolled out yet
		return false, nil
	}

	// create/update ExportProxy Deployments
	for _, deployment := range r.targetStrategy.ExportProxyDeployments() {
		if r.exportProxyEnabled() {
			deployment, err := r.syncDeployment(deployment)
			if err != nil {
				return false, err
			}
			err = r.syncPodDisruptionBudgetForDeployment(deployment)
			if err != nil {
				return false, err
			}
		} else if err := r.deleteDeployment(deployment); err != nil {
			return false, err
		}
	}

	// create/update Synchronization controller Deployments
	for _, deployment := range r.targetStrategy.SynchronizationControllerDeployments() {
		if virtconfig.IsFeatureGateEnabled(featuregate.DecentralizedLiveMigration, r.kv.Spec.Configuration) {
			deployment, err := r.syncDeployment(deployment)
			if err != nil {
				return false, err
			}
			err = r.syncPodDisruptionBudgetForDeployment(deployment)
			if err != nil {
				return false, err
			}
		} else if err := r.deleteDeployment(deployment); err != nil {
			return false, err
		}
	}

	// create/update API Deployments
	for _, deployment := range r.targetStrategy.ApiDeployments() {
		deployment, err := r.syncDeployment(deployment)
		if err != nil {
			return false, err
		}
		err = r.syncPodDisruptionBudgetForDeployment(deployment)
		if err != nil {
			return false, err
		}
	}

	return true, nil
}
