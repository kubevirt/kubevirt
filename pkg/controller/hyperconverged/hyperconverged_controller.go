package hyperconverged

import (
	"context"
	"errors"
	"fmt"
	"os"
	"reflect"
	"strings"

	"encoding/json"

	"github.com/go-logr/logr"
	"github.com/operator-framework/operator-sdk/pkg/ready"
	schedulingv1 "k8s.io/api/scheduling/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/record"
	"k8s.io/client-go/tools/reference"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/source"

	sspv1 "github.com/MarSik/kubevirt-ssp-operator/pkg/apis/kubevirt/v1"
	networkaddonsv1alpha1 "github.com/kubevirt/cluster-network-addons-operator/pkg/apis/networkaddonsoperator/v1alpha1"
	networkaddonsnames "github.com/kubevirt/cluster-network-addons-operator/pkg/names"
	hcov1alpha1 "github.com/kubevirt/hyperconverged-cluster-operator/pkg/apis/hco/v1alpha1"
	hcoutil "github.com/kubevirt/hyperconverged-cluster-operator/pkg/util"
	version "github.com/kubevirt/hyperconverged-cluster-operator/version"
	conditionsv1 "github.com/openshift/custom-resource-status/conditions/v1"
	objectreferencesv1 "github.com/openshift/custom-resource-status/objectreferences/v1"
	virtconfig "kubevirt.io/kubevirt/pkg/virt-config"

	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	kubevirtv1 "kubevirt.io/client-go/api/v1"
	"kubevirt.io/client-go/generated/network-attachment-definition-client/clientset/versioned/scheme"
	cdiv1alpha1 "kubevirt.io/containerized-data-importer/pkg/apis/core/v1alpha1"
	logf "sigs.k8s.io/controller-runtime/pkg/runtime/log"
)

var (
	log = logf.Log.WithName("controller_hyperconverged")
)

const (
	// We cannot set owner reference of cluster-wide resources to namespaced HyperConverged object. Therefore,
	// use finalizers to manage the cleanup.
	FinalizerName = "hyperconvergeds.hco.kubevirt.io"

	// UndefinedNamespace is for cluster scoped resources
	UndefinedNamespace string = ""

	// OpenshiftNamespace is for resources that belong in the openshift namespace
	OpenshiftNamespace string = "openshift"

	reconcileInit               = "Init"
	reconcileInitMessage        = "Initializing HyperConverged cluster"
	reconcileFailed             = "ReconcileFailed"
	reconcileCompleted          = "ReconcileCompleted"
	reconcileCompletedMessage   = "Reconcile completed successfully"
	invalidRequestReason        = "InvalidRequest"
	invalidRequestMessageFormat = "Request does not match expected name (%v) and namespace (%v)"
	commonDegradedReason        = "HCODegraded"
	commonProgressingReason     = "HCOProgressing"

	ErrCDIUninstall       = "ErrCDIUninstall"
	uninstallCDIErrorMsg  = "The uninstall request failed on CDI component: "
	ErrVirtUninstall      = "ErrVirtUninstall"
	uninstallVirtErrorMsg = "The uninstall request failed on virt component: "
	ErrHCOUninstall       = "ErrHCOUninstall"
	uninstallHCOErrorMsg  = "The uninstall request failed on dependent components, please check their logs."

	hcoVersionName = "operator"
)

// Add creates a new HyperConverged Controller and adds it to the Manager. The Manager will set fields on the Controller
// and Start it when the Manager is Started.
func Add(mgr manager.Manager) error {
	return add(mgr, newReconciler(mgr))
}

// newReconciler returns a new reconcile.Reconciler
func newReconciler(mgr manager.Manager) reconcile.Reconciler {

	ownVersion := os.Getenv(hcoutil.HcoKvIoVersionName)
	if ownVersion == "" {
		ownVersion = version.Version
	}

	return &ReconcileHyperConverged{
		client:      mgr.GetClient(),
		scheme:      mgr.GetScheme(),
		recorder:    mgr.GetEventRecorderFor(hcov1alpha1.HyperConvergedName),
		upgradeMode: false,
		ownVersion:  ownVersion,
	}
}

