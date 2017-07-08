/*
 * This file is part of the kubevirt project
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
 * Copyright 2017 Red Hat, Inc.
 *
 */

package watch

import (
	"fmt"
	"time"

	k8sv1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/util/workqueue"

	kubev1 "kubevirt.io/kubevirt/pkg/api/v1"
	"kubevirt.io/kubevirt/pkg/kubecli"
	"kubevirt.io/kubevirt/pkg/logging"
	"kubevirt.io/kubevirt/pkg/virt-controller/services"
)

func NewMigrationController(restClient *rest.RESTClient, migrationService services.VMService, clientset *kubernetes.Clientset, queue workqueue.RateLimitingInterface, migrationInformer cache.SharedIndexInformer, podInformer cache.SharedIndexInformer, migrationCache cache.Store) *MigrationController {

	return &MigrationController{
		restClient:        restClient,
		vmService:         migrationService,
		clientset:         clientset,
		queue:             queue,
		store:             migrationCache,
		migrationInformer: migrationInformer,
		podInformer:       podInformer,
	}
}

type MigrationController struct {
	restClient        *rest.RESTClient
	vmService         services.VMService
	clientset         *kubernetes.Clientset
	queue             workqueue.RateLimitingInterface
	store             cache.Store
	migrationInformer cache.SharedIndexInformer
	podInformer       cache.SharedIndexInformer
}

func (c *MigrationController) Run(threadiness int, stopCh chan struct{}) {
	defer kubecli.HandlePanic()
	defer c.queue.ShutDown()
	logging.DefaultLogger().Info().Msg("Starting controller.")

	cache.WaitForCacheSync(stopCh, c.migrationInformer.HasSynced, c.podInformer.HasSynced)

	// Start the actual work
	for i := 0; i < threadiness; i++ {
		go wait.Until(c.runWorker, time.Second, stopCh)
	}

	<-stopCh
	logging.DefaultLogger().Info().Msg("Stopping controller.")
}

func (c *MigrationController) runWorker() {
	for c.Execute() {
	}
}

func (md *MigrationController) Execute() bool {
	key, quit := md.queue.Get()
	if quit {
		return false
	}
	defer md.queue.Done(key)
	if err := md.execute(key.(string)); err != nil {
		logging.DefaultLogger().Info().Reason(err).Msgf("reenqueuing migration %v", key)
		md.queue.AddRateLimited(key)
	} else {
		logging.DefaultLogger().Info().V(4).Msgf("processed migration %v", key)
		md.queue.Forget(key)
	}
	return true
}

