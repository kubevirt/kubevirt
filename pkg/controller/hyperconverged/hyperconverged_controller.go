package hyperconverged

import (
	"context"

	sspv1 "github.com/MarSik/kubevirt-ssp-operator/pkg/apis/kubevirt/v1"
	networkaddons "github.com/kubevirt/cluster-network-addons-operator/pkg/apis/networkaddonsoperator/v1alpha1"
	networkaddonsnames "github.com/kubevirt/cluster-network-addons-operator/pkg/names"
	hcov1alpha1 "github.com/kubevirt/hyperconverged-cluster-operator/pkg/apis/hco/v1alpha1"
	kwebuis "github.com/kubevirt/web-ui-operator/pkg/apis/kubevirt/v1alpha1"
	cdi "kubevirt.io/containerized-data-importer/pkg/apis/core/v1alpha1"
	kubevirt "kubevirt.io/kubevirt/pkg/api/v1"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
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
)

var log = logf.Log.WithName("controller_hyperconverged")

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

	err = c.Watch(&source.Kind{Type: &kubevirt.KubeVirt{}}, &handler.EnqueueRequestForOwner{
		IsController: true,
		OwnerType:    &hcov1alpha1.HyperConverged{},
	})
	if err != nil {
		return err
	}

	err = c.Watch(&source.Kind{Type: &cdi.CDI{}}, &handler.EnqueueRequestForOwner{
		IsController: true,
		OwnerType:    &hcov1alpha1.HyperConverged{},
	})
	if err != nil {
		return err
	}

	err = c.Watch(&source.Kind{Type: &networkaddons.NetworkAddonsConfig{}}, &handler.EnqueueRequestForOwner{
		IsController: true,
		OwnerType:    &hcov1alpha1.HyperConverged{},
	})
	if err != nil {
		return err
	}

	// SSP needs to handle few types; SSP components are intentionally split in few CRs
	err = c.Watch(&source.Kind{Type: &sspv1.KubevirtCommonTemplatesBundle{}}, &handler.EnqueueRequestForOwner{
		IsController: true,
		OwnerType:    &hcov1alpha1.HyperConverged{},
	})
	if err != nil {
		return err
	}
	err = c.Watch(&source.Kind{Type: &sspv1.KubevirtNodeLabellerBundle{}}, &handler.EnqueueRequestForOwner{
		IsController: true,
		OwnerType:    &hcov1alpha1.HyperConverged{},
	})
	if err != nil {
		return err
	}
	err = c.Watch(&source.Kind{Type: &sspv1.KubevirtTemplateValidator{}}, &handler.EnqueueRequestForOwner{
		IsController: true,
		OwnerType:    &hcov1alpha1.HyperConverged{},
	})
	if err != nil {
		return err
	}
	err = c.Watch(&source.Kind{Type: &kwebuis.KWebUI{}}, &handler.EnqueueRequestForOwner{
		IsController: true,
		OwnerType:    &hcov1alpha1.HyperConverged{},
	})
	if err != nil {
		return err
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

	// Define KubeVirt's configuration ConfigMap first
	kvConfig := newKubeVirtConfigForCR(instance)
	kvConfig.ObjectMeta.Namespace = request.Namespace

	// Set HyperConverged instance as the owner and controller
	if err := controllerutil.SetControllerReference(instance, kvConfig, r.scheme); err != nil {
		return reconcile.Result{}, err
	}

	// Create the KubeVirt ConfigMap if it doesn't already exist
	result, err := manageComponentResource(kvConfig, "KubeVirtConfig", r.client)

	// KubeVirt ConfigMap failed to create, requeue
	if err != nil {
		return result, err
	}

	// Define a new KubeVirt object
	virtCR := newKubeVirtForCR(instance)
	virtCR.ObjectMeta.Namespace = request.Namespace

	// Set HyperConverged instance as the owner and controller
	if err := controllerutil.SetControllerReference(instance, virtCR, r.scheme); err != nil {
		return reconcile.Result{}, err
	}

	// Create the KubeVirt CR if it doesn't already exist
	result, err = manageComponentResource(virtCR, "KubeVirt", r.client)

	// KubeVirt failed to create, requeue
	if err != nil {
		return result, err
	}

	// Define a new CDI object
	cdiCR := newCDIForCR(instance)
	cdiCR.ObjectMeta.Namespace = request.Namespace

	// Set HyperConverged instance as the owner and controller
	if err := controllerutil.SetControllerReference(instance, cdiCR, r.scheme); err != nil {
		return reconcile.Result{}, err
	}

	// Create the CDI CR if it doesn't already exist
	result, err = manageComponentResource(cdiCR, "CDI", r.client)

	// CDI failed to create, requeue
	if err != nil {
		return result, err
	}

	// Define a new NetworkAddonsConfig object
	networkAddonsCR := newNetworkAddonsForCR(instance)

	// Set HyperConverged instance as the owner and controller
	if err := controllerutil.SetControllerReference(instance, networkAddonsCR, r.scheme); err != nil {
		return reconcile.Result{}, err
	}

	// Create the NetworkAddonsConfig CR if it doesn't already exist
	result, err = manageComponentResource(networkAddonsCR, "NetworkAddonsConfig", r.client)

	// NetworkAddonsConfig failed to create, requeue
	if err != nil {
		return result, err
	}

	// Define new SSP objects
	kubevirtCommonTemplatesBundleCR := newKubevirtCommonTemplateBundleForCR(instance)

	// Set HyperConverged instance as the owner and controller
	if err := controllerutil.SetControllerReference(instance, kubevirtCommonTemplatesBundleCR, r.scheme); err != nil {
		return reconcile.Result{}, err
	}

	// Create the KubevirtCommonTemplatesBundle CR if it doesn't already exist
	result, err = manageComponentResource(kubevirtCommonTemplatesBundleCR, "KubevirtCommonTemplatesBundle", r.client)
	// object failed to create, requeue
	if err != nil {
		return result, err
	}

	// Define a new kubevirtNodeLabellerBundleCR object
	kubevirtNodeLabellerBundleCR := newKubevirtNodeLabellerBundleForCR(instance)
	kubevirtNodeLabellerBundleCR.ObjectMeta.Namespace = request.Namespace

	// Set HyperConverged instance as the owner and controller
	if err := controllerutil.SetControllerReference(instance, kubevirtNodeLabellerBundleCR, r.scheme); err != nil {
		return reconcile.Result{}, err
	}

	// Create the KubevirtNodeLabellerBundle CR if it doesn't already exist
	result, err = manageComponentResource(kubevirtNodeLabellerBundleCR, "KubevirtNodeLabellerBundle", r.client)
	// object failed to create, requeue
	if err != nil {
		return result, err
	}

	// Define a new kubevirtNodeLabellerBundleCR object
	kubevirtTemplateValidatorCR := newKubevirtTemplateValidatorForCR(instance)
	kubevirtTemplateValidatorCR.ObjectMeta.Namespace = request.Namespace

	// Set HyperConverged instance as the owner and controller
	if err := controllerutil.SetControllerReference(instance, kubevirtTemplateValidatorCR, r.scheme); err != nil {
		return reconcile.Result{}, err
	}

	// Create the KubevirtTemplateValidator CR if it doesn't already exist
	result, err = manageComponentResource(kubevirtTemplateValidatorCR, "KubevirtTemplateValidator", r.client)
	// object failed to create, requeue
	if err != nil {
		return result, err
	}

	// Define a new KWebUI object
	kwebuiCR := newKWebUIForCR(instance)

	// Set HyperConverged instance as the owner and controller
	if err := controllerutil.SetControllerReference(instance, kwebuiCR, r.scheme); err != nil {
		return reconcile.Result{}, err
	}

	// Create the KWebUI CR if it doesn't already exist
	result, err = manageComponentResource(kwebuiCR, "KWebUI", r.client)

	// KWebUI failed to create, requeue
	if err != nil {
		return result, err
	}
	return result, nil
}

