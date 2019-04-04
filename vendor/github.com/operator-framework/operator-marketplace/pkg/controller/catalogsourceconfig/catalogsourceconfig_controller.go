package catalogsourceconfig

import (
	"context"

	marketplace "github.com/operator-framework/operator-marketplace/pkg/apis/operators/v1"
	catalogsourceconfighandler "github.com/operator-framework/operator-marketplace/pkg/catalogsourceconfig"
	"github.com/operator-framework/operator-marketplace/pkg/status"
	log "github.com/sirupsen/logrus"
	"k8s.io/apimachinery/pkg/api/errors"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/config"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/source"
)

// Add creates a new CatalogSourceConfig Controller and adds it to the Manager. The Manager will set fields on the Controller
// and Start it when the Manager is Started.
func Add(mgr manager.Manager) error {
	reconciler, err := newReconciler(mgr)
	if err != nil {
		return err
	}
	return add(mgr, reconciler)
}

// newReconciler returns a new reconcile.Reconciler
func newReconciler(mgr manager.Manager) (reconcile.Reconciler, error) {
	// The default client serves read requests from the cache which contains
	// objects only from the namespace the operator is watching. Given we need
	// to query other namespaces for CatalogSources, we create our own client
	// and pass it the manager's scheme which has all our registered types
	cfg, err := config.GetConfig()
	if err != nil {
		return nil, err
	}

	client, err := client.New(cfg, client.Options{Scheme: mgr.GetScheme()})
	if err != nil {
		return nil, err
	}

	return &ReconcileCatalogSourceConfig{
		CatalogSourceConfigHandler: catalogsourceconfighandler.NewHandler(mgr, client),
		client:                     client,
	}, nil
}

// add adds a new Controller to mgr with r as the reconcile.Reconciler
func add(mgr manager.Manager, r reconcile.Reconciler) error {
	// Create a new controller
	c, err := controller.New("catalogsourceconfig-controller", mgr, controller.Options{Reconciler: r})
	if err != nil {
		return err
	}

	// Watch for changes to primary resource CatalogSourceConfig
	err = c.Watch(&source.Kind{Type: &marketplace.CatalogSourceConfig{}}, &handler.EnqueueRequestForObject{})
	if err != nil {
		return err
	}

	return nil
}

var _ reconcile.Reconciler = &ReconcileCatalogSourceConfig{}

// ReconcileCatalogSourceConfig reconciles a CatalogSourceConfig object
type ReconcileCatalogSourceConfig struct {
	CatalogSourceConfigHandler catalogsourceconfighandler.Handler
	client                     client.Client
}

// Reconcile reads that state of the cluster for a CatalogSourceConfig object and makes changes based on the state read
// and what is in the CatalogSourceConfig.Spec
// Note:
// The Controller will requeue the Request to be processed again if the returned error is non-nil or
// Result.Requeue is true, otherwise upon completion it will remove the work from the queue.
func (r *ReconcileCatalogSourceConfig) Reconcile(request reconcile.Request) (reconcile.Result, error) {
	log.Printf("Reconciling CatalogSourceConfig %s/%s\n", request.Namespace, request.Name)
	// Reconcile kicked off, message Sync Channel
	status.SendSyncMessage(nil)

	// If err is not nil at the end of this function message the syncCh channel
	var err error
	defer func() {
		if err != nil {
			status.SendSyncMessage(err)
		}
	}()
	// Fetch the CatalogSourceConfig instance
	instance := &marketplace.CatalogSourceConfig{}
	err = r.client.Get(context.TODO(), request.NamespacedName, instance)
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

	// Needed because sdk does not get the gvk
	instance.EnsureGVK()

	err = r.CatalogSourceConfigHandler.Handle(context.TODO(), instance)
	if err != nil {
		return reconcile.Result{}, err
	}

	return reconcile.Result{}, err
}
