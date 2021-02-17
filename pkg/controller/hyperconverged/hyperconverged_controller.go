package hyperconverged

import (
	"context"
	"fmt"
	"os"

	"github.com/google/uuid"
	conditionsv1 "github.com/openshift/custom-resource-status/conditions/v1"
	operatorhandler "github.com/operator-framework/operator-lib/handler"

	corev1 "k8s.io/api/core/v1"
	schedulingv1 "k8s.io/api/scheduling/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/record"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/source"

	monitoringv1 "github.com/coreos/prometheus-operator/pkg/apis/monitoring/v1"
	networkaddonsv1 "github.com/kubevirt/cluster-network-addons-operator/pkg/apis/networkaddonsoperator/v1"
	hcov1beta1 "github.com/kubevirt/hyperconverged-cluster-operator/pkg/apis/hco/v1beta1"
	"github.com/kubevirt/hyperconverged-cluster-operator/pkg/controller/common"
	"github.com/kubevirt/hyperconverged-cluster-operator/pkg/controller/operands"
	hcoutil "github.com/kubevirt/hyperconverged-cluster-operator/pkg/util"
	version "github.com/kubevirt/hyperconverged-cluster-operator/version"
	vmimportv1beta1 "github.com/kubevirt/vm-import-operator/pkg/apis/v2v/v1beta1"
	kubevirtv1 "kubevirt.io/client-go/api/v1"
	cdiv1beta1 "kubevirt.io/containerized-data-importer/pkg/apis/core/v1beta1"
	sspv1beta1 "kubevirt.io/ssp-operator/api/v1beta1"
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
		client:             mgr.GetClient(),
		scheme:             mgr.GetScheme(),
		recorder:           mgr.GetEventRecorderFor(hcoutil.HyperConvergedName),
		cliDownloadHandler: &operands.CLIDownloadHandler{Client: mgr.GetClient(), Scheme: mgr.GetScheme()},
		operandHandler:     operands.NewOperandHandler(mgr.GetClient(), mgr.GetScheme(), ci.IsOpenshift(), hcoutil.GetEventEmitter()),
		upgradeMode:        false,
		ownVersion:         ownVersion,
		eventEmitter:       hcoutil.GetEventEmitter(),
		firstLoop:          true,
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

	secondaryResources := []client.Object{
		&kubevirtv1.KubeVirt{},
		&cdiv1beta1.CDI{},
		&networkaddonsv1.NetworkAddonsConfig{},
		&sspv1beta1.SSP{},
		&schedulingv1.PriorityClass{},
		&vmimportv1beta1.VMImportConfig{},
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
	client             client.Client
	scheme             *runtime.Scheme
	recorder           record.EventRecorder
	cliDownloadHandler *operands.CLIDownloadHandler
	operandHandler     *operands.OperandHandler
	upgradeMode        bool
	ownVersion         string
	eventEmitter       hcoutil.EventEmitter
	firstLoop          bool
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

	err = r.updateHyperConverged(hcoRequest)
	if apierrors.IsConflict(err) {
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

	// in-memory conditions should start off empty. It will only ever hold
	// negative conditions (!Available, Degraded, Progressing)
	req.Conditions = common.NewHcoConditions()

	finDropped := false
	// Handle finalizers
	if contains(req.Instance.ObjectMeta.Finalizers, badFinalizerName) {
		req.Logger.Info("removing a finalizer set in the past (without a fully qualified name)")
		req.Instance.ObjectMeta.Finalizers, finDropped = drop(req.Instance.ObjectMeta.Finalizers, badFinalizerName)
		req.Dirty = req.Dirty || finDropped
	}
	if req.Instance.ObjectMeta.DeletionTimestamp.IsZero() {
		// Add the finalizer if it's not there
		if !contains(req.Instance.ObjectMeta.Finalizers, FinalizerName) {
			req.Logger.Info("setting a finalizer (with fully qualified name)")
			req.Instance.ObjectMeta.Finalizers = append(req.Instance.ObjectMeta.Finalizers, FinalizerName)
			req.Dirty = req.Dirty || finDropped
		}
	} else {
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

	r.cliDownloadHandler.Ensure(req)

	err = r.operandHandler.Ensure(req)
	if err != nil {
		r.updateConditions(req)
		hcoutil.SetReady(false)
		return reconcile.Result{Requeue: init}, nil
	}

	req.Logger.Info("Reconcile complete")

	// Requeue if we just created everything
	if init {
		hcoutil.SetReady(false)
		return reconcile.Result{Requeue: true}, err
	}

	r.completeReconciliation(req)

	return reconcile.Result{}, nil
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
func (r *ReconcileHyperConverged) updateHyperConverged(request *common.HcoRequest) error {

	// Since the status subresource is enabled for the HyperConverged kind,
	// we need to update the status and the metadata separately.
	// Moreover, we need to update the status first, in order to prevent a conflict.

	err := r.updateHyperConvergedStatus(request)
	if err != nil {
		r.logHyperConvergedUpdateError(request, err, "Failed to update HCO Status")
		return err
	}

	// Doing it here because status.update overrides spec for some reason
	r.recoverHCOVersion(request)

	err = r.updateHyperConvergedSpecMetadata(request)
	if err != nil {
		r.logHyperConvergedUpdateError(request, err, "Failed to update HCO CR")
		return err
	}

	return nil
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
		req.Conditions.SetStatusCondition(conditionsv1.Condition{
			Type:    hcov1beta1.ConditionReconcileComplete,
			Status:  corev1.ConditionFalse,
			Reason:  invalidRequestReason,
			Message: fmt.Sprintf(invalidRequestMessageFormat, hco.Name, hco.Namespace),
		})
		r.updateConditions(req)
		return false, nil
	}
	return true, nil
}

func (r *ReconcileHyperConverged) setInitialConditions(req *common.HcoRequest) {
	req.Instance.Status.UpdateVersion(hcoVersionName, r.ownVersion)
	req.Instance.Spec.Version = r.ownVersion
	req.Dirty = true

	req.Conditions.SetStatusCondition(conditionsv1.Condition{
		Type:    hcov1beta1.ConditionReconcileComplete,
		Status:  corev1.ConditionUnknown, // we just started trying to reconcile
		Reason:  reconcileInit,
		Message: reconcileInitMessage,
	})
	req.Conditions.SetStatusCondition(conditionsv1.Condition{
		Type:    conditionsv1.ConditionAvailable,
		Status:  corev1.ConditionFalse,
		Reason:  reconcileInit,
		Message: reconcileInitMessage,
	})
	req.Conditions.SetStatusCondition(conditionsv1.Condition{
		Type:    conditionsv1.ConditionProgressing,
		Status:  corev1.ConditionTrue,
		Reason:  reconcileInit,
		Message: reconcileInitMessage,
	})
	req.Conditions.SetStatusCondition(conditionsv1.Condition{
		Type:    conditionsv1.ConditionDegraded,
		Status:  corev1.ConditionFalse,
		Reason:  reconcileInit,
		Message: reconcileInitMessage,
	})
	req.Conditions.SetStatusCondition(conditionsv1.Condition{
		Type:    conditionsv1.ConditionUpgradeable,
		Status:  corev1.ConditionUnknown,
		Reason:  reconcileInit,
		Message: reconcileInitMessage,
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
	if contains(req.Instance.ObjectMeta.Finalizers, FinalizerName) {
		req.Instance.ObjectMeta.Finalizers, finDropped = drop(req.Instance.ObjectMeta.Finalizers, FinalizerName)
		req.Dirty = true
		requeue = requeue || finDropped
	}
	if contains(req.Instance.ObjectMeta.Finalizers, badFinalizerName) {
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
	req.Conditions.SetStatusCondition(conditionsv1.Condition{
		Type:    hcov1beta1.ConditionReconcileComplete,
		Status:  corev1.ConditionTrue,
		Reason:  reconcileCompleted,
		Message: reconcileCompletedMessage,
	})

	if req.Conditions.HasCondition(conditionsv1.ConditionDegraded) { // (#chart 1)

		req.Conditions.SetStatusConditionIfUnset(conditionsv1.Condition{ // (#chart 2,3)
			Type:    conditionsv1.ConditionProgressing,
			Status:  corev1.ConditionFalse,
			Reason:  reconcileCompleted,
			Message: reconcileCompletedMessage,
		})

		req.Conditions.SetStatusConditionIfUnset(conditionsv1.Condition{ // (#chart 4,5)
			Type:    conditionsv1.ConditionUpgradeable,
			Status:  corev1.ConditionFalse,
			Reason:  commonDegradedReason,
			Message: "HCO is not Upgradeable due to degraded components",
		})

		req.Conditions.SetStatusConditionIfUnset(conditionsv1.Condition{ // (#chart 6,7)
			Type:    conditionsv1.ConditionAvailable,
			Status:  corev1.ConditionFalse,
			Reason:  commonDegradedReason,
			Message: "HCO is not available due to degraded components",
		})

	} else {

		// Degraded is not found. add it.
		req.Conditions.SetStatusCondition(conditionsv1.Condition{ // (#chart 8)
			Type:    conditionsv1.ConditionDegraded,
			Status:  corev1.ConditionFalse,
			Reason:  reconcileCompleted,
			Message: reconcileCompletedMessage,
		})

		if req.Conditions.HasCondition(conditionsv1.ConditionProgressing) { // (#chart 9)

			req.Conditions.SetStatusConditionIfUnset(conditionsv1.Condition{ // (#chart 10,11)
				Type:    conditionsv1.ConditionUpgradeable,
				Status:  corev1.ConditionFalse,
				Reason:  commonProgressingReason,
				Message: "HCO is not Upgradeable due to progressing components",
			})

			req.Conditions.SetStatusConditionIfUnset(conditionsv1.Condition{ // (#chart 12,13)
				Type:    conditionsv1.ConditionAvailable,
				Status:  corev1.ConditionTrue,
				Reason:  reconcileCompleted,
				Message: reconcileCompletedMessage,
			})

		} else {

			req.Conditions.SetStatusCondition(conditionsv1.Condition{ // (#chart 14)
				Type:    conditionsv1.ConditionProgressing,
				Status:  corev1.ConditionFalse,
				Reason:  reconcileCompleted,
				Message: reconcileCompletedMessage,
			})

			req.Conditions.SetStatusConditionIfUnset(conditionsv1.Condition{ // (#chart 15,16)
				Type:    conditionsv1.ConditionUpgradeable,
				Status:  corev1.ConditionTrue,
				Reason:  reconcileCompleted,
				Message: reconcileCompletedMessage,
			})

			req.Conditions.SetStatusConditionIfUnset(conditionsv1.Condition{ // (#chart 17,18)
				Type:    conditionsv1.ConditionAvailable,
				Status:  corev1.ConditionTrue,
				Reason:  reconcileCompleted,
				Message: reconcileCompletedMessage,
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

		// if in upgrade mode, and all the components are upgraded - upgrade is completed
		if r.upgradeMode && req.ComponentUpgradeInProgress {
			// update the new version only when upgrade is completed
			req.Instance.Status.UpdateVersion(hcoVersionName, r.ownVersion)
			req.StatusDirty = true

			req.Instance.Spec.Version = r.ownVersion
			req.Dirty = true

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
		req.Conditions.SetStatusCondition(conditionsv1.Condition{
			Type:    conditionsv1.ConditionProgressing,
			Status:  corev1.ConditionTrue,
			Reason:  "HCOUpgrading",
			Message: "HCO is now upgrading to version " + r.ownVersion,
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
	for _, condType := range common.HcoConditionTypes {
		cond, found := req.Conditions[condType]
		if !found {
			cond = conditionsv1.Condition{
				Type:    condType,
				Status:  corev1.ConditionUnknown,
				Message: "Unknown Status",
			}
		}
		conditionsv1.SetStatusCondition(&req.Instance.Status.Conditions, cond)
	}

	// Detect a "TaintedConfiguration" state, and raise a corresponding event
	r.detectTaintedConfiguration(req)

	req.StatusDirty = true
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

func (r *ReconcileHyperConverged) detectTaintedConfiguration(req *common.HcoRequest) {
	conditionExists := conditionsv1.IsStatusConditionTrue(req.Instance.Status.Conditions,
		hcov1beta1.ConditionTaintedConfiguration)

	// A tainted configuration state is indicated by the
	// presence of at least one of the JSON Patch annotations
	tainted := false
	for _, jpa := range JSONPatchAnnotationNames {
		_, exists := req.Instance.ObjectMeta.Annotations[jpa]
		if exists {
			tainted = true
			break
		}
	}

	if tainted {
		conditionsv1.SetStatusCondition(&req.Instance.Status.Conditions, conditionsv1.Condition{
			Type:    hcov1beta1.ConditionTaintedConfiguration,
			Status:  corev1.ConditionTrue,
			Reason:  taintedConfigurationReason,
			Message: taintedConfigurationMessage,
		})

		if !conditionExists {
			// Only log at "first occurrence" of detection
			req.Logger.Info("Detected tainted configuration state for HCO")
			req.StatusDirty = true
		}
	} else { // !tainted

		// For the sake of keeping the JSONPatch backdoor in low profile,
		// we just remove the condition instead of False'ing it.
		if conditionExists {
			conditionsv1.RemoveStatusCondition(&req.Instance.Status.Conditions, hcov1beta1.ConditionTaintedConfiguration)

			req.Logger.Info("Detected untainted configuration state for HCO")
			req.StatusDirty = true
		}
	}
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

// recoverHCOVersion recovers Spec.Version if upgrade missed when upgrade completed
func (r *ReconcileHyperConverged) recoverHCOVersion(request *common.HcoRequest) {
	knownHcoVersion, versionFound := request.Instance.Status.GetVersion(hcoVersionName)

	if !r.upgradeMode &&
		versionFound &&
		(knownHcoVersion == r.ownVersion) &&
		(request.Instance.Spec.Version != r.ownVersion) {

		request.Instance.Spec.Version = r.ownVersion
		request.Dirty = true
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

func contains(slice []string, s string) bool {
	for _, element := range slice {
		if element == s {
			return true
		}
	}
	return false
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
