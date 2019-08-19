package hyperconverged

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/go-logr/logr"
	"github.com/operator-framework/operator-sdk/pkg/ready"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/tools/reference"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/source"

	sspv1 "github.com/MarSik/kubevirt-ssp-operator/pkg/apis/kubevirt/v1"
	networkaddonsv1alpha1 "github.com/kubevirt/cluster-network-addons-operator/pkg/apis/networkaddonsoperator/v1alpha1"
	networkaddonsnames "github.com/kubevirt/cluster-network-addons-operator/pkg/names"
	hcov1alpha1 "github.com/kubevirt/hyperconverged-cluster-operator/pkg/apis/hco/v1alpha1"
	conditionsv1 "github.com/openshift/custom-resource-status/conditions/v1"
	objectreferencesv1 "github.com/openshift/custom-resource-status/objectreferences/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	kubevirtv1 "kubevirt.io/client-go/api/v1"
	cdiv1alpha1 "kubevirt.io/containerized-data-importer/pkg/apis/core/v1alpha1"
	mrv1alpha1 "kubevirt.io/machine-remediation-operator/pkg/apis/machineremediation/v1alpha1"
	logf "sigs.k8s.io/controller-runtime/pkg/runtime/log"
)

var log = logf.Log.WithName("controller_hyperconverged")

const (
	// We cannot set owner reference of cluster-wide resources to namespaced HyperConverged object. Therefore,
	// use finalizers to manage the cleanup.
	FinalizerName = "hyperconvergeds.hco.kubevirt.io"

	// Foreground deletion finalizer is blocking removal of HyperConverged until explicitly dropped.
	// TODO: Research whether there is a better way.
	foregroundDeletionFinalizer = "foregroundDeletion"

	// UndefinedNamespace is for cluster scoped resources
	UndefinedNamespace string = ""

	// OpenshiftNamespace is for resources that belong in the openshift namespace
	OpenshiftNamespace string = "openshift"

	reconcileInit             = "Init"
	reconcileFailed           = "ReconcileFailed"
	reconcileCompleted        = "ReconcileCompleted"
	reconcileCompletedMessage = "Reconcile completed successfully"
)

// Add creates a new HyperConverged Controller and adds it to the Manager. The Manager will set fields on the Controller
// and Start it when the Manager is Started.
func Add(mgr manager.Manager) error {
	return add(mgr, newReconciler(mgr))
}

// newReconciler returns a new reconcile.Reconciler
func newReconciler(mgr manager.Manager) reconcile.Reconciler {
	return &ReconcileHyperConverged{client: mgr.GetClient(), scheme: mgr.GetScheme()}
}

