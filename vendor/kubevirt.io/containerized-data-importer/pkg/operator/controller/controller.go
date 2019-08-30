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
	"strings"

	"github.com/blang/semver"
	"github.com/go-logr/logr"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	extv1beta1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1beta1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
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
	conditions "github.com/openshift/custom-resource-status/conditions/v1"

	cdiv1alpha1 "kubevirt.io/containerized-data-importer/pkg/apis/core/v1alpha1"
	"kubevirt.io/containerized-data-importer/pkg/operator"
	cdicluster "kubevirt.io/containerized-data-importer/pkg/operator/resources/cluster"
	cdinamespaced "kubevirt.io/containerized-data-importer/pkg/operator/resources/namespaced"
	"kubevirt.io/containerized-data-importer/pkg/util"
)

const (
	finalizerName = "operator.cdi.kubevirt.io"

	createVersionLabel          = "operator.cdi.kubevirt.io/createVersion"
	updateVersionLabel          = "operator.cdi.kubevirt.io/updateVersion"
	lastAppliedConfigAnnotation = "operator.cdi.kubevirt.io/lastAppliedConfiguration"
)

var log = logf.Log.WithName("cdi-operator")

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
		client:             mgr.GetClient(),
		scheme:             mgr.GetScheme(),
		namespace:          namespace,
		clusterArgs:        clusterArgs,
		namespacedArgs:     &namespacedArgs,
		explicitWatchTypes: getExplicitWatchTypes(),
		callbacks:          make(map[reflect.Type][]ReconcileCallback),
	}

	addReconcileCallbacks(r)

	return r, nil
}

var _ reconcile.Reconciler = &ReconcileCDI{}

// ReconcileCDI reconciles a CDI object
type ReconcileCDI struct {
	// This Client, initialized using mgr.client() above, is a split Client
	// that reads objects from the cache and writes to the apiserver
	client client.Client
	scheme *runtime.Scheme

	namespace      string
	clusterArgs    *cdicluster.FactoryArgs
	namespacedArgs *cdinamespaced.FactoryArgs

	explicitWatchTypes []runtime.Object
	callbacks          map[reflect.Type][]ReconcileCallback
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
			res, createErr := r.reconcileCreate(reqLogger, cr)
			// Always update conditions after a create.
			err = r.client.Update(context.TODO(), cr)
			if err != nil {
				return reconcile.Result{}, err
			}
			return res, createErr
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

	currentConditionValues := GetConditionValues(cr.Status.Conditions)
	reqLogger.Info("Doing reconcile update")

	existingAvailableCondition := conditions.FindStatusCondition(cr.Status.Conditions, conditions.ConditionAvailable)
	if existingAvailableCondition != nil {
		// should be the usual case
		MarkCrHealthyMessage(cr, existingAvailableCondition.Reason, existingAvailableCondition.Message)
	} else {
		MarkCrHealthyMessage(cr, "", "")
	}

	res, err := r.reconcileUpdate(reqLogger, cr)
	if conditionsChanged(currentConditionValues, GetConditionValues(cr.Status.Conditions)) {
		if err := r.crUpdate(cr.Status.Phase, cr); err != nil {
			return reconcile.Result{}, err
		}
	}
	return res, err
}

func shouldTakeUpdatePath(logger logr.Logger, targetVersion, currentVersion string) (bool, error) {

	// if no current version, then this can't be an update
	if currentVersion == "" {
		return false, nil
	}

	if targetVersion == currentVersion {
		return false, nil
	}

	// semver doesn't like the 'v' prefix
	targetVersion = strings.TrimPrefix(targetVersion, "v")
	currentVersion = strings.TrimPrefix(currentVersion, "v")

	// our default position is that this is an update.
	// So if the target and current version do not
	// adhere to the semver spec, we assume by default the
	// update path is the correct path.
	shouldTakeUpdatePath := true
	target, err := semver.Make(targetVersion)
	if err == nil {
		current, err := semver.Make(currentVersion)
		if err == nil {
			if target.Compare(current) < 0 {
				err := fmt.Errorf("operator downgraded, will not reconcile")
				logger.Error(err, "", "current", current, "target", target)
				return false, err
			} else if target.Compare(current) == 0 {
				shouldTakeUpdatePath = false
			}
		}
	}

	return shouldTakeUpdatePath, nil
}

