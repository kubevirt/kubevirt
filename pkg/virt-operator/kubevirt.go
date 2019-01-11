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
	"fmt"
	"reflect"
	"time"

	"kubevirt.io/kubevirt/pkg/virt-operator/creation"

	k8sv1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/tools/record"
	"k8s.io/client-go/util/workqueue"

	"kubevirt.io/kubevirt/pkg/api/v1"
	"kubevirt.io/kubevirt/pkg/controller"
	"kubevirt.io/kubevirt/pkg/kubecli"
	"kubevirt.io/kubevirt/pkg/log"
	"kubevirt.io/kubevirt/pkg/virt-operator/deletion"
	"kubevirt.io/kubevirt/pkg/virt-operator/util"
)

const (
	ConditionReasonDeploymentFailedExisting = "ExistingDeployment"
	ConditionReasonDeploymentFailedError    = "DeploymentFailed"
	ConditionReasonDeletionFailedError      = "DeletionFailed"
)

type KubeVirtController struct {
	clientset            kubecli.KubevirtClient
	queue                workqueue.RateLimitingInterface
	kubeVirtInformer     cache.SharedIndexInformer
	recorder             record.EventRecorder
	config               util.KubeVirtDeploymentConfig
	stores               util.Stores
	informers            util.Informers
	kubeVirtExpectations util.Expectations
}