// add adds a new Controller to mgr with r as the reconcile.Reconciler
func add(mgr manager.Manager, r reconcile.Reconciler) error {
	// Create a new controller
	c, err := controller.New("hyperconverged-controller", mgr, controller.Options{Reconciler: r})
	if err != nil {
		return err
	}

	// Watch for changes to primary resource HyperConverged
	err = c.Watch(&source.Kind{Type: &hcov1alpha1.HyperConverged{}}, &handler.EnqueueRequestForObject{})
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
		&mrv1alpha1.MachineRemediationOperator{},
	} {
		err = c.Watch(&source.Kind{Type: resource}, &handler.EnqueueRequestForOwner{
			IsController: true,
			OwnerType:    &hcov1alpha1.HyperConverged{},
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
	client     client.Client
	scheme     *runtime.Scheme
	conditions []conditionsv1.Condition
}

// Reconcile reads that state of the cluster for a HyperConverged object and makes changes based on the state read
// and what is in the HyperConverged.Spec
// Note:
// The Controller will requeue the Request to be processed again if the returned error is non-nil or
// Result.Requeue is true, otherwise upon completion it will remove the work from the queue.
func (r *ReconcileHyperConverged) Reconcile(request reconcile.Request) (reconcile.Result, error) {
	reqLogger := log.WithValues("Request.Namespace", request.Namespace, "Request.Name", request.Name)
	reqLogger.Info("Reconciling HyperConverged operator")

	// Fetch the HyperConverged instance
	instance := &hcov1alpha1.HyperConverged{}
	err := r.client.Get(context.TODO(), request.NamespacedName, instance)
	if err != nil {
		if errors.IsNotFound(err) {
			reqLogger.Info("No HyperConverged resource")
			// Request object not found, could have been deleted after reconcile request.
			// Owned objects are automatically garbage collected. For additional cleanup logic use finalizers.
			// Return and don't requeue
			return reconcile.Result{}, nil
		}
		// Error reading the object - requeue the request.
		return reconcile.Result{}, err
	}

	// Add conditions if there are none
	if instance.Status.Conditions == nil {
		message := "Initializing HyperConverged cluster"

		conditionsv1.SetStatusCondition(&instance.Status.Conditions, conditionsv1.Condition{
			Type:    hcov1alpha1.ConditionReconcileComplete,
			Status:  corev1.ConditionUnknown, // we just started trying to reconcile
			Reason:  reconcileInit,
			Message: message,
		})
		conditionsv1.SetStatusCondition(&instance.Status.Conditions, conditionsv1.Condition{
			Type:    conditionsv1.ConditionAvailable,
			Status:  corev1.ConditionFalse,
			Reason:  reconcileInit,
			Message: message,
		})
		conditionsv1.SetStatusCondition(&instance.Status.Conditions, conditionsv1.Condition{
			Type:    conditionsv1.ConditionProgressing,
			Status:  corev1.ConditionTrue,
			Reason:  reconcileInit,
			Message: message,
		})
		conditionsv1.SetStatusCondition(&instance.Status.Conditions, conditionsv1.Condition{
			Type:    conditionsv1.ConditionDegraded,
			Status:  corev1.ConditionFalse,
			Reason:  reconcileInit,
			Message: message,
		})
		conditionsv1.SetStatusCondition(&instance.Status.Conditions, conditionsv1.Condition{
			Type:    conditionsv1.ConditionUpgradeable,
			Status:  corev1.ConditionUnknown,
			Reason:  reconcileInit,
			Message: message,
		})

		err = r.client.Status().Update(context.TODO(), instance)
		if err != nil {
			reqLogger.Error(err, "Failed to add conditions to status")
			return reconcile.Result{}, err
		}
	}

	// in-memory conditions should start off empty. It will only ever hold
	// negative conditions (!Available, Degraded, Progressing)
	r.conditions = nil

	for _, f := range []func(*hcov1alpha1.HyperConverged, logr.Logger, reconcile.Request) error{
		r.ensureKubeVirtConfig,
		r.ensureKubeVirt,
		r.ensureCDI,
		r.ensureNetworkAddons,
		r.ensureKubeVirtCommonTemplateBundle,
		r.ensureKubeVirtNodeLabellerBundle,
		r.ensureKubeVirtTemplateValidator,
		r.ensureKubeVirtMetricsAggregation,
		r.ensureMachineRemediationOperator,
	} {
		err = f(instance, reqLogger, request)
		if err != nil {

			conditionsv1.SetStatusCondition(&instance.Status.Conditions, conditionsv1.Condition{
				Type:    hcov1alpha1.ConditionReconcileComplete,
				Status:  corev1.ConditionFalse,
				Reason:  reconcileFailed,
				Message: fmt.Sprintf("Error while reconciling: %v", err),
			})
			// don't want to overwrite the actual reconcile failure
			uErr := r.client.Status().Update(context.TODO(), instance)
			if uErr != nil {
				reqLogger.Error(uErr, "Failed to update conditions")
			}
			return reconcile.Result{}, err
		}
	}

	reqLogger.Info("Reconcile complete")
	conditionsv1.SetStatusCondition(&instance.Status.Conditions, conditionsv1.Condition{
		Type:    hcov1alpha1.ConditionReconcileComplete,
		Status:  corev1.ConditionTrue,
		Reason:  reconcileCompleted,
		Message: reconcileCompletedMessage,
	})
	if r.conditions == nil {
		reqLogger.Info("No component operator reported negatively")
		conditionsv1.SetStatusCondition(&instance.Status.Conditions, conditionsv1.Condition{
			Type:    conditionsv1.ConditionAvailable,
			Status:  corev1.ConditionTrue,
			Reason:  reconcileCompleted,
			Message: reconcileCompletedMessage,
		})
		conditionsv1.SetStatusCondition(&instance.Status.Conditions, conditionsv1.Condition{
			Type:    conditionsv1.ConditionProgressing,
			Status:  corev1.ConditionFalse,
			Reason:  reconcileCompleted,
			Message: reconcileCompletedMessage,
		})
		conditionsv1.SetStatusCondition(&instance.Status.Conditions, conditionsv1.Condition{
			Type:    conditionsv1.ConditionDegraded,
			Status:  corev1.ConditionFalse,
			Reason:  reconcileCompleted,
			Message: reconcileCompletedMessage,
		})
		conditionsv1.SetStatusCondition(&instance.Status.Conditions, conditionsv1.Condition{
			Type:    conditionsv1.ConditionUpgradeable,
			Status:  corev1.ConditionTrue,
			Reason:  reconcileCompleted,
			Message: reconcileCompletedMessage,
		})

		// If no operator whose conditions we are watching reports an error, then it is safe
		// to set readiness.
		r := ready.NewFileReady()
		err = r.Set()
		if err != nil {
			reqLogger.Error(err, "Failed to mark operator ready")
			return reconcile.Result{}, err
		}
	} else {
		// If any component operator reports negatively we want to write that to
		// the instance while preserving it's lastTransitionTime.
		// For example, consider the KubeVirt resource has the Available condition
		// type with type "False". When reconciling KubeVirt's resource we would
		// add it to the in-memory representation of HCO's conditions (r.conditions)
		// and here we are simply writing it back to the server.
		// One shortcoming is that only one failure of a particular condition can be
		// captured at one time (ie. if KubeVirt and CDI are both reporting !Available,
		// you will only see CDI as it updates last).
		for _, condition := range r.conditions {
			conditionsv1.SetStatusCondition(&instance.Status.Conditions, condition)
		}

		// If for any reason we marked ourselves !upgradeable...then unset readiness
		if conditionsv1.IsStatusConditionFalse(instance.Status.Conditions, conditionsv1.ConditionUpgradeable) {
			r := ready.NewFileReady()
			err = r.Unset()
			if err != nil {
				reqLogger.Error(err, "Failed to mark operator unready")
				return reconcile.Result{}, err
			}
		}
	}
	return reconcile.Result{}, r.client.Status().Update(context.TODO(), instance)
}

func newKubeVirtConfigForCR(cr *hcov1alpha1.HyperConverged, namespace string) *corev1.ConfigMap {
	labels := map[string]string{
		"app": cr.Name,
	}
	return &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "kubevirt-config",
			Labels:    labels,
			Namespace: namespace,
		},
		Data: map[string]string{
			"feature-gates": "DataVolumes,SRIOV,LiveMigration,CPUManager,CPUNodeDiscovery",
		},
	}
}