// add adds a new Controller to mgr with r as the reconcile.Reconciler
func add(mgr manager.Manager, r reconcile.Reconciler) error {
	// Create a new controller
	c, err := controller.New("hyperconverged-controller", mgr, controller.Options{Reconciler: r})
	if err != nil {
		return err
	}

	// Watch for changes to primary resource HyperConverged
	err = c.Watch(&source.Kind{Type: &hcov1alpha1.HyperConverged{}}, &handler.EnqueueRequestForObject{}, predicate.GenerationChangedPredicate{})
	if err != nil {
		return err
	}

	hco, err := getHyperconverged()
	if err != nil {
		return err
	}

	// Watch secondary resources
	for _, resource := range []runtime.Object{
		&kubevirtv1.KubeVirt{},
		&cdiv1alpha1.CDI{},
		&networkaddonsv1alpha1.NetworkAddonsConfig{},
		&sspv1.KubevirtCommonTemplatesBundle{},
		&sspv1.KubevirtNodeLabellerBundle{},
		&sspv1.KubevirtTemplateValidator{},
		&sspv1.KubevirtMetricsAggregation{},
		&schedulingv1.PriorityClass{},
	} {
		err = c.Watch(&source.Kind{Type: resource}, &handler.EnqueueRequestsFromMapFunc{
			ToRequests: handler.ToRequestsFunc(
				// always enqueue the same HyperConverged object, since there should be only one
				func(a handler.MapObject) []reconcile.Request {
					return []reconcile.Request{
						{NamespacedName: hco},
					}
				}),
		})
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
	client      client.Client
	scheme      *runtime.Scheme
	recorder    record.EventRecorder
	upgradeMode bool
	ownVersion  string
}

type hcoRequest struct {
	reconcile.Request
	logger     logr.Logger
	conditions hcoConditions
	ctx        context.Context
	instance   *hcov1alpha1.HyperConverged
}

// Reconcile reads that state of the cluster for a HyperConverged object and makes changes based on the state read
// and what is in the HyperConverged.Spec
// Note:
// The Controller will requeue the Request to be processed again if the returned error is non-nil or
// Result.Requeue is true, otherwise upon completion it will remove the work from the queue.
func (r *ReconcileHyperConverged) Reconcile(request reconcile.Request) (reconcile.Result, error) {

	req := &hcoRequest{
		Request:    request,
		logger:     log.WithValues("Request.Namespace", request.Namespace, "Request.Name", request.Name),
		conditions: newHcoConditions(),
		ctx:        context.TODO(),
	}

	req.logger.Info("Reconciling HyperConverged operator")

	// Fetch the HyperConverged instance
	instance, err := r.getHcoInstanceFromK8s(req)
	if instance == nil {
		return reconcile.Result{}, err
	}
	req.instance = instance

	valid, err := r.validateNamespace(req)
	if !valid {
		return reconcile.Result{}, err
	}
	// Add conditions if there are none
	init := req.instance.Status.Conditions == nil
	if init {
		err = r.setInitialConditions(req)
		if err != nil {
			req.logger.Error(err, "Failed to add conditions to status")
			return reconcile.Result{}, err
		}
	}

	// in-memory conditions should start off empty. It will only ever hold
	// negative conditions (!Available, Degraded, Progressing)
	req.conditions = newHcoConditions()

	// Handle finalizers
	if req.instance.ObjectMeta.DeletionTimestamp.IsZero() {
		// Add the finalizer if it's not there
		if !contains(req.instance.ObjectMeta.Finalizers, FinalizerName) {
			req.instance.ObjectMeta.Finalizers = append(req.instance.ObjectMeta.Finalizers, FinalizerName)
			err = r.client.Update(req.ctx, req.instance)
		}
	} else {
		if contains(req.instance.ObjectMeta.Finalizers, FinalizerName) {
			return r.ensureHcoDeleted(req)
		}
	}

	// If the current version is not updated in CR ,then we're updating. This is also works when updating from
	// an old version, since Status.Versions will be empty.
	knownHcoVersion, _ := req.instance.Status.GetVersion(hcoVersionName)

	if !r.upgradeMode && !init && knownHcoVersion != r.ownVersion {
		r.upgradeMode = true
		req.logger.Info(fmt.Sprintf("Start upgrating from version %s to version %s", knownHcoVersion, r.ownVersion))
	}

	err = r.ensureHco(req)
	if err != nil {
		return reconcile.Result{}, r.updateConditions(req)
	}

	req.logger.Info("Reconcile complete")

	// Requeue if we just created everything
	if init {
		return reconcile.Result{Requeue: true}, err
	}

	err = r.completeReconciliation(req)

	return reconcile.Result{}, err
}

func (r *ReconcileHyperConverged) getHcoInstanceFromK8s(req *hcoRequest) (*hcov1alpha1.HyperConverged, error) {
	instance := &hcov1alpha1.HyperConverged{}
	err := r.client.Get(req.ctx, req.NamespacedName, instance)
	if err != nil {
		if apierrors.IsNotFound(err) {
			req.logger.Info("No HyperConverged resource")
			// Request object not found, could have been deleted after reconcile request.
			// Owned objects are automatically garbage collected. For additional cleanup logic use finalizers.
			// Return and don't requeue
			return nil, nil
		}
		// Error reading the object - requeue the request.
		return nil, err
	}
	return instance, nil
}

func (r *ReconcileHyperConverged) validateNamespace(req *hcoRequest) (bool, error) {
	hco, err := getHyperconverged()
	if err != nil {
		req.logger.Error(err, "Failed to get HyperConverged namespaced name")
		return false, err
	}

	// Ignore invalid requests
	if req.NamespacedName != hco {
		req.logger.Info("Invalid request", "HyperConverged.Namespace", hco.Namespace, "HyperConverged.Name", hco.Name)
		req.conditions.setStatusCondition(conditionsv1.Condition{
			Type:    hcov1alpha1.ConditionReconcileComplete,
			Status:  corev1.ConditionFalse,
			Reason:  invalidRequestReason,
			Message: fmt.Sprintf(invalidRequestMessageFormat, hco.Name, hco.Namespace),
		})
		err := r.updateConditions(req)
		return false, err
	}
	return true, nil
}

func (r *ReconcileHyperConverged) setInitialConditions(req *hcoRequest) error {
	req.instance.Status.UpdateVersion(hcoVersionName, r.ownVersion)

	req.conditions.setStatusCondition(conditionsv1.Condition{
		Type:    hcov1alpha1.ConditionReconcileComplete,
		Status:  corev1.ConditionUnknown, // we just started trying to reconcile
		Reason:  reconcileInit,
		Message: reconcileInitMessage,
	})
	req.conditions.setStatusCondition(conditionsv1.Condition{
		Type:    conditionsv1.ConditionAvailable,
		Status:  corev1.ConditionFalse,
		Reason:  reconcileInit,
		Message: reconcileInitMessage,
	})
	req.conditions.setStatusCondition(conditionsv1.Condition{
		Type:    conditionsv1.ConditionProgressing,
		Status:  corev1.ConditionTrue,
		Reason:  reconcileInit,
		Message: reconcileInitMessage,
	})
	req.conditions.setStatusCondition(conditionsv1.Condition{
		Type:    conditionsv1.ConditionDegraded,
		Status:  corev1.ConditionFalse,
		Reason:  reconcileInit,
		Message: reconcileInitMessage,
	})
	req.conditions.setStatusCondition(conditionsv1.Condition{
		Type:    conditionsv1.ConditionUpgradeable,
		Status:  corev1.ConditionUnknown,
		Reason:  reconcileInit,
		Message: reconcileInitMessage,
	})

	return r.updateConditions(req)
}

func (r *ReconcileHyperConverged) ensureHcoDeleted(req *hcoRequest) (reconcile.Result, error) {
	for i, f := range []func(c client.Client, req *hcoRequest) error{
		// Keep ensuring that we are trying to delete
		// protected resources first
		ensureKubeVirtDeleted,
		ensureCDIDeleted,
		ensureNetworkAddonsDeleted,
		ensureKubeVirtCommonTemplateBundleDeleted,
		ensureKubeVirtPriorityClassDeleted,
	} {
		err := f(r.client, req)
		if err != nil {
			req.logger.Error(err, "Failed to manually delete objects")

			// TODO: ask to other components to expose something like
			// func IsDeleteRefused(err error) bool
			// to be able to clearly distinguish between an explicit
			// refuse from other operator and any other kind of error that
			// could potentially happen in the process
			errT := ErrHCOUninstall
			errMsg := uninstallHCOErrorMsg
			switch i {
			case 0:
				errT = ErrVirtUninstall
				errMsg = uninstallVirtErrorMsg + err.Error()
			case 1:
				errT = ErrCDIUninstall
				errMsg = uninstallCDIErrorMsg + err.Error()
			}

			errE := r.emitEvent(req.instance, req.logger, corev1.EventTypeWarning, errT, errMsg)
			if errE != nil {
				req.logger.Error(errE, "Failed emitting uninstall error event")
			}

			// TODO: implement a validating webhook to try to delete virt and CDI CRs
			// in dry run mode before really accepting the deletion request.
			// This event should still stay here because no strategy can ensure we are
			// 100% race conditions free

			return reconcile.Result{}, err
		}
	}

	// Remove the finalizer
	req.instance.ObjectMeta.Finalizers = drop(req.instance.ObjectMeta.Finalizers, FinalizerName)
	updateErr := r.client.Update(req.ctx, req.instance)

	// Need to requeue because finalizer update does not change metadata.generation
	return reconcile.Result{Requeue: true}, updateErr
}

func (r *ReconcileHyperConverged) emitEvent(instance *hcov1alpha1.HyperConverged, logger logr.Logger, kind string, errT string, errMsg string) error {
	r.recorder.Event(instance, kind, errT, errMsg)

	pod, pod_err := hcoutil.GetPod(r.client, logger)
	if pod_err != nil {
		if logger != nil {
			logger.Error(pod_err, "Failed to identify HCO POD, emitting warning event only on hyperconverged instance")
		}
		return pod_err
	}

	r.recorder.Event(pod, kind, errT, errMsg)

	csv, csv_err := hcoutil.GetCSVfromPod(pod, r.client, logger)
	if csv_err != nil {
		if logger != nil {
			logger.Error(csv_err, "Failed to identify HCO CSV, emitting warning event only on HCO pod and hyperconverged instance")
		}
		return csv_err
	}

	r.recorder.Event(csv, kind, errT, errMsg)
	return nil

}

func (r *ReconcileHyperConverged) ensureHco(req *hcoRequest) error {
	for _, f := range []func(*hcoRequest) error{
		r.ensureKubeVirtPriorityClass,
		r.ensureKubeVirtConfig,
		r.ensureKubeVirtStorageConfig,
		r.ensureKubeVirt,
		r.ensureCDI,
		r.ensureNetworkAddons,
		r.ensureKubeVirtCommonTemplateBundle,
		r.ensureKubeVirtNodeLabellerBundle,
		r.ensureKubeVirtTemplateValidator,
		r.ensureKubeVirtMetricsAggregation,
		r.ensureIMSConfig,
	} {
		err := f(req)
		if err != nil {
			req.conditions.setStatusCondition(conditionsv1.Condition{
				Type:    hcov1alpha1.ConditionReconcileComplete,
				Status:  corev1.ConditionFalse,
				Reason:  reconcileFailed,
				Message: fmt.Sprintf("Error while reconciling: %v", err),
			})
			return err
		}
	}
	return nil
}

func (r *ReconcileHyperConverged) aggregateComponentConditions(req *hcoRequest) bool {
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
	allComponentsAreUp := req.conditions.empty()
	req.conditions.setStatusCondition(conditionsv1.Condition{
		Type:    hcov1alpha1.ConditionReconcileComplete,
		Status:  corev1.ConditionTrue,
		Reason:  reconcileCompleted,
		Message: reconcileCompletedMessage,
	})

	if _, conditionFound := req.conditions[conditionsv1.ConditionDegraded]; conditionFound { // (#chart 1)
		if _, conditionFound = req.conditions[conditionsv1.ConditionProgressing]; !conditionFound { // (#chart 2)
			req.conditions.setStatusCondition(conditionsv1.Condition{ // (#chart 3)
				Type:    conditionsv1.ConditionProgressing,
				Status:  corev1.ConditionFalse,
				Reason:  reconcileCompleted,
				Message: reconcileCompletedMessage,
			})
		} // else - Progressing is already exists

		if _, conditionFound = req.conditions[conditionsv1.ConditionUpgradeable]; !conditionFound { // (#chart 4)
			req.conditions.setStatusCondition(conditionsv1.Condition{ // (#chart 5)
				Type:    conditionsv1.ConditionUpgradeable,
				Status:  corev1.ConditionFalse,
				Reason:  commonDegradedReason,
				Message: "HCO is not Upgradeable due to degraded components",
			})
		} // else - Upgradeable is already exists
		if _, conditionFound = req.conditions[conditionsv1.ConditionAvailable]; !conditionFound { // (#chart 6)
			req.conditions.setStatusCondition(conditionsv1.Condition{ // (#chart 7)
				Type:    conditionsv1.ConditionAvailable,
				Status:  corev1.ConditionFalse,
				Reason:  commonDegradedReason,
				Message: "HCO is not available due to degraded components",
			})
		} // else - Available is already exists
	} else {
		// Degraded is not found. add it.
		req.conditions.setStatusCondition(conditionsv1.Condition{ // (#chart 8)
			Type:    conditionsv1.ConditionDegraded,
			Status:  corev1.ConditionFalse,
			Reason:  reconcileCompleted,
			Message: reconcileCompletedMessage,
		})

		if _, conditionFound = req.conditions[conditionsv1.ConditionProgressing]; conditionFound { // (#chart 9)

			if _, conditionFound = req.conditions[conditionsv1.ConditionUpgradeable]; !conditionFound { // (#chart 10)
				req.conditions.setStatusCondition(conditionsv1.Condition{ // (#chart 11)
					Type:    conditionsv1.ConditionUpgradeable,
					Status:  corev1.ConditionFalse,
					Reason:  commonProgressingReason,
					Message: "HCO is not Upgradeable due to progressing components",
				})
			} // else - Upgradeable is already exists

			if _, conditionFound = req.conditions[conditionsv1.ConditionAvailable]; !conditionFound { // (#chart 12)
				req.conditions.setStatusCondition(conditionsv1.Condition{ // (#chart 13)
					Type:    conditionsv1.ConditionAvailable,
					Status:  corev1.ConditionTrue,
					Reason:  reconcileCompleted,
					Message: reconcileCompletedMessage,
				})
			} // else - Available is already exists
		} else {
			req.conditions.setStatusCondition(conditionsv1.Condition{ // (#chart 14)
				Type:    conditionsv1.ConditionProgressing,
				Status:  corev1.ConditionFalse,
				Reason:  reconcileCompleted,
				Message: reconcileCompletedMessage,
			})

			if _, conditionFound = req.conditions[conditionsv1.ConditionUpgradeable]; !conditionFound { // (#chart 15)
				req.conditions.setStatusCondition(conditionsv1.Condition{ // (#chart 16)
					Type:    conditionsv1.ConditionUpgradeable,
					Status:  corev1.ConditionTrue,
					Reason:  reconcileCompleted,
					Message: reconcileCompletedMessage,
				})
			}

			if _, conditionFound = req.conditions[conditionsv1.ConditionAvailable]; !conditionFound { // (#chart 17) {
				req.conditions.setStatusCondition(conditionsv1.Condition{ // (#chart 18)
					Type:    conditionsv1.ConditionAvailable,
					Status:  corev1.ConditionTrue,
					Reason:  reconcileCompleted,
					Message: reconcileCompletedMessage,
				})
			}
		}
	}
	return allComponentsAreUp
}

func (r *ReconcileHyperConverged) completeReconciliation(req *hcoRequest) error {
	allComponentsAreUp := r.aggregateComponentConditions(req)
	if allComponentsAreUp {
		req.logger.Info("No component operator reported negatively")

		if r.upgradeMode { // update the new image only when upgrade is completed
			req.instance.Status.UpdateVersion(hcoVersionName, r.ownVersion)
			r.upgradeMode = false
			req.logger.Info(fmt.Sprintf("Successfuly upgraded to version %s", r.ownVersion))
		}

		// If no operator whose conditions we are watching reports an error, then it is safe
		// to set readiness.
		r := ready.NewFileReady()
		err := r.Set()
		if err != nil {
			req.logger.Error(err, "Failed to mark operator ready")
			return err
		}
	} else if cond, conditionFound := req.conditions[conditionsv1.ConditionUpgradeable]; conditionFound && cond.Status == corev1.ConditionFalse {
		// If for any reason we marked ourselves !upgradeable...then unset readiness
		r := ready.NewFileReady()
		err := r.Unset()
		if err != nil {
			req.logger.Error(err, "Failed to mark operator unready")
			return err
		}
	}

	return r.updateConditions(req)
}

func newKubeVirtConfigForCR(cr *hcov1alpha1.HyperConverged, namespace string) *corev1.ConfigMap {
	labels := map[string]string{
		"app": cr.Name,
	}
	cm := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "kubevirt-config",
			Labels:    labels,
			Namespace: namespace,
		},
		Data: map[string]string{
			"feature-gates":       "DataVolumes,SRIOV,LiveMigration,CPUManager,CPUNodeDiscovery,Sidecar",
			"migrations":          `{"nodeDrainTaintKey" : "node.kubernetes.io/unschedulable"}`,
			"selinuxLauncherType": "spc_t",
		},
	}
	val, ok := os.LookupEnv("SMBIOS")
	if ok && val != "" {
		cm.Data[virtconfig.SmbiosConfigKey] = val
	}
	val, ok = os.LookupEnv("MACHINETYPE")
	if ok && val != "" {
		cm.Data[virtconfig.MachineTypeKey] = val
	}
	return cm
}