func (r *ReconcileCDI) reconcileCreate(logger logr.Logger, cr *cdiv1alpha1.CDI) (reconcile.Result, error) {
	MarkCrDeploying(cr, "DeployStarted", "Started Deployment")
	// claim the configmap
	if err := r.createConfigMap(cr); err != nil {
		MarkCrFailed(cr, "ConfigError", "Unable to claim ConfigMap")
		return reconcile.Result{}, err
	}

	logger.Info("ConfigMap created successfully")

	if err := r.crInit(cr); err != nil {
		MarkCrFailed(cr, "CrInitError", "Unable to Initialize CR")
		return reconcile.Result{}, err
	}

	logger.Info("Successfully entered Deploying state")

	return r.reconcileUpdate(logger, cr)
}

func (r *ReconcileCDI) checkUpgrade(logger logr.Logger, cr *cdiv1alpha1.CDI) error {
	// should maybe put this in separate function
	if cr.Status.OperatorVersion != r.namespacedArgs.DockerTag {
		cr.Status.OperatorVersion = r.namespacedArgs.DockerTag
		if err := r.crUpdate(cr.Status.Phase, cr); err != nil {
			return err
		}
	}

	isUpgrade, err := shouldTakeUpdatePath(logger, r.namespacedArgs.DockerTag, cr.Status.ObservedVersion)
	if err != nil {
		return err
	}

	if isUpgrade && !r.isUpgrading(cr) {
		logger.Info("Observed version is not target version. Begin upgrade", "Observed version ", cr.Status.ObservedVersion, "TargetVersion", r.namespacedArgs.DockerTag)
		MarkCrUpgradeHealingDegraded(cr, "UpgradeStarted", fmt.Sprintf("Started upgrade to version %s", r.namespacedArgs.DockerTag))
		cr.Status.TargetVersion = r.namespacedArgs.DockerTag
		if err := r.crUpdate(cdiv1alpha1.CDIPhaseUpgrading, cr); err != nil {
			return err
		}
	}

	return nil
}

