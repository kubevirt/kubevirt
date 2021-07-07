package hyperconverged

import (
	"context"
	"fmt"
	jsonpatch "github.com/evanphx/json-patch"
	"github.com/kubevirt/hyperconverged-cluster-operator/pkg/metrics"
	"os"
	"reflect"
	"strings"

	"github.com/google/uuid"
	operatorhandler "github.com/operator-framework/operator-lib/handler"
	corev1 "k8s.io/api/core/v1"
	schedulingv1 "k8s.io/api/scheduling/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	apimetav1 "k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/yaml"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/source"

	networkaddonsv1 "github.com/kubevirt/cluster-network-addons-operator/pkg/apis/networkaddonsoperator/v1"
	vmimportv1beta1 "github.com/kubevirt/vm-import-operator/pkg/apis/v2v/v1beta1"
	monitoringv1 "github.com/prometheus-operator/prometheus-operator/pkg/apis/monitoring/v1"
	kubevirtv1 "kubevirt.io/client-go/api/v1"
	cdiv1beta1 "kubevirt.io/containerized-data-importer/pkg/apis/core/v1beta1"
	sspv1beta1 "kubevirt.io/ssp-operator/api/v1beta1"

	hcov1beta1 "github.com/kubevirt/hyperconverged-cluster-operator/pkg/apis/hco/v1beta1"
	"github.com/kubevirt/hyperconverged-cluster-operator/pkg/controller/common"
	"github.com/kubevirt/hyperconverged-cluster-operator/pkg/controller/operands"
	hcoutil "github.com/kubevirt/hyperconverged-cluster-operator/pkg/util"
	version "github.com/kubevirt/hyperconverged-cluster-operator/version"
)

var (
	log               = logf.Log.WithName("controller_hyperconverged")
	randomConstSuffix = ""
)

const (
	// We cannot set owner reference of cluster-wide resources to namespaced HyperConverged object. Therefore,
	// use finalizers to manage the cleanup.
	FinalizerName    = "kubevirt.io/hyperconverged"
	badFinalizerName = "hyperconvergeds.hco.kubevirt.io"

	// OpenshiftNamespace is for resources that belong in the openshift namespace

	reconcileInit               = "Init"
	reconcileInitMessage        = "Initializing HyperConverged cluster"
	reconcileCompleted          = "ReconcileCompleted"
	reconcileCompletedMessage   = "Reconcile completed successfully"
	invalidRequestReason        = "InvalidRequest"
	invalidRequestMessageFormat = "Request does not match expected name (%v) and namespace (%v)"
	commonDegradedReason        = "HCODegraded"
	commonProgressingReason     = "HCOProgressing"
	taintedConfigurationReason  = "UnsupportedFeatureAnnotation"
	taintedConfigurationMessage = "Unsupported feature was activated via an HCO annotation"

	hcoVersionName    = "operator"
	secondaryCRPrefix = "hco-controlled-cr-"
)

// JSONPatchAnnotationNames - annotations used to patch operand CRs with unsupported/unofficial/hidden features.
// The presence of any of these annotations raises the hcov1beta1.ConditionTaintedConfiguration condition.
var JSONPatchAnnotationNames = []string{
	common.JSONPatchKVAnnotationName,
	common.JSONPatchCDIAnnotationName,
	common.JSONPatchCNAOAnnotationName,
}

// RegisterReconciler creates a new HyperConverged Reconciler and registers it into manager.
func RegisterReconciler(mgr manager.Manager, ci hcoutil.ClusterInfo) error {
	return add(mgr, newReconciler(mgr, ci), ci)
}

// newReconciler returns a new reconcile.Reconciler
func newReconciler(mgr manager.Manager, ci hcoutil.ClusterInfo) reconcile.Reconciler {

	ownVersion := os.Getenv(hcoutil.HcoKvIoVersionName)
	if ownVersion == "" {
		ownVersion = version.Version
	}

	return &ReconcileHyperConverged{
		client:         mgr.GetClient(),
		scheme:         mgr.GetScheme(),
		operandHandler: operands.NewOperandHandler(mgr.GetClient(), mgr.GetScheme(), ci.IsOpenshift(), hcoutil.GetEventEmitter()),
		upgradeMode:    false,
		ownVersion:     ownVersion,
		eventEmitter:   hcoutil.GetEventEmitter(),
		firstLoop:      true,
	}
}

// add adds a new Controller to mgr with r as the reconcile.Reconciler
func add(mgr manager.Manager, r reconcile.Reconciler, ci hcoutil.ClusterInfo) error {
	// Create a new controller
	c, err := controller.New("hyperconverged-controller", mgr, controller.Options{Reconciler: r})
	if err != nil {
		return err
	}

	// Watch for changes to primary resource HyperConverged
	err = c.Watch(
		&source.Kind{Type: &hcov1beta1.HyperConverged{}},
		&operatorhandler.InstrumentedEnqueueRequestForObject{},
		predicate.Or(predicate.GenerationChangedPredicate{}, predicate.AnnotationChangedPredicate{}))
	if err != nil {
		return err
	}

	secCRPlaceholder, err := getSecondaryCRPlaceholder()
	if err != nil {
		return err
	}

	// To limit the memory usage, the controller manager got instantiated with a custom cache
	// that is watching only a specific set of objects with selectors.
	// When a new object got added here, it has also to be added to the custom cache
	// managed by getNewManagerCache()
	secondaryResources := []client.Object{
		&kubevirtv1.KubeVirt{},
		&cdiv1beta1.CDI{},
		&networkaddonsv1.NetworkAddonsConfig{},
		&sspv1beta1.SSP{},
		&schedulingv1.PriorityClass{},
		&vmimportv1beta1.VMImportConfig{},
		&corev1.ConfigMap{},
	}
	if ci.IsOpenshift() {
		secondaryResources = append(secondaryResources, []client.Object{
			&corev1.Service{},
			&monitoringv1.ServiceMonitor{},
			&monitoringv1.PrometheusRule{},
		}...)
	}

	// Watch secondary resources
	for _, resource := range secondaryResources {
		msg := fmt.Sprintf("Reconciling for %T", resource)
		err = c.Watch(
			&source.Kind{Type: resource},
			handler.EnqueueRequestsFromMapFunc(func(a client.Object) []reconcile.Request {
				// enqueue using a placeholder to be able to discriminate request triggered
				// by changes on the HyperConverged object from request triggered by changes
				// on a secondary CR controlled by HCO
				log.Info(msg)
				return []reconcile.Request{
					{NamespacedName: secCRPlaceholder},
				}
			}),
		)
		if err != nil {
			return err
		}
	}

	return nil
}