func (r *ReconcileHyperConverged) ensureKubeVirtConfig(instance *hcov1alpha1.HyperConverged, logger logr.Logger, request reconcile.Request) error {
	kubevirtConfig := newKubeVirtConfigForCR(instance, request.Namespace)
	if err := controllerutil.SetControllerReference(instance, kubevirtConfig, r.scheme); err != nil {
		return err
	}

	key, err := client.ObjectKeyFromObject(kubevirtConfig)
	if err != nil {
		logger.Error(err, "Failed to get object key for kubevirt config")
	}

	found := &corev1.ConfigMap{}
	err = r.client.Get(context.TODO(), key, found)
	if err != nil && errors.IsNotFound(err) {
		logger.Info("Creating kubevirt config")
		return r.client.Create(context.TODO(), kubevirtConfig)
	}

	if err != nil {
		return err
	}

	logger.Info("KubeVirt config already exists", "KubeVirtConfig.Namespace", found.Namespace, "KubeVirtConfig.Name", found.Name)
	// Add it to the list of RelatedObjects if found
	objectRef, err := reference.GetReference(r.scheme, found)
	if err != nil {
		return err
	}
	objectreferencesv1.SetObjectReference(&instance.Status.RelatedObjects, *objectRef)

	return nil
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
	}
}