func NewKubeVirtController(
	clientset kubecli.KubevirtClient,
	informer cache.SharedIndexInformer,
	recorder record.EventRecorder,
	stores util.Stores,
	informers util.Informers) *KubeVirtController {

	c := KubeVirtController{
		clientset:        clientset,
		queue:            workqueue.NewRateLimitingQueue(workqueue.DefaultControllerRateLimiter()),
		kubeVirtInformer: informer,
		recorder:         recorder,
		config:           util.GetConfig(),
		stores:           stores,
		informers:        informers,
		kubeVirtExpectations: util.Expectations{
			ServiceAccount:     controller.NewUIDTrackingControllerExpectations(controller.NewControllerExpectationsWithName("ServiceAccount")),
			ClusterRole:        controller.NewUIDTrackingControllerExpectations(controller.NewControllerExpectationsWithName("ClusterRole")),
			ClusterRoleBinding: controller.NewUIDTrackingControllerExpectations(controller.NewControllerExpectationsWithName("ClusterRoleBinding")),
			Role:               controller.NewUIDTrackingControllerExpectations(controller.NewControllerExpectationsWithName("Role")),
			RoleBinding:        controller.NewUIDTrackingControllerExpectations(controller.NewControllerExpectationsWithName("RoleBinding")),
			Crd:                controller.NewUIDTrackingControllerExpectations(controller.NewControllerExpectationsWithName("Crd")),
			Service:            controller.NewUIDTrackingControllerExpectations(controller.NewControllerExpectationsWithName("Service")),
			Deployment:         controller.NewUIDTrackingControllerExpectations(controller.NewControllerExpectationsWithName("Deployment")),
			DaemonSet:          controller.NewUIDTrackingControllerExpectations(controller.NewControllerExpectationsWithName("DaemonSet")),
		},
	}

	c.kubeVirtInformer.AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc:    c.addKubeVirt,
		DeleteFunc: c.deleteKubeVirt,
		UpdateFunc: c.updateKubeVirt,
	})

	c.informers.ServiceAccount.AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc: func(obj interface{}) {
			c.genericAddHandler(obj, c.kubeVirtExpectations.ServiceAccount)
		},
		DeleteFunc: func(obj interface{}) {
			c.genericDeleteHandler(obj, c.kubeVirtExpectations.ServiceAccount)
		},
		UpdateFunc: func(oldObj, newObj interface{}) {
			c.genericUpdateHandler(oldObj, newObj, c.kubeVirtExpectations.ServiceAccount)
		},
	})

	c.informers.ClusterRole.AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc: func(obj interface{}) {
			c.genericAddHandler(obj, c.kubeVirtExpectations.ClusterRole)
		},
		DeleteFunc: func(obj interface{}) {
			c.genericDeleteHandler(obj, c.kubeVirtExpectations.ClusterRole)
		},
		UpdateFunc: func(oldObj, newObj interface{}) {
			c.genericUpdateHandler(oldObj, newObj, c.kubeVirtExpectations.ClusterRole)
		},
	})

	c.informers.ClusterRoleBinding.AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc: func(obj interface{}) {
			c.genericAddHandler(obj, c.kubeVirtExpectations.ClusterRoleBinding)
		},
		DeleteFunc: func(obj interface{}) {
			c.genericDeleteHandler(obj, c.kubeVirtExpectations.ClusterRoleBinding)
		},
		UpdateFunc: func(oldObj, newObj interface{}) {
			c.genericUpdateHandler(oldObj, newObj, c.kubeVirtExpectations.ClusterRoleBinding)
		},
	})

	c.informers.Role.AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc: func(obj interface{}) {
			c.genericAddHandler(obj, c.kubeVirtExpectations.Role)
		},
		DeleteFunc: func(obj interface{}) {
			c.genericDeleteHandler(obj, c.kubeVirtExpectations.Role)
		},
		UpdateFunc: func(oldObj, newObj interface{}) {
			c.genericUpdateHandler(oldObj, newObj, c.kubeVirtExpectations.Role)
		},
	})

	c.informers.RoleBinding.AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc: func(obj interface{}) {
			c.genericAddHandler(obj, c.kubeVirtExpectations.RoleBinding)
		},
		DeleteFunc: func(obj interface{}) {
			c.genericDeleteHandler(obj, c.kubeVirtExpectations.RoleBinding)
		},
		UpdateFunc: func(oldObj, newObj interface{}) {
			c.genericUpdateHandler(oldObj, newObj, c.kubeVirtExpectations.RoleBinding)
		},
	})

	c.informers.Crd.AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc: func(obj interface{}) {
			c.genericAddHandler(obj, c.kubeVirtExpectations.Crd)
		},
		DeleteFunc: func(obj interface{}) {
			c.genericDeleteHandler(obj, c.kubeVirtExpectations.Crd)
		},
		UpdateFunc: func(oldObj, newObj interface{}) {
			c.genericUpdateHandler(oldObj, newObj, c.kubeVirtExpectations.Crd)
		},
	})

	c.informers.Service.AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc: func(obj interface{}) {
			c.genericAddHandler(obj, c.kubeVirtExpectations.Service)
		},
		DeleteFunc: func(obj interface{}) {
			c.genericDeleteHandler(obj, c.kubeVirtExpectations.Service)
		},
		UpdateFunc: func(oldObj, newObj interface{}) {
			c.genericUpdateHandler(oldObj, newObj, c.kubeVirtExpectations.Service)
		},
	})

	c.informers.Deployment.AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc: func(obj interface{}) {
			c.genericAddHandler(obj, c.kubeVirtExpectations.Deployment)
		},
		DeleteFunc: func(obj interface{}) {
			c.genericDeleteHandler(obj, c.kubeVirtExpectations.Deployment)
		},
		UpdateFunc: func(oldObj, newObj interface{}) {
			c.genericUpdateHandler(oldObj, newObj, c.kubeVirtExpectations.Deployment)
		},
	})

	c.informers.DaemonSet.AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc: func(obj interface{}) {
			c.genericAddHandler(obj, c.kubeVirtExpectations.DaemonSet)
		},
		DeleteFunc: func(obj interface{}) {
			c.genericDeleteHandler(obj, c.kubeVirtExpectations.DaemonSet)
		},
		UpdateFunc: func(oldObj, newObj interface{}) {
			c.genericUpdateHandler(oldObj, newObj, c.kubeVirtExpectations.DaemonSet)
		},
	})

	return &c
}

func (c *KubeVirtController) getOperatorKey() (string, error) {
	// XXX use owner references instead in general
	kvs := c.kubeVirtInformer.GetStore().List()
	if len(kvs) > 1 {
		log.Log.Errorf("More than one KubeVirt custom resource detectged: %v", len(kvs))
		return "", fmt.Errorf("more than one KubeVirt custom resource detectged: %v", len(kvs))
	}

	if len(kvs) == 1 {
		kv := kvs[0].(*v1.KubeVirt)
		return controller.KeyFunc(kv)
	}
	return "", nil
}