var _ reconcile.Reconciler = &ReconcileHyperConverged{}

// ReconcileHyperConverged reconciles a HyperConverged object
type ReconcileHyperConverged struct {
	// This client, initialized using mgr.Client() above, is a split client
	// that reads objects from the cache and writes to the apiserver
	client         client.Client
	scheme         *runtime.Scheme
	operandHandler *operands.OperandHandler
	upgradeMode    bool
	ownVersion     string
	eventEmitter   hcoutil.EventEmitter
	firstLoop      bool
}

// Reconcile reads that state of the cluster for a HyperConverged object and makes changes based on the state read
// and what is in the HyperConverged.Spec
// Note:
// The Controller will requeue the Request to be processed again if the returned error is non-nil or
// Result.Requeue is true, otherwise upon completion it will remove the work from the queue.
func (r *ReconcileHyperConverged) Reconcile(ctx context.Context, request reconcile.Request) (reconcile.Result, error) {
	logger := log.WithValues("Request.Namespace", request.Namespace, "Request.Name", request.Name)

	hcoTriggered, err := isTriggeredByHyperConverged(request)
	if err != nil {
		return reconcile.Result{}, err
	}

	resolvedRequest, err := resolveReconcileRequest(request, hcoTriggered)
	if err != nil {
		return reconcile.Result{}, err
	}
	hcoRequest := common.NewHcoRequest(ctx, resolvedRequest, log, r.upgradeMode, hcoTriggered)

	if hcoTriggered {
		logger.Info("Reconciling HyperConverged operator")
		r.operandHandler.Reset()
	} else {
		logger.Info("The reconciliation got triggered by a secondary CR object")
	}

	// Fetch the HyperConverged instance
	instance, err := r.getHyperConverged(hcoRequest)
	if instance == nil {
		return reconcile.Result{}, err
	}
	hcoRequest.Instance = instance

	if r.firstLoop {
		r.firstLoopInitialization(hcoRequest)
	}

	result, err := r.doReconcile(hcoRequest)
	if err != nil {
		r.eventEmitter.EmitEvent(hcoRequest.Instance, corev1.EventTypeWarning, "ReconcileError", err.Error())
		return reconcile.Result{}, err
	}

	requeue, err := r.updateHyperConverged(hcoRequest)
	if requeue || apierrors.IsConflict(err) {
		result.Requeue = true
	}

	return result, err
}

// resolveReconcileRequest returns a reconcile.Request to be used throughout the reconciliation cycle,
// regardless of which resource has triggered it.
func resolveReconcileRequest(originalRequest reconcile.Request, hcoTriggered bool) (reconcile.Request, error) {
	if hcoTriggered {
		return originalRequest, nil
	}

	hc, err := getHyperConvergedNamespacedName()
	if err != nil {
		return reconcile.Request{}, err
	}

	resolvedRequest := reconcile.Request{
		NamespacedName: hc,
	}

	return resolvedRequest, nil
}

func isTriggeredByHyperConverged(request reconcile.Request) (bool, error) {
	placeholder, err := getSecondaryCRPlaceholder()
	if err != nil {
		return false, err
	}

	isHyperConverged := request.NamespacedName != placeholder
	return isHyperConverged, nil
}

func (r *ReconcileHyperConverged) doReconcile(req *common.HcoRequest) (reconcile.Result, error) {

	valid, err := r.validateNamespace(req)
	if !valid {
		return reconcile.Result{}, err
	}
	// Add conditions if there are none
	init := req.Instance.Status.Conditions == nil
	if init {
		r.eventEmitter.EmitEvent(req.Instance, corev1.EventTypeNormal, "InitHCO", "Initiating the HyperConverged")
		r.setInitialConditions(req)
	}

	r.setLabels(req)

	updateStatusGeneration(req)

	// in-memory conditions should start off empty. It will only ever hold
	// negative conditions (!Available, Degraded, Progressing)
	req.Conditions = common.NewHcoConditions()

	// Handle finalizers
	if !checkFinalizers(req) {
		if !req.HCOTriggered {
			// this is just the effect of a delete request created by HCO
			// in the previous iteration, ignore it
			return reconcile.Result{}, nil
		}
		return r.ensureHcoDeleted(req)
	}

	// If the current version is not updated in CR ,then we're updating. This is also works when updating from
	// an old version, since Status.Versions will be empty.
	knownHcoVersion, _ := req.Instance.Status.GetVersion(hcoVersionName)

	if !r.upgradeMode && !init && knownHcoVersion != r.ownVersion {
		r.upgradeMode = true

		r.eventEmitter.EmitEvent(req.Instance, corev1.EventTypeNormal, "UpgradeHCO", "Upgrading the HyperConverged to version "+r.ownVersion)
		req.Logger.Info(fmt.Sprintf("Start upgrading from version %s to version %s", knownHcoVersion, r.ownVersion))
	}

	req.SetUpgradeMode(r.upgradeMode)

	if r.upgradeMode {
		modified, err := r.migrateBeforeUpgrade(req)
		if err != nil {
			return reconcile.Result{Requeue: init}, err
		}

		if modified {
			r.updateConditions(req)
			return reconcile.Result{Requeue: true}, nil
		}
	}

	return r.EnsureOperandAndComplete(req, init)
}

