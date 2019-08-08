package hyperconverged

import (
	"context"
	"fmt"
	"reflect"

	sspv1 "github.com/MarSik/kubevirt-ssp-operator/pkg/apis/kubevirt/v1"
	"github.com/go-logr/logr"
	networkaddons "github.com/kubevirt/cluster-network-addons-operator/pkg/apis/networkaddonsoperator/v1alpha1"
	hcov1alpha1 "github.com/kubevirt/hyperconverged-cluster-operator/pkg/apis/hco/v1alpha1"
	kubevirt "kubevirt.io/client-go/api/v1"
	cdi "kubevirt.io/containerized-data-importer/pkg/apis/core/v1alpha1"

	"encoding/json"

	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	logf "sigs.k8s.io/controller-runtime/pkg/runtime/log"
	"sigs.k8s.io/controller-runtime/pkg/source"
)

var log = logf.Log.WithName("controller_hyperconverged")

const (
	// We cannot set owner reference of cluster-wide resources to namespaced HyperConverged object. Therefore,
	// use finalizers to manage the cleanup.
	FinalizerName = "hyperconvergeds.hco.kubevirt.io"

	// Foreground deletion finalizer is blocking removal of HyperConverged until explicitly dropped.
	// TODO: Research whether there is a better way.
	foregroundDeletionFinalizer = "foregroundDeletion"
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
		&kubevirt.KubeVirt{},
		&cdi.CDI{},
		&networkaddons.NetworkAddonsConfig{},
		&sspv1.KubevirtCommonTemplatesBundle{},
		&sspv1.KubevirtNodeLabellerBundle{},
		&sspv1.KubevirtTemplateValidator{},
		&sspv1.KubevirtMetricsAggregation{},
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
	client client.Client
	scheme *runtime.Scheme
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
			// Request object not found, could have been deleted after reconcile request.
			// Owned objects are automatically garbage collected. For additional cleanup logic use finalizers.
			// Return and don't requeue
			return reconcile.Result{}, nil
		}
		// Error reading the object - requeue the request.
		return reconcile.Result{}, err
	}

	for _, f := range []func(*hcov1alpha1.HyperConverged, logr.Logger, reconcile.Request) error{
		r.ensureKubeVirtConfig,
		r.ensureKubeVirt,
		r.ensureCDI,
		r.ensureNetworkAddons,
		r.ensureKubeVirtCommonTemplatebundle,
		r.ensureKubeVirtNodeLabellerBundle,
		r.ensureKubeVirtTemplateValidator,
		r.ensureKubeVirtMetricsAggregation,
	} {
		err = f(instance, reqLogger, request)
		if err != nil {
			return reconcile.Result{}, err
		}
	}

	return reconcile.Result{}, nil
}

func (r *ReconcileHyperConverged) ensureKubeVirtConfig(instance *hcov1alpha1.HyperConverged, logger logr.Logger, request reconcile.Request) error {
	kubevirtConfig := newKubeVirtConfigForCR(instance, request.Namespace)
	return r.ensureResourceExists(instance, logger, request, kubevirtConfig)
}

func (r *ReconcileHyperConverged) ensureKubeVirt(instance *hcov1alpha1.HyperConverged, logger logr.Logger, request reconcile.Request) error {
	kubevirt := newKubeVirtForCR(instance, request.Namespace)
	return r.ensureResourceExists(instance, logger, request, kubevirt)
}

func (r *ReconcileHyperConverged) ensureCDI(instance *hcov1alpha1.HyperConverged, logger logr.Logger, request reconcile.Request) error {
	cdi := newCDIForCR(instance, UndefinedNamespace)
	return r.ensureResourceExists(instance, logger, request, cdi)
}

func (r *ReconcileHyperConverged) ensureNetworkAddons(instance *hcov1alpha1.HyperConverged, logger logr.Logger, request reconcile.Request) error {
	networkAddons := newNetworkAddonsForCR(instance, UndefinedNamespace)
	return r.ensureResourceExists(instance, logger, request, networkAddons)
}

func (r *ReconcileHyperConverged) ensureKubeVirtCommonTemplatebundle(instance *hcov1alpha1.HyperConverged, logger logr.Logger, request reconcile.Request) error {
	kubevirtCommonTemplateBundle := newKubeVirtCommonTemplateBundleForCR(instance, OpenshiftNamespace)
	return r.ensureResourceExists(instance, logger, request, kubevirtCommonTemplateBundle)
}

func (r *ReconcileHyperConverged) ensureKubeVirtNodeLabellerBundle(instance *hcov1alpha1.HyperConverged, logger logr.Logger, request reconcile.Request) error {
	kubevirtNodeLabellerBundle := newKubeVirtNodeLabellerBundleForCR(instance, request.Namespace)
	return r.ensureResourceExists(instance, logger, request, kubevirtNodeLabellerBundle)
}

func (r *ReconcileHyperConverged) ensureKubeVirtTemplateValidator(instance *hcov1alpha1.HyperConverged, logger logr.Logger, request reconcile.Request) error {
	kubevirtTemplateValidator := newKubeVirtTemplateValidatorForCR(instance, request.Namespace)
	return r.ensureResourceExists(instance, logger, request, kubevirtTemplateValidator)
}

func (r *ReconcileHyperConverged) ensureKubeVirtMetricsAggregation(instance *hcov1alpha1.HyperConverged, logger logr.Logger, request reconcile.Request) error {
	kubevirtMetricsAggregation := newKubeVirtMetricsAggregationForCR(instance, request.Namespace)
	return r.ensureResourceExists(instance, logger, request, kubevirtMetricsAggregation)
}

func (r *ReconcileHyperConverged) ensureResourceExists(instance *hcov1alpha1.HyperConverged, logger logr.Logger, request reconcile.Request, desiredRuntimeObj runtime.Object) error {
	desiredMetaObj := desiredRuntimeObj.(metav1.Object)

	// use reflection to create a new default instance of desiredRuntimeObj type
	// which is only used as a temporary variable for r.client.Get
	typ := reflect.ValueOf(desiredRuntimeObj).Elem().Type()
	currentRuntimeObj := reflect.New(typ).Interface().(runtime.Object)

	key := client.ObjectKey{
		Namespace: desiredMetaObj.GetNamespace(),
		Name:      desiredMetaObj.GetName(),
	}
	err := r.client.Get(context.TODO(), key, currentRuntimeObj)

	if err == nil {
		logger.Info("Skip reconcile: resource already exists", "key", key)
		return nil
	} else {
		// anything other than a "not found" error, don't create the resource, simply
		// return the error
		if !errors.IsNotFound(err) {
			return err
		}

		// set the reference for the object that is about to be created
		if err = controllerutil.SetControllerReference(instance, desiredMetaObj, r.scheme); err != nil {
			return err
		}

		// Note: common-templates fails the Get check above and appears to still be missing.
		// The check fails because request.Namespace is "kubevirt-hyperconverged", and
		// common-templates is installed in the "openshift" namespace.
		if err = r.client.Create(context.TODO(), desiredRuntimeObj); err != nil {
			if err != nil && errors.IsAlreadyExists(err) {
				logger.Info("Skip reconcile: tried create but resource already exists", "key", key)
				return nil
			} else if err != nil {
				return err
			}
		} else {
			logger.Info("Resource created",
				"namespace", desiredMetaObj.GetNamespace(),
				"name", desiredMetaObj.GetName(),
				"type", fmt.Sprintf("%T", desiredMetaObj))
		}
	}

	return nil
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
