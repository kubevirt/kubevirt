package hyperconverged

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"reflect"

	"github.com/blang/semver/v4"
	jsonpatch "github.com/evanphx/json-patch"
	"github.com/google/uuid"
	consolev1 "github.com/openshift/api/console/v1"
	imagev1 "github.com/openshift/api/image/v1"
	routev1 "github.com/openshift/api/route/v1"
	operatorhandler "github.com/operator-framework/operator-lib/handler"
	"github.com/pkg/errors"
	monitoringv1 "github.com/prometheus-operator/prometheus-operator/pkg/apis/monitoring/v1"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	schedulingv1 "k8s.io/api/scheduling/v1"
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	apimetav1 "k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/source"

	networkaddonsv1 "github.com/kubevirt/cluster-network-addons-operator/pkg/apis/networkaddonsoperator/v1"
	hcov1beta1 "github.com/kubevirt/hyperconverged-cluster-operator/pkg/apis/hco/v1beta1"
	"github.com/kubevirt/hyperconverged-cluster-operator/pkg/controller/common"
	"github.com/kubevirt/hyperconverged-cluster-operator/pkg/controller/operands"
	"github.com/kubevirt/hyperconverged-cluster-operator/pkg/metrics"
	hcoutil "github.com/kubevirt/hyperconverged-cluster-operator/pkg/util"
	"github.com/kubevirt/hyperconverged-cluster-operator/version"
	kubevirtcorev1 "kubevirt.io/api/core/v1"
	cdiv1beta1 "kubevirt.io/containerized-data-importer-api/pkg/apis/core/v1beta1"
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

	// These group are no longer supported. Use these constants to remove unused resources
	v2vGroup = "v2v.kubevirt.io"

	requestedStatusKey = "requested status"
)

// JSONPatchAnnotationNames - annotations used to patch operand CRs with unsupported/unofficial/hidden features.
// The presence of any of these annotations raises the hcov1beta1.ConditionTaintedConfiguration condition.
var JSONPatchAnnotationNames = []string{
	common.JSONPatchKVAnnotationName,
	common.JSONPatchCDIAnnotationName,
	common.JSONPatchCNAOAnnotationName,
}

// RegisterReconciler creates a new HyperConverged Reconciler and registers it into manager.
func RegisterReconciler(mgr manager.Manager, ci hcoutil.ClusterInfo, upgradeableCond hcoutil.Condition) error {
	return add(mgr, newReconciler(mgr, ci, upgradeableCond), ci)
}

// newReconciler returns a new reconcile.Reconciler
func newReconciler(mgr manager.Manager, ci hcoutil.ClusterInfo, upgradeableCond hcoutil.Condition) reconcile.Reconciler {

	ownVersion := os.Getenv(hcoutil.HcoKvIoVersionName)
	if ownVersion == "" {
		ownVersion = version.Version
	}

	return &ReconcileHyperConverged{
		client:               mgr.GetClient(),
		scheme:               mgr.GetScheme(),
		operandHandler:       operands.NewOperandHandler(mgr.GetClient(), mgr.GetScheme(), ci.IsOpenshift(), hcoutil.GetEventEmitter()),
		upgradeMode:          false,
		ownVersion:           ownVersion,
		eventEmitter:         hcoutil.GetEventEmitter(),
		firstLoop:            true,
		upgradeableCondition: upgradeableCond,
	}
}