func (r *ReconcileHyperConverged) EnsureOperandAndComplete(req *common.HcoRequest, init bool) (reconcile.Result, error) {
	if err := r.operandHandler.Ensure(req); err != nil {
		r.updateConditions(req)
		hcoutil.SetReady(false)
		return reconcile.Result{Requeue: init}, nil
	}

	req.Logger.Info("Reconcile complete")

	// Requeue if we just created everything
	if init {
		hcoutil.SetReady(false)
		return reconcile.Result{Requeue: true}, nil
	}

	r.completeReconciliation(req)

	return reconcile.Result{}, nil
}

func updateStatusGeneration(req *common.HcoRequest) {
	if req.Instance.ObjectMeta.Generation != req.Instance.Status.ObservedGeneration {
		req.Instance.Status.ObservedGeneration = req.Instance.ObjectMeta.Generation
		req.StatusDirty = true
	}
}

// getHyperConverged gets the HyperConverged resource from the Kubernetes API.
func (r *ReconcileHyperConverged) getHyperConverged(req *common.HcoRequest) (*hcov1beta1.HyperConverged, error) {
	instance := &hcov1beta1.HyperConverged{}
	err := r.client.Get(req.Ctx, req.NamespacedName, instance)

	// Green path first
	if err == nil {
		return instance, nil
	}

	// Error path
	if apierrors.IsNotFound(err) {
		req.Logger.Info("No HyperConverged resource")
		// Request object not found, could have been deleted after reconcile request.
		// Owned objects are automatically garbage collected. For additional cleanup logic use finalizers.
		// Return and don't requeue
		return nil, nil
	}

	// Another error reading the object.
	// Just return the error so that the request is requeued.
	return nil, err
}

// updateHyperConverged updates the HyperConverged resource according to its state in the request.
func (r *ReconcileHyperConverged) updateHyperConverged(request *common.HcoRequest) (bool, error) {

	// Since the status subresource is enabled for the HyperConverged kind,
	// we need to update the status and the metadata separately.
	// Moreover, we need to update the status first, in order to prevent a conflict.
	// In addition, spec changes are removed by status update, but since status update done first, we need
	// to store the spec and recover it after status update

	if request.Dirty {
		err := r.updateHyperConvergedSpecMetadata(request)
		if err != nil {
			r.logHyperConvergedUpdateError(request, err, "Failed to update HCO CR")
			return false, err
		}
		return true, nil
	}

	err := r.updateHyperConvergedStatus(request)
	if err != nil {
		r.logHyperConvergedUpdateError(request, err, "Failed to update HCO Status")
		return false, err
	}

	return false, nil
}

// updateHyperConvergedSpecMetadata updates the HyperConverged resource's spec and metadata.
func (r *ReconcileHyperConverged) updateHyperConvergedSpecMetadata(request *common.HcoRequest) error {
	if !request.Dirty {
		return nil
	}

	return r.client.Update(request.Ctx, request.Instance)
}

// updateHyperConvergedSpecMetadata updates the HyperConverged resource's status (and metadata).
func (r *ReconcileHyperConverged) updateHyperConvergedStatus(request *common.HcoRequest) error {
	if !request.StatusDirty {
		return nil
	}

	return r.client.Status().Update(request.Ctx, request.Instance)
}

// logHyperConvergedUpdateError logs an error that occurred during resource update,
// as well as emits a corresponding event.
func (r *ReconcileHyperConverged) logHyperConvergedUpdateError(request *common.HcoRequest, err error, errMsg string) {
	r.eventEmitter.EmitEvent(request.Instance,
		corev1.EventTypeWarning,
		"HcoUpdateError",
		errMsg)

	request.Logger.Error(err, errMsg)
}

func (r *ReconcileHyperConverged) validateNamespace(req *common.HcoRequest) (bool, error) {
	hco, err := getHyperConvergedNamespacedName()
	if err != nil {
		req.Logger.Error(err, "Failed to get HyperConverged namespaced name")
		return false, err
	}

	// Ignore invalid requests
	if req.NamespacedName != hco {
		req.Logger.Info("Invalid request", "HyperConverged.Namespace", hco.Namespace, "HyperConverged.Name", hco.Name)
		req.Conditions.SetStatusCondition(metav1.Condition{
			Type:               hcov1beta1.ConditionReconcileComplete,
			Status:             metav1.ConditionFalse,
			Reason:             invalidRequestReason,
			Message:            fmt.Sprintf(invalidRequestMessageFormat, hco.Name, hco.Namespace),
			ObservedGeneration: req.Instance.Generation,
		})
		r.updateConditions(req)
		return false, nil
	}
	return true, nil
}