func (r *ReconcileCDI) reconcileUpdate(logger logr.Logger, cr *cdiv1alpha1.CDI) (reconcile.Result, error) {
	if err := r.checkUpgrade(logger, cr); err != nil {
		return reconcile.Result{}, err
	}

	resources, err := r.getAllResources(cr)
	if err != nil {
		return reconcile.Result{}, err
	}

	for _, desiredRuntimeObj := range resources {
		desiredMetaObj := desiredRuntimeObj.(metav1.Object)
		currentRuntimeObj := newDefaultInstance(desiredRuntimeObj)

		key := client.ObjectKey{
			Namespace: desiredMetaObj.GetNamespace(),
			Name:      desiredMetaObj.GetName(),
		}
		err = r.client.Get(context.TODO(), key, currentRuntimeObj)

		if err != nil {
			if !errors.IsNotFound(err) {
				return reconcile.Result{}, err
			}

			setLastAppliedConfiguration(desiredMetaObj)
			setLabel(createVersionLabel, r.namespacedArgs.DockerTag, desiredMetaObj)

			if err = controllerutil.SetControllerReference(cr, desiredMetaObj, r.scheme); err != nil {
				return reconcile.Result{}, err
			}

			// PRE_CREATE callback
			if err = r.invokeCallbacks(logger, ReconcileStatePreCreate, desiredRuntimeObj, nil); err != nil {
				return reconcile.Result{}, err
			}

			currentRuntimeObj = desiredRuntimeObj.DeepCopyObject()
			if err = r.client.Create(context.TODO(), currentRuntimeObj); err != nil {
				logger.Error(err, "")
				return reconcile.Result{}, err
			}

			// POST_CREATE callback
			if err = r.invokeCallbacks(logger, ReconcileStatePostCreate, desiredRuntimeObj, nil); err != nil {
				return reconcile.Result{}, err
			}

			logger.Info("Resource created",
				"namespace", desiredMetaObj.GetNamespace(),
				"name", desiredMetaObj.GetName(),
				"type", fmt.Sprintf("%T", desiredMetaObj))
		} else {
			// POST_READ callback
			if err = r.invokeCallbacks(logger, ReconcileStatePostRead, desiredRuntimeObj, currentRuntimeObj); err != nil {
				return reconcile.Result{}, err
			}

			currentRuntimeObjCopy := currentRuntimeObj.DeepCopyObject()
			currentMetaObj := currentRuntimeObj.(metav1.Object)

			// allow users to add new annotations (but not change ours)
			mergeLabelsAndAnnotations(desiredMetaObj, currentMetaObj)

			if !r.isMutable(currentRuntimeObj) {
				setLastAppliedConfiguration(desiredMetaObj)

				// overwrite currentRuntimeObj
				currentRuntimeObj, err = mergeObject(desiredRuntimeObj, currentRuntimeObj)
				if err != nil {
					return reconcile.Result{}, err
				}
				currentMetaObj = currentRuntimeObj.(metav1.Object)
			}

			if !reflect.DeepEqual(currentRuntimeObjCopy, currentRuntimeObj) {
				logJSONDiff(logger, currentRuntimeObjCopy, currentRuntimeObj)

				setLabel(updateVersionLabel, r.namespacedArgs.DockerTag, currentMetaObj)

				// PRE_UPDATE callback
				if err = r.invokeCallbacks(logger, ReconcileStatePreUpdate, desiredRuntimeObj, currentRuntimeObj); err != nil {
					return reconcile.Result{}, err
				}

				if err = r.client.Update(context.TODO(), currentRuntimeObj); err != nil {
					return reconcile.Result{}, err
				}

				// POST_UPDATE callback
				if err = r.invokeCallbacks(logger, ReconcileStatePostUpdate, desiredRuntimeObj, nil); err != nil {
					return reconcile.Result{}, err
				}

				logger.Info("Resource updated",
					"namespace", desiredMetaObj.GetNamespace(),
					"name", desiredMetaObj.GetName(),
					"type", fmt.Sprintf("%T", desiredMetaObj))
			} else {
				logger.Info("Resource unchanged",
					"namespace", desiredMetaObj.GetNamespace(),
					"name", desiredMetaObj.GetName(),
					"type", fmt.Sprintf("%T", desiredMetaObj))
			}
		}
	}

	if cr.Status.Phase != cdiv1alpha1.CDIPhaseDeployed && !r.isUpgrading(cr) {
		//We are not moving to Deployed phase until new operator deployment is ready in case of Upgrade
		cr.Status.ObservedVersion = r.namespacedArgs.DockerTag
		MarkCrHealthyMessage(cr, "DeployCompleted", "Deployment Completed")
		if err = r.crUpdate(cdiv1alpha1.CDIPhaseDeployed, cr); err != nil {
			return reconcile.Result{}, err
		}

		logger.Info("Successfully entered Deployed state")
	}

	degraded, err := r.checkDegraded(logger, cr)
	if err != nil {
		return reconcile.Result{}, err
	}

	if !degraded && r.isUpgrading(cr) {
		logger.Info("Completing upgrade process...")

		if err = r.completeUpgrade(logger, cr); err != nil {
			return reconcile.Result{}, err
		}
	}

	return reconcile.Result{}, nil
}

