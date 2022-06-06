package alerts

import (
	"context"
	"fmt"
	"reflect"
	"sync"
	"time"

	"github.com/go-logr/logr"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/utils/pointer"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/kubevirt/hyperconverged-cluster-operator/controllers/common"
	hcoutil "github.com/kubevirt/hyperconverged-cluster-operator/pkg/util"
)

type MetricReconciler interface {
	Kind() string
	ResourceName() string
	GetFullResource() client.Object
	EmptyObject() client.Object
	UpdateExistingResource(context.Context, client.Client, client.Object, logr.Logger) (client.Object, bool, error)
}

type MonitoringReconciler struct {
	reconcilers   []MetricReconciler
	scheme        *runtime.Scheme
	latestObjects []client.Object
	client        client.Client
	namespace     string
	eventEmitter  hcoutil.EventEmitter
}

func NewMonitoringReconciler(ci hcoutil.ClusterInfo, cl client.Client, ee hcoutil.EventEmitter, scheme *runtime.Scheme) *MonitoringReconciler {
	deployment := ci.GetDeployment()
	namespace := deployment.Namespace
	owner := getDeploymentReference(deployment)

	return &MonitoringReconciler{
		reconcilers: []MetricReconciler{
			newAlertRuleReconciler(namespace, owner),
			newRoleReconciler(namespace, owner),
			newRoleBindingReconciler(namespace, owner),
			newMetricServiceReconciler(namespace, owner),
			newServiceMonitorReconciler(namespace, owner),
		},
		scheme:       scheme,
		client:       cl,
		namespace:    namespace,
		eventEmitter: ee,
	}
}

func (r *MonitoringReconciler) Reconcile(ctx context.Context, logger logr.Logger) error {
	if r == nil {
		return nil
	}

	toCtx, cancel := context.WithTimeout(ctx, time.Second*10)
	defer cancel()

	errorCh := make(chan error)
	done := make(chan struct{})

	defer close(errorCh)
	defer close(done)

	wg := &sync.WaitGroup{}
	wg.Add(len(r.reconcilers))
	go func(w *sync.WaitGroup) {
		w.Wait()
		done <- struct{}{}
	}(wg)

	objects := make([]client.Object, len(r.reconcilers))

	for i, rc := range r.reconcilers {
		go func(ctx context.Context, rc MetricReconciler, w *sync.WaitGroup, index int, logger logr.Logger) {
			defer w.Done()

			obj, err := r.ReconcileOneResource(toCtx, rc, logger)
			if err != nil {
				errorCh <- err
			} else if obj != nil {
				objects[index] = obj
			}
		}(toCtx, rc, wg, i, logger)
	}

	var err error
	for {
		select {
		case <-toCtx.Done():
			err = toCtx.Err()

		case <-done:
			for i := 0; i < len(errorCh); i++ {
				err = <-errorCh
			}
			if err != nil {
				return err
			}

			r.latestObjects = objects
			return nil

		case e := <-errorCh:
			err = e
		}
	}
}

func (r *MonitoringReconciler) ReconcileOneResource(ctx context.Context, reconciler MetricReconciler, logger logr.Logger) (client.Object, error) {
	if r == nil {
		return nil, nil // not initialized (not running on openshift). do nothing
	}

	logger.V(5).Info(fmt.Sprintf("Reconciling the %s", reconciler.Kind()))

	existing := reconciler.EmptyObject()

	logger.V(5).Info(fmt.Sprintf("Reading the current %s", reconciler.Kind()))
	err := r.client.Get(ctx, client.ObjectKey{Namespace: r.namespace, Name: reconciler.ResourceName()}, existing)

	if err != nil {
		if errors.IsNotFound(err) {
			logger.Info(fmt.Sprintf("Can't find the %s; creating a new one", reconciler.Kind()), "name", reconciler.ResourceName())
			required := reconciler.GetFullResource()
			err := r.client.Create(ctx, required)
			if err != nil {
				logger.Error(err, "failed to create PrometheusRule")
				r.eventEmitter.EmitEvent(nil, corev1.EventTypeWarning, "UnexpectedError", fmt.Sprintf("failed to create the %s %s", reconciler.ResourceName(), reconciler.Kind()))
				return nil, err
			}
			logger.Info(fmt.Sprintf("successfully created the %s", reconciler.Kind()), "name", reconciler.ResourceName())
			r.eventEmitter.EmitEvent(required, corev1.EventTypeNormal, "Created", fmt.Sprintf("Created %s %s", reconciler.Kind(), reconciler.ResourceName()))

			return required, nil
		}

		logger.Error(err, "unexpected error while reading the PrometheusRule")
		return nil, err
	}

	resource, updated, err := reconciler.UpdateExistingResource(ctx, r.client, existing, logger)
	if err != nil {
		r.eventEmitter.EmitEvent(nil, corev1.EventTypeWarning, "UnexpectedError", fmt.Sprintf("failed to update the %s %s", reconciler.ResourceName(), reconciler.Kind()))
	} else if updated {
		r.eventEmitter.EmitEvent(nil, corev1.EventTypeNormal, "Updated", fmt.Sprintf("Updated %s %s", reconciler.Kind(), reconciler.ResourceName()))
	}

	return resource, err
}

func (r *MonitoringReconciler) UpdateRelatedObjects(req *common.HcoRequest) error {
	if r == nil {
		return nil // not initialized (not running on openshift). do nothing
	}

	wasChanged := false
	for _, obj := range r.latestObjects {
		changed, err := hcoutil.AddCrToTheRelatedObjectList(&req.Instance.Status.RelatedObjects, obj, r.scheme)
		if err != nil {
			return err
		}

		wasChanged = wasChanged || changed
	}

	if wasChanged {
		req.StatusDirty = true
	}

	return nil
}

func getDeploymentReference(deployment *appsv1.Deployment) metav1.OwnerReference {
	return metav1.OwnerReference{
		APIVersion:         appsv1.SchemeGroupVersion.String(),
		Kind:               "Deployment",
		Name:               deployment.GetName(),
		UID:                deployment.GetUID(),
		BlockOwnerDeletion: pointer.BoolPtr(false),
		Controller:         pointer.BoolPtr(false),
	}
}

// update the labels and the ownerReferences in a metric resource
// return true if something was changed
func updateCommonDetails(required, existing *metav1.ObjectMeta) bool {
	if reflect.DeepEqual(required.OwnerReferences, existing.OwnerReferences) &&
		reflect.DeepEqual(required.Labels, existing.Labels) {
		return false
	}

	hcoutil.DeepCopyLabels(required, existing)
	if reqLen := len(required.OwnerReferences); reqLen > 0 {
		refs := make([]metav1.OwnerReference, reqLen)
		for i, ref := range required.OwnerReferences {
			ref.DeepCopyInto(&refs[i])
		}
		existing.OwnerReferences = refs
	} else {
		existing.OwnerReferences = nil
	}

	return true
}
