package networkaddonsconfig

import (
	"context"
	"fmt"
	"log"
	"os"
	"reflect"
	"strings"

	osv1 "github.com/openshift/api/operator/v1"
	osnetnames "github.com/openshift/cluster-network-operator/pkg/names"
	"github.com/pkg/errors"
	appsv1 "k8s.io/api/apps/v1"
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
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/source"

	opv1alpha1 "github.com/kubevirt/cluster-network-addons-operator/pkg/apis/networkaddonsoperator/v1alpha1"
	"github.com/kubevirt/cluster-network-addons-operator/pkg/apply"
	"github.com/kubevirt/cluster-network-addons-operator/pkg/controller/statusmanager"
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

	namespace, namespaceSet := os.LookupEnv("OPERATOR_NAMESPACE")
	if !namespaceSet {
		return fmt.Errorf("environment variable OPERATOR_NAMESPACE has to be set")
	}

	clusterInfo := &network.ClusterInfo{}

	openShift4, err := isRunningOnOpenShift4(clientset)
	if err != nil {
		return fmt.Errorf("failed to check whether running on OpenShift 4: %v", err)
	}
	if openShift4 {
		log.Printf("Running on OpenShift 4")
	}
	clusterInfo.OpenShift4 = openShift4

	sccAvailable, err := isSCCAvailable(clientset)
	if err != nil {
		return fmt.Errorf("failed to check for availability of SCC: %v", err)
	}
	clusterInfo.SCCAvailable = sccAvailable

	return add(mgr, newReconciler(mgr, namespace, clusterInfo))
}

// newReconciler returns a new ReconcileNetworkAddonsConfig
func newReconciler(mgr manager.Manager, namespace string, clusterInfo *network.ClusterInfo) *ReconcileNetworkAddonsConfig {
	// Status manager is shared between both reconcilers and it is used to update conditions of
	// NetworkAddonsConfig.State. NetworkAddonsConfig reconciler updates it with progress of rendering
	// and applying of manifests. Pods reconciler updates it with progress of deployed pods.
	statusManager := statusmanager.New(mgr.GetClient(), names.OPERATOR_CONFIG)
	return &ReconcileNetworkAddonsConfig{
		client:        mgr.GetClient(),
		scheme:        mgr.GetScheme(),
		namespace:     namespace,
		podReconciler: newPodReconciler(statusManager),
		statusManager: statusManager,
		clusterInfo:   clusterInfo,
	}
}

// add adds a new Controller to mgr with r as the ReconcileNetworkAddonsConfig
func add(mgr manager.Manager, r *ReconcileNetworkAddonsConfig) error {
	// Create a new controller for operator's NetworkAddonsConfig resource
	c, err := controller.New("networkaddonsconfig-controller", mgr, controller.Options{Reconciler: r})
	if err != nil {
		return err
	}

	// Create custom predicate for NetworkAddonsConfig watcher. This makes sure that Status field
	// updates will not trigger reconciling of the object. Reconciliation is trigger only if
	// Spec fields differ.
	pred := predicate.Funcs{
		UpdateFunc: func(e event.UpdateEvent) bool {
			oldConfig, err := runtimeObjectToNetworkAddonsConfig(e.ObjectOld)
			if err != nil {
				log.Printf("Failed to convert runtime.Object to NetworkAddonsConfig: %v", err)
				return false
			}
			newConfig, err := runtimeObjectToNetworkAddonsConfig(e.ObjectNew)
			if err != nil {
				log.Printf("Failed to convert runtime.Object to NetworkAddonsConfig: %v", err)
				return false
			}
			return !reflect.DeepEqual(oldConfig.Spec, newConfig.Spec)
		},
	}

	// Watch for changes to primary resource NetworkAddonsConfig
	if err := c.Watch(&source.Kind{Type: &opv1alpha1.NetworkAddonsConfig{}}, &handler.EnqueueRequestForObject{}, pred); err != nil {
		return err
	}

	// Create a new controller for Pod resources, this will be used to track state of deployed components
	c, err = controller.New("pod-controller", mgr, controller.Options{Reconciler: r.podReconciler})
	if err != nil {
		return err
	}

	// Watch for changes on DaemonSet and Deployment resources
	err = c.Watch(&source.Kind{Type: &appsv1.DaemonSet{}}, &handler.EnqueueRequestForObject{})
	if err != nil {
		return err
	}
	err = c.Watch(&source.Kind{Type: &appsv1.Deployment{}}, &handler.EnqueueRequestForObject{})
	if err != nil {
		return err
	}

	return nil
}