func manageComponentResource(o runtime.Object, kind string, c client.Client) (reconcile.Result, error) {
	err := c.Create(context.TODO(), o)
	if err != nil && errors.IsAlreadyExists(err) {
		log.Info("Skip reconcile: resource already exists", "Kind", kind)
		return reconcile.Result{}, nil
	} else if err != nil {
		return reconcile.Result{}, err
	}

	log.Info("Creating new resource", "Kind", kind)
	return reconcile.Result{}, nil
}

func newKubeVirtConfigForCR(cr *hcov1alpha1.HyperConverged) *corev1.ConfigMap {
	labels := map[string]string{
		"app": cr.Name,
	}
	return &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:   "kubevirt-config",
			Labels: labels,
		},
		Data: map[string]string{
			"feature-gates": "DataVolumes,SRIOV,LiveMigration,CPUManager,CPUNodeDiscovery",
		},
	}
}

// newKubeVirtForCR returns a KubeVirt CR
func newKubeVirtForCR(cr *hcov1alpha1.HyperConverged) *kubevirt.KubeVirt {
	labels := map[string]string{
		"app": cr.Name,
	}
	return &kubevirt.KubeVirt{
		ObjectMeta: metav1.ObjectMeta{
			Name:   "kubevirt-" + cr.Name,
			Labels: labels,
		},
	}
}