func (md *MigrationController) execute(key string) error {

	setMigrationPhase := func(migration *kubev1.Migration, phase kubev1.MigrationPhase) error {

		if migration.Status.Phase == phase {
			return nil
		}

		logger := logging.DefaultLogger().Object(migration)

		migration.Status.Phase = phase
		// TODO indicate why it was set to failed
		err := md.vmService.UpdateMigration(migration)
		if err != nil {
			logger.Error().Reason(err).Msgf("updating migration state failed: %v ", err)
			return err
		}
		return nil
	}

	setMigrationFailed := func(mig *kubev1.Migration) error {
		return setMigrationPhase(mig, kubev1.MigrationFailed)
	}

	obj, exists, err := md.store.GetByKey(key)
	if err != nil {
		return err
	}
	if !exists {
		return nil
	}

	logger := logging.DefaultLogger().Object(obj.(*kubev1.Migration))
	// Copy migration for future modifications
	if obj, err = scheme.Scheme.Copy(obj.(runtime.Object)); err != nil {
		logger.Error().Reason(err).Msg("could not copy migration object")
		return err
	}
	migration := obj.(*kubev1.Migration)

	vm, exists, err := md.vmService.FetchVM(migration.GetObjectMeta().GetNamespace(), migration.Spec.Selector.Name)
	if err != nil {
		logger.Error().Reason(err).Msgf("fetching the vm %s failed", migration.Spec.Selector.Name)
		return err
	}

	if !exists {
		logger.Info().Msgf("VM with name %s does not exist, marking migration as failed", migration.Spec.Selector.Name)
		return setMigrationFailed(migration)
	}

	switch migration.Status.Phase {
	case kubev1.MigrationUnknown:
		if vm.Status.Phase != kubev1.Running {
			logger.Error().Msgf("VM with name %s is in state %s, no migration possible. Marking migration as failed", vm.GetObjectMeta().GetName(), vm.Status.Phase)
			return setMigrationFailed(migration)
		}

		if err := mergeConstraints(migration, vm); err != nil {
			logger.Error().Reason(err).Msg("merging Migration and VM placement constraints failed.")
			return err
		}
		podList, err := md.vmService.GetRunningVMPods(vm)
		if err != nil {
			logger.Error().Reason(err).Msg("could not fetch a list of running VM target pods")
			return err
		}

		//FIXME when we have more than one worker, we need a lock on the VM
		numOfPods, targetPod, err := investigateTargetPodSituation(migration, podList, md.store)
		if err != nil {
			logger.Error().Reason(err).Msg("could not investigate pods")
			return err
		}

		if targetPod == nil {
			if numOfPods >= 1 {
				logger.Error().Msg("another migration seems to be in progress, marking Migration as failed")
				// Another migration is currently going on
				if err = setMigrationFailed(migration); err != nil {
					return err
				}
				return nil
			} else if numOfPods == 0 {
				// We need to start a migration target pod
				// TODO, this detection is not optimal, it can lead to strange situations
				err := md.vmService.CreateMigrationTargetPod(migration, vm)
				if err != nil {
					logger.Error().Reason(err).Msg("creating a migration target pod failed")
					return err
				}
			}
		} else {
			if targetPod.Status.Phase == k8sv1.PodFailed {
				logger.Error().Msg("migration target pod is in failed state")
				return setMigrationFailed(migration)
			}
			// Unlikely to hit this case, but prevents erroring out
			// if we re-enter this loop
			logger.Info().Msgf("migration appears to be set up, but was not set to %s", kubev1.MigrationRunning)
		}
		return setMigrationPhase(migration, kubev1.MigrationRunning)
	case kubev1.MigrationRunning:
		podList, err := md.vmService.GetRunningVMPods(vm)
		if err != nil {
			logger.Error().Reason(err).Msg("could not fetch a list of running VM target pods")
			return err
		}
		_, targetPod, err := investigateTargetPodSituation(migration, podList, md.store)
		if err != nil {
			logger.Error().Reason(err).Msg("could not investigate pods")
			return err
		}
		if targetPod == nil {
			logger.Error().Msg("migration target pod does not exist or is in an end state")
			return setMigrationFailed(migration)
		}
		switch targetPod.Status.Phase {
		case k8sv1.PodRunning:
			break
		case k8sv1.PodSucceeded, k8sv1.PodFailed:
			logger.Error().Msgf("migration target pod is in end state %s", targetPod.Status.Phase)
			return setMigrationFailed(migration)
		default:
			//Not requeuing, just not far enough along to proceed
			logger.Info().V(3).Msg("target Pod not running yet")
			return nil
		}

		if vm.Status.MigrationNodeName != targetPod.Spec.NodeName {
			vm.Status.Phase = kubev1.Migrating
			vm.Status.MigrationNodeName = targetPod.Spec.NodeName
			if _, err = md.vmService.PutVm(vm); err != nil {
				logger.Error().Reason(err).Msgf("failed to update VM to state %s", kubev1.Migrating)
				return err
			}
		}

		// Let's check if the job already exists, it can already exist in case we could not update the VM object in a previous run
		migrationPod, exists, err := md.vmService.GetMigrationJob(migration)

		if err != nil {
			logger.Error().Reason(err).Msg("Checking for an existing migration job failed.")
			return err
		}

		if !exists {
			sourceNode, err := md.clientset.CoreV1().Nodes().Get(vm.Status.NodeName, metav1.GetOptions{})
			if err != nil {
				logger.Error().Reason(err).Msgf("fetching source node %s failed", vm.Status.NodeName)
				return err
			}
			targetNode, err := md.clientset.CoreV1().Nodes().Get(vm.Status.MigrationNodeName, metav1.GetOptions{})
			if err != nil {
				logger.Error().Reason(err).Msgf("fetching target node %s failed", vm.Status.MigrationNodeName)
				return err
			}

			if err := md.vmService.StartMigration(migration, vm, sourceNode, targetNode, targetPod); err != nil {
				logger.Error().Reason(err).Msg("Starting the migration job failed.")
				return err
			}
			return nil
		}

		// FIXME, the final state updates must come from virt-handler
		switch migrationPod.Status.Phase {
		case k8sv1.PodFailed:
			vm.Status.Phase = kubev1.Running
			vm.Status.MigrationNodeName = ""
			if _, err = md.vmService.PutVm(vm); err != nil {
				return err
			}
			return setMigrationFailed(migration)
		case k8sv1.PodSucceeded:
			vm.Status.NodeName = targetPod.Spec.NodeName
			vm.Status.MigrationNodeName = ""
			vm.Status.Phase = kubev1.Running
			if vm.ObjectMeta.Labels == nil {
				vm.ObjectMeta.Labels = map[string]string{}
			}
			vm.ObjectMeta.Labels[kubev1.NodeNameLabel] = vm.Status.NodeName
			if _, err = md.vmService.PutVm(vm); err != nil {
				logger.Error().Reason(err).Msg("updating the VM failed.")
				return err
			}
			return setMigrationPhase(migration, kubev1.MigrationSucceeded)
		}
	}
	return nil
}