func (r *ReconcileHyperConverged) ensureKubeVirt(instance *hcov1alpha1.HyperConverged, logger logr.Logger, request reconcile.Request) error {
	virt := newKubeVirtForCR(instance, request.Namespace)
	if err := controllerutil.SetControllerReference(instance, virt, r.scheme); err != nil {
		return err
	}

	key, err := client.ObjectKeyFromObject(virt)
	if err != nil {
		logger.Error(err, "Failed to get object key for KubeVirt")
	}

	found := &kubevirtv1.KubeVirt{}
	err = r.client.Get(context.TODO(), key, found)
	if err != nil && errors.IsNotFound(err) {
		logger.Info("Creating kubevirt")
		return r.client.Create(context.TODO(), virt)
	}

	if err != nil {
		return err
	}

	logger.Info("KubeVirt already exists", "KubeVirt.Namespace", found.Namespace, "KubeVirt.Name", found.Name)

	// Add it to the list of RelatedObjects if found
	objectRef, err := reference.GetReference(r.scheme, found)
	if err != nil {
		return err
	}
	objectreferencesv1.SetObjectReference(&instance.Status.RelatedObjects, *objectRef)

	// Handle KubeVirt resource conditions
	if found.Status.Conditions == nil {
		logger.Info("KubeVirt's resource is not reporting Conditions on it's Status")
		conditionsv1.SetStatusCondition(&r.conditions, conditionsv1.Condition{
			Type:    conditionsv1.ConditionAvailable,
			Status:  corev1.ConditionFalse,
			Reason:  "KubeVirtConditions",
			Message: "KubeVirt resource has no conditions",
		})
		conditionsv1.SetStatusCondition(&r.conditions, conditionsv1.Condition{
			Type:    conditionsv1.ConditionProgressing,
			Status:  corev1.ConditionTrue,
			Reason:  "KubeVirtConditions",
			Message: "KubeVirt resource has no conditions",
		})
		conditionsv1.SetStatusCondition(&r.conditions, conditionsv1.Condition{
			Type:    conditionsv1.ConditionUpgradeable,
			Status:  corev1.ConditionFalse,
			Reason:  "KubeVirtConditions",
			Message: "KubeVirt resource has no conditions",
		})
	} else {
		for _, condition := range found.Status.Conditions {
			// convert the KubeVirt condition type to one we understand
			switch conditionsv1.ConditionType(condition.Type) {
			case conditionsv1.ConditionAvailable:
				if condition.Status == corev1.ConditionFalse {
					logger.Info("KubeVirt is not 'Available'")
					conditionsv1.SetStatusCondition(&r.conditions, conditionsv1.Condition{
						Type:    conditionsv1.ConditionAvailable,
						Status:  corev1.ConditionFalse,
						Reason:  "KubeVirtNotAvailable",
						Message: fmt.Sprintf("KubeVirt is not available: %v", string(condition.Message)),
					})
				}
			case conditionsv1.ConditionProgressing:
				if condition.Status == corev1.ConditionTrue {
					logger.Info("KubeVirt is 'Progressing'")
					conditionsv1.SetStatusCondition(&r.conditions, conditionsv1.Condition{
						Type:    conditionsv1.ConditionProgressing,
						Status:  corev1.ConditionTrue,
						Reason:  "KubeVirtProgressing",
						Message: fmt.Sprintf("KubeVirt is progressing: %v", string(condition.Message)),
					})
					conditionsv1.SetStatusCondition(&r.conditions, conditionsv1.Condition{
						Type:    conditionsv1.ConditionUpgradeable,
						Status:  corev1.ConditionFalse,
						Reason:  "KubeVirtProgressing",
						Message: fmt.Sprintf("KubeVirt is progressing: %v", string(condition.Message)),
					})
				}
			case conditionsv1.ConditionDegraded:
				if condition.Status == corev1.ConditionTrue {
					logger.Info("KubeVirt is 'Degraded'")
					conditionsv1.SetStatusCondition(&r.conditions, conditionsv1.Condition{
						Type:    conditionsv1.ConditionDegraded,
						Status:  corev1.ConditionTrue,
						Reason:  "KubeVirtDegraded",
						Message: fmt.Sprintf("KubeVirt is degraded: %v", string(condition.Message)),
					})
				}
			}
		}
	}

	return r.client.Status().Update(context.TODO(), instance)
}

// newCDIForCr returns a CDI CR
func newCDIForCR(cr *hcov1alpha1.HyperConverged, namespace string) *cdiv1alpha1.CDI {
	labels := map[string]string{
		"app": cr.Name,
	}
	return &cdiv1alpha1.CDI{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "cdi-" + cr.Name,
			Labels:    labels,
			Namespace: namespace,
		},
	}
}

