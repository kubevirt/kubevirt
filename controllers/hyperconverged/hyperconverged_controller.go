package hyperconverged

import (
	"cmp"
	"context"
	"fmt"
	"os"
	"reflect"
	"slices"

	"github.com/blang/semver/v4"
	jsonpatch "github.com/evanphx/json-patch/v5"
	"github.com/go-logr/logr"
	openshiftconfigv1 "github.com/openshift/api/config/v1"
	consolev1 "github.com/openshift/api/console/v1"
	imagev1 "github.com/openshift/api/image/v1"
	routev1 "github.com/openshift/api/route/v1"
	operatorhandler "github.com/operator-framework/operator-lib/handler"
	monitoringv1 "github.com/prometheus-operator/prometheus-operator/pkg/apis/monitoring/v1"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	schedulingv1 "k8s.io/api/scheduling/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	apimetav1 "k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/utils/ptr"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/source"

	networkaddonsv1 "github.com/kubevirt/cluster-network-addons-operator/pkg/apis/networkaddonsoperator/v1"
	kubevirtcorev1 "kubevirt.io/api/core/v1"
	aaqv1alpha1 "kubevirt.io/application-aware-quota/staging/src/kubevirt.io/application-aware-quota-api/pkg/apis/core/v1alpha1"
	cdiv1beta1 "kubevirt.io/containerized-data-importer-api/pkg/apis/core/v1beta1"
	sspv1beta2 "kubevirt.io/ssp-operator/api/v1beta2"

	hcov1beta1 "github.com/kubevirt/hyperconverged-cluster-operator/api/v1beta1"
	"github.com/kubevirt/hyperconverged-cluster-operator/controllers/alerts"
	"github.com/kubevirt/hyperconverged-cluster-operator/controllers/common"
	"github.com/kubevirt/hyperconverged-cluster-operator/controllers/operands"
	"github.com/kubevirt/hyperconverged-cluster-operator/controllers/reqresolver"
	"github.com/kubevirt/hyperconverged-cluster-operator/pkg/monitoring/hyperconverged/metrics"
	"github.com/kubevirt/hyperconverged-cluster-operator/pkg/upgradepatch"
	hcoutil "github.com/kubevirt/hyperconverged-cluster-operator/pkg/util"
	"github.com/kubevirt/hyperconverged-cluster-operator/version"
)

var (
	log = logf.Log.WithName("controller_hyperconverged")
)

const (
	// We cannot set owner reference of cluster-wide resources to namespaced HyperConverged object. Therefore,
	// use finalizers to manage the cleanup.
	FinalizerName = "kubevirt.io/hyperconverged"

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
	systemHealthStatusHealthy   = "healthy"
	systemHealthStatusWarning   = "warning"
	systemHealthStatusError     = "error"

	hcoVersionName = "operator"

	requestedStatusKey = "requested status"
)

// JSONPatchAnnotationNames - annotations used to patch operand CRs with unsupported/unofficial/hidden features.
// The presence of any of these annotations raises the hcov1beta1.ConditionTaintedConfiguration condition.
var JSONPatchAnnotationNames = []string{
	common.JSONPatchKVAnnotationName,
	common.JSONPatchCDIAnnotationName,
	common.JSONPatchCNAOAnnotationName,
	common.JSONPatchSSPAnnotationName,
}

// RegisterReconciler creates a new HyperConverged Reconciler and registers it into manager.
func RegisterReconciler(mgr manager.Manager, ci hcoutil.ClusterInfo, upgradeableCond hcoutil.Condition, ingressEventCh <-chan event.TypedGenericEvent[client.Object]) error {
	return add(mgr, newReconciler(mgr, ci, upgradeableCond), ci, ingressEventCh)
}