func (r *ReconcileHyperConverged) ensureKubeVirtConfig(req *hcoRequest) error {
	kubevirtConfig := newKubeVirtConfigForCR(req.instance, req.Namespace)
	if err := controllerutil.SetControllerReference(req.instance, kubevirtConfig, r.scheme); err != nil {
		return err
	}

	key, err := client.ObjectKeyFromObject(kubevirtConfig)
	if err != nil {
		req.logger.Error(err, "Failed to get object key for kubevirt config")
	}

	found := &corev1.ConfigMap{}
	err = r.client.Get(req.ctx, key, found)
	if err != nil && apierrors.IsNotFound(err) {
		req.logger.Info("Creating kubevirt config")
		return r.client.Create(req.ctx, kubevirtConfig)
	}

	if err != nil {
		return err
	}

	req.logger.Info("KubeVirt config already exists", "KubeVirtConfig.Namespace", found.Namespace, "KubeVirtConfig.Name", found.Name)
	// Add it to the list of RelatedObjects if found
	objectRef, err := reference.GetReference(r.scheme, found)
	if err != nil {
		return err
	}
	objectreferencesv1.SetObjectReference(&req.instance.Status.RelatedObjects, *objectRef)

	if !reflect.DeepEqual(found.Data, kubevirtConfig.Data) {
		req.logger.Info("Updating existing KubeVirt config")
		found.Data = kubevirtConfig.Data
		err = r.client.Update(req.ctx, found)
		if err != nil {
			req.logger.Error(err, "Failed updating an existing kubevirt config")
			return err
		}
	}

	return r.client.Status().Update(req.ctx, req.instance)
}