func (r *ReconcileHyperConverged) ensureCDI(instance *hcov1alpha1.HyperConverged, logger logr.Logger, request reconcile.Request) error {
	cdi := newCDIForCR(instance, UndefinedNamespace)
	if err := controllerutil.SetControllerReference(instance, cdi, r.scheme); err != nil {
		return err
	}

	key, err := client.ObjectKeyFromObject(cdi)
	if err != nil {
		logger.Error(err, "Failed to get object key for CDI")
	}

	found := &cdiv1alpha1.CDI{}
	err = r.client.Get(context.TODO(), key, found)
	if err != nil && errors.IsNotFound(err) {
		logger.Info("Creating CDI")
		return r.client.Create(context.TODO(), cdi)
	}

	if err != nil {
		return err
	}

	logger.Info("CDI already exists", "CDI.Namespace", found.Namespace, "CDI.Name", found.Name)

	// Add it to the list of RelatedObjects if found
	objectRef, err := reference.GetReference(r.scheme, found)
	if err != nil {
		return err
	}
	objectreferencesv1.SetObjectReference(&instance.Status.RelatedObjects, *objectRef)

	// Handle CDI resource conditions
	if found.Status.Conditions == nil {
		logger.Info("CDI's resource is not reporting Conditions on it's Status")
		conditionsv1.SetStatusCondition(&r.conditions, conditionsv1.Condition{
			Type:    conditionsv1.ConditionAvailable,
			Status:  corev1.ConditionFalse,
			Reason:  "CDIConditions",
			Message: "CDI resource has no conditions",
		})
		conditionsv1.SetStatusCondition(&r.conditions, conditionsv1.Condition{
			Type:    conditionsv1.ConditionProgressing,
			Status:  corev1.ConditionTrue,
			Reason:  "CDIConditions",
			Message: "CDI resource has no conditions",
		})
		conditionsv1.SetStatusCondition(&r.conditions, conditionsv1.Condition{
			Type:    conditionsv1.ConditionUpgradeable,
			Status:  corev1.ConditionFalse,
			Reason:  "CDIConditions",
			Message: "CDI resource has no conditions",
		})
	} else {
		for _, condition := range found.Status.Conditions {
			// convert the CDI condition type to one we understand
			switch conditionsv1.ConditionType(condition.Type) {
			case conditionsv1.ConditionAvailable:
				if condition.Status == corev1.ConditionFalse {
					logger.Info("CDI is not 'Available'")
					conditionsv1.SetStatusCondition(&r.conditions, conditionsv1.Condition{
						Type:    conditionsv1.ConditionAvailable,
						Status:  corev1.ConditionFalse,
						Reason:  "CDINotAvailable",
						Message: fmt.Sprintf("CDI is not available: %v", string(condition.Message)),
					})
				}
			case conditionsv1.ConditionProgressing:
				if condition.Status == corev1.ConditionTrue {
					logger.Info("CDI is 'Progressing'")
					conditionsv1.SetStatusCondition(&r.conditions, conditionsv1.Condition{
						Type:    conditionsv1.ConditionProgressing,
						Status:  corev1.ConditionTrue,
						Reason:  "CDIProgressing",
						Message: fmt.Sprintf("CDI is progressing: %v", string(condition.Message)),
					})
					conditionsv1.SetStatusCondition(&r.conditions, conditionsv1.Condition{
						Type:    conditionsv1.ConditionUpgradeable,
						Status:  corev1.ConditionFalse,
						Reason:  "CDIProgressing",
						Message: fmt.Sprintf("CDI is progressing: %v", string(condition.Message)),
					})
				}
			case conditionsv1.ConditionDegraded:
				if condition.Status == corev1.ConditionTrue {
					logger.Info("CDI is 'Degraded'")
					conditionsv1.SetStatusCondition(&r.conditions, conditionsv1.Condition{
						Type:    conditionsv1.ConditionDegraded,
						Status:  corev1.ConditionTrue,
						Reason:  "CDIDegraded",
						Message: fmt.Sprintf("CDI is degraded: %v", string(condition.Message)),
					})
				}
			}
		}
	}

	return r.client.Status().Update(context.TODO(), instance)
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
			KubeMacPool: &networkaddonsv1alpha1.KubeMacPool{},
			Ovs:         &networkaddonsv1alpha1.Ovs{},
			NMState:     &networkaddonsv1alpha1.NMState{},
		},
	}
}