// newReconciler returns a new reconcile.Reconciler
func newReconciler(mgr manager.Manager, ci hcoutil.ClusterInfo, upgradeableCond hcoutil.Condition) reconcile.Reconciler {

	ownVersion := cmp.Or(os.Getenv(hcoutil.HcoKvIoVersionName), version.Version)

	r := &ReconcileHyperConverged{
		client:               mgr.GetClient(),
		scheme:               mgr.GetScheme(),
		operandHandler:       operands.NewOperandHandler(mgr.GetClient(), mgr.GetScheme(), ci, hcoutil.GetEventEmitter()),
		upgradeMode:          false,
		ownVersion:           ownVersion,
		eventEmitter:         hcoutil.GetEventEmitter(),
		firstLoop:            true,
		upgradeableCondition: upgradeableCond,
	}

	if ci.IsMonitoringAvailable() {
		r.monitoringReconciler = alerts.NewMonitoringReconciler(ci, r.client, hcoutil.GetEventEmitter(), r.scheme)
	}

	return r
}

// newCRDremover returns a new CRDRemover
func add(mgr manager.Manager, r reconcile.Reconciler, ci hcoutil.ClusterInfo, ingressEventCh <-chan event.TypedGenericEvent[client.Object]) error {
	// Create a new controller
	c, err := controller.New("hyperconverged-controller", mgr, controller.Options{Reconciler: r})
	if err != nil {
		return err
	}

	// Watch for changes to primary resource HyperConverged
	err = c.Watch(
		source.Kind(
			mgr.GetCache(), client.Object(&hcov1beta1.HyperConverged{}),
			&operatorhandler.InstrumentedEnqueueRequestForObject[client.Object]{},
			predicate.Or[client.Object](predicate.GenerationChangedPredicate{}, predicate.AnnotationChangedPredicate{},
				predicate.ResourceVersionChangedPredicate{}),
		))
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
		&aaqv1alpha1.AAQ{},
		&schedulingv1.PriorityClass{},
		&corev1.ConfigMap{},
		&corev1.Service{},
		&rbacv1.Role{},
		&rbacv1.RoleBinding{},
	}
	if ci.IsMonitoringAvailable() {
		secondaryResources = append(secondaryResources, []client.Object{
			&monitoringv1.ServiceMonitor{},
			&monitoringv1.PrometheusRule{},
		}...)
	}
	if ci.IsOpenshift() {
		secondaryResources = append(secondaryResources, []client.Object{
			&sspv1beta2.SSP{},
			&corev1.Service{},
			&routev1.Route{},
			&consolev1.ConsoleCLIDownload{},
			&consolev1.ConsoleQuickStart{},
			&consolev1.ConsolePlugin{},
			&imagev1.ImageStream{},
			&corev1.Namespace{},
			&appsv1.Deployment{},
		}...)
	}

	// Watch secondary resources
	for _, resource := range secondaryResources {
		msg := fmt.Sprintf("Reconciling for %T", resource)
		err = c.Watch(
			source.Kind(mgr.GetCache(), resource,
				handler.EnqueueRequestsFromMapFunc(func(ctx context.Context, a client.Object) []reconcile.Request {
					// enqueue using a placeholder to be able to discriminate request triggered
					// by changes on the HyperConverged object from request triggered by changes
					// on a secondary CR controlled by HCO
					log.Info(msg)
					return []reconcile.Request{
						reqresolver.GetSecondaryCRRequest(),
					}
				}),
			))
		if err != nil {
			return err
		}
	}

	if ci.IsOpenshift() {
		err = c.Watch(
			source.Kind(
				mgr.GetCache(),
				client.Object(&openshiftconfigv1.APIServer{}),
				handler.EnqueueRequestsFromMapFunc(func(ctx context.Context, a client.Object) []reconcile.Request {
					// enqueue using a placeholder to signal that the change is not
					// directly on HCO CR but on the APIServer CR that we want to reload
					// only if really changed
					log.Info("Reconciling for openshiftconfigv1.APIServer")
					return []reconcile.Request{
						reqresolver.GetAPIServerCRRequest(),
					}
				}),
			))
		if err != nil {
			return err
		}

		err = c.Watch(
			source.Channel(
				ingressEventCh,
				handler.EnqueueRequestsFromMapFunc(func(ctx context.Context, a client.Object) []reconcile.Request {
					// the ingress-cluster controller initiate this by pushing an event to the ingressEventCh channel
					// This will force this controller to update the URL of the cli download route, if the user
					// customized the hostname.
					log.Info("Reconciling for openshiftconfigv1.Ingress")
					return []reconcile.Request{
						reqresolver.GetIngressCRResource(),
					}
				}),
			))
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
	monitoringReconciler *alerts.MonitoringReconciler
}