func (r *ReconcileCDI) completeUpgrade(logger logr.Logger, cr *cdiv1alpha1.CDI) error {
	if err := r.cleanupUnusedResources(logger, cr); err != nil {
		return err
	}

	previousVersion := cr.Status.ObservedVersion
	cr.Status.ObservedVersion = r.namespacedArgs.DockerTag

	MarkCrHealthyMessage(cr, "DeployCompleted", "Deployment Completed")
	if err := r.crUpdate(cdiv1alpha1.CDIPhaseDeployed, cr); err != nil {
		return err
	}

	logger.Info("Successfully finished Upgrade and entered Deployed state", "from version", previousVersion, "to version", cr.Status.ObservedVersion)

	return nil
}

func (r *ReconcileCDI) cleanupUnusedResources(logger logr.Logger, cr *cdiv1alpha1.CDI) error {
	//Iterate over installed resources of
	//Deployment/CRDs/Services etc and delete all resources that
	//do not exist in current version

	desiredResources, err := r.getAllResources(cr)
	if err != nil {
		return err
	}

	listTypes := []runtime.Object{
		&extv1beta1.CustomResourceDefinitionList{},
		&rbacv1.ClusterRoleBindingList{},
		&rbacv1.ClusterRoleList{},
		&appsv1.DeploymentList{},
		&corev1.ServiceList{},
		&rbacv1.RoleBindingList{},
		&rbacv1.RoleList{},
		&corev1.ServiceAccountList{},
	}

	for _, lt := range listTypes {
		lo := &client.ListOptions{}
		lo.SetLabelSelector(createVersionLabel)

		if err := r.client.List(context.TODO(), lo, lt); err != nil {
			logger.Error(err, "Error listing resources")
			return err
		}

		sv := reflect.ValueOf(lt).Elem()
		iv := sv.FieldByName("Items")

		for i := 0; i < iv.Len(); i++ {
			found := false
			observedObj := iv.Index(i).Addr().Interface().(runtime.Object)
			observedMetaObj := observedObj.(metav1.Object)

			for _, desiredObj := range desiredResources {
				if sameResource(observedObj, desiredObj) {
					found = true
					break
				}
			}

			if !found && metav1.IsControlledBy(observedMetaObj, cr) {
				//Invoke pre delete callback
				if err = r.invokeCallbacks(logger, ReconcileStatePreDelete, nil, observedObj); err != nil {
					return err
				}

				logger.Info("Deleting  ", "type", reflect.TypeOf(observedObj), "Name", observedMetaObj.GetName())
				err = r.client.Delete(context.TODO(), observedObj, func(opts *client.DeleteOptions) {
					p := metav1.DeletePropagationForeground
					opts.PropagationPolicy = &p
				})
				if err != nil && !errors.IsNotFound(err) {
					return err
				}

				//invoke post delete callback
				if err = r.invokeCallbacks(logger, ReconcileStatePostDelete, nil, observedObj); err != nil {
					return err
				}
			}
		}
	}

	return nil
}

func (r *ReconcileCDI) isMutable(obj runtime.Object) bool {
	switch obj.(type) {
	case *corev1.ConfigMap:
		return true
	}
	return false
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

	if i < 0 {
		return reconcile.Result{}, nil
	}

	if cr.Status.Phase != cdiv1alpha1.CDIPhaseDeleting {
		if err := r.crUpdate(cdiv1alpha1.CDIPhaseDeleting, cr); err != nil {
			return reconcile.Result{}, err
		}
	}

	if err := r.invokeDeleteCDICallbacks(logger, cr); err != nil {
		return reconcile.Result{}, err
	}

	cr.Finalizers = append(cr.Finalizers[0:i], cr.Finalizers[i+1:]...)

	if err := r.crUpdate(cdiv1alpha1.CDIPhaseDeleted, cr); err != nil {
		return reconcile.Result{}, err
	}

	logger.Info("Finalizer complete")

	return reconcile.Result{}, nil
}

