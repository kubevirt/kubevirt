/*
Copyright 2018 The CDI Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package controller

import (
	"context"
	"encoding/json"
	"fmt"
	"reflect"

	"github.com/go-logr/logr"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	logf "sigs.k8s.io/controller-runtime/pkg/runtime/log"
	"sigs.k8s.io/controller-runtime/pkg/source"

	"github.com/kelseyhightower/envconfig"

	cdiv1alpha1 "kubevirt.io/containerized-data-importer/pkg/apis/core/v1alpha1"
	cdicluster "kubevirt.io/containerized-data-importer/pkg/operator/resources/cluster"
	cdinamespaced "kubevirt.io/containerized-data-importer/pkg/operator/resources/namespaced"
	"kubevirt.io/containerized-data-importer/pkg/util"
)

const (
	// ConfigMapName is the name of the CDI Operator config map
	// used to determine which CDI instance is "active"
	// and maybe other stuff someday
	ConfigMapName = "cdi-config"

	finalizerName = "operator.cdi.kubevirt.io"
)

var log = logf.Log.WithName("controler")

// Add creates a new CDI Controller and adds it to the Manager. The Manager will set fields on the Controller
// and Start it when the Manager is Started.
func Add(mgr manager.Manager) error {
	r, err := newReconciler(mgr)
	if err != nil {
		return err
	}
	return r.add(mgr)
}

// newReconciler returns a new reconcile.Reconciler
func newReconciler(mgr manager.Manager) (*ReconcileCDI, error) {
	var namespacedArgs cdinamespaced.FactoryArgs
	namespace := util.GetNamespace()
	clusterArgs := &cdicluster.FactoryArgs{Namespace: namespace}

	err := envconfig.Process("", &namespacedArgs)
	if err != nil {
		return nil, err
	}

	namespacedArgs.Namespace = namespace

	log.Info("", "VARS", fmt.Sprintf("%+v", namespacedArgs))

	r := &ReconcileCDI{
		client:         mgr.GetClient(),
		scheme:         mgr.GetScheme(),
		namespace:      namespace,
		clusterArgs:    clusterArgs,
		namespacedArgs: &namespacedArgs,
	}
	return r, nil
}

var _ reconcile.Reconciler = &ReconcileCDI{}

// ReconcileCDI reconciles a CDI object
type ReconcileCDI struct {
	// This client, initialized using mgr.Client() above, is a split client
	// that reads objects from the cache and writes to the apiserver
	client client.Client
	scheme *runtime.Scheme

	namespace      string
	clusterArgs    *cdicluster.FactoryArgs
	namespacedArgs *cdinamespaced.FactoryArgs
}

// Reconcile reads that state of the cluster for a CDI object and makes changes based on the state read
// and what is in the CDI.Spec
// Note:
// The Controller will requeue the Request to be processed again if the returned error is non-nil or
// Result.Requeue is true, otherwise upon completion it will remove the work from the queue.
func (r *ReconcileCDI) Reconcile(request reconcile.Request) (reconcile.Result, error) {
	reqLogger := log.WithValues("Request.Namespace", request.Namespace, "Request.Name", request.Name)
	reqLogger.Info("Reconciling CDI")

	// Fetch the CDI instance
	// check at cluster level
	cr := &cdiv1alpha1.CDI{}
	crKey := client.ObjectKey{Namespace: "", Name: request.NamespacedName.Name}
	if err := r.client.Get(context.TODO(), crKey, cr); err != nil {
		if errors.IsNotFound(err) {
			// Request object not found, could have been deleted after reconcile request.
			// Return and don't requeue
			reqLogger.Info("CDI CR no longer exists")
			return reconcile.Result{}, nil
		}
		return reconcile.Result{}, err
	}

	// mid delete
	if cr.DeletionTimestamp != nil {
		reqLogger.Info("Doing reconcile delete")
		return r.reconcileDelete(reqLogger, cr)
	}

	configMap, err := r.getConfigMap()
	if err != nil {
		return reconcile.Result{}, err
	}

	if configMap == nil {
		// let's try to create stuff
		if cr.Status.Phase == "" {
			reqLogger.Info("Doing reconcile create")
			return r.reconcileCreate(reqLogger, cr)
		}

		reqLogger.Info("Reconciling to error state, no configmap")

		// we are in a weird state
		return r.reconcileError(reqLogger, cr)
	}

	// do we even care about this CR?
	if !metav1.IsControlledBy(configMap, cr) {
		reqLogger.Info("Reconciling to error state, unwanted CDI object")

		return r.reconcileError(reqLogger, cr)
	}

	reqLogger.Info("Doing reconcile update")

	// should be the usual case
	return r.reconcileUpdate(reqLogger, cr)
}

func (r *ReconcileCDI) reconcileCreate(logger logr.Logger, cr *cdiv1alpha1.CDI) (reconcile.Result, error) {
	// claim the configmap
	if err := r.createConfigMap(cr); err != nil {
		return reconcile.Result{}, err
	}

	logger.Info("ConfigMap created successfully")

	if err := r.crInit(cr); err != nil {
		return reconcile.Result{}, err
	}

	logger.Info("Successfully entered Deploying state")

	return r.reconcileUpdate(logger, cr)
}

func (r *ReconcileCDI) reconcileUpdate(logger logr.Logger, cr *cdiv1alpha1.CDI) (reconcile.Result, error) {
	if err := r.syncPrivilegedAccounts(logger, cr, true); err != nil {
		return reconcile.Result{}, err
	}

	resources, err := r.getAllResources(cr)
	if err != nil {
		return reconcile.Result{}, err
	}

	for _, desiredRuntimeObj := range resources {
		desiredMetaObj := desiredRuntimeObj.(metav1.Object)

		// use reflection to create default instance of desiredRuntimeObj type
		typ := reflect.ValueOf(desiredRuntimeObj).Elem().Type()
		currentRuntimeObj := reflect.New(typ).Interface().(runtime.Object)

		key := client.ObjectKey{
			Namespace: desiredMetaObj.GetNamespace(),
			Name:      desiredMetaObj.GetName(),
		}
		err = r.client.Get(context.TODO(), key, currentRuntimeObj)

		if err != nil {
			if !errors.IsNotFound(err) {
				return reconcile.Result{}, err
			}

			if err = controllerutil.SetControllerReference(cr, desiredMetaObj, r.scheme); err != nil {
				return reconcile.Result{}, err
			}

			if err = r.client.Create(context.TODO(), desiredRuntimeObj); err != nil {
				logger.Error(err, "")
				return reconcile.Result{}, err
			}

			logger.Info("Resource created",
				"namespace", desiredMetaObj.GetNamespace(),
				"name", desiredMetaObj.GetName(),
				"type", fmt.Sprintf("%T", desiredMetaObj))
		} else {
			currentMetaObj := currentRuntimeObj.(metav1.Object)

			// allow users to add new annotations (but not change ours)
			mergeLabelsAndAnnotations(currentMetaObj, desiredMetaObj)

			desiredBytes, err := json.Marshal(desiredRuntimeObj)
			if err != nil {
				return reconcile.Result{}, err
			}

			if err = json.Unmarshal(desiredBytes, currentRuntimeObj); err != nil {
				return reconcile.Result{}, err
			}

			if err = r.client.Update(context.TODO(), currentRuntimeObj); err != nil {
				return reconcile.Result{}, err
			}

			logger.Info("Resource updated",
				"namespace", desiredMetaObj.GetNamespace(),
				"name", desiredMetaObj.GetName(),
				"type", fmt.Sprintf("%T", desiredMetaObj))
		}
	}

	if cr.Status.Phase != cdiv1alpha1.CDIPhaseDeployed {
		cr.Status.ObservedVersion = r.namespacedArgs.DockerTag
		if err = r.crUpdate(cdiv1alpha1.CDIPhaseDeployed, cr); err != nil {
			return reconcile.Result{}, err
		}

		logger.Info("Successfully entered Deployed state")
	}

	if err = r.checkReady(logger, cr); err != nil {
		return reconcile.Result{}, err
	}

	return reconcile.Result{}, nil
}

// I hate that this function exists, but major refactoring required to make CDI CR the owner of all the things
func (r *ReconcileCDI) reconcileDelete(logger logr.Logger, cr *cdiv1alpha1.CDI) (reconcile.Result, error) {
	i := -1
	for j, f := range cr.Finalizers {
		if f == finalizerName {
			i = j
			break
		}
	}

	// already done whatever we wanted to do
	if i == -1 {
		return reconcile.Result{}, nil
	}

	if cr.Status.Phase != cdiv1alpha1.CDIPhaseDeleting {
		if err := r.crUpdate(cdiv1alpha1.CDIPhaseDeleting, cr); err != nil {
			return reconcile.Result{}, err
		}
	}

	// delete all deployments
	deployments, err := r.getAllDeployments(cr)
	if err != nil {
		return reconcile.Result{}, err
	}

	for _, deployment := range deployments {
		err = r.client.Delete(context.TODO(), deployment, func(opts *client.DeleteOptions) {
			p := metav1.DeletePropagationForeground
			opts.PropagationPolicy = &p
		})
		if err != nil {
			if errors.IsNotFound(err) {
				continue
			}

			return reconcile.Result{}, err
		}
	}

	lo := &client.ListOptions{}
	// maybe use different selectors?
	lo.SetLabelSelector("cdi.kubevirt.io")

	// delete pods
	podList := &corev1.PodList{}
	if err = r.client.List(context.TODO(), lo, podList); err != nil {
		return reconcile.Result{}, err
	}

	for _, pod := range podList.Items {
		logger.Info("Deleting pod", "Name", pod.Name, "Namespace", pod.Namespace)
		if err = r.client.Delete(context.TODO(), &pod); err != nil {
			return reconcile.Result{}, err
		}
	}

	// delete services (from upload)
	serviceList := &corev1.ServiceList{}
	if err = r.client.List(context.TODO(), lo, serviceList); err != nil {
		return reconcile.Result{}, err
	}

	for _, service := range serviceList.Items {
		logger.Info("Deleting service", "Name", service.Name, "Namespace", service.Namespace)
		if err = r.client.Delete(context.TODO(), &service); err != nil {
			return reconcile.Result{}, err
		}
	}

	if err = r.syncPrivilegedAccounts(logger, cr, false); err != nil {
		return reconcile.Result{}, err
	}

	cr.Finalizers = append(cr.Finalizers[:i], cr.Finalizers[i+1:]...)

	if err := r.crUpdate(cdiv1alpha1.CDIPhaseDeleted, cr); err != nil {
		return reconcile.Result{}, err
	}

	logger.Info("Finalizer complete")

	return reconcile.Result{}, nil
}

func (r *ReconcileCDI) reconcileError(logger logr.Logger, cr *cdiv1alpha1.CDI) (reconcile.Result, error) {
	if err := r.crError(cr); err != nil {
		return reconcile.Result{}, err
	}

	return reconcile.Result{}, nil
}

func (r *ReconcileCDI) checkReady(logger logr.Logger, cr *cdiv1alpha1.CDI) error {
	deployments, err := r.getAllDeployments(cr)
	if err != nil {
		return err
	}

	for _, deployment := range deployments {
		key := client.ObjectKey{Namespace: deployment.Namespace, Name: deployment.Name}

		if err = r.client.Get(context.TODO(), key, deployment); err != nil {
			return err
		}

		if deployment.Status.Replicas != deployment.Status.ReadyReplicas {
			if err = r.conditionRemove(cdiv1alpha1.CDIConditionRunning, cr); err != nil {
				return err
			}

			return nil
		}

	}

	logger.Info("CDI is running")

	if err = r.conditionUpdate(conditionReady, cr); err != nil {
		return err
	}

	return nil
}

func (r *ReconcileCDI) add(mgr manager.Manager) error {
	// Create a new controller
	c, err := controller.New("cdi-operator-controller", mgr, controller.Options{Reconciler: r})
	if err != nil {
		return err
	}

	return r.watch(c)
}

func (r *ReconcileCDI) watch(c controller.Controller) error {
	// Watch for changes to CDI CR
	if err := c.Watch(&source.Kind{Type: &cdiv1alpha1.CDI{}}, &handler.EnqueueRequestForObject{}); err != nil {
		return err
	}

	resources, err := r.getAllResources(nil)
	if err != nil {
		return err
	}

	return r.watchTypes(c, resources)
}

func (r *ReconcileCDI) getConfigMap() (*corev1.ConfigMap, error) {
	cm := &corev1.ConfigMap{}
	key := client.ObjectKey{Name: ConfigMapName, Namespace: r.namespace}

	if err := r.client.Get(context.TODO(), key, cm); err != nil {
		if errors.IsNotFound(err) {
			return nil, nil
		}
		return nil, err
	}

	return cm, nil
}

func (r *ReconcileCDI) createConfigMap(cr *cdiv1alpha1.CDI) error {
	cm := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      ConfigMapName,
			Namespace: r.namespace,
			Labels:    map[string]string{"operator.cdi.kubevirt.io": ""},
		},
	}

	if err := controllerutil.SetControllerReference(cr, cm, r.scheme); err != nil {
		return err
	}

	if err := r.client.Create(context.TODO(), cm); err != nil {
		return err
	}

	return nil
}

func (r *ReconcileCDI) getAllDeployments(cr *cdiv1alpha1.CDI) ([]*appsv1.Deployment, error) {
	var result []*appsv1.Deployment

	resources, err := r.getAllResources(cr)
	if err != nil {
		return nil, err
	}

	for _, resource := range resources {
		if deployment, ok := resource.(*appsv1.Deployment); ok {
			result = append(result, deployment)
		}
	}

	return result, nil
}

func (r *ReconcileCDI) getNamespacedArgs(cr *cdiv1alpha1.CDI) *cdinamespaced.FactoryArgs {
	result := *r.namespacedArgs
	if cr != nil && cr.Spec.ImagePullPolicy != "" {
		result.PullPolicy = string(cr.Spec.ImagePullPolicy)
	}
	return &result
}

func (r *ReconcileCDI) getAllResources(cr *cdiv1alpha1.CDI) ([]runtime.Object, error) {
	var resources []runtime.Object

	if deployClusterResources() {
		crs, err := cdicluster.CreateAllResources(r.clusterArgs)
		if err != nil {
			return nil, err
		}

		resources = append(resources, crs...)
	}

	nsrs, err := cdinamespaced.CreateAllResources(r.getNamespacedArgs(cr))
	if err != nil {
		return nil, err
	}

	resources = append(resources, nsrs...)

	return resources, nil
}

func (r *ReconcileCDI) watchTypes(c controller.Controller, resources []runtime.Object) error {
	types := map[string]bool{}

	for _, resource := range resources {
		t := fmt.Sprintf("%T", resource)
		if types[t] {
			continue
		}

		eventHandler := &handler.EnqueueRequestForOwner{
			IsController: true,
			OwnerType:    &cdiv1alpha1.CDI{},
		}

		if err := c.Watch(&source.Kind{Type: resource}, eventHandler); err != nil {
			return err
		}

		log.Info("Watching", "type", t)

		types[t] = true
	}

	return nil
}