func (r *ReconcileHyperConverged) setInitialConditions(req *common.HcoRequest) {
	req.Instance.Status.UpdateVersion(hcoVersionName, r.ownVersion)

	req.Conditions.SetStatusCondition(metav1.Condition{
		Type:               hcov1beta1.ConditionReconcileComplete,
		Status:             metav1.ConditionUnknown, // we just started trying to reconcile
		Reason:             reconcileInit,
		Message:            reconcileInitMessage,
		ObservedGeneration: req.Instance.Generation,
	})
	req.Conditions.SetStatusCondition(metav1.Condition{
		Type:               hcov1beta1.ConditionAvailable,
		Status:             metav1.ConditionFalse,
		Reason:             reconcileInit,
		Message:            reconcileInitMessage,
		ObservedGeneration: req.Instance.Generation,
	})
	req.Conditions.SetStatusCondition(metav1.Condition{
		Type:               hcov1beta1.ConditionProgressing,
		Status:             metav1.ConditionTrue,
		Reason:             reconcileInit,
		Message:            reconcileInitMessage,
		ObservedGeneration: req.Instance.Generation,
	})
	req.Conditions.SetStatusCondition(metav1.Condition{
		Type:               hcov1beta1.ConditionDegraded,
		Status:             metav1.ConditionFalse,
		Reason:             reconcileInit,
		Message:            reconcileInitMessage,
		ObservedGeneration: req.Instance.Generation,
	})
	req.Conditions.SetStatusCondition(metav1.Condition{
		Type:               hcov1beta1.ConditionUpgradeable,
		Status:             metav1.ConditionUnknown,
		Reason:             reconcileInit,
		Message:            reconcileInitMessage,
		ObservedGeneration: req.Instance.Generation,
	})

	r.updateConditions(req)
}

func (r *ReconcileHyperConverged) ensureHcoDeleted(req *common.HcoRequest) (reconcile.Result, error) {
	err := r.operandHandler.EnsureDeleted(req)
	if err != nil {
		return reconcile.Result{}, err
	}

	requeue := false

	// Remove the finalizers
	finDropped := false
	if hcoutil.ContainsString(req.Instance.ObjectMeta.Finalizers, FinalizerName) {
		req.Instance.ObjectMeta.Finalizers, finDropped = drop(req.Instance.ObjectMeta.Finalizers, FinalizerName)
		req.Dirty = true
		requeue = requeue || finDropped
	}
	if hcoutil.ContainsString(req.Instance.ObjectMeta.Finalizers, badFinalizerName) {
		req.Instance.ObjectMeta.Finalizers, finDropped = drop(req.Instance.ObjectMeta.Finalizers, badFinalizerName)
		req.Dirty = true
		requeue = requeue || finDropped
	}

	// Need to requeue because finalizer update does not change metadata.generation
	return reconcile.Result{Requeue: requeue}, nil
}