// newCDIForCr returns a CDI CR
func newCDIForCR(cr *hcov1alpha1.HyperConverged) *cdi.CDI {
	labels := map[string]string{
		"app": cr.Name,
	}
	return &cdi.CDI{
		ObjectMeta: metav1.ObjectMeta{
			Name:   "cdi-" + cr.Name,
			Labels: labels,
		},
	}
}

// newNetworkAddonsForCR returns a NetworkAddonsConfig CR
func newNetworkAddonsForCR(cr *hcov1alpha1.HyperConverged) *networkaddons.NetworkAddonsConfig {
	labels := map[string]string{
		"app": cr.Name,
	}
	return &networkaddons.NetworkAddonsConfig{
		ObjectMeta: metav1.ObjectMeta{
			Name:   networkaddonsnames.OPERATOR_CONFIG,
			Labels: labels,
		},
		Spec: networkaddons.NetworkAddonsConfigSpec{
			Multus:      &networkaddons.Multus{},
			LinuxBridge: &networkaddons.LinuxBridge{},
			KubeMacPool: &networkaddons.KubeMacPool{},
		},
	}
}

func newKubevirtCommonTemplateBundleForCR(cr *hcov1alpha1.HyperConverged) *sspv1.KubevirtCommonTemplatesBundle {
	labels := map[string]string{
		"app": cr.Name,
	}
	return &sspv1.KubevirtCommonTemplatesBundle{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "common-templates-" + cr.Name,
			Labels:    labels,
			Namespace: "openshift",
		},
	}
}

func newKubevirtNodeLabellerBundleForCR(cr *hcov1alpha1.HyperConverged) *sspv1.KubevirtNodeLabellerBundle {
	labels := map[string]string{
		"app": cr.Name,
	}
	return &sspv1.KubevirtNodeLabellerBundle{
		ObjectMeta: metav1.ObjectMeta{
			Name:   "node-labeller-" + cr.Name,
			Labels: labels,
		},
	}
}

func newKubevirtTemplateValidatorForCR(cr *hcov1alpha1.HyperConverged) *sspv1.KubevirtTemplateValidator {
	labels := map[string]string{
		"app": cr.Name,
	}
	return &sspv1.KubevirtTemplateValidator{
		ObjectMeta: metav1.ObjectMeta{
			Name:   "template-validator-" + cr.Name,
			Labels: labels,
		},
	}
}

func newKWebUIForCR(cr *hcov1alpha1.HyperConverged) *kwebuis.KWebUI {
	labels := map[string]string{
		"app": cr.Name,
	}
	return &kwebuis.KWebUI{
		ObjectMeta: metav1.ObjectMeta{
			Name:   "kubevirt-web-ui-" + cr.Name,
			Labels: labels,
		},
		// Missing CR values will be set via ENV variables of the web-ui-operator
		Spec: kwebuis.KWebUISpec{
			OpenshiftMasterDefaultSubdomain: cr.Spec.KWebUIMasterDefaultSubdomain, // set if provided, otherwise keep empty
			PublicMasterHostname:            cr.Spec.KWebUIPublicMasterHostname,   // set if provided, otherwise keep empty
			Version:                         "automatic",                          // special value to determine version dynamically from env variables; empty or missing value is reserved for deprovision
		},
	}
}
