package networkaddonsconfig

import (
	"context"
	"fmt"
	"log"
	"strings"

	osnetv1 "github.com/openshift/cluster-network-operator/pkg/apis/networkoperator/v1"
	osnetnames "github.com/openshift/cluster-network-operator/pkg/names"
	"github.com/pkg/errors"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes"
	"sigs.k8s.io/controller-runtime/pkg/client"
	k8sclient "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/config"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/source"

	opv1alpha1 "github.com/kubevirt/cluster-network-addons-operator/pkg/apis/networkaddonsoperator/v1alpha1"
	"github.com/kubevirt/cluster-network-addons-operator/pkg/apply"
	"github.com/kubevirt/cluster-network-addons-operator/pkg/names"
	"github.com/kubevirt/cluster-network-addons-operator/pkg/network"
)

// ManifestPath is the path to the manifest templates
var ManifestPath = "./data"

// Add creates a new NetworkAddonsConfig Controller and adds it to the Manager. The Manager will set fields on the Controller
// and Start it when the Manager is Started.
func Add(mgr manager.Manager) error {
	cfg, err := config.GetConfig()
	if err != nil {
		return fmt.Errorf("failed to get apiserver config: %v", err)
	}

	clientset, err := kubernetes.NewForConfig(cfg)
	if err != nil {
		return fmt.Errorf("failed to initialize apiserver client: %v", err)
	}

	sccIsAvailable, err := isSCCAvailable(clientset)
	if err != nil {
		return fmt.Errorf("failed to check for availability of SCC: %v", err)
	}

	return add(mgr, newReconciler(mgr, sccIsAvailable))
}

// newReconciler returns a new reconcile.Reconciler
func newReconciler(mgr manager.Manager, sccIsAvailable bool) reconcile.Reconciler {
	return &ReconcileNetworkAddonsConfig{
		client:         mgr.GetClient(),
		scheme:         mgr.GetScheme(),
		sccIsAvailable: sccIsAvailable,
	}
}

// add adds a new Controller to mgr with r as the reconcile.Reconciler
func add(mgr manager.Manager, r reconcile.Reconciler) error {
	// Create a new controller
	c, err := controller.New("networkaddonsconfig-controller", mgr, controller.Options{Reconciler: r})
	if err != nil {
		return err
	}

	// Watch for changes to primary resource NetworkAddonsConfig
	if err := c.Watch(&source.Kind{Type: &opv1alpha1.NetworkAddonsConfig{}}, &handler.EnqueueRequestForObject{}); err != nil {
		return err
	}

	return nil
}

var _ reconcile.Reconciler = &ReconcileNetworkAddonsConfig{}

// ReconcileNetworkAddonsConfig reconciles a NetworkAddonsConfig object
type ReconcileNetworkAddonsConfig struct {
	// This client, initialized using mgr.Client() above, is a split client
	// that reads objects from the cache and writes to the apiserver
	client client.Client
	scheme *runtime.Scheme

	sccIsAvailable bool
}