func (c *KubeVirtController) genericAddHandler(obj interface{}, expecter *controller.UIDTrackingControllerExpectations) {
	o := obj.(metav1.Object)

	if o.GetDeletionTimestamp() != nil {
		// on a restart of the controller manager, it's possible a new o shows up in a state that
		// is already pending deletion. Prevent the o from being a creation observation.
		c.genericDeleteHandler(obj, expecter)
		return
	}

	controllerKey, err := c.getOperatorKey()
	if controllerKey != "" && err == nil {
		expecter.CreationObserved(controllerKey)
		c.queue.Add(controllerKey)
	}
}

// When an object is updated, inform the kubevirt CR about the change
func (c *KubeVirtController) genericUpdateHandler(old, cur interface{}, expecter *controller.UIDTrackingControllerExpectations) {
	curObj := cur.(metav1.Object)
	oldObj := old.(metav1.Object)
	if curObj.GetResourceVersion() == oldObj.GetResourceVersion() {
		// Periodic resync will send update events for all known objects.
		// Two different versions of the same object will always have different RVs.
		return
	}

	if curObj.GetDeletionTimestamp() != nil {
		// having an object marked for deletion is enough to count as a deletion expectation
		c.genericDeleteHandler(curObj, expecter)
		return
	}
	key, err := c.getOperatorKey()
	if key != "" && err == nil {
		c.queue.Add(key)
	}
	return
}

// When an object is deleted, mark objects as deleted and wake up the kubevirt CR
func (c *KubeVirtController) genericDeleteHandler(obj interface{}, expecter *controller.UIDTrackingControllerExpectations) {
	var o metav1.Object
	tombstone, ok := obj.(cache.DeletedFinalStateUnknown)
	if ok {
		o, ok = tombstone.Obj.(metav1.Object)
		if !ok {
			log.Log.Reason(fmt.Errorf("tombstone contained object that is not a k8s object %#v", obj)).Error("Failed to process delete notification")
			return
		}
	} else if o, ok = obj.(metav1.Object); !ok {
		log.Log.Reason(fmt.Errorf("couldn't get object from %+v", obj)).Error("Failed to process delete notification")
		return
	}

	k, err := controller.KeyFunc(o)
	if err != nil {
		log.Log.Reason(err).Errorf("could not extract key from k8s object")
		return
	}

	key, err := c.getOperatorKey()
	if key != "" && err == nil {
		expecter.DeletionObserved(key, k)
		c.queue.Add(key)
	}
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

	// Wait for cache sync before we start the controller
	cache.WaitForCacheSync(stopCh, c.kubeVirtInformer.HasSynced)
	cache.WaitForCacheSync(stopCh, c.informers.ServiceAccount.HasSynced)
	cache.WaitForCacheSync(stopCh, c.informers.ClusterRole.HasSynced)
	cache.WaitForCacheSync(stopCh, c.informers.ClusterRoleBinding.HasSynced)
	cache.WaitForCacheSync(stopCh, c.informers.Role.HasSynced)
	cache.WaitForCacheSync(stopCh, c.informers.RoleBinding.HasSynced)
	cache.WaitForCacheSync(stopCh, c.informers.Crd.HasSynced)
	cache.WaitForCacheSync(stopCh, c.informers.Service.HasSynced)
	cache.WaitForCacheSync(stopCh, c.informers.Deployment.HasSynced)
	cache.WaitForCacheSync(stopCh, c.informers.DaemonSet.HasSynced)

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
		// when the resource is gone, deletion was handled already
		log.Log.Infof("KubeVirt resource not found")
		c.kubeVirtExpectations.DeleteExpectations(key)
		return nil
	}

	kv := obj.(*v1.KubeVirt)
	logger := log.Log.Object(kv)

	// If we can't extract the key we can't do anything
	_, err = controller.KeyFunc(kv)
	if err != nil {
		log.Log.Reason(err).Errorf("Could not extract the key from the custom resource, will do nothing and not requeue.")
		return nil
	}

	logger.Info("Handling KubeVirt resource")

	// only process the kubevirt deployment if all expectations are satisfied.
	needsSync := c.kubeVirtExpectations.SatisfiedExpectations(key)
	if !needsSync {
		logger.Info("Waiting for expectations to be fulfilled")
		return nil
	}

	// Adds of all types are not done in one go. We need to set an expectation of 0 so that we can add something
	c.kubeVirtExpectations.ResetExpectations(key)

	var syncError error
	kvCopy := kv.DeepCopy()

	if kv.DeletionTimestamp != nil {
		syncError = c.syncDeletion(kvCopy)
	} else {
		syncError = c.syncDeployment(kvCopy)
	}

	// If we detect a change on KubeVirt we update it
	if !reflect.DeepEqual(kv.Status, kvCopy.Status) ||
		!reflect.DeepEqual(kv.Finalizers, kvCopy.Finalizers) {

		_, err := c.clientset.KubeVirt(kv.Namespace).Update(kvCopy)

		if err != nil {
			logger.Reason(err).Errorf("Could not update the KubeVirt resource.")
			return err
		}
	}

	return syncError
}