// newKubeVirtForCR returns a KubeVirt CR
func newKubeVirtForCR(cr *hcov1alpha1.HyperConverged, namespace string) *kubevirtv1.KubeVirt {
	labels := map[string]string{
		"app": cr.Name,
	}
	return &kubevirtv1.KubeVirt{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "kubevirt-" + cr.Name,
			Labels:    labels,
			Namespace: namespace,
		},
		Spec: kubevirtv1.KubeVirtSpec{
			UninstallStrategy: kubevirtv1.KubeVirtUninstallStrategyBlockUninstallIfWorkloadsExist,
		},
	}
}

func newKubeVirtPriorityClass() *schedulingv1.PriorityClass {
	return &schedulingv1.PriorityClass{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "scheduling.k8s.io/v1",
			Kind:       "PriorityClass",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: "kubevirt-cluster-critical",
		},
		// 1 billion is the highest value we can set
		// https://kubernetes.io/docs/concepts/configuration/pod-priority-preemption/#priorityclass
		Value:         1000000000,
		GlobalDefault: false,
		Description:   "This priority class should be used for KubeVirt core components only.",
	}
}

func (r *ReconcileHyperConverged) ensureKubeVirtPriorityClass(req *hcoRequest) error {
	req.logger.Info("Reconciling KubeVirt PriorityClass")
	pc := newKubeVirtPriorityClass()

	key, err := client.ObjectKeyFromObject(pc)
	if err != nil {
		req.logger.Error(err, "Failed to get object key for KubeVirt PriorityClass")
		return err
	}

	found := &schedulingv1.PriorityClass{}
	err = r.client.Get(req.ctx, key, found)

	if err != nil {
		if apierrors.IsNotFound(err) {
			// create the new object
			return r.client.Create(req.ctx, pc, &client.CreateOptions{})
		}
		return err
	}

	// at this point we found the object in the cache and we check if something was changed
	if pc.Name == found.Name && pc.Value == found.Value && pc.Description == found.Description {
		req.logger.Info("KubeVirt PriorityClass already exists", "PriorityClass.Name", pc.Name)
		objectRef, err := reference.GetReference(scheme.Scheme, found)
		if err != nil {
			req.logger.Error(err, "failed getting object reference for found object")
			return err
		}
		objectreferencesv1.SetObjectReference(&req.instance.Status.RelatedObjects, *objectRef)
		return nil
	}

	// something was changed but since we can't patch a priority class object, we remove it
	err = r.client.Delete(req.ctx, found, &client.DeleteOptions{})
	if err != nil {
		return err
	}

	// create the new object
	return r.client.Create(req.ctx, pc, &client.CreateOptions{})
}

func (r *ReconcileHyperConverged) ensureKubeVirt(req *hcoRequest) error {
	virt := newKubeVirtForCR(req.instance, req.Namespace)
	if err := controllerutil.SetControllerReference(req.instance, virt, r.scheme); err != nil {
		return err
	}

	key, err := client.ObjectKeyFromObject(virt)
	if err != nil {
		req.logger.Error(err, "Failed to get object key for KubeVirt")
	}

	found := &kubevirtv1.KubeVirt{}
	err = r.client.Get(req.ctx, key, found)
	if err != nil && apierrors.IsNotFound(err) {
		req.logger.Info("Creating kubevirt")
		return r.client.Create(req.ctx, virt)
	}

	if err != nil {
		return err
	}

	req.logger.Info("KubeVirt already exists", "KubeVirt.Namespace", found.Namespace, "KubeVirt.Name", found.Name)

	if !reflect.DeepEqual(found.Spec, virt.Spec) {
		if found.Spec.UninstallStrategy == "" {
			req.logger.Info("Updating UninstallStrategy on existing KubeVirt to its default value")
			found.Spec.UninstallStrategy = virt.Spec.UninstallStrategy
		}
		return r.client.Update(req.ctx, found)
	}

	// Add it to the list of RelatedObjects if found
	objectRef, err := reference.GetReference(r.scheme, found)
	if err != nil {
		return err
	}
	objectreferencesv1.SetObjectReference(&req.instance.Status.RelatedObjects, *objectRef)

	// Handle KubeVirt resource conditions
	handleComponentConditions(r, req, "KubeVirt", translateKubeVirtConds(found.Status.Conditions))

	return r.client.Status().Update(req.ctx, req.instance)
}