func (r *ReconcileCDI) reconcileError(logger logr.Logger, cr *cdiv1alpha1.CDI) (reconcile.Result, error) {
	MarkCrFailed(cr, "ConfigError", "ConfigMap not owned by cr")
	if err := r.crUpdate(cr.Status.Phase, cr); err != nil {
		return reconcile.Result{}, err
	}
	if err := r.crError(cr); err != nil {
		return reconcile.Result{}, err
	}

	return reconcile.Result{}, nil
}

func (r *ReconcileCDI) checkDegraded(logger logr.Logger, cr *cdiv1alpha1.CDI) (bool, error) {
	degraded := false

	deployments, err := r.getAllDeployments(cr)
	if err != nil {
		return true, err
	}

	for _, deployment := range deployments {
		key := client.ObjectKey{Namespace: deployment.Namespace, Name: deployment.Name}

		if err = r.client.Get(context.TODO(), key, deployment); err != nil {
			return true, err
		}

		if !checkDeploymentReady(deployment) {
			degraded = true
			break
		}
	}

	logger.Info("CDI degraded check", "Degraded", degraded)

	if degraded {
		conditions.SetStatusCondition(&cr.Status.Conditions, conditions.Condition{
			Type:   conditions.ConditionDegraded,
			Status: corev1.ConditionTrue,
		})
	} else {
		conditions.SetStatusCondition(&cr.Status.Conditions, conditions.Condition{
			Type:   conditions.ConditionDegraded,
			Status: corev1.ConditionFalse,
		})
	}

	logger.Info("Finished degraded check", "conditions", cr.Status.Conditions)
	return degraded, nil
}

func (r *ReconcileCDI) add(mgr manager.Manager) error {
	// Create a new controller
	c, err := controller.New("cdi-operator-controller", mgr, controller.Options{Reconciler: r})
	if err != nil {
		return err
	}

	if err = r.watch(c); err != nil {
		return err
	}

	return nil
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

	resources = append(resources, r.explicitWatchTypes...)

	if err = r.watchResourceTypes(c, resources); err != nil {
		return err
	}

	// would like to get rid of this
	if err = r.watchSecurityContextConstraints(c); err != nil {
		return err
	}

	return nil
}

func (r *ReconcileCDI) getConfigMap() (*corev1.ConfigMap, error) {
	cm := &corev1.ConfigMap{}
	key := client.ObjectKey{Name: operator.ConfigMapName, Namespace: r.namespace}

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
			Name:      operator.ConfigMapName,
			Namespace: r.namespace,
			Labels:    map[string]string{"operator.cdi.kubevirt.io": ""},
		},
	}

	if err := controllerutil.SetControllerReference(cr, cm, r.scheme); err != nil {
		return err
	}

	return r.client.Create(context.TODO(), cm)
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

	if cr != nil {
		if cr.Spec.ImageRegistry != "" {
			result.DockerRepo = cr.Spec.ImageRegistry
		}

		if cr.Spec.ImageTag != "" {
			result.DockerTag = cr.Spec.ImageTag
		}

		if cr.Spec.ImagePullPolicy != "" {
			result.PullPolicy = string(cr.Spec.ImagePullPolicy)
		}
	}

	return &result
}

func (r *ReconcileCDI) getAllResources(cr *cdiv1alpha1.CDI) ([]runtime.Object, error) {
	var resources []runtime.Object

	if deployClusterResources() {
		crs, err := cdicluster.CreateAllResources(r.clusterArgs)
		if err != nil {
			MarkCrFailedHealing(cr, "CreateResources", "Unable to create all resources")
			return nil, err
		}

		resources = append(resources, crs...)
	}

	nsrs, err := cdinamespaced.CreateAllResources(r.getNamespacedArgs(cr))
	if err != nil {
		MarkCrFailedHealing(cr, "CreateNamespaceResources", "Unable to create all namespaced resources")
		return nil, err
	}

	resources = append(resources, nsrs...)

	return resources, nil
}