var _ reconcile.Reconciler = &ReconcileNetworkAddonsConfig{}

// ReconcileNetworkAddonsConfig reconciles a NetworkAddonsConfig object
type ReconcileNetworkAddonsConfig struct {
	// This client, initialized using mgr.Client() above, is a split client
	// that reads objects from the cache and writes to the apiserver
	client        client.Client
	scheme        *runtime.Scheme
	namespace     string
	podReconciler *ReconcilePods
	statusManager *statusmanager.StatusManager
	clusterInfo   *network.ClusterInfo
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

	// Canonicalize and validate NetworkAddonsConfig, finally render objects of requested components
	objs, err := r.renderObjects(networkAddonsConfig)
	if err != nil {
		// If failed, set NetworkAddonsConfig to failing and requeue
		r.statusManager.SetFailing(statusmanager.OperatorConfig, "FailedToRender", err.Error())
		return reconcile.Result{}, err
	}

	// Apply generated objects on Kubernetes API server
	err = r.applyObjects(networkAddonsConfig, objs)
	if err != nil {
		// If failed, set NetworkAddonsConfig to failing and requeue
		r.statusManager.SetFailing(statusmanager.OperatorConfig, "FailedToApply", err.Error())
		return reconcile.Result{}, err
	}

	// Track state of all deployed pods
	r.trackDeployedObjects(objs)

	// Everything went smooth, remove failures from NetworkAddonsConfig if there are any from
	// previous runs.
	r.statusManager.SetNotFailing(statusmanager.OperatorConfig)

	// From now on, r.podReconciler takes over NetworkAddonsConfig handling, it will track deployed
	// objects and set NetworkAddonsConfig.Status accordingly

	return reconcile.Result{}, nil
}

// Handle NetworkAddonsConfig object. Canonicalize, validate and finally render objects for all
// desired components. Please note that this function has side effects, it reads config map
// containing previously saved NetworkAddonsConfig and OpenShift's Network operator config.
func (r *ReconcileNetworkAddonsConfig) renderObjects(networkAddonsConfig *opv1alpha1.NetworkAddonsConfig) ([]*unstructured.Unstructured, error) {
	objs := []*unstructured.Unstructured{}

	// Convert to a canonicalized form
	network.Canonicalize(&networkAddonsConfig.Spec)

	// Read OpenShift network operator configuration (if exists)
	openshiftNetworkConfig, err := getOpenShiftNetworkConfig(context.TODO(), r.client)
	if err != nil {
		log.Printf("failed to load OpenShift NetworkConfig: %v", err)
		err = errors.Wrapf(err, "failed to load OpenShift NetworkConfig: %v", err)
		return objs, err
	}

	// Validate the configuration
	if err := network.Validate(&networkAddonsConfig.Spec, openshiftNetworkConfig); err != nil {
		log.Printf("failed to validate NetworkConfig.Spec: %v", err)
		err = errors.Wrapf(err, "failed to validate NetworkConfig.Spec: %v", err)
		return objs, err
	}

	// Retrieve the previously applied operator configuration
	prev, err := getAppliedConfiguration(context.TODO(), r.client, networkAddonsConfig.ObjectMeta.Name, r.namespace)
	if err != nil {
		log.Printf("failed to retrieve previously applied configuration: %v", err)
		err = errors.Wrapf(err, "failed to retrieve previously applied configuration: %v", err)
		return objs, err
	}

	// Fill all defaults explicitly
	if err := network.FillDefaults(&networkAddonsConfig.Spec, prev); err != nil {
		log.Printf("failed to fill defaults: %v", err)
		err = errors.Wrapf(err, "failed to fill defaults: %v", err)
		return objs, err
	}

	// Compare against previous applied configuration to see if this change
	// is safe.
	if prev != nil {
		// We may need to fill defaults here -- sort of as a poor-man's
		// upconversion scheme -- if we add additional fields to the config.
		err = network.IsChangeSafe(prev, &networkAddonsConfig.Spec)
		if err != nil {
			log.Printf("not applying unsafe change: %v", err)
			err = errors.Wrapf(err, "not applying unsafe change")
			return objs, err
		}
	}

	// Generate the objects
	objs, err = network.Render(&networkAddonsConfig.Spec, ManifestPath, openshiftNetworkConfig, r.clusterInfo)
	if err != nil {
		log.Printf("failed to render: %v", err)
		err = errors.Wrapf(err, "failed to render")
		return objs, err
	}

	// The first object we create should be the record of our applied configuration
	applied, err := appliedConfiguration(networkAddonsConfig, r.namespace)
	if err != nil {
		log.Printf("failed to render applied: %v", err)
		err = errors.Wrapf(err, "failed to render applied")
		return objs, err
	}
	objs = append([]*unstructured.Unstructured{applied}, objs...)

	return objs, nil
}