// newCDIForCr returns a CDI CR
func newCDIForCR(cr *hcov1alpha1.HyperConverged, namespace string) *cdiv1alpha1.CDI {
	labels := map[string]string{
		"app": cr.Name,
	}
	uninstallStrategy := cdiv1alpha1.CDIUninstallStrategyBlockUninstallIfWorkloadsExist
	return &cdiv1alpha1.CDI{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "cdi-" + cr.Name,
			Labels:    labels,
			Namespace: namespace,
		},
		Spec: cdiv1alpha1.CDISpec{
			UninstallStrategy: &uninstallStrategy,
		},
	}
}

func (r *ReconcileHyperConverged) ensureCDI(req *hcoRequest) error {
	cdi := newCDIForCR(req.instance, UndefinedNamespace)

	key, err := client.ObjectKeyFromObject(cdi)
	if err != nil {
		req.logger.Error(err, "Failed to get object key for CDI")
	}

	found := &cdiv1alpha1.CDI{}
	err = r.client.Get(req.ctx, key, found)

	if err != nil && apierrors.IsNotFound(err) {
		req.logger.Info("Creating CDI")
		return r.client.Create(req.ctx, cdi)
	}

	if err != nil {
		return err
	}

	req.logger.Info("CDI already exists", "CDI.Namespace", found.Namespace, "CDI.Name", found.Name)

	existingOwners := found.GetOwnerReferences()

	// Previous versions used to have HCO-operator (scope namespace)
	// as the owner of CDI (scope cluster).
	// It's not legal, so remove that.
	if len(existingOwners) > 0 {
		req.logger.Info("CDI has owners, removing...")
		found.SetOwnerReferences([]metav1.OwnerReference{})
		err = r.client.Update(req.ctx, found)
		if err != nil {
			req.logger.Error(err, "Failed to remove CDI's previous owners")
		}
	}

	if !reflect.DeepEqual(found.Spec, cdi.Spec) {
		if found.Spec.UninstallStrategy == nil {
			req.logger.Info("Updating UninstallStrategy on existing CDI to its default value")
			defaultUninstallStrategy := cdiv1alpha1.CDIUninstallStrategyBlockUninstallIfWorkloadsExist
			found.Spec.UninstallStrategy = &defaultUninstallStrategy
		}
		return r.client.Update(req.ctx, found)
	}

	// Add it to the list of RelatedObjects if found
	objectRef, err := reference.GetReference(r.scheme, found)
	if err != nil {
		return err
	}
	objectreferencesv1.SetObjectReference(&req.instance.Status.RelatedObjects, *objectRef)

	// Handle CDI resource conditions
	handleComponentConditions(r, req, "CDI", found.Status.Conditions)

	return r.client.Status().Update(req.ctx, req.instance)
}

// newNetworkAddonsForCR returns a NetworkAddonsConfig CR
func newNetworkAddonsForCR(cr *hcov1alpha1.HyperConverged, namespace string) *networkaddonsv1alpha1.NetworkAddonsConfig {
	labels := map[string]string{
		"app": cr.Name,
	}
	return &networkaddonsv1alpha1.NetworkAddonsConfig{
		ObjectMeta: metav1.ObjectMeta{
			Name:      networkaddonsnames.OPERATOR_CONFIG,
			Labels:    labels,
			Namespace: namespace,
		},
		Spec: networkaddonsv1alpha1.NetworkAddonsConfigSpec{
			Multus:      &networkaddonsv1alpha1.Multus{},
			LinuxBridge: &networkaddonsv1alpha1.LinuxBridge{},
			Ovs:         &networkaddonsv1alpha1.Ovs{},
			NMState:     &networkaddonsv1alpha1.NMState{},
		},
	}
}

func (r *ReconcileHyperConverged) ensureNetworkAddons(req *hcoRequest) error {
	networkAddons := newNetworkAddonsForCR(req.instance, UndefinedNamespace)

	key, err := client.ObjectKeyFromObject(networkAddons)
	if err != nil {
		req.logger.Error(err, "Failed to get object key for Network Addons")
	}

	found := &networkaddonsv1alpha1.NetworkAddonsConfig{}
	err = r.client.Get(req.ctx, key, found)

	if err != nil && apierrors.IsNotFound(err) {
		req.logger.Info("Creating Network Addons")
		return r.client.Create(req.ctx, networkAddons)
	} else if err != nil {
		return err
	}

	existingOwners := found.GetOwnerReferences()

	// Previous versions used to have HCO-operator (scope namespace)
	// as the owner of NetworkAddons (scope cluster).
	// It's not legal, so remove that.
	if len(existingOwners) > 0 {
		req.logger.Info("NetworkAddons has owners, removing...")
		found.SetOwnerReferences([]metav1.OwnerReference{})
		err = r.client.Update(req.ctx, found)
		if err != nil {
			req.logger.Error(err, "Failed to remove NetworkAddons' previous owners")
		}
	}

	if !reflect.DeepEqual(found.Spec, networkAddons.Spec) {
		req.logger.Info("Updating existing Network Addons")
		found.Spec = networkAddons.Spec
		return r.client.Update(req.ctx, found)
	}

	req.logger.Info("NetworkAddonsConfig already exists", "NetworkAddonsConfig.Namespace", found.Namespace, "NetworkAddonsConfig.Name", found.Name)

	// Add it to the list of RelatedObjects if found
	objectRef, err := reference.GetReference(r.scheme, found)
	if err != nil {
		return err
	}
	objectreferencesv1.SetObjectReference(&req.instance.Status.RelatedObjects, *objectRef)

	// Handle conditions
	handleComponentConditions(r, req, "NetworkAddonsConfig", found.Status.Conditions)

	return r.client.Status().Update(req.ctx, req.instance)
}

func handleComponentConditions(r *ReconcileHyperConverged, req *hcoRequest, component string, componentConds []conditionsv1.Condition) {
	if len(componentConds) == 0 {
		reason := fmt.Sprintf("%sConditions", component)
		message := fmt.Sprintf("%s resource has no conditions", component)
		req.logger.Info(fmt.Sprintf("%s's resource is not reporting Conditions on it's Status", component))
		req.conditions.setStatusCondition(conditionsv1.Condition{
			Type:    conditionsv1.ConditionAvailable,
			Status:  corev1.ConditionFalse,
			Reason:  reason,
			Message: message,
		})
		req.conditions.setStatusCondition(conditionsv1.Condition{
			Type:    conditionsv1.ConditionProgressing,
			Status:  corev1.ConditionTrue,
			Reason:  reason,
			Message: message,
		})
		req.conditions.setStatusCondition(conditionsv1.Condition{
			Type:    conditionsv1.ConditionUpgradeable,
			Status:  corev1.ConditionFalse,
			Reason:  reason,
			Message: message,
		})
	} else {
		foundAvailableCond := false
		for _, condition := range componentConds {
			switch condition.Type {
			case conditionsv1.ConditionAvailable:
				foundAvailableCond = true
				if condition.Status == corev1.ConditionFalse {
					msg := fmt.Sprintf("%s is not available: %v", component, string(condition.Message))
					r.componentNotAvailable(req, component, msg)
				}
			case conditionsv1.ConditionProgressing:
				if condition.Status == corev1.ConditionTrue {
					req.logger.Info(fmt.Sprintf("%s is 'Progressing'", component))
					req.conditions.setStatusCondition(conditionsv1.Condition{
						Type:    conditionsv1.ConditionProgressing,
						Status:  corev1.ConditionTrue,
						Reason:  fmt.Sprintf("%sProgressing", component),
						Message: fmt.Sprintf("%s is progressing: %v", component, string(condition.Message)),
					})
					req.conditions.setStatusCondition(conditionsv1.Condition{
						Type:    conditionsv1.ConditionUpgradeable,
						Status:  corev1.ConditionFalse,
						Reason:  fmt.Sprintf("%sProgressing", component),
						Message: fmt.Sprintf("%s is progressing: %v", component, string(condition.Message)),
					})
				}
			case conditionsv1.ConditionDegraded:
				if condition.Status == corev1.ConditionTrue {
					req.logger.Info(fmt.Sprintf("%s is 'Degraded'", component))
					req.conditions.setStatusCondition(conditionsv1.Condition{
						Type:    conditionsv1.ConditionDegraded,
						Status:  corev1.ConditionTrue,
						Reason:  fmt.Sprintf("%sDegraded", component),
						Message: fmt.Sprintf("%s is degraded: %v", component, string(condition.Message)),
					})
				}
			}
		}

		if !foundAvailableCond {
			r.componentNotAvailable(req, component, `missing "Available" condition`)
		}
	}
}