// Returns the number of  running pods and if a pod for exactly that migration is currently running
func investigateTargetPodSituation(migration *kubev1.Migration, podList *k8sv1.PodList, migrationStore cache.Store) (int, *k8sv1.Pod, error) {
	var targetPod *k8sv1.Pod = nil
	podCount := 0
	for idx, pod := range podList.Items {
		if pod.Labels[kubev1.MigrationUIDLabel] == string(migration.GetObjectMeta().GetUID()) {
			targetPod = &podList.Items[idx]
			podCount += 1
			continue
		}

		// The first pod was never part of a migration, it does not count
		l, exists := pod.Labels[kubev1.MigrationLabel]
		if !exists {
			continue
		}
		key := fmt.Sprintf("%s/%s", pod.ObjectMeta.Namespace, l)
		cachedObj, exists, err := migrationStore.GetByKey(key)
		if err != nil {
			return 0, nil, err
		}
		if exists {
			cachedMigration := cachedObj.(*kubev1.Migration)
			if (cachedMigration.Status.Phase != kubev1.MigrationFailed) &&
				(cachedMigration.Status.Phase) != kubev1.MigrationSucceeded {
				podCount += 1
			}
		} else {
			podCount += 1
		}
	}
	return podCount, targetPod, nil
}

func mergeConstraints(migration *kubev1.Migration, vm *kubev1.VM) error {

	merged := map[string]string{}
	for k, v := range vm.Spec.NodeSelector {
		merged[k] = v
	}
	conflicts := []string{}
	for k, v := range migration.Spec.NodeSelector {
		val, exists := vm.Spec.NodeSelector[k]
		if exists && val != v {
			conflicts = append(conflicts, k)
		} else {
			merged[k] = v
		}
	}
	if len(conflicts) > 0 {
		return fmt.Errorf("Conflicting node selectors: %v", conflicts)
	}
	vm.Spec.NodeSelector = merged
	return nil
}

func migrationJobLabelHandler(migrationQueue workqueue.RateLimitingInterface) func(obj interface{}) {
	return func(obj interface{}) {
		phase := obj.(*k8sv1.Pod).Status.Phase
		namespace := obj.(*k8sv1.Pod).ObjectMeta.Namespace
		appLabel, hasAppLabel := obj.(*k8sv1.Pod).ObjectMeta.Labels[kubev1.AppLabel]
		migrationLabel, hasMigrationLabel := obj.(*k8sv1.Pod).ObjectMeta.Labels[kubev1.MigrationLabel]
		_, hasDomainLabel := obj.(*k8sv1.Pod).ObjectMeta.Labels[kubev1.DomainLabel]

		if phase == k8sv1.PodRunning ||
			phase == k8sv1.PodUnknown ||
			phase == k8sv1.PodPending {

			return
		} else if hasDomainLabel == false || hasMigrationLabel == false || hasAppLabel == false {
			// missing required labels
			return
		} else if appLabel != "migration" {
			return
		}

		migrationQueue.Add(namespace + "/" + migrationLabel)
	}
}

func migrationPodLabelHandler(migrationQueue workqueue.RateLimitingInterface) func(obj interface{}) {
	return func(obj interface{}) {
		phase := obj.(*k8sv1.Pod).Status.Phase
		namespace := obj.(*k8sv1.Pod).ObjectMeta.Namespace
		appLabel, hasAppLabel := obj.(*k8sv1.Pod).ObjectMeta.Labels[kubev1.AppLabel]
		migrationLabel, hasMigrationLabel := obj.(*k8sv1.Pod).ObjectMeta.Labels[kubev1.MigrationLabel]

		if phase != k8sv1.PodRunning {
			return
		} else if hasMigrationLabel == false || hasAppLabel == false {
			// missing required labels
			return
		} else if appLabel != "virt-launcher" {
			return
		}

		migrationQueue.Add(namespace + "/" + migrationLabel)
	}
}