func (r *ReconcileHyperConverged) ensureNetworkAddons(instance *hcov1alpha1.HyperConverged, logger logr.Logger, request reconcile.Request) error {
	networkAddons := newNetworkAddonsForCR(instance, UndefinedNamespace)
	if err := controllerutil.SetControllerReference(instance, networkAddons, r.scheme); err != nil {
		return err
	}

	key, err := client.ObjectKeyFromObject(networkAddons)
	if err != nil {
		logger.Error(err, "Failed to get object key for Network Addons")
	}

	found := &networkaddonsv1alpha1.NetworkAddonsConfig{}
	err = r.client.Get(context.TODO(), key, found)
	if err != nil && errors.IsNotFound(err) {
		logger.Info("Creating Network Addons")
		return r.client.Create(context.TODO(), networkAddons)
	}

	if err != nil {
		return err
	}

	logger.Info("NetworkAddonsConfig already exists", "NetworkAddonsConfig.Namespace", found.Namespace, "NetworkAddonsConfig.Name", found.Name)

	// Add it to the list of RelatedObjects if found
	objectRef, err := reference.GetReference(r.scheme, found)
	if err != nil {
		return err
	}
	objectreferencesv1.SetObjectReference(&instance.Status.RelatedObjects, *objectRef)

	// Handle conditions
	if found.Status.Conditions == nil {
		logger.Info("NetworkAddonsConfig's resource is not reporting Conditions on it's Status")
		conditionsv1.SetStatusCondition(&r.conditions, conditionsv1.Condition{
			Type:    conditionsv1.ConditionAvailable,
			Status:  corev1.ConditionFalse,
			Reason:  "NetworkAddonsConfigConditions",
			Message: "NetworkAddonsConfig resource has no conditions",
		})
		conditionsv1.SetStatusCondition(&r.conditions, conditionsv1.Condition{
			Type:    conditionsv1.ConditionProgressing,
			Status:  corev1.ConditionTrue,
			Reason:  "NetworkAddonsConfigConditions",
			Message: "NetworkAddonsConfig resource has no conditions",
		})
		conditionsv1.SetStatusCondition(&r.conditions, conditionsv1.Condition{
			Type:    conditionsv1.ConditionUpgradeable,
			Status:  corev1.ConditionFalse,
			Reason:  "NetworkAddonsConfigConditions",
			Message: "NetworkAddonsConfig resource has no conditions",
		})
	} else {
		for _, condition := range found.Status.Conditions {
			switch conditionsv1.ConditionType(condition.Type) {
			case conditionsv1.ConditionAvailable:
				if condition.Status == corev1.ConditionFalse {
					logger.Info("NetworkAddonsConfig is not 'Available'")
					conditionsv1.SetStatusCondition(&r.conditions, conditionsv1.Condition{
						Type:    conditionsv1.ConditionAvailable,
						Status:  corev1.ConditionFalse,
						Reason:  "NetworkAddonsConfigNotAvailable",
						Message: fmt.Sprintf("NetworkAddonsConfig is not available: %v", string(condition.Message)),
					})
				}
			case conditionsv1.ConditionProgressing:
				if condition.Status == corev1.ConditionTrue {
					logger.Info("NetworkAddonsConfig is 'Progressing'")
					conditionsv1.SetStatusCondition(&r.conditions, conditionsv1.Condition{
						Type:    conditionsv1.ConditionProgressing,
						Status:  corev1.ConditionTrue,
						Reason:  "NetworkAddonsConfigProgressing",
						Message: fmt.Sprintf("NetworkAddonsConfig is progressing: %v", string(condition.Message)),
					})
					conditionsv1.SetStatusCondition(&r.conditions, conditionsv1.Condition{
						Type:    conditionsv1.ConditionUpgradeable,
						Status:  corev1.ConditionFalse,
						Reason:  "NetworkAddonsConfigProgressing",
						Message: fmt.Sprintf("NetworkAddonsConfig is progressing: %v", string(condition.Message)),
					})
				}
			case conditionsv1.ConditionDegraded:
				if condition.Status == corev1.ConditionTrue {
					logger.Info("NetworkAddonsConfig is 'Degraded'")
					conditionsv1.SetStatusCondition(&r.conditions, conditionsv1.Condition{
						Type:    conditionsv1.ConditionDegraded,
						Status:  corev1.ConditionTrue,
						Reason:  "NetworkAddonsConfigDegraded",
						Message: fmt.Sprintf("NetworkAddonsConfig is degraded: %v", string(condition.Message)),
					})
				}
			}
		}
	}

	return r.client.Status().Update(context.TODO(), instance)
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

func (r *ReconcileHyperConverged) ensureKubeVirtCommonTemplateBundle(instance *hcov1alpha1.HyperConverged, logger logr.Logger, request reconcile.Request) error {
	kvCTB := newKubeVirtCommonTemplateBundleForCR(instance, OpenshiftNamespace)
	if err := controllerutil.SetControllerReference(instance, kvCTB, r.scheme); err != nil {
		return err
	}

	key, err := client.ObjectKeyFromObject(kvCTB)
	if err != nil {
		logger.Error(err, "Failed to get object key for KubeVirt Common Templates Bundle")
	}

	found := &sspv1.KubevirtCommonTemplatesBundle{}
	err = r.client.Get(context.TODO(), key, found)
	if err != nil && errors.IsNotFound(err) {
		logger.Info("Creating KubeVirt Common Templates Bundle")
		return r.client.Create(context.TODO(), kvCTB)
	}

	if err != nil {
		return err
	}

	logger.Info("KubeVirt Common Templates Bundle already exists", "bundle.Namespace", found.Namespace, "bundle.Name", found.Name)

	// Add it to the list of RelatedObjects if found
	objectRef, err := reference.GetReference(r.scheme, found)
	if err != nil {
		return err
	}
	objectreferencesv1.SetObjectReference(&instance.Status.RelatedObjects, *objectRef)

	// TODO: Handle conditions
	return r.client.Status().Update(context.TODO(), instance)
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
	}
}