func (r *ReconcileHyperConverged) componentNotAvailable(req *hcoRequest, component string, msg string) {
	req.logger.Info(fmt.Sprintf("%s is not 'Available'", component))
	req.conditions.setStatusCondition(conditionsv1.Condition{
		Type:    conditionsv1.ConditionAvailable,
		Status:  corev1.ConditionFalse,
		Reason:  fmt.Sprintf("%sNotAvailable", component),
		Message: msg,
	})
}

func newKubeVirtCommonTemplateBundleForCR(cr *hcov1alpha1.HyperConverged, namespace string) *sspv1.KubevirtCommonTemplatesBundle {
	labels := map[string]string{
		"app": cr.Name,
	}
	return &sspv1.KubevirtCommonTemplatesBundle{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "common-templates-" + cr.Name,
			Labels:    labels,
			Namespace: OpenshiftNamespace,
		},
	}
}

func (r *ReconcileHyperConverged) ensureKubeVirtCommonTemplateBundle(req *hcoRequest) error {
	kvCTB := newKubeVirtCommonTemplateBundleForCR(req.instance, OpenshiftNamespace)

	key, err := client.ObjectKeyFromObject(kvCTB)
	if err != nil {
		req.logger.Error(err, "Failed to get object key for KubeVirt Common Templates Bundle")
	}

	found := &sspv1.KubevirtCommonTemplatesBundle{}
	err = r.client.Get(req.ctx, key, found)
	if err != nil && apierrors.IsNotFound(err) {
		req.logger.Info("Creating KubeVirt Common Templates Bundle")
		return r.client.Create(req.ctx, kvCTB)
	}

	if err != nil {
		return err
	}

	existingOwners := found.GetOwnerReferences()

	// Previous versions used to have HCO-operator (namespace: kubevirt-hyperconverged)
	// as the owner of kvCTB (namespace: OpenshiftNamespace).
	// It's not legal, so remove that.
	if len(existingOwners) > 0 {
		req.logger.Info("kvCTB has owners, removing...")
		found.SetOwnerReferences([]metav1.OwnerReference{})
		err = r.client.Update(req.ctx, found)
		if err != nil {
			req.logger.Error(err, "Failed to remove kvCTB's previous owners")
		}
	}

	req.logger.Info("KubeVirt Common Templates Bundle already exists", "bundle.Namespace", found.Namespace, "bundle.Name", found.Name)

	// Add it to the list of RelatedObjects if found
	objectRef, err := reference.GetReference(r.scheme, found)
	if err != nil {
		return err
	}
	objectreferencesv1.SetObjectReference(&req.instance.Status.RelatedObjects, *objectRef)

	// TODO: temporary avoid checking conditions on KubevirtCommonTemplatesBundle because it's currently
	// broken on k8s. Revert this when we will be able to fix it
	// handleComponentConditions(r, req, "KubevirtCommonTemplatesBundle", found.Status.Conditions)
	return r.client.Status().Update(req.ctx, req.instance)
}

func newKubeVirtNodeLabellerBundleForCR(cr *hcov1alpha1.HyperConverged, namespace string) *sspv1.KubevirtNodeLabellerBundle {
	labels := map[string]string{
		"app": cr.Name,
	}
	return &sspv1.KubevirtNodeLabellerBundle{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "node-labeller-" + cr.Name,
			Labels:    labels,
			Namespace: namespace,
		},
		Spec: sspv1.ComponentSpec{
			UseKVM: isKVMAvailable(),
		},
	}
}

func (r *ReconcileHyperConverged) ensureKubeVirtNodeLabellerBundle(req *hcoRequest) error {
	kvNLB := newKubeVirtNodeLabellerBundleForCR(req.instance, req.Namespace)
	if err := controllerutil.SetControllerReference(req.instance, kvNLB, r.scheme); err != nil {
		return err
	}

	key, err := client.ObjectKeyFromObject(kvNLB)
	if err != nil {
		req.logger.Error(err, "Failed to get object key for KubeVirt Node Labeller Bundle")
	}

	found := &sspv1.KubevirtNodeLabellerBundle{}
	err = r.client.Get(req.ctx, key, found)
	if err != nil && apierrors.IsNotFound(err) {
		req.logger.Info("Creating KubeVirt Node Labeller Bundle")
		return r.client.Create(req.ctx, kvNLB)
	}

	if err != nil {
		return err
	}

	req.logger.Info("KubeVirt Node Labeller Bundle already exists", "bundle.Namespace", found.Namespace, "bundle.Name", found.Name)

	// Add it to the list of RelatedObjects if found
	objectRef, err := reference.GetReference(r.scheme, found)
	if err != nil {
		return err
	}
	objectreferencesv1.SetObjectReference(&req.instance.Status.RelatedObjects, *objectRef)

	// TODO: temporary avoid checking conditions on KubevirtNodeLabellerBundle because it's currently
	// broken on k8s. Revert this when we will be able to fix it
	//handleComponentConditions(r, req, "KubevirtNodeLabellerBundle", found.Status.Conditions)
	return r.client.Status().Update(req.ctx, req.instance)
}

func newIMSConfigForCR(cr *hcov1alpha1.HyperConverged, namespace string) *corev1.ConfigMap {
	labels := map[string]string{
		"app": cr.Name,
	}
	return &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "v2v-vmware",
			Labels:    labels,
			Namespace: namespace,
		},
		Data: map[string]string{
			"v2v-conversion-image":              os.Getenv("CONVERSION_CONTAINER"),
			"kubevirt-vmware-image":             os.Getenv("VMWARE_CONTAINER"),
			"kubevirt-vmware-image-pull-policy": "IfNotPresent",
		},
	}
}