func (c *KubeVirtController) syncDeployment(kv *v1.KubeVirt) error {
	logger := log.Log.Object(kv)
	logger.Infof("Handling deployment")

	if kv.Status.Phase == v1.KubeVirtPhaseDeployed {
		logger.Info("Is already deployed")
		return nil
	}

	// Set versions...
	if kv.Status.OperatorVersion == "" {
		util.SetVersions(kv, c.config)
	}

	// Set phase to deploying
	kv.Status.Phase = v1.KubeVirtPhaseDeploying

	// check if there is already an active KubeVirt deployment
	// TODO move this into a new validating webhook
	kvs := c.kubeVirtInformer.GetStore().List()
	for _, obj := range kvs {
		if fromStore, ok := obj.(v1.KubeVirt); ok {
			if fromStore.UID == kv.UID {
				continue
			}
			if isKubeVirtActive(&fromStore) {
				logger.Warningf("There is already a KubeVirt deployment!")
				util.UpdateCondition(kv, v1.KubeVirtConditionSynchronized, k8sv1.ConditionFalse, ConditionReasonDeploymentFailedExisting, "There is an active KubeVirt deployment")
				return nil
			}
		}
	}

	// add finalizer to prevent deletion of CR before KubeVirt was undeployed
	util.AddFinalizer(kv)

	// deploy
	objectsAdded, err := creation.Create(kv, c.config, c.stores, c.clientset, &c.kubeVirtExpectations)

	if err != nil {
		// deployment failed
		util.UpdateCondition(kv, v1.KubeVirtConditionSynchronized, k8sv1.ConditionFalse, ConditionReasonDeploymentFailedError, fmt.Sprintf("An error occurred during deployment: %v", err))
		if err != nil {
			logger.Errorf("Failed to set condition: %v", err)
		}
		return err
	}

	if objectsAdded == 0 {
		// deployment successful
		kv.Status.Phase = v1.KubeVirtPhaseDeployed
		kv.Status.Conditions = []v1.KubeVirtCondition{}

		logger.Info("KubeVirt deployed")
		return nil
	}

	logger.Info("Processed deployment for this round")
	return nil
}

func (c *KubeVirtController) syncDeletion(kv *v1.KubeVirt) error {
	logger := log.Log.Object(kv)
	logger.Info("Handling deletion")

	// delete
	if kv.Status.Phase == v1.KubeVirtPhaseDeleted {
		logger.Info("Is already deleted")
		return nil
	}

	// set phase to deleting
	kv.Status.Phase = v1.KubeVirtPhaseDeleting

	err := deletion.Delete(kv, c.clientset, c.stores, &c.kubeVirtExpectations)
	if err != nil {
		// deletion failed
		util.UpdateCondition(kv, v1.KubeVirtConditionSynchronized, k8sv1.ConditionFalse, ConditionReasonDeletionFailedError, fmt.Sprintf("An error occurred during deletion: %v", err))
		return err
	}

	if c.stores.AllEmpty() {

		// deletion successful
		kv.Status.Phase = v1.KubeVirtPhaseDeleted
		kv.Status.Conditions = []v1.KubeVirtCondition{}
		kv.Finalizers = []string{}

		logger.Info("KubeVirt deleted")

		return nil
	}

	logger.Info("Processed deletion for this round")
	return nil
}

func isKubeVirtActive(kv *v1.KubeVirt) bool {
	return kv.Status.Phase != v1.KubeVirtPhaseDeleted
}