func (r *ReconcileHyperConverged) aggregateComponentConditions(req *common.HcoRequest) bool {
	/*
		See the chart at design/aggregateComponentConditions.svg; The numbers below follows the numbers in the chart
		Here is the PlantUML code for the chart that describes the aggregation of the sub-components conditions.
		Find the PlantURL syntax here: https://plantuml.com/activity-diagram-beta

		@startuml ../../../design/aggregateComponentConditions.svg
		title Aggregate Component Conditions

		start
		  #springgreen:Set **ReconcileComplete = True**]
		  !x=1
		if ((x) [Degraded = True] Exists) then
		  !x=x+1
		  #orangered:<<implicit>>\n**Degraded = True** /
		  -[#orangered]-> yes;
		  if ((x) [Progressing = True] Exists) then
			!x=x+1
			-[#springgreen]-> no;
			#springgreen:(x) Set **Progressing = False**]
			!x=x+1
		  else
			-[#orangered]-> yes;
			#orangered:<<implicit>>\n**Progressing = True** /
		  endif
		  if ((x) [Upgradable = False] Exists) then
			!x=x+1
			-[#springgreen]-> no;
			#orangered:(x) Set **Upgradable = False**]
			!x=x+1
		  else
			-[#orangered]-> yes;
			#orangered:<<implicit>>\n**Upgradable = False** /
		  endif
		  if ((x) [Available = False] Exists) then
			!x=x+1
			-[#springgreen]-> no;
			#orangered:(x) Set **Available = False**]
			!x=x+1
		  else
			-[#orangered]-> yes;
			#orangered:<<implicit>>\n**Available = False** /
		  endif
		else
		  -[#springgreen]-> no;
		  #springgreen:(x) Set **Degraded = False**]
		  !x=x+1
		  if ((x) [Progressing = True] Exists) then
			!x=x+1
			-[#orangered]-> yes;
			#orangered:<<implicit>>\n**Progressing = True** /
			if ((x) [Upgradable = False] Exists) then
			  !x=x+1
			  -[#springgreen]-> no;
			  #orangered:(x) Set **Upgradable = False**]
			  !x=x+1
			else
			  -[#orangered]-> yes;
			  #orangered:<<implicit>>\n**Upgradable = False** /
			endif
			if ((x) [Available = False] Exists) then
			  !x=x+1
			  -[#springgreen]-> no;
			  #springgreen:(x) Set **Available = True**]
			  !x=x+1
			else
			  #orangered:<<implicit>>\n**Available = False** /
			  -[#orangered]-> yes;
			endif
		  else
			-[#springgreen]-> no;
			#springgreen:(x) Set **Progressing = False**]
			!x=x+1
			if ((x) [Upgradable = False] Exists) then
			  !x=x+1
			  -[#springgreen]-> no;
			  #springgreen:(x) Set **Upgradable = True**]
			  !x=x+1
			else
			#orangered:<<implicit>>\n**Upgradable = False** /
			  -[#orangered]-> yes;
			endif
			if ((x) [Available = False] Exists) then
			  !x=x+1
			  -[#springgreen]-> no;
			  #springgreen:(x) Set **Available = True**]
			  !x=x+1
			else
			  -[#orangered]-> yes;
			  #orangered:<<implicit>>\n**Available = False** /
			endif
		  endif
		endif
		end
		@enduml
	*/

	/*
		    If any component operator reports negatively we want to write that to
			the instance while preserving it's lastTransitionTime.
			For example, consider the KubeVirt resource has the Available condition
			type with type "False". When reconciling KubeVirt's resource we would
			add it to the in-memory representation of HCO's conditions (r.conditions)
			and here we are simply writing it back to the server.
			One shortcoming is that only one failure of a particular condition can be
			captured at one time (ie. if KubeVirt and CDI are both reporting !Available,
		    you will only see CDI as it updates last).
	*/
	allComponentsAreUp := req.Conditions.IsEmpty()
	req.Conditions.SetStatusCondition(metav1.Condition{
		Type:               hcov1beta1.ConditionReconcileComplete,
		Status:             metav1.ConditionTrue,
		Reason:             reconcileCompleted,
		Message:            reconcileCompletedMessage,
		ObservedGeneration: req.Instance.Generation,
	})

	if req.Conditions.HasCondition(hcov1beta1.ConditionDegraded) { // (#chart 1)

		req.Conditions.SetStatusConditionIfUnset(metav1.Condition{ // (#chart 2,3)
			Type:               hcov1beta1.ConditionProgressing,
			Status:             metav1.ConditionFalse,
			Reason:             reconcileCompleted,
			Message:            reconcileCompletedMessage,
			ObservedGeneration: req.Instance.Generation,
		})

		req.Conditions.SetStatusConditionIfUnset(metav1.Condition{ // (#chart 4,5)
			Type:               hcov1beta1.ConditionUpgradeable,
			Status:             metav1.ConditionFalse,
			Reason:             commonDegradedReason,
			Message:            "HCO is not Upgradeable due to degraded components",
			ObservedGeneration: req.Instance.Generation,
		})

		req.Conditions.SetStatusConditionIfUnset(metav1.Condition{ // (#chart 6,7)
			Type:               hcov1beta1.ConditionAvailable,
			Status:             metav1.ConditionFalse,
			Reason:             commonDegradedReason,
			Message:            "HCO is not available due to degraded components",
			ObservedGeneration: req.Instance.Generation,
		})

	} else {

		// Degraded is not found. add it.
		req.Conditions.SetStatusCondition(metav1.Condition{ // (#chart 8)
			Type:               hcov1beta1.ConditionDegraded,
			Status:             metav1.ConditionFalse,
			Reason:             reconcileCompleted,
			Message:            reconcileCompletedMessage,
			ObservedGeneration: req.Instance.Generation,
		})

		if req.Conditions.HasCondition(hcov1beta1.ConditionProgressing) { // (#chart 9)

			req.Conditions.SetStatusConditionIfUnset(metav1.Condition{ // (#chart 10,11)
				Type:               hcov1beta1.ConditionUpgradeable,
				Status:             metav1.ConditionFalse,
				Reason:             commonProgressingReason,
				Message:            "HCO is not Upgradeable due to progressing components",
				ObservedGeneration: req.Instance.Generation,
			})

			req.Conditions.SetStatusConditionIfUnset(metav1.Condition{ // (#chart 12,13)
				Type:               hcov1beta1.ConditionAvailable,
				Status:             metav1.ConditionTrue,
				Reason:             reconcileCompleted,
				Message:            reconcileCompletedMessage,
				ObservedGeneration: req.Instance.Generation,
			})

		} else {

			req.Conditions.SetStatusCondition(metav1.Condition{ // (#chart 14)
				Type:               hcov1beta1.ConditionProgressing,
				Status:             metav1.ConditionFalse,
				Reason:             reconcileCompleted,
				Message:            reconcileCompletedMessage,
				ObservedGeneration: req.Instance.Generation,
			})

			req.Conditions.SetStatusConditionIfUnset(metav1.Condition{ // (#chart 15,16)
				Type:               hcov1beta1.ConditionUpgradeable,
				Status:             metav1.ConditionTrue,
				Reason:             reconcileCompleted,
				Message:            reconcileCompletedMessage,
				ObservedGeneration: req.Instance.Generation,
			})

			req.Conditions.SetStatusConditionIfUnset(metav1.Condition{ // (#chart 17,18)
				Type:               hcov1beta1.ConditionAvailable,
				Status:             metav1.ConditionTrue,
				Reason:             reconcileCompleted,
				Message:            reconcileCompletedMessage,
				ObservedGeneration: req.Instance.Generation,
			})

		}
	}

	return allComponentsAreUp
}

func (r *ReconcileHyperConverged) completeReconciliation(req *common.HcoRequest) {
	allComponentsAreUp := r.aggregateComponentConditions(req)

	hcoReady := false

	if allComponentsAreUp {
		req.Logger.Info("No component operator reported negatively")

		// if in upgrade mode, and all the components are upgraded, and nothing pending to be written - upgrade is completed
		if r.upgradeMode && req.ComponentUpgradeInProgress && !req.Dirty {
			// update the new version only when upgrade is completed
			req.Instance.Status.UpdateVersion(hcoVersionName, r.ownVersion)
			req.StatusDirty = true

			r.upgradeMode = false
			req.ComponentUpgradeInProgress = false
			req.Logger.Info(fmt.Sprintf("Successfuly upgraded to version %s", r.ownVersion))
			r.eventEmitter.EmitEvent(req.Instance, corev1.EventTypeNormal, "UpgradeHCO", fmt.Sprintf("Successfuly upgraded to version %s", r.ownVersion))
		}

		// If not in upgrade mode, then we're ready, because all the operators reported positive conditions.
		// if upgrade was done successfully, r.upgradeMode is already false here.
		hcoReady = !r.upgradeMode
	}

	if r.upgradeMode {
		// override the Progressing condition during upgrade
		req.Conditions.SetStatusCondition(metav1.Condition{
			Type:               hcov1beta1.ConditionProgressing,
			Status:             metav1.ConditionTrue,
			Reason:             "HCOUpgrading",
			Message:            "HCO is now upgrading to version " + r.ownVersion,
			ObservedGeneration: req.Instance.Generation,
		})
	}

	hcoutil.SetReady(hcoReady)
	if hcoReady {
		// If no operator whose conditions we are watching reports an error, then it is safe
		// to set readiness.
		r.eventEmitter.EmitEvent(req.Instance, corev1.EventTypeNormal, "ReconcileHCO", "HCO Reconcile completed successfully")
	} else {
		// If for any reason we marked ourselves !upgradeable...then unset readiness
		if r.upgradeMode {
			r.eventEmitter.EmitEvent(req.Instance, corev1.EventTypeNormal, "ReconcileHCO", "HCO Upgrade in progress")
		} else {
			r.eventEmitter.EmitEvent(req.Instance, corev1.EventTypeWarning, "ReconcileHCO", "Not all the operators are ready")
		}
	}

	r.updateConditions(req)
}