func (r *ReconcileHyperConverged) ensureKubeVirtNodeLabellerBundle(instance *hcov1alpha1.HyperConverged, logger logr.Logger, request reconcile.Request) error {
	kvNLB := newKubeVirtNodeLabellerBundleForCR(instance, request.Namespace)
	if err := controllerutil.SetControllerReference(instance, kvNLB, r.scheme); err != nil {
		return err
	}

	key, err := client.ObjectKeyFromObject(kvNLB)
	if err != nil {
		logger.Error(err, "Failed to get object key for KubeVirt Node Labeller Bundle")
	}

	found := &sspv1.KubevirtNodeLabellerBundle{}
	err = r.client.Get(context.TODO(), key, found)
	if err != nil && errors.IsNotFound(err) {
		logger.Info("Creating KubeVirt Node Labeller Bundle")
		return r.client.Create(context.TODO(), kvNLB)
	}

	if err != nil {
		return err
	}

	logger.Info("KubeVirt Node Labeller Bundle already exists", "bundle.Namespace", found.Namespace, "bundle.Name", found.Name)

	// Add it to the list of RelatedObjects if found
	objectRef, err := reference.GetReference(r.scheme, found)
	if err != nil {
		return err
	}
	objectreferencesv1.SetObjectReference(&instance.Status.RelatedObjects, *objectRef)

	// TODO: Handle conditions
	return r.client.Status().Update(context.TODO(), instance)
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

func (r *ReconcileHyperConverged) ensureKubeVirtTemplateValidator(instance *hcov1alpha1.HyperConverged, logger logr.Logger, request reconcile.Request) error {
	kvTV := newKubeVirtTemplateValidatorForCR(instance, request.Namespace)
	if err := controllerutil.SetControllerReference(instance, kvTV, r.scheme); err != nil {
		return err
	}

	key, err := client.ObjectKeyFromObject(kvTV)
	if err != nil {
		logger.Error(err, "Failed to get object key for KubeVirt Template Validator")
	}

	found := &sspv1.KubevirtTemplateValidator{}
	err = r.client.Get(context.TODO(), key, found)
	if err != nil && errors.IsNotFound(err) {
		logger.Info("Creating KubeVirt Template Validator")
		return r.client.Create(context.TODO(), kvTV)
	}

	if err != nil {
		return err
	}

	logger.Info("KubeVirt Template Validator already exists", "validator.Namespace", found.Namespace, "validator.Name", found.Name)

	// Add it to the list of RelatedObjects if found
	objectRef, err := reference.GetReference(r.scheme, found)
	if err != nil {
		return err
	}
	objectreferencesv1.SetObjectReference(&instance.Status.RelatedObjects, *objectRef)

	// TODO: Handle conditions
	return r.client.Status().Update(context.TODO(), instance)
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

func (r *ReconcileHyperConverged) ensureKubeVirtMetricsAggregation(instance *hcov1alpha1.HyperConverged, logger logr.Logger, request reconcile.Request) error {
	kubevirtMetricsAggregation := newKubeVirtMetricsAggregationForCR(instance, request.Namespace)
	if err := controllerutil.SetControllerReference(instance, kubevirtMetricsAggregation, r.scheme); err != nil {
		return err
	}

	key, err := client.ObjectKeyFromObject(kubevirtMetricsAggregation)
	if err != nil {
		logger.Error(err, "Failed to get object key for KubeVirt Metrics Aggregation")
	}

	found := &sspv1.KubevirtMetricsAggregation{}
	err = r.client.Get(context.TODO(), key, found)
	if err != nil && errors.IsNotFound(err) {
		logger.Info("Creating KubeVirt Metrics Aggregation")
		return r.client.Create(context.TODO(), kubevirtMetricsAggregation)
	}

	if err != nil {
		return err
	}

	logger.Info("KubeVirt Metrics Aggregation already exists", "metrics.Namespace", found.Namespace, "metrics.Name", found.Name)

	// Add it to the list of RelatedObjects if found
	objectRef, err := reference.GetReference(r.scheme, found)
	if err != nil {
		return err
	}
	objectreferencesv1.SetObjectReference(&instance.Status.RelatedObjects, *objectRef)

	// TODO: Handle conditions
	return r.client.Status().Update(context.TODO(), instance)
}

// newMROForCR returns a MachineRemediationOperator CR
func newMachineRemediationOperatorForCR(cr *hcov1alpha1.HyperConverged, namespace string) *mrv1alpha1.MachineRemediationOperator {
	labels := map[string]string{
		"app": cr.Name,
	}
	return &mrv1alpha1.MachineRemediationOperator{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "mro-" + cr.Name,
			Labels:    labels,
			Namespace: namespace,
		},
	}
}

func (r *ReconcileHyperConverged) ensureMachineRemediationOperator(instance *hcov1alpha1.HyperConverged, logger logr.Logger, request reconcile.Request) error {
	mro := newMachineRemediationOperatorForCR(instance, request.Namespace)
	if err := controllerutil.SetControllerReference(instance, mro, r.scheme); err != nil {
		return err
	}

	key, err := client.ObjectKeyFromObject(mro)
	if err != nil {
		logger.Error(err, "Failed to get object key for MachineRemediationOperator")
	}

	found := &mrv1alpha1.MachineRemediationOperator{}
	err = r.client.Get(context.TODO(), key, found)
	if err != nil && errors.IsNotFound(err) {
		logger.Info("Creating MachineRemediationOperator")
		return r.client.Create(context.TODO(), mro)
	}

	if err != nil {
		return err
	}

	logger.Info("MachineRemediationOperator already exists", "MachineRemediationOperator.Namespace", found.Namespace, "MachineRemediationOperator.Name", found.Name)

	// Add it to the list of RelatedObjects if found
	objectRef, err := reference.GetReference(r.scheme, found)
	if err != nil {
		return err
	}
	objectreferencesv1.SetObjectReference(&instance.Status.RelatedObjects, *objectRef)

	// Handle MachineRemediationOperator resource conditions
	if found.Status.Conditions == nil {
		logger.Info("MachineRemediationOperator resource is not reporting Conditions on it's Status")
		conditionsv1.SetStatusCondition(&r.conditions, conditionsv1.Condition{
			Type:    conditionsv1.ConditionAvailable,
			Status:  corev1.ConditionFalse,
			Reason:  "MachineRemediationOperatorConditions",
			Message: "MachineRemediationOperator resource has no conditions",
		})
		conditionsv1.SetStatusCondition(&r.conditions, conditionsv1.Condition{
			Type:    conditionsv1.ConditionProgressing,
			Status:  corev1.ConditionTrue,
			Reason:  "MachineRemediationOperatorConditions",
			Message: "MachineRemediationOperator resource has no conditions",
		})
		conditionsv1.SetStatusCondition(&r.conditions, conditionsv1.Condition{
			Type:    conditionsv1.ConditionUpgradeable,
			Status:  corev1.ConditionFalse,
			Reason:  "MachineRemediationOperatorConditions",
			Message: "MachineRemediationOperator resource has no conditions",
		})
	} else {
		for _, condition := range found.Status.Conditions {
			// convert the KubeVirt condition type to one we understand
			switch conditionsv1.ConditionType(condition.Type) {
			case conditionsv1.ConditionAvailable:
				if condition.Status == corev1.ConditionFalse {
					logger.Info("MachineRemediationOperator is not 'Available'")
					conditionsv1.SetStatusCondition(&r.conditions, conditionsv1.Condition{
						Type:    conditionsv1.ConditionAvailable,
						Status:  corev1.ConditionFalse,
						Reason:  "MachineRemediationOperatorNotAvailable",
						Message: fmt.Sprintf("MachineRemediationOperator is not available: %v", string(condition.Message)),
					})
				}
			case conditionsv1.ConditionProgressing:
				if condition.Status == corev1.ConditionTrue {
					logger.Info("MachineRemediationOperator is 'Progressing'")
					conditionsv1.SetStatusCondition(&r.conditions, conditionsv1.Condition{
						Type:    conditionsv1.ConditionProgressing,
						Status:  corev1.ConditionTrue,
						Reason:  "MachineRemediationOperatorProgressing",
						Message: fmt.Sprintf("MachineRemediationOperator is progressing: %v", string(condition.Message)),
					})
					conditionsv1.SetStatusCondition(&r.conditions, conditionsv1.Condition{
						Type:    conditionsv1.ConditionUpgradeable,
						Status:  corev1.ConditionFalse,
						Reason:  "MachineRemediationOperatorProgressing",
						Message: fmt.Sprintf("MachineRemediationOperator is progressing: %v", string(condition.Message)),
					})
				}
			case conditionsv1.ConditionDegraded:
				if condition.Status == corev1.ConditionTrue {
					logger.Info("MachineRemediationOperator is 'Degraded'")
					conditionsv1.SetStatusCondition(&r.conditions, conditionsv1.Condition{
						Type:    conditionsv1.ConditionDegraded,
						Status:  corev1.ConditionTrue,
						Reason:  "MachineRemediationOperatorDegraded",
						Message: fmt.Sprintf("MachineRemediationOperator is degraded: %v", string(condition.Message)),
					})
				}
			}
		}
	}

	return r.client.Status().Update(context.TODO(), instance)
}

// The set of resources managed by the HCO
func (r *ReconcileHyperConverged) getAllResources(cr *hcov1alpha1.HyperConverged, request reconcile.Request) []runtime.Object {
	return []runtime.Object{
		newKubeVirtConfigForCR(cr, request.Namespace),
		newKubeVirtForCR(cr, request.Namespace),
		newCDIForCR(cr, UndefinedNamespace),
		newNetworkAddonsForCR(cr, UndefinedNamespace),
		newKubeVirtCommonTemplateBundleForCR(cr, OpenshiftNamespace),
		newKubeVirtNodeLabellerBundleForCR(cr, request.Namespace),
		newKubeVirtTemplateValidatorForCR(cr, request.Namespace),
		newMachineRemediationOperatorForCR(cr, request.Namespace),
	}
}

func contains(l []string, s string) bool {
	for _, elem := range l {
		if elem == s {
			return true
		}
	}
	return false
}

func drop(l []string, s string) []string {
	newL := []string{}
	for _, elem := range l {
		if elem != s {
			newL = append(newL, elem)
		}
	}
	return newL
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
