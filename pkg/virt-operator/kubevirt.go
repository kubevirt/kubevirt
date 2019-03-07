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
	"sync"
	"time"

	appsv1 "k8s.io/api/apps/v1"
	batchv1 "k8s.io/api/batch/v1"
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
	"kubevirt.io/kubevirt/pkg/virt-operator/creation"
	"kubevirt.io/kubevirt/pkg/virt-operator/deletion"
	"kubevirt.io/kubevirt/pkg/virt-operator/install-strategy"
	"kubevirt.io/kubevirt/pkg/virt-operator/util"
)

const (
	ConditionReasonDeploymentFailedExisting = "ExistingDeployment"
	ConditionReasonDeploymentFailedError    = "DeploymentFailed"
	ConditionReasonDeletionFailedError      = "DeletionFailed"
	ConditionReasonDeploymentCreated        = "AllResourcesCreated"
	ConditionReasonDeploymentReady          = "AllComponentsReady"
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
	installStrategyMutex sync.Mutex
	installStrategyMap   map[string]*installstrategy.InstallStrategy
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
			ServiceAccount:           controller.NewUIDTrackingControllerExpectations(controller.NewControllerExpectationsWithName("ServiceAccount")),
			ClusterRole:              controller.NewUIDTrackingControllerExpectations(controller.NewControllerExpectationsWithName("ClusterRole")),
			ClusterRoleBinding:       controller.NewUIDTrackingControllerExpectations(controller.NewControllerExpectationsWithName("ClusterRoleBinding")),
			Role:                     controller.NewUIDTrackingControllerExpectations(controller.NewControllerExpectationsWithName("Role")),
			RoleBinding:              controller.NewUIDTrackingControllerExpectations(controller.NewControllerExpectationsWithName("RoleBinding")),
			Crd:                      controller.NewUIDTrackingControllerExpectations(controller.NewControllerExpectationsWithName("Crd")),
			Service:                  controller.NewUIDTrackingControllerExpectations(controller.NewControllerExpectationsWithName("Service")),
			Deployment:               controller.NewUIDTrackingControllerExpectations(controller.NewControllerExpectationsWithName("Deployment")),
			DaemonSet:                controller.NewUIDTrackingControllerExpectations(controller.NewControllerExpectationsWithName("DaemonSet")),
			InstallStrategyConfigMap: controller.NewUIDTrackingControllerExpectations(controller.NewControllerExpectationsWithName("ConfigMap")),
			InstallStrategyJob:       controller.NewUIDTrackingControllerExpectations(controller.NewControllerExpectationsWithName("Jobs")),
		},
		installStrategyMap: make(map[string]*installstrategy.InstallStrategy),
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

	c.informers.InstallStrategyConfigMap.AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc: func(obj interface{}) {
			c.genericAddHandler(obj, c.kubeVirtExpectations.InstallStrategyConfigMap)
		},
		DeleteFunc: func(obj interface{}) {
			c.genericDeleteHandler(obj, c.kubeVirtExpectations.InstallStrategyConfigMap)
		},
		UpdateFunc: func(oldObj, newObj interface{}) {
			c.genericUpdateHandler(oldObj, newObj, c.kubeVirtExpectations.InstallStrategyConfigMap)
		},
	})

	c.informers.InstallStrategyJob.AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc: func(obj interface{}) {
			c.genericAddHandler(obj, c.kubeVirtExpectations.InstallStrategyJob)
		},
		DeleteFunc: func(obj interface{}) {
			c.genericDeleteHandler(obj, c.kubeVirtExpectations.InstallStrategyJob)
		},
		UpdateFunc: func(oldObj, newObj interface{}) {
			c.genericUpdateHandler(oldObj, newObj, c.kubeVirtExpectations.InstallStrategyJob)
		},
	})

	return &c
}