// Apply the objects to the cluster. Set their controller reference to NetworkAddonsConfig, so they
// are removed when NetworkAddonsConfig config is
func (r *ReconcileNetworkAddonsConfig) applyObjects(networkAddonsConfig *opv1alpha1.NetworkAddonsConfig, objs []*unstructured.Unstructured) error {
	for _, obj := range objs {
		// Mark the object to be GC'd if the owner is deleted
		if err := controllerutil.SetControllerReference(networkAddonsConfig, obj, r.scheme); err != nil {
			log.Printf("could not set reference for (%s) %s/%s: %v", obj.GroupVersionKind(), obj.GetNamespace(), obj.GetName(), err)
			err = errors.Wrapf(err, "could not set reference for (%s) %s/%s", obj.GroupVersionKind(), obj.GetNamespace(), obj.GetName())
			return err
		}

		// Apply all objects on apiserver
		if err := apply.ApplyObject(context.TODO(), r.client, obj); err != nil {
			log.Printf("could not apply (%s) %s/%s: %v", obj.GroupVersionKind(), obj.GetNamespace(), obj.GetName(), err)
			err = errors.Wrapf(err, "could not apply (%s) %s/%s", obj.GroupVersionKind(), obj.GetNamespace(), obj.GetName())
			return err
		}
	}

	return nil
}

// Track current state of Deployments and DaemonSets deployed by the operator. This is needed to
// keep state of NetworkAddonsConfig up-to-date, e.g. mark as Ready once all objects are successfully
// created
func (r *ReconcileNetworkAddonsConfig) trackDeployedObjects(objs []*unstructured.Unstructured) {
	daemonSets := []types.NamespacedName{}
	deployments := []types.NamespacedName{}

	for _, obj := range objs {
		if obj.GetAPIVersion() == "apps/v1" && obj.GetKind() == "DaemonSet" {
			daemonSets = append(daemonSets, types.NamespacedName{Namespace: obj.GetNamespace(), Name: obj.GetName()})
		} else if obj.GetAPIVersion() == "apps/v1" && obj.GetKind() == "Deployment" {
			deployments = append(deployments, types.NamespacedName{Namespace: obj.GetNamespace(), Name: obj.GetName()})
		}
	}

	r.statusManager.SetDaemonSets(daemonSets)
	r.statusManager.SetDeployments(deployments)

	allResources := []types.NamespacedName{}
	allResources = append(allResources, daemonSets...)
	allResources = append(allResources, deployments...)

	r.podReconciler.SetResources(allResources)
}

func getOpenShiftNetworkConfig(ctx context.Context, c k8sclient.Client) (*osv1.Network, error) {
	nc := &osv1.Network{}

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

// Check whether running on OpenShift 4 by looking for operator objects that has been introduced
// only in OpenShift 4
func isRunningOnOpenShift4(c kubernetes.Interface) (bool, error) {
	return isResourceAvailable(c, "configs", "imageregistry.operator.openshift.io", "v1")
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

func runtimeObjectToNetworkAddonsConfig(obj runtime.Object) (*opv1alpha1.NetworkAddonsConfig, error) {
	// convert the runtime.Object to unstructured.Unstructured
	unstructuredObj, err := runtime.DefaultUnstructuredConverter.ToUnstructured(obj)
	if err != nil {
		return nil, err
	}

	// convert unstructured.Unstructured to a NetworkAddonsConfig
	networkAddonsConfig := &opv1alpha1.NetworkAddonsConfig{}
	if err = runtime.DefaultUnstructuredConverter.FromUnstructured(unstructuredObj, networkAddonsConfig); err != nil {
		return nil, err
	}

	return networkAddonsConfig, nil
}
