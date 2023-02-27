package nodeconfig

import (
	"context"

	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/event"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/source"

	operatorhandler "github.com/operator-framework/operator-lib/handler"

	hcoutil "github.com/kubevirt/hyperconverged-cluster-operator/pkg/util"
)

var log = logf.Log.WithName("controller_nodeconfig")

type ReconcileNodeConfig struct {
	client        client.Client
	kubeletConfig *kubeletConfig
}

func RegisterReconciler(mgr manager.Manager) error {
	r, err := newReconciler(mgr)
	if err != nil {
		return err
	}

	return add(mgr, r)
}

func newReconciler(mgr manager.Manager) (reconcile.Reconciler, error) {
	node := corev1.Node{}
	restClient, err := hcoutil.GetRESTClientFor(&node, mgr.GetConfig(), mgr.GetScheme())
	if err != nil {
		return nil, err
	}

	r := &ReconcileNodeConfig{
		client:        mgr.GetClient(),
		kubeletConfig: newKubeletConfig(restClient),
	}

	return r, nil
}

func onlyConfigurationOrStatusImagesChangedPredicate() predicate.Predicate {
	return predicate.Funcs{
		UpdateFunc: func(e event.UpdateEvent) bool {
			oldNode := e.ObjectOld.(*corev1.Node)
			newNode := e.ObjectNew.(*corev1.Node)

			diffGen := oldNode.GetGeneration() != newNode.GetGeneration()
			diffImageNum := len(oldNode.Status.Images) != len(newNode.Status.Images)

			return diffGen || diffImageNum
		},
		DeleteFunc: func(e event.DeleteEvent) bool {
			return false
		},
	}
}

func add(mgr manager.Manager, r reconcile.Reconciler) error {
	c, err := controller.New("nodeconfig-controller", mgr, controller.Options{Reconciler: r})
	if err != nil {
		return err
	}

	err = c.Watch(
		&source.Kind{Type: &corev1.Node{}},
		&operatorhandler.InstrumentedEnqueueRequestForObject{},
		onlyConfigurationOrStatusImagesChangedPredicate(),
	)
	if err != nil {
		return err
	}

	return nil
}

func (r ReconcileNodeConfig) Reconcile(ctx context.Context, request reconcile.Request) (reconcile.Result, error) {
	logger := log.WithValues("Request.Namespace", request.Namespace, "Request.Name", request.Name)

	node := corev1.Node{}
	err := r.client.Get(ctx, client.ObjectKey{Name: request.Name}, &node)
	if err != nil {
		return reconcile.Result{}, err
	}

	err = r.kubeletConfig.setNodeNumberOfImagesMetrics(logger, &node)
	if err != nil {
		return reconcile.Result{}, err
	}

	err = r.kubeletConfig.setNodeStatusMaxImagesMetrics(ctx, logger, &node)
	if err != nil {
		return reconcile.Result{}, err
	}

	return reconcile.Result{}, nil
}