// This function is used to exit from the reconcile function, updating the conditions and returns the reconcile result
func (r *ReconcileHyperConverged) updateConditions(req *common.HcoRequest) {
	conditions := make([]metav1.Condition, len(req.Instance.Status.Conditions))
	copy(conditions, req.Instance.Status.Conditions)

	for _, condType := range common.HcoConditionTypes {
		cond, found := req.Conditions[condType]
		if !found {
			cond = metav1.Condition{
				Type:               condType,
				Status:             metav1.ConditionUnknown,
				Message:            "Unknown Status",
				Reason:             "StatusUnknown",
				ObservedGeneration: req.Instance.ObjectMeta.Generation,
			}
		}

		apimetav1.SetStatusCondition(&conditions, cond)
	}

	// Detect a "TaintedConfiguration" state, and raise a corresponding event
	r.detectTaintedConfiguration(req, &conditions)

	if !reflect.DeepEqual(conditions, req.Instance.Status.Conditions) {
		req.Instance.Status.Conditions = conditions
		req.StatusDirty = true
	}
}

func (r *ReconcileHyperConverged) setLabels(req *common.HcoRequest) {
	if req.Instance.ObjectMeta.Labels == nil {
		req.Instance.ObjectMeta.Labels = map[string]string{}
	}
	if req.Instance.ObjectMeta.Labels[hcoutil.AppLabel] == "" {
		req.Instance.ObjectMeta.Labels[hcoutil.AppLabel] = req.Instance.Name
		req.Dirty = true
	}
}

func (r *ReconcileHyperConverged) detectTaintedConfiguration(req *common.HcoRequest, conditions *[]metav1.Condition) {
	conditionExists := apimetav1.IsStatusConditionTrue(req.Instance.Status.Conditions, hcov1beta1.ConditionTaintedConfiguration)

	// A tainted configuration state is indicated by the
	// presence of at least one of the JSON Patch annotations
	tainted := false
	for _, jpa := range JSONPatchAnnotationNames {
		NumOfChanges := 0
		jsonPatch, exists := req.Instance.ObjectMeta.Annotations[jpa]
		if exists {
			if NumOfChanges = getNumOfChangesJsonPatch(jsonPatch); NumOfChanges > 0 {
				tainted = true
			}
		}
		metrics.HcoMetrics.SetUnsafeModificationCount(NumOfChanges, jpa)
	}

	if tainted {
		apimetav1.SetStatusCondition(conditions, metav1.Condition{
			Type:               hcov1beta1.ConditionTaintedConfiguration,
			Status:             metav1.ConditionTrue,
			Reason:             taintedConfigurationReason,
			Message:            taintedConfigurationMessage,
			ObservedGeneration: req.Instance.ObjectMeta.Generation,
		})

		if !conditionExists {
			// Only log at "first occurrence" of detection
			req.Logger.Info("Detected tainted configuration state for HCO")
		}
	} else { // !tainted

		// For the sake of keeping the JSONPatch backdoor in low profile,
		// we just remove the condition instead of False'ing it.
		if conditionExists {
			apimetav1.RemoveStatusCondition(conditions, hcov1beta1.ConditionTaintedConfiguration)

			req.Logger.Info("Detected untainted configuration state for HCO")
		}
	}
}

func getNumOfChangesJsonPatch(jsonPatch string) int {
	patches, err := jsonpatch.DecodePatch([]byte(jsonPatch))
	if err != nil {
		return 0
	}
	return len(patches)
}

func (r *ReconcileHyperConverged) firstLoopInitialization(request *common.HcoRequest) {
	// Reload eventEmitter.
	// The client should now find all the required resources.
	r.eventEmitter.UpdateClient(request.Ctx, r.client, request.Logger)

	// Initialize operand handler.
	r.operandHandler.FirstUseInitiation(r.scheme, hcoutil.GetClusterInfo().IsOpenshift(), request.Instance)

	// Avoid re-initializing.
	r.firstLoop = false
}

// This function performs migrations before starting the upgrade process
// return true if the HyperConverged CR was modified; else, return false
//
// If the kubevirt-config configMap exists:
// 1. create a backup of the configMap, if not exists
// 2. if the configMap includes the live migration configurations, and they are not match to the default values,
//    update the HyperConverged CR
// 3. remove the kubevirt-config configMap
// 4. return true if the CR was modified in #2
const (
	kvCmName         = "kubevirt-config"
	backupKvCmName   = kvCmName + "-backup"
	imsCmName        = "v2v-vmware"
	liveMigrationKey = "migrations"
	vddkInitImakeKey = "vddk-init-image"
)

