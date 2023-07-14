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
 * Copyright 2023 Red Hat, Inc.
 *
 */
package instancetype

import (
	"fmt"
	"time"

	appsv1 "k8s.io/api/apps/v1"
	k8sv1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/tools/record"
	"k8s.io/client-go/util/workqueue"

	"kubevirt.io/client-go/kubecli"
	"kubevirt.io/client-go/log"

	instancetypeapi "kubevirt.io/api/instancetype"
	instancetypev1beta1 "kubevirt.io/api/instancetype/v1beta1"

	"kubevirt.io/kubevirt/pkg/controller"
	"kubevirt.io/kubevirt/pkg/instancetype"
	"kubevirt.io/kubevirt/pkg/util/status"
)

const (
	upgradeFailureReason = "ControllerRevisionUpgrade failed"
)

type UpgradeController struct {
	queue             workqueue.RateLimitingInterface
	upgrader          instancetype.Upgrader
	statusUpdater     *status.ControllerRevisionUpgradeStatusUpdater
	recorder          record.EventRecorder
	crInformer        cache.SharedIndexInformer
	crUpgradeInformer cache.SharedIndexInformer
}

func NewUpgradeController(
	client kubecli.KubevirtClient,
	recorder record.EventRecorder,
	vmInformer cache.SharedIndexInformer,
	crInformer cache.SharedIndexInformer,
	crUpgradeInformer cache.SharedIndexInformer,
) (*UpgradeController, error) {
	c := &UpgradeController{
		queue:             workqueue.NewNamedRateLimitingQueue(workqueue.DefaultControllerRateLimiter(), "virt-controller-instancetype-migration"),
		upgrader:          instancetype.NewUpgrader(client, vmInformer),
		statusUpdater:     status.NewControllerRevisionUpgradeStatusUpdater(client),
		recorder:          recorder,
		crInformer:        crInformer,
		crUpgradeInformer: crUpgradeInformer,
	}

	if _, err := c.crUpgradeInformer.AddEventHandler(
		cache.ResourceEventHandlerFuncs{
			AddFunc:    c.upgradeAdded,
			UpdateFunc: c.upgradeUpdated,
		},
	); err != nil {
		return nil, err
	}

	return c, nil
}

func (c *UpgradeController) enqueueUpgrade(obj interface{}) {
	crUpgrade, ok := obj.(*instancetypev1beta1.ControllerRevisionUpgrade)
	if !ok {
		log.Log.Error("Failed to extract ControllerRevisionUpgrade.")
		return
	}
	key, err := controller.KeyFunc(crUpgrade)
	if err != nil {
		log.Log.Object(crUpgrade).Reason(err).Error("Failed to extract key from ControllerRevisionUpgrade.")
		return
	}
	c.queue.Add(key)
}

func (c *UpgradeController) upgradeAdded(obj interface{}) {
	c.enqueueUpgrade(obj)
}

func (c *UpgradeController) upgradeUpdated(_, currObj interface{}) {
	c.enqueueUpgrade(currObj)
}

func (c *UpgradeController) Run(threadiness int, stopCh <-chan struct{}) {
	defer controller.HandlePanic()
	defer c.queue.ShutDown()

	log.Log.Info("Starting ControllerRevisionUpgrade controller.")
	defer log.Log.Info("Stopping ControllerRevisionUpgrade controller.")

	cache.WaitForCacheSync(stopCh, c.crInformer.HasSynced, c.crUpgradeInformer.HasSynced)

	for i := 0; i < threadiness; i++ {
		go wait.Until(c.runWorker, time.Second, stopCh)
	}

	<-stopCh
}

func (c *UpgradeController) runWorker() {
	for c.process() {
	}
}

func (c *UpgradeController) process() bool {
	key, quit := c.queue.Get()
	if quit {
		return false
	}
	defer c.queue.Done(key)

	if err := c.execute(key); err != nil {
		// We don't reenqueue requests here as it isn't expected that the upgrade would eventually complete.
		log.Log.Reason(err).Infof("failure to process ControllerRevisionUpgrade %v", key)
	} else {
		log.Log.V(4).Infof("successfully processed ControllerRevisionUpgrade %v", key)
	}

	return true
}

func (c *UpgradeController) findUpgrade(crUpgradeKey string) (*instancetypev1beta1.ControllerRevisionUpgrade, error) {
	obj, exists, err := c.crUpgradeInformer.GetStore().GetByKey(crUpgradeKey)
	if err != nil {
		return nil, err
	}
	if !exists {
		return nil, fmt.Errorf("unable to find ControllerRevisionUpgrade %s", crUpgradeKey)
	}
	crUpgrade, ok := obj.(*instancetypev1beta1.ControllerRevisionUpgrade)
	if !ok {
		return nil, fmt.Errorf("unknown object returned from ControllerRevisionUpgrade informer")
	}
	return crUpgrade, nil
}

