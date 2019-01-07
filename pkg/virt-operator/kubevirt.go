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
 * Copyright 2018 Red Hat, Inc.
 *
 */

package virt_operator

import (
	"time"

	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/tools/record"
	"k8s.io/client-go/util/workqueue"

	"kubevirt.io/kubevirt/pkg/api/v1"
	"kubevirt.io/kubevirt/pkg/controller"
	"kubevirt.io/kubevirt/pkg/kubecli"
	"kubevirt.io/kubevirt/pkg/log"
	"kubevirt.io/kubevirt/pkg/virt-operator/creation"
	"kubevirt.io/kubevirt/pkg/virt-operator/deletion"
	"kubevirt.io/kubevirt/pkg/virt-operator/util"
)

type KubeVirtController struct {
	clientset        kubecli.KubevirtClient
	queue            workqueue.RateLimitingInterface
	kubeVirtInformer cache.SharedIndexInformer
	recorder         record.EventRecorder
	config           util.KubeVirtDeploymentConfig
}

func NewKubeVirtController(
	clientset kubecli.KubevirtClient,
	informer cache.SharedIndexInformer,
	recorder record.EventRecorder) *KubeVirtController {

	c := KubeVirtController{
		clientset:        clientset,
		queue:            workqueue.NewRateLimitingQueue(workqueue.DefaultControllerRateLimiter()),
		kubeVirtInformer: informer,
		recorder:         recorder,
		config:           util.GetConfig(),
	}

	c.kubeVirtInformer.AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc:    c.addKubeVirt,
		DeleteFunc: c.deleteKubeVirt,
		UpdateFunc: c.updateKubeVirt,
	})

	return &c
}

func (c *KubeVirtController) addKubeVirt(obj interface{}) {
	c.enqueueKubeVirt(obj)
}

func (c *KubeVirtController) deleteKubeVirt(obj interface{}) {
	c.enqueueKubeVirt(obj)
}

func (c *KubeVirtController) updateKubeVirt(old, curr interface{}) {
	c.enqueueKubeVirt(curr)
}

func (c *KubeVirtController) enqueueKubeVirt(obj interface{}) {
	logger := log.Log
	kv := obj.(*v1.KubeVirt)
	key, err := controller.KeyFunc(kv)
	if err != nil {
		logger.Object(kv).Reason(err).Error("Failed to extract key from KubeVirt.")
	}
	c.queue.Add(key)
}

func (c *KubeVirtController) Run(threadiness int, stopCh chan struct{}) {
	defer controller.HandlePanic()
	defer c.queue.ShutDown()
	log.Log.Info("Starting KubeVirt controller.")

	// Wait for cache sync before we start the pod controller
	cache.WaitForCacheSync(stopCh, c.kubeVirtInformer.HasSynced)

	// Start the actual work
	for i := 0; i < threadiness; i++ {
		go wait.Until(c.runWorker, time.Second, stopCh)
	}

	<-stopCh
	log.Log.Info("Stopping KubeVirt controller.")
}

func (c *KubeVirtController) runWorker() {
	for c.Execute() {
	}
}

func (c *KubeVirtController) Execute() bool {
	key, quit := c.queue.Get()
	if quit {
		return false
	}
	defer c.queue.Done(key)
	err := c.execute(key.(string))

	if err != nil {
		log.Log.Reason(err).Infof("reenqueuing KubeVirt %v", key)
		c.queue.AddRateLimited(key)
	} else {
		log.Log.V(4).Infof("processed KubeVirt %v", key)
		c.queue.Forget(key)
	}
	return true
}

func (c *KubeVirtController) execute(key string) error {

	// Fetch the latest KubeVirt from cache
	obj, exists, err := c.kubeVirtInformer.GetStore().GetByKey(key)

	if err != nil {
		return err
	}

	if !exists {
		// when the resource is gone, deletion was hanlded already
		log.Log.Infof("KubeVirt resource not found")
		return nil
	}

	kv := obj.(*v1.KubeVirt)
	logger := log.Log.Object(kv)

	if kv.DeletionTimestamp != nil {

		if isKubeVirtDeleting(kv) {
			logger.Infof("KubeVirt deletion is/was already processed")
			return nil
		}

		log.Log.Infof("Handling deleted KubeVirt object %+v", kv)
		err = deletion.Delete(kv, c.clientset)
		if err != nil {
			err1 := util.UpdatePhase(kv, v1.KubeVirtPhaseDeletionFailed, c.clientset)
			if err1 != nil {
				log.Log.Errorf("Failed to update phase: %v", err1)
			}
		} else {
			err1 := util.UpdatePhase(kv, v1.KubeVirtPhaseDeleted, c.clientset)
			if err1 != nil {
				log.Log.Errorf("Failed to update phase: %v", err1)
			} else {
				err2 := util.RemoveFinalizer(kv, c.clientset)
				if err2 != nil {
					log.Log.Errorf("Failed to remove finalizer: %v", err2)
				}
			}
		}
		return err
	}

	if kv.Status.Phase != v1.KubeVirtPhaseUnset {
		logger.Infof("KubeVirt creation is/was already processed")
		return nil
	}

	// Add finalizer to prevent deletion of CR before KuveVirt was undeployed
	err = util.AddFinalizer(kv, c.clientset)
	if err != nil {
		log.Log.Errorf("Failed to add finalizer: %v", err)
		return err
	}

	// check if there is already an active KubeVirt deployment
	kvs := c.kubeVirtInformer.GetStore().List()
	for _, obj := range kvs {
		if fromStore, ok := obj.(v1.KubeVirt); ok {
			if fromStore.UID == kv.UID {
				continue
			}
			if isKubeVirtActive(&fromStore) {
				logger.Warningf("There is already a KubeVirt deployment!")
				util.UpdatePhase(kv, v1.KubeVirtPhaseIgnored, c.clientset)
				return nil
			}
		}
	}

	logger.Infof("handling KubeVirt object")

	err = creation.Create(kv, c.config, c.clientset)
	if err != nil {
		err1 := util.UpdatePhase(kv, v1.KubeVirtPhaseDeployFailed, c.clientset)
		if err1 != nil {
			log.Log.Errorf("Failed to update phase: %v", err1)
		}
		// TODO clean up what already was deployed...?
	} else {
		err1 := util.UpdatePhase(kv, v1.KubeVirtPhaseDeployed, c.clientset)
		if err1 != nil {
			log.Log.Errorf("Failed to update phase: %v", err1)
		}
	}

	return err
}

func isKubeVirtActive(kv *v1.KubeVirt) bool {
	return kv.Status.Phase != v1.KubeVirtPhaseUnset &&
		kv.Status.Phase != v1.KubeVirtPhaseDeployFailed &&
		kv.Status.Phase != v1.KubeVirtPhaseDeletionFailed &&
		kv.Status.Phase != v1.KubeVirtPhaseDeleted
}

func isKubeVirtDeleting(kv *v1.KubeVirt) bool {
	return kv.Status.Phase == v1.KubeVirtPhaseDeleting ||
		kv.Status.Phase == v1.KubeVirtPhaseDeletionFailed ||
		kv.Status.Phase == v1.KubeVirtPhaseDeleted
}