func (c *KubeVirtController) getKubeVirtKey() (string, error) {
	// XXX use owner references instead in general
	kvs := c.kubeVirtInformer.GetStore().List()
	if len(kvs) > 1 {
		log.Log.Errorf("More than one KubeVirt custom resource detected: %v", len(kvs))
		return "", fmt.Errorf("more than one KubeVirt custom resource detected: %v", len(kvs))
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

	controllerKey, err := c.getKubeVirtKey()
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

	if oldObj.GetDeletionTimestamp() == nil && curObj.GetDeletionTimestamp() != nil {
		// having an object marked for deletion is enough to count as a deletion expectation
		c.genericDeleteHandler(curObj, expecter)
		return
	}

	key, err := c.getKubeVirtKey()
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

	key, err := c.getKubeVirtKey()
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
	cache.WaitForCacheSync(stopCh, c.informers.SCC.HasSynced)
	cache.WaitForCacheSync(stopCh, c.informers.InstallStrategyConfigMap.HasSynced)
	cache.WaitForCacheSync(stopCh, c.informers.InstallStrategyJob.HasSynced)

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

func (c *KubeVirtController) generateInstallStrategyJob(kv *v1.KubeVirt, imageTag string, imageRegistry string) *batchv1.Job {

	pullPolicy := k8sv1.PullIfNotPresent
	if string(kv.Spec.ImagePullPolicy) != "" {
		pullPolicy = kv.Spec.ImagePullPolicy
	}
	job := &batchv1.Job{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "batch/v1",
			Kind:       "Job",
		},

		ObjectMeta: metav1.ObjectMeta{
			Namespace: kv.Namespace,
			Name:      fmt.Sprintf("virt-install-strategy-job-%s", imageTag),
			Labels: map[string]string{
				v1.AppLabel:                    "",
				v1.ManagedByLabel:              v1.ManagedByLabelOperatorValue,
				v1.InstallStrategyVersionLabel: imageTag,
			},
		},
		Spec: batchv1.JobSpec{
			Template: k8sv1.PodTemplateSpec{

				Spec: k8sv1.PodSpec{
					ServiceAccountName: "kubevirt-operator",
					RestartPolicy:      k8sv1.RestartPolicyNever,
					Containers: []k8sv1.Container{
						{
							Name:            "install-strategy-upload",
							Image:           fmt.Sprintf("%s/%s:%s", imageRegistry, "virt-operator", imageTag),
							ImagePullPolicy: pullPolicy,
							Command: []string{
								"virt-operator",
								"--dump-install-strategy",
							},
							Env: []k8sv1.EnvVar{
								{
									Name:  util.OperatorImageEnvName,
									Value: fmt.Sprintf("%s/%s:%s", imageRegistry, "virt-operator", imageTag),
								},
							},
						},
					},
				},
			},
		},
	}

	return job
}

func (c *KubeVirtController) garbageCollectInstallStrategyJobs() error {
	batch := c.clientset.BatchV1()
	jobs := c.stores.InstallStrategyJobCache.List()

	for _, obj := range jobs {
		job, ok := obj.(*batchv1.Job)
		if !ok {
			continue
		}
		if job.Status.CompletionTime == nil {
			continue
		}

		propagationPolicy := metav1.DeletePropagationForeground
		err := batch.Jobs(job.Namespace).Delete(job.Name, &metav1.DeleteOptions{
			PropagationPolicy: &propagationPolicy,
		})
		if err != nil {
			return err
		}
		log.Log.Object(job).Infof("Garbage collected completed install strategy job")
	}

	return nil
}

func (c *KubeVirtController) getInstallStrategyFromMap(version string) (*installstrategy.InstallStrategy, bool) {
	c.installStrategyMutex.Lock()
	defer c.installStrategyMutex.Unlock()

	strategy, ok := c.installStrategyMap[version]
	return strategy, ok
}

func (c *KubeVirtController) deleteAllInstallStrategy() error {
	for _, obj := range c.stores.InstallStrategyConfigMapCache.List() {

		configMap, ok := obj.(*k8sv1.ConfigMap)
		if ok {
			err := c.clientset.CoreV1().ConfigMaps(configMap.Namespace).Delete(configMap.Name, &metav1.DeleteOptions{})
			if err != nil {
				return err
			}
		}
	}
	c.installStrategyMutex.Lock()
	defer c.installStrategyMutex.Unlock()
	// reset the local map
	c.installStrategyMap = make(map[string]*installstrategy.InstallStrategy)

	return nil
}

// Loads install strategies into memory, and generates jobs to
// create install strategies that don't exist yet.
func (c *KubeVirtController) loadInstallStrategy(kv *v1.KubeVirt) (*installstrategy.InstallStrategy, bool, error) {

	kvkey, err := controller.KeyFunc(kv)
	if err != nil {
		return nil, true, err
	}

	// 1. see if we already loaded the install strategy
	strategy, ok := c.getInstallStrategyFromMap(c.config.ImageTag)
	if ok {
		// we already loaded this strategy into memory
		return strategy, false, nil
	}

	// 2. look for install strategy config map in cache.
	strategy, err = installstrategy.LoadInstallStrategyFromCache(c.stores, kv.Namespace, c.config.ImageTag)
	if err == nil {
		c.installStrategyMutex.Lock()
		defer c.installStrategyMutex.Unlock()
		c.installStrategyMap[c.config.ImageTag] = strategy
		log.Log.Infof("Loaded install strategy for kubevirt version %s into cache", c.config.ImageTag)
		return strategy, false, nil
	}

	log.Log.Infof("Install strategy config map not loaded. reason: %v", err)

	// 3. See if we have a pending job in flight for this install strategy.
	batch := c.clientset.BatchV1()
	job := c.generateInstallStrategyJob(kv, c.config.ImageTag, c.config.ImageRegistry)

	obj, exists, _ := c.stores.InstallStrategyJobCache.Get(job)
	if exists {
		cachedJob := obj.(*batchv1.Job)

		if cachedJob.Status.CompletionTime != nil {
			// job completed but we don't have a install strategy still
			// delete the job and we'll re-execute it once it is removed.

			log.Log.Object(cachedJob).Errorf("Job failed to create install strategy for version %s", c.config.ImageTag)
			if cachedJob.DeletionTimestamp == nil {

				// Just in case there's an issue causing the job to fail
				// immediately after being posted, lets perform a rudimentary
				// for of rate-limiting for how quickly we'll re-attempt.
				// TODO there's an alpha feature that lets us set a TTL on the job
				// itself which will ensure it is automatically cleaned up for us
				// after completion. That feature is feature-gated and isn't something
				// we can depend on right now though.
				now := time.Now().UTC().Unix()
				secondsSinceCompletion := now - cachedJob.Status.CompletionTime.UTC().Unix()
				if secondsSinceCompletion < 10 {
					secondsLeft := int64(10)
					if secondsSinceCompletion > 0 {
						secondsLeft = secondsSinceCompletion
					}
					c.queue.AddAfter(kvkey, time.Duration(secondsLeft)*time.Second)

				} else {
					key, err := controller.KeyFunc(cachedJob)
					if err != nil {
						return nil, true, err
					}

					c.kubeVirtExpectations.InstallStrategyJob.AddExpectedDeletion(kvkey, key)
					propagationPolicy := metav1.DeletePropagationForeground
					err = batch.Jobs(kv.Namespace).Delete(cachedJob.Name, &metav1.DeleteOptions{
						PropagationPolicy: &propagationPolicy,
					})
					if err != nil {
						c.kubeVirtExpectations.InstallStrategyJob.DeletionObserved(kvkey, key)

						return nil, true, err
					}
					log.Log.Object(cachedJob).Errorf("Deleting job for install strategy version %s because configmap was not generated", c.config.ImageTag)
				}
				// waiting on deleted job to disappear before re-creating it.
				return nil, true, err
			}

			// waiting on deleted job to disappear before re-creating it.
			return nil, true, nil
		}
	}

	// 4. execute a job to generate the install strategy for the target version of KubeVirt that's being installed/updated
	c.kubeVirtExpectations.InstallStrategyJob.RaiseExpectations(kvkey, 1, 0)
	_, err = batch.Jobs(kv.Namespace).Create(job)
	if err != nil {
		c.kubeVirtExpectations.InstallStrategyJob.LowerExpectations(kvkey, 1, 0)
		return nil, true, err
	}
	log.Log.Infof("Created job to generate install strategy configmap for version %s", c.config.ImageTag)

	// pending is true here because we're waiting on the job
	// to generate the install strategy
	return nil, true, nil
}

func (c *KubeVirtController) syncDeployment(kv *v1.KubeVirt) error {
	logger := log.Log.Object(kv)
	logger.Infof("Handling deployment")

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
		if fromStore, ok := obj.(*v1.KubeVirt); ok {
			if fromStore.UID == kv.UID {
				continue
			}
			if isKubeVirtActive(fromStore) {
				logger.Warningf("There is already a KubeVirt deployment!")
				util.UpdateCondition(kv, v1.KubeVirtConditionSynchronized, k8sv1.ConditionFalse, ConditionReasonDeploymentFailedExisting, "There is an active KubeVirt deployment")
				return nil
			}
		}
	}

	// add finalizer to prevent deletion of CR before KubeVirt was undeployed
	util.AddFinalizer(kv)

	strategy, pending, err := c.loadInstallStrategy(kv)
	if err != nil {
		return err
	}

	// we're waiting on the job to finish and the config map to be created
	if pending {
		return nil
	}

	// once all the install strategies are loaded, garbage collect any
	// install strategy jobs that were created.
	c.garbageCollectInstallStrategyJobs()

	// deploy
	objectsAdded, err := creation.Create(kv, c.stores, c.clientset, &c.kubeVirtExpectations, strategy)

	if err != nil {
		// deployment failed
		util.UpdateCondition(kv, v1.KubeVirtConditionSynchronized, k8sv1.ConditionFalse, ConditionReasonDeploymentFailedError, fmt.Sprintf("An error occurred during deployment: %v", err))

		logger.Errorf("Failed to create all resources: %v", err)
		return err
	}
	util.RemoveCondition(kv, v1.KubeVirtConditionSynchronized)

	if objectsAdded == 0 {

		// add Created condition
		util.UpdateCondition(kv, v1.KubeVirtConditionCreated, k8sv1.ConditionTrue, ConditionReasonDeploymentCreated, "All resources were created.")
		logger.Info("All KubeVirt resources created")

		// check if components are ready
		if c.isReady() {
			logger.Info("All KubeVirt components ready")
			kv.Status.Phase = v1.KubeVirtPhaseDeployed
			util.UpdateCondition(kv, v1.KubeVirtConditionReady, k8sv1.ConditionTrue, ConditionReasonDeploymentReady, "All components are ready.")
			return nil
		}
		util.RemoveCondition(kv, v1.KubeVirtConditionReady)

	} else {
		util.RemoveCondition(kv, v1.KubeVirtConditionCreated)
	}

	logger.Info("Processed deployment for this round")
	return nil
}