func (r *ReconcileCDI) watchResourceTypes(c controller.Controller, resources []runtime.Object) error {
	types := map[reflect.Type]bool{}

	for _, resource := range resources {
		t := reflect.TypeOf(resource)
		if types[t] {
			continue
		}

		if r.isMutable(resource) {
			log.Info("NOT Watching", "type", t)
			continue
		}

		eventHandler := &handler.EnqueueRequestForOwner{
			IsController: true,
			OwnerType:    &cdiv1alpha1.CDI{},
		}

		if err := c.Watch(&source.Kind{Type: resource}, eventHandler); err != nil {
			if meta.IsNoMatchError(err) {
				log.Info("No match for type, NOT WATCHING", "type", t)
				continue
			}
			return err
		}

		log.Info("Watching", "type", t)

		types[t] = true
	}

	return nil
}

func (r *ReconcileCDI) addCallback(obj runtime.Object, cb ReconcileCallback) {
	t := reflect.TypeOf(obj)
	cbs := r.callbacks[t]
	r.callbacks[t] = append(cbs, cb)
}

func (r *ReconcileCDI) invokeDeleteCDICallbacks(logger logr.Logger, cr *cdiv1alpha1.CDI) error {
	desiredResources, err := r.getAllResources(cr)
	if err != nil {
		return err
	}

	for _, desiredObj := range desiredResources {
		if err = r.invokeCallbacks(logger, ReconcileStateCDIDelete, desiredObj, nil); err != nil {
			return err
		}
	}

	return nil
}

func (r *ReconcileCDI) invokeCallbacks(l logr.Logger, s ReconcileState, desiredObj, currentObj runtime.Object) error {
	var t reflect.Type

	if desiredObj != nil {
		t = reflect.TypeOf(desiredObj)
	} else if currentObj != nil {
		t = reflect.TypeOf(currentObj)
	}

	// callbacks with nil key always get invoked
	cbs := append(r.callbacks[t], r.callbacks[nil]...)

	for _, cb := range cbs {
		if s != ReconcileStatePreCreate && currentObj == nil {
			metaObj := desiredObj.(metav1.Object)
			key := client.ObjectKey{
				Namespace: metaObj.GetNamespace(),
				Name:      metaObj.GetName(),
			}

			currentObj = newDefaultInstance(desiredObj)
			if err := r.client.Get(context.TODO(), key, currentObj); err != nil {
				if !errors.IsNotFound(err) {
					return err
				}
				currentObj = nil
			}
		}

		args := &ReconcileCallbackArgs{
			Logger:        l,
			Client:        r.client,
			Scheme:        r.scheme,
			State:         s,
			DesiredObject: desiredObj,
			CurrentObject: currentObj,
		}

		log.V(3).Info("Invoking callbacks for", "type", t)
		if err := cb(args); err != nil {
			log.Error(err, "error invoking callback for", "type", t)
			return err
		}
	}

	return nil
}

func setLabel(key, value string, obj metav1.Object) {
	if obj.GetLabels() == nil {
		obj.SetLabels(make(map[string]string))
	}
	obj.GetLabels()[key] = value
}

func setLastAppliedConfiguration(obj metav1.Object) error {
	bytes, err := json.Marshal(obj)
	if err != nil {
		return err
	}

	if obj.GetAnnotations() == nil {
		obj.SetAnnotations(make(map[string]string))
	}

	obj.GetAnnotations()[lastAppliedConfigAnnotation] = string(bytes)

	return nil
}

func sameResource(obj1, obj2 runtime.Object) bool {
	metaObj1 := obj1.(metav1.Object)
	metaObj2 := obj2.(metav1.Object)

	if reflect.TypeOf(obj1) != reflect.TypeOf(obj2) ||
		metaObj1.GetNamespace() != metaObj2.GetNamespace() ||
		metaObj1.GetName() != metaObj2.GetName() {
		return false
	}

	return true
}