func (r *ReconcileHyperConverged) ensureIMSConfig(req *hcoRequest) error {
	if os.Getenv("CONVERSION_CONTAINER") == "" {
		return errors.New("ims-conversion-container not specified")
	}

	if os.Getenv("VMWARE_CONTAINER") == "" {
		return errors.New("ims-vmware-container not specified")
	}

	imsConfig := newIMSConfigForCR(req.instance, req.Namespace)
	if err := controllerutil.SetControllerReference(req.instance, imsConfig, r.scheme); err != nil {
		return err
	}

	key, err := client.ObjectKeyFromObject(imsConfig)
	if err != nil {
		req.logger.Error(err, "Failed to get object key for IMS Configmap")
	}

	found := &corev1.ConfigMap{}
	err = r.client.Get(req.ctx, key, found)
	if err != nil && apierrors.IsNotFound(err) {
		req.logger.Info("Creating IMS Configmap")
		return r.client.Create(req.ctx, imsConfig)
	}

	if err != nil {
		return err
	}

	req.logger.Info("IMS Configmap already exists", "imsConfigMap.Namespace", found.Namespace, "imsConfigMap.Name", found.Name)

	// Add it to the list of RelatedObjects if found
	objectRef, err := reference.GetReference(r.scheme, found)
	if err != nil {
		return err
	}
	objectreferencesv1.SetObjectReference(&req.instance.Status.RelatedObjects, *objectRef)

	// TODO: Handle conditions
	return r.client.Status().Update(req.ctx, req.instance)
}

func newKubeVirtTemplateValidatorForCR(cr *hcov1alpha1.HyperConverged, namespace string) *sspv1.KubevirtTemplateValidator {
	labels := map[string]string{
		"app": cr.Name,
	}
	return &sspv1.KubevirtTemplateValidator{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "template-validator-" + cr.Name,
			Labels:    labels,
			Namespace: namespace,
		},
	}
}

func (r *ReconcileHyperConverged) ensureKubeVirtTemplateValidator(req *hcoRequest) error {
	kvTV := newKubeVirtTemplateValidatorForCR(req.instance, req.Namespace)
	if err := controllerutil.SetControllerReference(req.instance, kvTV, r.scheme); err != nil {
		return err
	}

	key, err := client.ObjectKeyFromObject(kvTV)
	if err != nil {
		req.logger.Error(err, "Failed to get object key for KubeVirt Template Validator")
	}

	found := &sspv1.KubevirtTemplateValidator{}
	err = r.client.Get(req.ctx, key, found)
	if err != nil && apierrors.IsNotFound(err) {
		req.logger.Info("Creating KubeVirt Template Validator")
		return r.client.Create(req.ctx, kvTV)
	}

	if err != nil {
		return err
	}

	req.logger.Info("KubeVirt Template Validator already exists", "validator.Namespace", found.Namespace, "validator.Name", found.Name)

	// Add it to the list of RelatedObjects if found
	objectRef, err := reference.GetReference(r.scheme, found)
	if err != nil {
		return err
	}
	objectreferencesv1.SetObjectReference(&req.instance.Status.RelatedObjects, *objectRef)

	// TODO: temporary avoid checking conditions on KubevirtTemplateValidator because it's currently
	// broken on k8s. Revert this when we will be able to fix it
	// handleComponentConditions(r, req, "KubevirtTemplateValidator", found.Status.Conditions)
	return r.client.Status().Update(req.ctx, req.instance)
}

func newKubeVirtStorageConfigForCR(cr *hcov1alpha1.HyperConverged, namespace string) *corev1.ConfigMap {
	localSC := "local-sc"
	if *(&cr.Spec.LocalStorageClassName) != "" {
		localSC = *(&cr.Spec.LocalStorageClassName)
	}

	labels := map[string]string{
		"app": cr.Name,
	}
	return &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "kubevirt-storage-class-defaults",
			Labels:    labels,
			Namespace: namespace,
		},
		Data: map[string]string{
			"accessMode":            "ReadWriteOnce",
			"volumeMode":            "Filesystem",
			localSC + ".accessMode": "ReadWriteOnce",
			localSC + ".volumeMode": "Filesystem",
		},
	}
}

func (r *ReconcileHyperConverged) ensureKubeVirtStorageConfig(req *hcoRequest) error {
	kubevirtStorageConfig := newKubeVirtStorageConfigForCR(req.instance, req.Namespace)
	if err := controllerutil.SetControllerReference(req.instance, kubevirtStorageConfig, r.scheme); err != nil {
		return err
	}

	key, err := client.ObjectKeyFromObject(kubevirtStorageConfig)
	if err != nil {
		req.logger.Error(err, "Failed to get object key for kubevirt storage config")
	}

	found := &corev1.ConfigMap{}
	err = r.client.Get(req.ctx, key, found)
	if err != nil && apierrors.IsNotFound(err) {
		req.logger.Info("Creating kubevirt storage config")
		return r.client.Create(req.ctx, kubevirtStorageConfig)
	}

	if err != nil {
		return err
	}

	req.logger.Info("KubeVirt storage config already exists", "KubeVirtConfig.Namespace", found.Namespace, "KubeVirtConfig.Name", found.Name)
	// Add it to the list of RelatedObjects if found
	objectRef, err := reference.GetReference(r.scheme, found)
	if err != nil {
		return err
	}
	objectreferencesv1.SetObjectReference(&req.instance.Status.RelatedObjects, *objectRef)

	return nil
}

func newKubeVirtMetricsAggregationForCR(cr *hcov1alpha1.HyperConverged, namespace string) *sspv1.KubevirtMetricsAggregation {
	labels := map[string]string{
		"app": cr.Name,
	}
	return &sspv1.KubevirtMetricsAggregation{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "metrics-aggregation-" + cr.Name,
			Labels:    labels,
			Namespace: namespace,
		},
	}
}

func (r *ReconcileHyperConverged) ensureKubeVirtMetricsAggregation(req *hcoRequest) error {
	kubevirtMetricsAggregation := newKubeVirtMetricsAggregationForCR(req.instance, req.Namespace)
	if err := controllerutil.SetControllerReference(req.instance, kubevirtMetricsAggregation, r.scheme); err != nil {
		return err
	}

	key, err := client.ObjectKeyFromObject(kubevirtMetricsAggregation)
	if err != nil {
		req.logger.Error(err, "Failed to get object key for KubeVirt Metrics Aggregation")
	}

	found := &sspv1.KubevirtMetricsAggregation{}
	err = r.client.Get(req.ctx, key, found)
	if err != nil && apierrors.IsNotFound(err) {
		req.logger.Info("Creating KubeVirt Metrics Aggregation")
		return r.client.Create(req.ctx, kubevirtMetricsAggregation)
	}

	if err != nil {
		return err
	}

	req.logger.Info("KubeVirt Metrics Aggregation already exists", "metrics.Namespace", found.Namespace, "metrics.Name", found.Name)

	// Add it to the list of RelatedObjects if found
	objectRef, err := reference.GetReference(r.scheme, found)
	if err != nil {
		return err
	}
	objectreferencesv1.SetObjectReference(&req.instance.Status.RelatedObjects, *objectRef)

	// TODO: we don't call handleComponentConditions because KubeVirtMetricsAggregation uses non-standard conditions
	// fix this when KubeVirtMetricsAggregation will be ready for this
	return r.client.Status().Update(req.ctx, req.instance)
}