func (c *KubeVirtController) isReady() bool {

	for _, obj := range c.stores.DeploymentCache.List() {
		if deployment, ok := obj.(*appsv1.Deployment); ok {
			var specReplicas int32 = 1
			if deployment.Spec.Replicas != nil {
				specReplicas = *deployment.Spec.Replicas
			}
			if specReplicas != deployment.Status.Replicas ||
				deployment.Status.Replicas != deployment.Status.ReadyReplicas {
				log.Log.V(4).Infof("Deployment %v not ready yet", deployment.Name)
				return false
			}
		}
	}

	for _, obj := range c.stores.DaemonSetCache.List() {
		if daemonset, ok := obj.(*appsv1.DaemonSet); ok {
			if daemonset.Status.DesiredNumberScheduled == 0 ||
				daemonset.Status.DesiredNumberScheduled != daemonset.Status.NumberReady {

				log.Log.V(4).Infof("DaemonSet %v not ready yet", daemonset.Name)
				return false
			}
		}
	}

	return true
}

func (c *KubeVirtController) syncDeletion(kv *v1.KubeVirt) error {
	logger := log.Log.Object(kv)
	logger.Info("Handling deletion")

	// set phase to deleting
	kv.Status.Phase = v1.KubeVirtPhaseDeleting

	// remove created and ready conditions
	util.RemoveCondition(kv, v1.KubeVirtConditionCreated)
	util.RemoveCondition(kv, v1.KubeVirtConditionReady)

	err := deletion.Delete(kv, c.clientset, c.stores, &c.kubeVirtExpectations)
	if err != nil {
		// deletion failed
		util.UpdateCondition(kv, v1.KubeVirtConditionSynchronized, k8sv1.ConditionFalse, ConditionReasonDeletionFailedError, fmt.Sprintf("An error occurred during deletion: %v", err))
		return err
	}

	err = c.deleteAllInstallStrategy()
	if err != nil {
		// garbage collection of install strategies failed
		util.UpdateCondition(kv, v1.KubeVirtConditionSynchronized, k8sv1.ConditionFalse, ConditionReasonDeletionFailedError, fmt.Sprintf("An error occurred during deletion: %v", err))
		return err
	}

	util.RemoveCondition(kv, v1.KubeVirtConditionSynchronized)

	if c.stores.AllEmpty() {

		// deletion successful
		kv.Status.Phase = v1.KubeVirtPhaseDeleted

		// remove finalizer
		kv.Finalizers = nil

		logger.Info("KubeVirt deleted")

		return nil
	}

	logger.Info("Processed deletion for this round")
	return nil
}

func isKubeVirtActive(kv *v1.KubeVirt) bool {
	return kv.Status.Phase != v1.KubeVirtPhaseDeleted
}