// Reconcile reads that state of the cluster for a NetworkAddonsConfig object and makes changes based on the state read
// and what is in the NetworkAddonsConfig.Spec
func (r *ReconcileNetworkAddonsConfig) Reconcile(request reconcile.Request) (reconcile.Result, error) {
	log.Print("reconciling NetworkAddonsConfig")

	// We won't create more than one network addons instance
	if request.Name != names.OPERATOR_CONFIG {
		log.Print("ignoring NetworkAddonsConfig without default name")
		return reconcile.Result{}, nil
	}

	// Fetch the NetworkAddonsConfig instance
	networkAddonsConfig := &opv1alpha1.NetworkAddonsConfig{}
	err := r.client.Get(context.TODO(), request.NamespacedName, networkAddonsConfig)
	if err != nil {
		if apierrors.IsNotFound(err) {
			// Request object not found, could have been deleted after reconcile request.
			// Owned objects are automatically garbage collected. Return and don't requeue
			return reconcile.Result{}, nil
		}
		// Error reading the object - requeue the request.
		return reconcile.Result{}, err
	}

	// Convert to a canonicalized form
	network.Canonicalize(&networkAddonsConfig.Spec)

	// TODO doc
	openshiftNetworkConfig, err := getOpenShiftNetworkConfig(context.TODO(), r.client)
	if err != nil {
		log.Printf("failed to load OpenShift NetworkConfig: %v", err)
		return reconcile.Result{}, err
	}

	// Validate the configuration
	if err := network.Validate(&networkAddonsConfig.Spec, openshiftNetworkConfig); err != nil {
		log.Printf("failed to validate NetworkConfig.Spec: %v", err)
		return reconcile.Result{}, err
	}

	// Retrieve the previously applied operator configuration
	prev, err := getAppliedConfiguration(context.TODO(), r.client, networkAddonsConfig.ObjectMeta.Name)
	if err != nil {
		log.Printf("failed to retrieve previously applied configuration: %v", err)
		return reconcile.Result{}, err
	}

	// Fill all defaults explicitly
	network.FillDefaults(&networkAddonsConfig.Spec, prev)

	// Compare against previous applied configuration to see if this change
	// is safe.
	if prev != nil {
		// We may need to fill defaults here -- sort of as a poor-man's
		// upconversion scheme -- if we add additional fields to the config.
		err = network.IsChangeSafe(prev, &networkAddonsConfig.Spec)
		if err != nil {
			log.Printf("not applying unsafe change: %v", err)
			errors.Wrapf(err, "not applying unsafe change")
			return reconcile.Result{}, err
		}
	}

	// Generate the objects
	objs, err := network.Render(&networkAddonsConfig.Spec, ManifestPath, openshiftNetworkConfig, r.sccIsAvailable)
	if err != nil {
		log.Printf("failed to render: %v", err)
		err = errors.Wrapf(err, "failed to render")
		return reconcile.Result{}, err
	}

	// The first object we create should be the record of our applied configuration
	applied, err := appliedConfiguration(networkAddonsConfig)
	if err != nil {
		log.Printf("failed to render applied: %v", err)
		err = errors.Wrapf(err, "failed to render applied")
		return reconcile.Result{}, err
	}
	objs = append([]*unstructured.Unstructured{applied}, objs...)

	// Apply the objects to the cluster
	for _, obj := range objs {
		// Mark the object to be GC'd if the owner is deleted
		if err := controllerutil.SetControllerReference(networkAddonsConfig, obj, r.scheme); err != nil {
			log.Printf("could not set reference for (%s) %s/%s: %v", obj.GroupVersionKind(), obj.GetNamespace(), obj.GetName(), err)
			err = errors.Wrapf(err, "could not set reference for (%s) %s/%s", obj.GroupVersionKind(), obj.GetNamespace(), obj.GetName())
			return reconcile.Result{}, err
		}

		// Apply all objects on apiserver
		if err := apply.ApplyObject(context.TODO(), r.client, obj); err != nil {
			log.Printf("could not apply (%s) %s/%s: %v", obj.GroupVersionKind(), obj.GetNamespace(), obj.GetName(), err)
			err = errors.Wrapf(err, "could not apply (%s) %s/%s", obj.GroupVersionKind(), obj.GetNamespace(), obj.GetName())
			return reconcile.Result{}, err
		}
	}

	return reconcile.Result{}, nil
}

func getOpenShiftNetworkConfig(ctx context.Context, c k8sclient.Client) (*osnetv1.NetworkConfig, error) {
	nc := &osnetv1.NetworkConfig{}

	// TODO: names imported and in constant
	err := c.Get(ctx, types.NamespacedName{Namespace: "", Name: osnetnames.OPERATOR_CONFIG}, nc)
	if err != nil {
		if apierrors.IsNotFound(err) || strings.Contains(err.Error(), "no matches for kind") {
			log.Printf("OpenShift cluster network configuration resource has not been found: %v", err)
			return nil, nil
		}
		log.Printf("failed to obtain OpenShift cluster network configuration with unexpected error: %v", err)
		return nil, err
	}

	return nc, nil
}

func isSCCAvailable(c kubernetes.Interface) (bool, error) {
	return isResourceAvailable(c, "securitycontextconstraints", "security.openshift.io", "v1")
}

func isResourceAvailable(kubeClient kubernetes.Interface, name string, group string, version string) (bool, error) {
	result := kubeClient.ExtensionsV1beta1().RESTClient().Get().RequestURI("/apis/" + group + "/" + version + "/" + name).Do()
	if result.Error() != nil {
		if strings.Contains(result.Error().Error(), "the server could not find the requested resource") {
			return false, nil
		}
		return false, result.Error()
	}

	return true, nil
}
