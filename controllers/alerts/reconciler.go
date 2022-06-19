package alerts

import (
	"context"
	"fmt"
	"reflect"

	"github.com/go-logr/logr"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/utils/pointer"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/kubevirt/hyperconverged-cluster-operator/pkg/metrics"

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

func (r *MonitoringReconciler) Reconcile(req *common.HcoRequest) error {
	if r == nil {
		return nil
	}

	if err := reconcileNamespace(req.Ctx, r.client, r.namespace, req.Logger); err != nil {
		return err
	}

	objects := make([]client.Object, len(r.reconcilers))

	for i, rc := range r.reconcilers {
		obj, err := r.ReconcileOneResource(req, rc)
		if err != nil {
			return err
		} else if obj != nil {
			objects[i] = obj
		}
	}

	r.latestObjects = objects
	return nil
}

func (r *MonitoringReconciler) ReconcileOneResource(req *common.HcoRequest, reconciler MetricReconciler) (client.Object, error) {
	if r == nil {
		return nil, nil // not initialized (not running on openshift). do nothing
	}

	req.Logger.V(5).Info(fmt.Sprintf("Reconciling the %s", reconciler.Kind()))

	existing := reconciler.EmptyObject()

	req.Logger.V(5).Info(fmt.Sprintf("Reading the current %s", reconciler.Kind()))
	err := r.client.Get(req.Ctx, client.ObjectKey{Namespace: r.namespace, Name: reconciler.ResourceName()}, existing)

	if err != nil {
		if errors.IsNotFound(err) {
			req.Logger.Info(fmt.Sprintf("Can't find the %s; creating a new one", reconciler.Kind()), "name", reconciler.ResourceName())
			required := reconciler.GetFullResource()
			err := r.client.Create(req.Ctx, required)
			if err != nil {
				req.Logger.Error(err, fmt.Sprintf("failed to create %s", reconciler.Kind()))
				r.eventEmitter.EmitEvent(nil, corev1.EventTypeWarning, "UnexpectedError", fmt.Sprintf("failed to create the %s %s", reconciler.ResourceName(), reconciler.Kind()))
				return nil, err
			}
			req.Logger.Info(fmt.Sprintf("successfully created the %s", reconciler.Kind()), "name", reconciler.ResourceName())
			r.eventEmitter.EmitEvent(required, corev1.EventTypeNormal, "Created", fmt.Sprintf("Created %s %s", reconciler.Kind(), reconciler.ResourceName()))

			return required, nil
		}

		req.Logger.Error(err, "unexpected error while reading the PrometheusRule")
		return nil, err
	}

	resource, updated, err := reconciler.UpdateExistingResource(req.Ctx, r.client, existing, req.Logger)
	if err != nil {
		r.eventEmitter.EmitEvent(nil, corev1.EventTypeWarning, "UnexpectedError", fmt.Sprintf("failed to update the %s %s", reconciler.ResourceName(), reconciler.Kind()))
	} else if updated {
		if req.HCOTriggered {
			r.eventEmitter.EmitEvent(nil, corev1.EventTypeNormal, "Updated", fmt.Sprintf("Updated %s %s", reconciler.Kind(), reconciler.ResourceName()))
		} else {
			r.eventEmitter.EmitEvent(nil, corev1.EventTypeWarning, "Overwritten", fmt.Sprintf("Overwritten %s %s", reconciler.Kind(), reconciler.ResourceName()))
			err := metrics.HcoMetrics.IncOverwrittenModifications(reconciler.Kind(), reconciler.ResourceName())
			if err != nil {
				req.Logger.Error(err, "couldn't update 'OverwrittenModifications' metric")
			}
		}
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
