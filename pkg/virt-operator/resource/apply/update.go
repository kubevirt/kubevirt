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
 */

package apply

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