func (c *UpgradeController) findCR(crKey string) (*appsv1.ControllerRevision, error) {
	obj, exists, err := c.crInformer.GetStore().GetByKey(crKey)
	if err != nil {
		return nil, err
	}
	if !exists {
		return nil, fmt.Errorf("unable to find ControllerRevision %s", crKey)
	}
	cr, ok := obj.(*appsv1.ControllerRevision)
	if !ok {
		return nil, fmt.Errorf("unknown object found in ControllerRevision informer")
	}
	return cr, nil
}

func (c *UpgradeController) execute(key interface{}) error {
	crUpgradeKey, ok := key.(string)
	if !ok {
		return fmt.Errorf("unable to use ControllerRevisionUpgrade key %v", key)
	}

	crUpgrade, err := c.findUpgrade(crUpgradeKey)
	if err != nil {
		return err
	}

	logger := log.Log.Object(crUpgrade)
	logger.V(4).Info("Started processing ControllerRevisionUpgrade")
	defer logger.V(4).Info("finished processing ControllerRevisionUpgrade")

	if upgradeStatusSuccessful(crUpgrade) {
		logger.V(4).Info("ControllerRevisionUpgrade already completed, ignoring")
		return nil
	}

	if upgradeStatusUnknown(crUpgrade) {
		logger.V(4).Info("updating ControllerRevisionUpgrade to running")
		return c.updateStatusWithRunning(crUpgrade)
	}

	newCR, err := c.upgrade(crUpgrade)
	if err != nil {
		if updateErr := c.updateStatusWithFailure(crUpgrade, err.Error()); updateErr != nil {
			return updateErr
		}
		return err
	}

	return c.updateStatusWithSuccess(crUpgrade, newCR)
}

func (c *UpgradeController) upgrade(crUpgrade *instancetypev1beta1.ControllerRevisionUpgrade) (*appsv1.ControllerRevision, error) {
	targetCR, err := c.findCR(fmt.Sprintf("%s/%s", crUpgrade.Namespace, crUpgrade.Spec.TargetName))
	if err != nil {
		return nil, err
	}
	newCR, err := c.upgrader.Upgrade(targetCR)
	if err != nil {
		return nil, fmt.Errorf("failure to upgrade ControllerRevision: %w", err)
	}
	return newCR, nil
}

func upgradeStatusUnknown(crUpgrade *instancetypev1beta1.ControllerRevisionUpgrade) bool {
	return crUpgrade.Status.Phase == ""
}

func upgradeStatusSuccessful(crUpgrade *instancetypev1beta1.ControllerRevisionUpgrade) bool {
	return crUpgrade.Status.Phase == instancetypev1beta1.UpgradeSucceeded
}

func (c *UpgradeController) updateStatusWithSuccess(crUpgrade *instancetypev1beta1.ControllerRevisionUpgrade, newCR *appsv1.ControllerRevision) error {
	crUpgrade.Status = instancetypev1beta1.ControllerRevisionUpgradeStatus{
		Phase: instancetypev1beta1.UpgradeSucceeded,
		Result: &instancetypev1beta1.ControllerRevisionUpgradeResult{
			Name:    newCR.Name,
			Version: newCR.Labels[instancetypeapi.ControllerRevisionObjectVersionLabel],
		},
	}
	if err := c.statusUpdater.UpdateStatus(crUpgrade); err != nil {
		return fmt.Errorf("failure to update ControllerRevisionUpgrade with success: %w", err)
	}
	return nil
}

func (c *UpgradeController) updateStatusWithFailure(crUpgrade *instancetypev1beta1.ControllerRevisionUpgrade, updateErr string) error {
	crUpgrade.Status = instancetypev1beta1.ControllerRevisionUpgradeStatus{
		Phase: instancetypev1beta1.UpgradeFailed,
		Conditions: []instancetypev1beta1.ControllerRevisionUpgradeCondition{{
			Type:    instancetypev1beta1.ControllerRevisionUpgradeFailure,
			Status:  k8sv1.ConditionTrue,
			Reason:  upgradeFailureReason,
			Message: updateErr,
		}},
	}
	if err := c.statusUpdater.UpdateStatus(crUpgrade); err != nil {
		return fmt.Errorf("failure to update ControllerRevisionUpgrade with failure: %w", err)
	}
	return nil
}

func (c *UpgradeController) updateStatusWithRunning(crUpgrade *instancetypev1beta1.ControllerRevisionUpgrade) error {
	crUpgrade.Status = instancetypev1beta1.ControllerRevisionUpgradeStatus{
		Phase: instancetypev1beta1.UpgradeRunning,
	}
	if err := c.statusUpdater.UpdateStatus(crUpgrade); err != nil {
		return fmt.Errorf("failure to update ControllerRevisionUpgrade with running: %w", err)
	}
	return nil
}