// This function is used to exit from the reconcile function, updating the conditions and returns the reconcile result
func (r *ReconcileHyperConverged) updateConditions(req *hcoRequest) error {
	for _, condType := range hcoConditionTypes {
		cond, found := req.conditions[condType]
		if !found {
			cond = conditionsv1.Condition{
				Type:    condType,
				Status:  corev1.ConditionUnknown,
				Message: "Unknown Status",
			}
		}
		conditionsv1.SetStatusCondition(&req.instance.Status.Conditions, cond)
	}

	err := r.client.Status().Update(req.ctx, req.instance)
	req.logger.Info("Failed to update CR status")
	return err
}

func isKVMAvailable() bool {
	if val, ok := os.LookupEnv("KVM_EMULATION"); ok && (strings.ToLower(val) == "true") {
		log.Info("Running with KVM emulation")
		return false
	}
	log.Info("Running with KVM available")
	return true
}

// getHyperconverged returns the name/namespace of the HyperConverged resource
func getHyperconverged() (types.NamespacedName, error) {
	hco := types.NamespacedName{
		Name: hcov1alpha1.HyperConvergedName,
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

func drop(slice []string, s string) []string {
	newSlice := []string{}
	for _, element := range slice {
		if element != s {
			newSlice = append(newSlice, element)
		}
	}
	return newSlice
}

// toUnstructured convers an arbitrary object (which MUST obey the
// k8s object conventions) to an Unstructured
func toUnstructured(obj interface{}) (*unstructured.Unstructured, error) {
	b, err := json.Marshal(obj)
	if err != nil {
		return nil, err
	}
	u := &unstructured.Unstructured{}
	if err := json.Unmarshal(b, u); err != nil {
		return nil, err
	}
	return u, nil
}

func componentResourceRemoval(o interface{}, c client.Client, req *hcoRequest) error {
	resource, err := toUnstructured(o)
	if err != nil {
		req.logger.Error(err, "Failed to convert object to Unstructured")
		return err
	}

	err = c.Get(req.ctx, types.NamespacedName{Name: resource.GetName(), Namespace: resource.GetNamespace()}, resource)
	if err != nil {
		if apierrors.IsNotFound(err) {
			req.logger.Info("Resource doesn't exist, there is nothing to remove", "Kind", resource.GetObjectKind())
			return nil
		}
		return err
	}

	labels := resource.GetLabels()
	if app, labelExists := labels["app"]; !labelExists || app != req.instance.Name {
		req.logger.Info("Existing resource wasn't deployed by HCO, ignoring", "Kind", resource.GetObjectKind())
		return nil
	}

	err = c.Delete(req.ctx, resource)
	return err
}

func ensureKubeVirtPriorityClassDeleted(c client.Client, req *hcoRequest) error {
	pc := newKubeVirtPriorityClass()
	key, err := client.ObjectKeyFromObject(pc)
	if err != nil {
		req.logger.Error(err, "Failed to get object key for KubeVirt PriorityClass")
		return err
	}

	found := &schedulingv1.PriorityClass{}
	err = c.Get(req.ctx, key, found)

	if err != nil {
		if apierrors.IsNotFound(err) {
			req.logger.Info("KubeVirt Priority Class resource doesn't exist, there is nothing to remove")
			return nil
		}
		req.logger.Error(err, "Failed to get KubeVirt Priority Class from kubernetes")
		return err
	}

	return componentResourceRemoval(found, c, req)
}

func ensureKubeVirtDeleted(c client.Client, req *hcoRequest) error {
	virt := newKubeVirtForCR(req.instance, req.instance.Namespace)
	key, err := client.ObjectKeyFromObject(virt)
	if err != nil {
		req.logger.Error(err, "Failed to get object key for KubeVirt")
		return err
	}

	found := &kubevirtv1.KubeVirt{}
	err = c.Get(req.ctx, key, found)

	if err != nil {
		if apierrors.IsNotFound(err) {
			req.logger.Info("KubeVirt resource doesn't exist, there is nothing to remove")
			return nil
		}
		req.logger.Error(err, "Failed to get KubeVirt from kubernetes")
		return err
	}

	return componentResourceRemoval(found, c, req)
}

func ensureCDIDeleted(c client.Client, req *hcoRequest) error {
	cdi := newCDIForCR(req.instance, UndefinedNamespace)
	key, err := client.ObjectKeyFromObject(cdi)
	if err != nil {
		req.logger.Error(err, "Failed to get object key for CDI")
		return err
	}

	found := &cdiv1alpha1.CDI{}
	err = c.Get(req.ctx, key, found)

	if err != nil {
		if apierrors.IsNotFound(err) {
			req.logger.Info("CDI resource doesn't exist, there is nothing to remove")
			return nil
		}
		req.logger.Error(err, "Failed to get CDI from kubernetes")
		return err
	}

	return componentResourceRemoval(found, c, req)
}

func ensureNetworkAddonsDeleted(c client.Client, req *hcoRequest) error {
	networkAddons := newNetworkAddonsForCR(req.instance, UndefinedNamespace)
	key, err := client.ObjectKeyFromObject(networkAddons)
	if err != nil {
		req.logger.Error(err, "Failed to get object key for Network Addons")
		return err
	}

	found := &networkaddonsv1alpha1.NetworkAddonsConfig{}
	err = c.Get(req.ctx, key, found)

	if err != nil {
		if apierrors.IsNotFound(err) {
			req.logger.Info("NetworkAddonsConfig doesn't exist, there is nothing to remove")
			return nil
		}
		req.logger.Error(err, "Failed to get NetworkAddonsConfig from kubernetes")
		return err
	}

	return componentResourceRemoval(found, c, req)
}

func ensureKubeVirtCommonTemplateBundleDeleted(c client.Client, req *hcoRequest) error {
	kvCTB := newKubeVirtCommonTemplateBundleForCR(req.instance, OpenshiftNamespace)

	key, err := client.ObjectKeyFromObject(kvCTB)
	if err != nil {
		req.logger.Error(err, "Failed to get object key for KubeVirt Common Templates Bundle")
		return err
	}

	found := &sspv1.KubevirtCommonTemplatesBundle{}
	err = c.Get(req.ctx, key, found)

	if err != nil {
		if apierrors.IsNotFound(err) {
			req.logger.Info("KubevirtCommonTemplatesBundle doesn't exist, there is nothing to remove")
			return nil
		}
		req.logger.Error(err, "Failed to get KubeVirt Common Templates Bundle from kubernetes")
		return err
	}

	return componentResourceRemoval(found, c, req)
}

// translateKubeVirtConds translates list of KubeVirt conditions to a list of custom resource
// conditions.
func translateKubeVirtConds(orig []kubevirtv1.KubeVirtCondition) []conditionsv1.Condition {
	translated := make([]conditionsv1.Condition, len(orig))

	for i, origCond := range orig {
		translated[i] = conditionsv1.Condition{
			Type:    conditionsv1.ConditionType(origCond.Type),
			Status:  origCond.Status,
			Reason:  origCond.Reason,
			Message: origCond.Message,
		}
	}

	return translated
}