func (r *ReconcileHyperConverged) migrateBeforeUpgrade(req *common.HcoRequest) (bool, error) {
	kvConfigModified, err := r.migrateKvConfigurations(req)
	if err != nil {
		return false, err
	}

	cdiConfigModified, err := r.migrateCdiConfigurations(req)
	if err != nil {
		return false, err
	}

	imsConfigModified, err := r.migrateImsConfigurations(req)
	if err != nil {
		return false, err
	}

	return kvConfigModified || cdiConfigModified || imsConfigModified, nil
}

func (r ReconcileHyperConverged) migrateKvConfigurations(req *common.HcoRequest) (bool, error) {
	cm, err := r.getCm(kvCmName, req)
	if err != nil {
		return false, err
	} else if cm == nil {
		return false, nil
	}

	if err = r.makeCmBackup(cm, backupKvCmName, req); err != nil {
		return false, err
	}

	modified := adoptOldKvConfigs(req, cm)

	if !modified {
		err = r.removeConfigMap(req, cm, kvCmName)
		if err != nil {
			return false, err
		}
	}

	return modified, nil
}

func (r ReconcileHyperConverged) migrateCdiConfigurations(req *common.HcoRequest) (bool, error) {
	cdi := operands.NewCDIWithNameOnly(req.Instance)

	err := hcoutil.GetRuntimeObject(req.Ctx, r.client, cdi, req.Logger)
	if err != nil {
		if apierrors.IsNotFound(err) {
			return false, nil
		}
		return false, err
	}

	return adoptCdiConfigs(req, cdi.Spec.Config), nil
}

func (r ReconcileHyperConverged) migrateImsConfigurations(req *common.HcoRequest) (bool, error) {
	req.Logger.Info("read IMS configmap")
	cm := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      imsCmName,
			Namespace: req.Namespace,
		},
	}

	if err := hcoutil.GetRuntimeObject(req.Ctx, r.client, cm, req.Logger); err != nil {
		if apierrors.IsNotFound(err) {
			req.Logger.Info("IMS configmap already removed")
			return false, nil
		}
		req.Logger.Info("failed to get IMS configmap", "error", err.Error())
		return false, err
	}

	modified := false
	vddkInitImage, ok := cm.Data[vddkInitImakeKey]
	if ok {
		if req.Instance.Spec.VddkInitImage == nil {
			req.Logger.Info("updating the HyperConverged CR from the IMS configMap")
			req.Instance.Spec.VddkInitImage = &vddkInitImage
			req.Dirty = true
			modified = true
		}
	}

	return modified, nil
}

func (r *ReconcileHyperConverged) removeConfigMap(req *common.HcoRequest, cm *corev1.ConfigMap, cmName string) error {
	req.Logger.Info("removing the kubevirt configMap")
	err := hcoutil.ComponentResourceRemoval(req.Ctx, r.client, cm, req.Name, req.Logger, false, true)
	if err != nil {
		return err
	}

	r.eventEmitter.EmitEvent(req.Instance, corev1.EventTypeNormal, "Killing", fmt.Sprintf("Removed ConfigMap %s", cmName))

	refs := make([]corev1.ObjectReference, 0, len(req.Instance.Status.RelatedObjects))
	for _, obj := range req.Instance.Status.RelatedObjects {
		if obj.Kind == "ConfigMap" && obj.Name == cmName {
			continue
		}
		refs = append(refs, obj)
	}

	req.Instance.Status.RelatedObjects = refs
	req.StatusDirty = true

	return nil
}

func (r *ReconcileHyperConverged) getCm(cmName string, req *common.HcoRequest) (*corev1.ConfigMap, error) {
	cm := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      cmName,
			Namespace: req.Namespace,
		},
	}

	if err := hcoutil.GetRuntimeObject(req.Ctx, r.client, cm, req.Logger); err != nil {
		if apierrors.IsNotFound(err) {
			req.Logger.Info(fmt.Sprintf("%s configmap already removed", cmName))
			return nil, nil
		}
		req.Logger.Info(fmt.Sprintf("failed to get %s configmap", cmName), "error", err.Error())
		return nil, err
	}

	return cm, nil
}

func (r *ReconcileHyperConverged) makeCmBackup(cm *corev1.ConfigMap, backupName string, req *common.HcoRequest) error {
	backupCm := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      backupName,
			Namespace: cm.Namespace,
			Labels:    cm.Labels,
		},
		Data: cm.Data,
	}

	req.Logger.Info(fmt.Sprintf("creating %s configmap backup", backupName))
	if err := r.client.Create(req.Ctx, backupCm); err != nil {
		if apierrors.IsAlreadyExists(err) {
			req.Logger.Info(fmt.Sprintf("%s configmap backup already exists", backupName))
		} else {
			req.Logger.Info(fmt.Sprintf("failed to create %s configmap backup", backupName), "error", err.Error())
			return err
		}
	} else {
		r.eventEmitter.EmitEvent(req.Instance, corev1.EventTypeNormal, "Created", fmt.Sprintf("Created ConfigMap %s", backupName))
	}

	return nil
}