// newCRDremover returns a new CRDRemover
func newCRDremover(client client.Client) *CRDRemover {
	crdRemover := &CRDRemover{
		client: client,
		crdsToRemove: []schema.GroupKind{
			// These are the v2v CRDs we have to remove moving to MTV
			{Group: v2vGroup, Kind: "V2VVmware"},
			{Group: v2vGroup, Kind: "OVirtProvider"},
			{Group: v2vGroup, Kind: "VMImportConfig"},
		},
	}

	// The list of related objects to remove is initialized empty;
	// Once the corresponding CRD (and hence CR) is removed successfully,
	// the CR can be removed from the list of related objects.
	crdRemover.relatedObjectsToRemove = make([]schema.GroupKind, 0, len(crdRemover.crdsToRemove))

	return crdRemover
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
		&kubevirtcorev1.KubeVirt{},
		&cdiv1beta1.CDI{},
		&networkaddonsv1.NetworkAddonsConfig{},
		&sspv1beta1.SSP{},
		&schedulingv1.PriorityClass{},
		&corev1.ConfigMap{},
		&corev1.Service{},
		&rbacv1.Role{},
		&rbacv1.RoleBinding{},
	}
	if ci.IsOpenshift() {
		secondaryResources = append(secondaryResources, []client.Object{
			&corev1.Service{},
			&monitoringv1.ServiceMonitor{},
			&monitoringv1.PrometheusRule{},
			&routev1.Route{},
			&consolev1.ConsoleCLIDownload{},
			&imagev1.ImageStream{},
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
	client               client.Client
	scheme               *runtime.Scheme
	operandHandler       *operands.OperandHandler
	upgradeMode          bool
	ownVersion           string
	eventEmitter         hcoutil.EventEmitter
	firstLoop            bool
	upgradeableCondition hcoutil.Condition
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

	hcoRequest.Instance = instance

	if instance == nil {
		// if the HyperConverged CR was deleted during an upgrade process, then this is not an upgrade anymore
		r.upgradeMode = false
		if err == nil {
			err = r.setOperatorUpgradeableStatus(hcoRequest)
		}

		return reconcile.Result{}, err
	}

	if r.firstLoop {
		r.firstLoopInitialization(hcoRequest)
	}

	result, err := r.doReconcile(hcoRequest)
	if err != nil {
		r.eventEmitter.EmitEvent(hcoRequest.Instance, corev1.EventTypeWarning, "ReconcileError", err.Error())
		return result, err
	}

	if err = r.setOperatorUpgradeableStatus(hcoRequest); err != nil {
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

	applyDataImportSchedule(req)

	// If the current version is not updated in CR ,then we're updating. This is also works when updating from
	// an old version, since Status.Versions will be empty.
	knownHcoVersion, _ := req.Instance.Status.GetVersion(hcoVersionName)

	// detect upgrade mode
	if !r.upgradeMode && !init && knownHcoVersion != r.ownVersion {
		// get into upgrade mode

		r.upgradeMode = true

		r.eventEmitter.EmitEvent(req.Instance, corev1.EventTypeNormal, "UpgradeHCO", "Upgrading the HyperConverged to version "+r.ownVersion)
		req.Logger.Info(fmt.Sprintf("Start upgrading from version %s to version %s", knownHcoVersion, r.ownVersion))
	}

	req.SetUpgradeMode(r.upgradeMode)

	if r.upgradeMode {

		err = validateUpgradePatches(req)
		if err != nil {
			return reconcile.Result{Requeue: true}, err
		}

		crdStatusUpdated, err := r.updateCrdStoredVersions(req)
		if err != nil {
			return reconcile.Result{Requeue: true}, err
		}

		if crdStatusUpdated {
			return reconcile.Result{Requeue: true}, nil
		}

		// Attempt to remove old CRDs and related objects:
		// Directly removing v2v CRDs will be enough to trigger the
		// removal of the corresponding CRs.
		// vmimportconfigs CR is protected by "vm-import-finalizer"
		// which is managed by vm-import-operator that should remove
		// all of its operands before removing the finalizer so that the
		// CR and then the CRD can be really deleted.
		// We don't have any race condition here because the CRD will be
		// deleted without blocking the caller.
		// vm-import-operator pod on the other side is not deleted during the
		// upgrade process, but only at the end of it as a consequence of
		// the removal of the old CSV (vm-import-operator deployment has
		// an owner reference on it) once the upgrade successfully completed.

		cdrRemover := newCRDremover(r.client)
		err = cdrRemover.Remove(req)
		if err != nil {
			return reconcile.Result{Requeue: init}, err
		}
		// if we still have something to remove, requeue to retry
		if !cdrRemover.Done() {
			return reconcile.Result{Requeue: true}, nil
		}

		modified, err := r.migrateBeforeUpgrade(req)
		if err != nil {
			return reconcile.Result{Requeue: true}, err
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
		return reconcile.Result{Requeue: init}, nil
	}

	req.Logger.Info("Reconcile complete")

	// Requeue if we just created everything
	if init {
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

	// should never happen - we are dropping the wrong finalizer in checkFinalizers, that always called before this
	// function
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
			if NumOfChanges = getNumOfChangesJSONPatch(jsonPatch); NumOfChanges > 0 {
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

func getNumOfChangesJSONPatch(jsonPatch string) int {
	patches, err := jsonpatch.DecodePatch([]byte(jsonPatch))
	if err != nil {
		return 0
	}
	return len(patches)
}

func (r *ReconcileHyperConverged) firstLoopInitialization(request *common.HcoRequest) {
	// Initialize operand handler.
	r.operandHandler.FirstUseInitiation(r.scheme, hcoutil.GetClusterInfo().IsOpenshift(), request.Instance)

	// Avoid re-initializing.
	r.firstLoop = false
}

func (r *ReconcileHyperConverged) setOperatorUpgradeableStatus(request *common.HcoRequest) error {
	if hcoutil.GetClusterInfo().IsManagedByOLM() {

		request.Logger.Info("setting the Upgradeable operator condition", requestedStatusKey, !r.upgradeMode)

		msg := hcoutil.UpgradeableAllowMessage
		status := metav1.ConditionTrue
		reason := hcoutil.UpgradeableAllowReason

		if r.upgradeMode {
			msg = hcoutil.UpgradeableUpgradingMessage + r.ownVersion
			status = metav1.ConditionFalse
			reason = hcoutil.UpgradeableUpgradingReason
		}

		if err := r.upgradeableCondition.Set(request.Ctx, status, reason, msg); err != nil {
			request.Logger.Error(err, "can't set the Upgradeable operator condition", requestedStatusKey, !r.upgradeMode)
			return err
		}

	}

	return nil
}

const crdName = "hyperconvergeds.hco.kubevirt.io"

func (r *ReconcileHyperConverged) updateCrdStoredVersions(req *common.HcoRequest) (bool, error) {
	versionsToBeRemoved := []string{"v1alpha1"}

	found := &apiextensionsv1.CustomResourceDefinition{}
	key := client.ObjectKey{Namespace: hcoutil.UndefinedNamespace, Name: crdName}
	err := r.client.Get(req.Ctx, key, found)
	if err != nil {
		req.Logger.Error(err, fmt.Sprintf("failed to read the %s CRD; %s", crdName, err.Error()))
		return false, err
	}

	needsUpdate := false
	var newStoredVersions []string
	for _, vToBeRemoved := range versionsToBeRemoved {
		for _, sVersion := range found.Status.StoredVersions {
			if vToBeRemoved != sVersion {
				newStoredVersions = append(newStoredVersions, sVersion)
			} else {
				needsUpdate = true
			}
		}
	}
	if needsUpdate {
		found.Status.StoredVersions = newStoredVersions
		err = r.client.Status().Update(req.Ctx, found)
		if err != nil {
			req.Logger.Error(err, fmt.Sprintf("failed updating the %s CRD status: %s", crdName, err.Error()))
			return false, err
		}
		req.Logger.Info("successfully updated status.storedVersions on HCO CRD", "CRD Name", crdName)
		return true, nil
	}

	return false, nil
}

func (r *ReconcileHyperConverged) migrateBeforeUpgrade(req *common.HcoRequest) (bool, error) {
	upgradePatched, err := r.applyUpgradePatches(req)
	if err != nil {
		return false, err
	}

	removeOldQuickStartGuides(req, r.client, r.operandHandler.GetQuickStartNames())

	return upgradePatched, nil
}

func (r ReconcileHyperConverged) applyUpgradePatches(req *common.HcoRequest) (bool, error) {
	modified := false

	knownHcoVersion, _ := req.Instance.Status.GetVersion(hcoVersionName)
	if knownHcoVersion == "" {
		knownHcoVersion = "0.0.0"
	}
	knownHcoSV, err := semver.ParseTolerant(knownHcoVersion)
	if err != nil {
		req.Logger.Error(err, "Error!")
		return false, err
	}

	hcoJson, err := json.Marshal(req.Instance)
	if err != nil {
		return false, err
	}

	for _, p := range hcoUpgradeChanges.HCOCRPatchList {
		hcoJson, err = r.applyUpgradePatch(req, hcoJson, knownHcoSV, p)
		if err != nil {
			return false, err
		}
	}

	tmpInstance := &hcov1beta1.HyperConverged{}
	err = json.Unmarshal(hcoJson, tmpInstance)
	if err != nil {
		return false, err
	}
	if !reflect.DeepEqual(tmpInstance.Spec, req.Instance.Spec) {
		req.Logger.Info("updating HCO spec as a result of upgrade patches")
		tmpInstance.Spec.DeepCopyInto(&req.Instance.Spec)
		modified = true
		req.Dirty = true
	}

	return modified, nil
}

func (r ReconcileHyperConverged) applyUpgradePatch(req *common.HcoRequest, hcoJson []byte, knownHcoSV semver.Version, p hcoCRPatch) ([]byte, error) {
	affectedRange, err := semver.ParseRange(p.SemverRange)
	if err != nil {
		return hcoJson, err
	}
	if affectedRange(knownHcoSV) {
		req.Logger.Info("applying upgrade patch", "knownHcoSV", knownHcoSV, "affectedRange", p.SemverRange, "patches", p.JSONPatch)
		patchedBytes, err := p.JSONPatch.Apply(hcoJson)
		if err != nil {
			if errors.Cause(err) == jsonpatch.ErrTestFailed {
				return hcoJson, nil
			}

			return hcoJson, err
		}
		return patchedBytes, nil
	}
	return hcoJson, nil
}

func removeOldQuickStartGuides(req *common.HcoRequest, cl client.Client, requiredQSList []string) {
	existingQSList := &consolev1.ConsoleQuickStartList{}
	req.Logger.Info("reading quickstart guides")
	err := cl.List(req.Ctx, existingQSList, client.MatchingLabels{hcoutil.AppLabelManagedBy: hcoutil.OperatorName})
	if err != nil {
		req.Logger.Error(err, "failed to read list of quickstart guides")
		return
	}

	var existingQSNames map[string]consolev1.ConsoleQuickStart
	if len(existingQSList.Items) > 0 {
		existingQSNames = make(map[string]consolev1.ConsoleQuickStart)
		for _, qs := range existingQSList.Items {
			existingQSNames[qs.Name] = qs
		}

		for name, existQs := range existingQSNames {
			if !hcoutil.ContainsString(requiredQSList, name) {
				req.Logger.Info("deleting ConsoleQuickStart", "name", name)
				if _, err = hcoutil.EnsureDeleted(req.Ctx, cl, &existQs, req.Instance.Name, req.Logger, false, false); err != nil {
					req.Logger.Error(err, "failed to delete ConsoleQuickStart", "name", name)
				}
			}
		}

		removeRelatedQSObjects(req, requiredQSList)
	}
}

// removeRelatedQSObjects removes old quickstart from the related object list
// can't use the removeRelatedObject function because the status not get updated during each reconcile loop,
// but the old qs already removed (above) so you loos track of it. That why we must re-check all the qs names
func removeRelatedQSObjects(req *common.HcoRequest, requiredNames []string) {
	refs := make([]corev1.ObjectReference, 0, len(req.Instance.Status.RelatedObjects))
	foundOldQs := false

	for _, obj := range req.Instance.Status.RelatedObjects {
		if obj.Kind == "ConsoleQuickStart" && !hcoutil.ContainsString(requiredNames, obj.Name) {
			foundOldQs = true
			continue
		}
		refs = append(refs, obj)
	}

	if foundOldQs {
		req.Instance.Status.RelatedObjects = refs
		req.StatusDirty = true
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
	var newSlice []string
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