// Reconcile reads that state of the cluster for a HyperConverged object and makes changes based on the state read
// and what is in the HyperConverged.Spec
// Note:
// The Controller will requeue the Request to be processed again if the returned error is non-nil or
// Result.Requeue is true, otherwise upon completion it will remove the work from the queue.
func (r *ReconcileHyperConverged) Reconcile(ctx context.Context, request reconcile.Request) (reconcile.Result, error) {
	logger := log.WithValues("Request.Namespace", request.Namespace, "Request.Name", request.Name)
	err := r.refreshAPIServerCR(ctx, logger, request)
	if err != nil {
		return reconcile.Result{}, err
	}

	resolvedRequest, hcoTriggered := reqresolver.ResolveReconcileRequest(log, request)
	hcoRequest := common.NewHcoRequest(ctx, resolvedRequest, log, r.upgradeMode, hcoTriggered)

	if hcoTriggered {
		r.operandHandler.Reset()
	}

	err = r.monitoringReconciler.Reconcile(hcoRequest, r.firstLoop)
	if err != nil {
		return reconcile.Result{}, err
	}

	// Fetch the HyperConverged instance
	instance, err := r.getHyperConverged(hcoRequest)
	if err != nil {
		return reconcile.Result{}, err
	}

	hcoRequest.Instance = instance

	if instance == nil {
		// if the HyperConverged CR was deleted during an upgrade process, then this is not an upgrade anymore
		r.upgradeMode = false
		err = r.setOperatorUpgradeableStatus(hcoRequest)

		return reconcile.Result{}, err
	}

	if r.firstLoop {
		r.firstLoopInitialization(hcoRequest)
	}

	if err = r.monitoringReconciler.UpdateRelatedObjects(hcoRequest); err != nil {
		logger.Error(err, "Failed to update the PrometheusRule as a related object")
		return reconcile.Result{}, err
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

// refreshAPIServerCR refreshes the APIServer cR, if the request is triggered by this CR.
func (r *ReconcileHyperConverged) refreshAPIServerCR(ctx context.Context, logger logr.Logger, originalRequest reconcile.Request) error {
	if reqresolver.IsTriggeredByAPIServerCR(originalRequest) {
		logger.Info("Refreshing the ApiServer CR")
		return hcoutil.GetClusterInfo().RefreshAPIServerCR(ctx, r.client)
	}

	return nil
}

func (r *ReconcileHyperConverged) doReconcile(req *common.HcoRequest) (reconcile.Result, error) {

	valid := r.validateNamespace(req)
	if !valid {
		return reconcile.Result{}, nil
	}

	// Add conditions if there are none
	init := req.Instance.Status.Conditions == nil
	if init {
		r.eventEmitter.EmitEvent(req.Instance, corev1.EventTypeNormal, "InitHCO", "Initiating the HyperConverged")
		r.setInitialConditions(req)

		req.Instance.Status.InfrastructureHighlyAvailable = ptr.To(hcoutil.GetClusterInfo().IsInfrastructureHighlyAvailable())
		req.StatusDirty = true
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
	knownHcoVersion, _ := GetVersion(&req.Instance.Status, hcoVersionName)

	// detect upgrade mode
	if !r.upgradeMode && !init && knownHcoVersion != r.ownVersion {
		// get into upgrade mode

		r.upgradeMode = true
		r.eventEmitter.EmitEvent(req.Instance, corev1.EventTypeNormal, "UpgradeHCO", "Upgrading the HyperConverged to version "+r.ownVersion)
		req.Logger.Info(fmt.Sprintf("Start upgrading from version %s to version %s", knownHcoVersion, r.ownVersion))
	}

	req.SetUpgradeMode(r.upgradeMode)

	if r.upgradeMode {
		if result, err := r.handleUpgrade(req); result != nil {
			return *result, err
		}
	}

	return r.EnsureOperandAndComplete(req, init)
}

func (r *ReconcileHyperConverged) handleUpgrade(req *common.HcoRequest) (*reconcile.Result, error) {
	modified, err := r.migrateBeforeUpgrade(req)
	if err != nil {
		return &reconcile.Result{Requeue: true}, err
	}

	if modified {
		r.updateConditions(req)
		return &reconcile.Result{Requeue: true}, nil
	}
	return nil, nil
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
	if req.Instance.Generation != req.Instance.Status.ObservedGeneration {
		req.Instance.Status.ObservedGeneration = req.Instance.Generation
		req.StatusDirty = true
	}
}

// getHyperConverged gets the HyperConverged resource from the Kubernetes API.
func (r *ReconcileHyperConverged) getHyperConverged(req *common.HcoRequest) (*hcov1beta1.HyperConverged, error) {
	instance := &hcov1beta1.HyperConverged{}
	err := r.client.Get(req.Ctx, req.NamespacedName, instance)

	// Green path first
	if err == nil {
		metrics.SetHCOMetricHyperConvergedExists()
		return instance, nil
	}

	// Error path
	if apierrors.IsNotFound(err) {
		req.Logger.Info("No HyperConverged resource")
		metrics.SetHCOMetricHyperConvergedNotExists()

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
	// In addition, metadata and spec changes are removed by status update, but since status update done first, we need
	// to store metadata and spec and recover it after status update

	var spec hcov1beta1.HyperConvergedSpec
	var meta metav1.ObjectMeta
	if request.Dirty {
		request.Instance.Spec.DeepCopyInto(&spec)
		request.Instance.ObjectMeta.DeepCopyInto(&meta)
	}

	err := r.updateHyperConvergedStatus(request)
	if err != nil {
		request.Logger.Error(err, "Failed to update HCO Status")
		return false, err
	}

	if request.Dirty {
		request.Instance.Annotations = meta.Annotations
		request.Instance.Finalizers = meta.Finalizers
		request.Instance.Labels = meta.Labels
		request.Instance.Spec = spec

		err = r.updateHyperConvergedSpecMetadata(request)
		if err != nil {
			request.Logger.Error(err, "Failed to update HCO CR")
			return false, err
		}
		// version update is a two steps process
		knownHcoVersion, _ := GetVersion(&request.Instance.Status, hcoVersionName)
		if r.ownVersion != knownHcoVersion && request.StatusDirty {
			return true, nil
		}
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

func (r *ReconcileHyperConverged) validateNamespace(req *common.HcoRequest) bool {
	// Ignore invalid requests
	if !reqresolver.IsTriggeredByHyperConverged(req.NamespacedName) {
		req.Logger.Info("Invalid request", "HyperConverged.Namespace", req.Namespace, "HyperConverged.Name", req.Name)
		hc := reqresolver.GetHyperConvergedNamespacedName()
		req.Conditions.SetStatusCondition(metav1.Condition{
			Type:               hcov1beta1.ConditionReconcileComplete,
			Status:             metav1.ConditionFalse,
			Reason:             invalidRequestReason,
			Message:            fmt.Sprintf(invalidRequestMessageFormat, hc.Name, hc.Namespace),
			ObservedGeneration: req.Instance.Generation,
		})
		r.updateConditions(req)
		return false
	}
	return true
}

func (r *ReconcileHyperConverged) setInitialConditions(req *common.HcoRequest) {
	UpdateVersion(&req.Instance.Status, hcoVersionName, r.ownVersion)

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
	if slices.Contains(req.Instance.Finalizers, FinalizerName) {
		req.Instance.Finalizers, finDropped = drop(req.Instance.Finalizers, FinalizerName)
		req.Dirty = true
		requeue = finDropped
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
			UpdateVersion(&req.Instance.Status, hcoVersionName, r.ownVersion)
			req.StatusDirty = true

			r.upgradeMode = false
			req.ComponentUpgradeInProgress = false
			req.Logger.Info(fmt.Sprintf("Successfully upgraded to version %s", r.ownVersion))
			r.eventEmitter.EmitEvent(req.Instance, corev1.EventTypeNormal, "UpgradeHCO", fmt.Sprintf("Successfully upgraded to version %s", r.ownVersion))
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

	// check if HCO was available before this reconcile loop
	hcoWasAvailable := apimetav1.IsStatusConditionTrue(req.Instance.Status.Conditions, hcov1beta1.ConditionAvailable) &&
		apimetav1.IsStatusConditionFalse(req.Instance.Status.Conditions, hcov1beta1.ConditionProgressing)

	if hcoReady {
		// If no operator whose conditions we are watching reports an error, then it is safe
		// to set readiness.
		if !hcoWasAvailable { // only when become available
			r.eventEmitter.EmitEvent(req.Instance, corev1.EventTypeNormal, "ReconcileHCO", "HCO Reconcile completed successfully")
		}
	} else {
		// If for any reason we marked ourselves !upgradeable...then unset readiness
		if !r.upgradeMode && hcoWasAvailable { // only when become not ready
			r.eventEmitter.EmitEvent(req.Instance, corev1.EventTypeWarning, "ReconcileHCO", "Not all the operators are ready")
		}
	}

	r.updateConditions(req)
}

// This function is used to exit from the reconcile function, updating the conditions and returns the reconcile result
func (r *ReconcileHyperConverged) updateConditions(req *common.HcoRequest) {
	conditions := slices.Clone(req.Instance.Status.Conditions)

	for _, condType := range common.HcoConditionTypes {
		cond, found := req.Conditions[condType]
		if !found {
			cond = metav1.Condition{
				Type:               condType,
				Status:             metav1.ConditionUnknown,
				Message:            "Unknown Status",
				Reason:             "StatusUnknown",
				ObservedGeneration: req.Instance.Generation,
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

	systemHealthStatus := r.getSystemHealthStatus(req.Conditions)

	if systemHealthStatus != req.Instance.Status.SystemHealthStatus {
		req.Instance.Status.SystemHealthStatus = systemHealthStatus
		req.StatusDirty = true
	}

	metrics.SetHCOMetricSystemHealthStatus(getNumericalHealthStatus(systemHealthStatus))
}

func (r *ReconcileHyperConverged) setLabels(req *common.HcoRequest) {
	if req.Instance.Labels == nil {
		req.Instance.Labels = map[string]string{}
	}
	if req.Instance.Labels[hcoutil.AppLabel] == "" {
		req.Instance.Labels[hcoutil.AppLabel] = req.Instance.Name
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
		jsonPatch, exists := req.Instance.Annotations[jpa]
		if exists {
			if NumOfChanges = getNumOfChangesJSONPatch(jsonPatch); NumOfChanges > 0 {
				tainted = true
			}
		}
		metrics.SetUnsafeModificationCount(NumOfChanges, jpa)
	}

	if tainted {
		apimetav1.SetStatusCondition(conditions, metav1.Condition{
			Type:               hcov1beta1.ConditionTaintedConfiguration,
			Status:             metav1.ConditionTrue,
			Reason:             taintedConfigurationReason,
			Message:            taintedConfigurationMessage,
			ObservedGeneration: req.Instance.Generation,
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

func (r *ReconcileHyperConverged) getSystemHealthStatus(conditions common.HcoConditions) string {
	if isSystemHealthStatusError(conditions) {
		return systemHealthStatusError
	}

	if isSystemHealthStatusWarning(conditions) {
		return systemHealthStatusWarning
	}

	return systemHealthStatusHealthy
}

func isSystemHealthStatusError(conditions common.HcoConditions) bool {
	return !conditions.IsStatusConditionTrue(hcov1beta1.ConditionAvailable) || conditions.IsStatusConditionTrue(hcov1beta1.ConditionDegraded)
}

func isSystemHealthStatusWarning(conditions common.HcoConditions) bool {
	return !conditions.IsStatusConditionTrue(hcov1beta1.ConditionReconcileComplete) || conditions.IsStatusConditionTrue(hcov1beta1.ConditionProgressing)
}

func getNumOfChangesJSONPatch(jsonPatch string) int {
	patches, err := jsonpatch.DecodePatch([]byte(jsonPatch))
	if err != nil {
		return 0
	}
	return len(patches)
}

func getNumericalHealthStatus(status string) float64 {
	healthStatusCodes := map[string]float64{
		systemHealthStatusHealthy: metrics.SystemHealthStatusHealthy,
		systemHealthStatusWarning: metrics.SystemHealthStatusWarning,
		systemHealthStatusError:   metrics.SystemHealthStatusError,
	}

	return healthStatusCodes[status]
}

func (r *ReconcileHyperConverged) firstLoopInitialization(request *common.HcoRequest) {
	// Initialize operand handler.
	r.operandHandler.FirstUseInitiation(r.scheme, hcoutil.GetClusterInfo(), request.Instance)

	// Avoid re-initializing.
	r.firstLoop = false
}

func (r *ReconcileHyperConverged) setOperatorUpgradeableStatus(request *common.HcoRequest) error {
	if hcoutil.GetClusterInfo().IsManagedByOLM() {

		upgradeable := !r.upgradeMode && request.Upgradeable

		request.Logger.Info("setting the Upgradeable operator condition", requestedStatusKey, upgradeable)

		msg := hcoutil.UpgradeableAllowMessage
		status := metav1.ConditionTrue
		reason := hcoutil.UpgradeableAllowReason

		if !upgradeable {
			status = metav1.ConditionFalse

			if r.upgradeMode {
				msg = hcoutil.UpgradeableUpgradingMessage + r.ownVersion
				reason = hcoutil.UpgradeableUpgradingReason
			} else {
				condition, found := request.Conditions.GetCondition(hcov1beta1.ConditionUpgradeable)
				if found && condition.Status == metav1.ConditionFalse {
					reason = condition.Reason
					msg = condition.Message
				}
			}
		}

		if err := r.upgradeableCondition.Set(request.Ctx, status, reason, msg); err != nil {
			request.Logger.Error(err, "can't set the Upgradeable operator condition", requestedStatusKey, upgradeable)
			return err
		}

	}

	return nil
}

func (r *ReconcileHyperConverged) migrateBeforeUpgrade(req *common.HcoRequest) (bool, error) {
	upgradePatched, err := r.applyUpgradePatches(req)
	if err != nil {
		return false, err
	}

	removeOldQuickStartGuides(req, r.client, r.operandHandler.GetQuickStartNames())

	return upgradePatched, nil
}

func (r *ReconcileHyperConverged) applyUpgradePatches(req *common.HcoRequest) (bool, error) {
	modified := false

	knownHcoVersion, _ := GetVersion(&req.Instance.Status, hcoVersionName)
	if knownHcoVersion == "" {
		knownHcoVersion = "0.0.0"
	}
	knownHcoSV, err := semver.ParseTolerant(knownHcoVersion)
	if err != nil {
		req.Logger.Error(err, "Error!")
		return false, err
	}

	tmpInstance, err := upgradepatch.ApplyUpgradePatch(req.Logger, req.Instance, knownHcoSV)
	if err != nil {
		return false, err
	}

	for _, p := range upgradepatch.GetObjectsToBeRemoved() {
		removed, err := r.removeLeftover(req, knownHcoSV, p)
		if err != nil {
			return removed, err
		}
	}

	if !reflect.DeepEqual(tmpInstance.Spec, req.Instance.Spec) {
		req.Logger.Info("updating HCO spec as a result of upgrade patches")
		tmpInstance.Spec.DeepCopyInto(&req.Instance.Spec)
		modified = true
		req.Dirty = true
	}

	return modified, nil
}

func (r *ReconcileHyperConverged) removeLeftover(req *common.HcoRequest, knownHcoSV semver.Version, p upgradepatch.ObjectToBeRemoved) (bool, error) {
	if p.IsAffectedRange(knownHcoSV) {
		removeRelatedObject(req, r.client, p.GroupVersionKind, p.ObjectKey)
		u := &unstructured.Unstructured{}
		u.SetGroupVersionKind(p.GroupVersionKind)
		gerr := r.client.Get(req.Ctx, p.ObjectKey, u)
		if gerr != nil {
			if apierrors.IsNotFound(gerr) {
				return false, nil
			}

			req.Logger.Error(gerr, "failed looking for leftovers", "objectToBeRemoved", p)
			return false, gerr
		}
		return r.deleteObj(req, u, false)

	}
	return false, nil
}

func (r *ReconcileHyperConverged) deleteObj(req *common.HcoRequest, obj client.Object, protectNonHCOObjects bool) (bool, error) {
	removed, err := hcoutil.EnsureDeleted(req.Ctx, r.client, obj, req.Instance.Name, req.Logger, false, false, protectNonHCOObjects)

	if err != nil {
		req.Logger.Error(
			err,
			fmt.Sprintf("failed to delete %s", obj.GetObjectKind().GroupVersionKind().Kind),
			"name",
			obj.GetName(),
		)

		return removed, err
	}

	if removed {
		r.eventEmitter.EmitEvent(
			req.Instance, corev1.EventTypeNormal, "Killing",
			fmt.Sprintf("Removed %s %s", obj.GetName(), obj.GetObjectKind().GroupVersionKind().Kind),
		)
	}

	return removed, nil
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
			if !slices.Contains(requiredQSList, name) {
				req.Logger.Info("deleting ConsoleQuickStart", "name", name)
				if _, err = hcoutil.EnsureDeleted(req.Ctx, cl, &existQs, req.Instance.Name, req.Logger, false, false, true); err != nil {
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
		if obj.Kind == "ConsoleQuickStart" && !slices.Contains(requiredNames, obj.Name) {
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

func removeRelatedObject(req *common.HcoRequest, cl client.Client, gvk schema.GroupVersionKind, objectKey types.NamespacedName) {
	refs := make([]corev1.ObjectReference, 0, len(req.Instance.Status.RelatedObjects))
	foundRO := false

	crdGVK := schema.GroupVersionKind{Group: "apiextensions.k8s.io", Version: "v1", Kind: "CustomResourceDefinition"}

	for _, obj := range req.Instance.Status.RelatedObjects {
		apiVersion, kind := gvk.ToAPIVersionAndKind()
		if obj.APIVersion == apiVersion && obj.Kind == kind && obj.Namespace == objectKey.Namespace && obj.Name == objectKey.Name {
			foundRO = true
			req.Logger.Info("Removed relatedObject entry for", "gvk", gvk, "objectKey", objectKey)
			continue
		}
		if reflect.DeepEqual(gvk, crdGVK) {
			mapping, err := cl.RESTMapper().RESTMapping(obj.GroupVersionKind().GroupKind(), obj.GroupVersionKind().Version)
			if err == nil && mapping != nil && mapping.Resource.GroupResource().String() == objectKey.Name {
				foundRO = true
				req.Logger.Info("Removed relatedObject on CRD removal for", "gvk", gvk, "objectKey", objectKey)
				continue
			}
		}
		refs = append(refs, obj)
	}

	if foundRO {
		req.Instance.Status.RelatedObjects = refs
		req.StatusDirty = true
	}

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

func checkFinalizers(req *common.HcoRequest) bool {
	if req.Instance.DeletionTimestamp.IsZero() {
		// Add the finalizer if it's not there
		if !slices.Contains(req.Instance.Finalizers, FinalizerName) {
			req.Logger.Info("setting a finalizer (with fully qualified name)")
			req.Instance.Finalizers = append(req.Instance.Finalizers, FinalizerName)
			req.Dirty = true
		}
		return true
	}
	return false
}