// Read the old KubeVit configuration from the config map, and move them to the HyperConverged CR
//
// In case of wrong foramt of the configmap, the HCO ignores this error (but print it to the log) in order to prevent
// an infinite loop (returning error will cause the same error again and again, and the only way to stop the loop
// is to manually fix or delete the wrong configMap).
func adoptOldKvConfigs(req *common.HcoRequest, cm *corev1.ConfigMap) bool {
	modified := false
	kvLiveMigrationConfig, ok := cm.Data[liveMigrationKey]
	if !ok {
		return false
	}
	kvCmLiveMigrationConfig := hcov1beta1.LiveMigrationConfigurations{}
	req.Instance.Spec.LiveMigrationConfig.DeepCopyInto(&kvCmLiveMigrationConfig)
	err := yaml.NewYAMLOrJSONDecoder(strings.NewReader(kvLiveMigrationConfig), 1024).Decode(&kvCmLiveMigrationConfig)
	if err != nil {
		req.Logger.Error(err, "Failed to read the KubeVirt ConfigMap, and its content was ignored. This ConfigMap will be deleted. The backup ConfigMap called "+backupKvCmName)
		return false
	}

	if !reflect.DeepEqual(req.Instance.Spec.LiveMigrationConfig, kvCmLiveMigrationConfig) {
		req.Logger.Info("updating the HyperConverged CR from the KubeVirt configMap")
		kvConfigMapToHyperConvergedCr(req, kvCmLiveMigrationConfig)

		modified = true
		req.Dirty = true
	}

	return modified
}

func adoptCdiConfigs(req *common.HcoRequest, cdiCfg *cdiv1beta1.CDIConfigSpec) bool {
	modified := false
	if cdiCfg != nil {
		if req.Instance.Spec.ScratchSpaceStorageClass == nil && cdiCfg.ScratchSpaceStorageClass != nil {
			req.Instance.Spec.ScratchSpaceStorageClass = new(string)
			*req.Instance.Spec.ScratchSpaceStorageClass = *cdiCfg.ScratchSpaceStorageClass
			modified = true
		}

		if cdiCfg.PodResourceRequirements != nil {
			if req.Instance.Spec.ResourceRequirements == nil {
				req.Instance.Spec.ResourceRequirements = &hcov1beta1.OperandResourceRequirements{}
			}

			if req.Instance.Spec.ResourceRequirements.StorageWorkloads == nil {
				req.Instance.Spec.ResourceRequirements.StorageWorkloads = cdiCfg.PodResourceRequirements.DeepCopy()
				modified = true
			}
		}
	}

	if modified {
		req.Dirty = true
	}

	return modified
}

func kvConfigMapToHyperConvergedCr(req *common.HcoRequest, kvCmLMConfig hcov1beta1.LiveMigrationConfigurations) {
	if kvCmLMConfig.BandwidthPerMigration != nil {
		req.Instance.Spec.LiveMigrationConfig.BandwidthPerMigration = kvCmLMConfig.BandwidthPerMigration
	}
	if kvCmLMConfig.CompletionTimeoutPerGiB != nil {
		req.Instance.Spec.LiveMigrationConfig.CompletionTimeoutPerGiB = kvCmLMConfig.CompletionTimeoutPerGiB
	}
	if kvCmLMConfig.ParallelMigrationsPerCluster != nil {
		req.Instance.Spec.LiveMigrationConfig.ParallelMigrationsPerCluster = kvCmLMConfig.ParallelMigrationsPerCluster
	}
	if kvCmLMConfig.ParallelOutboundMigrationsPerNode != nil {
		req.Instance.Spec.LiveMigrationConfig.ParallelOutboundMigrationsPerNode = kvCmLMConfig.ParallelOutboundMigrationsPerNode
	}
	if kvCmLMConfig.ProgressTimeout != nil {
		req.Instance.Spec.LiveMigrationConfig.ProgressTimeout = kvCmLMConfig.ProgressTimeout
	}
}

// getHyperConvergedNamespacedName returns the name/namespace of the HyperConverged resource
func getHyperConvergedNamespacedName() (types.NamespacedName, error) {
	hco := types.NamespacedName{
		Name: hcoutil.HyperConvergedName,
	}

	namespace, err := hcoutil.GetOperatorNamespaceFromEnv()
	if err != nil {
		return hco, err
	}
	hco.Namespace = namespace

	return hco, nil
}

// getOtherCrPlaceholder returns a placeholder to be able to discriminate
// reconciliation requests triggered by secondary watched resources
// use a random generated suffix for security reasons
func getSecondaryCRPlaceholder() (types.NamespacedName, error) {
	hco := types.NamespacedName{
		Name: secondaryCRPrefix + randomConstSuffix,
	}

	namespace, err := hcoutil.GetOperatorNamespaceFromEnv()
	if err != nil {
		return hco, err
	}
	hco.Namespace = namespace

	return hco, nil
}

func drop(slice []string, s string) ([]string, bool) {
	newSlice := []string{}
	dropped := false
	for _, element := range slice {
		if element != s {
			newSlice = append(newSlice, element)
		} else {
			dropped = true
		}
	}
	return newSlice, dropped
}

func init() {
	randomConstSuffix = uuid.New().String()
}

func checkFinalizers(req *common.HcoRequest) bool {
	finDropped := false

	if hcoutil.ContainsString(req.Instance.ObjectMeta.Finalizers, badFinalizerName) {
		req.Logger.Info("removing a finalizer set in the past (without a fully qualified name)")
		req.Instance.ObjectMeta.Finalizers, finDropped = drop(req.Instance.ObjectMeta.Finalizers, badFinalizerName)
		req.Dirty = req.Dirty || finDropped
	}
	if req.Instance.ObjectMeta.DeletionTimestamp.IsZero() {
		// Add the finalizer if it's not there
		if !hcoutil.ContainsString(req.Instance.ObjectMeta.Finalizers, FinalizerName) {
			req.Logger.Info("setting a finalizer (with fully qualified name)")
			req.Instance.ObjectMeta.Finalizers = append(req.Instance.ObjectMeta.Finalizers, FinalizerName)
			req.Dirty = req.Dirty || finDropped
		}
		return true
	}
	return false
}
