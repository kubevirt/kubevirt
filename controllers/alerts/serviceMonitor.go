package alerts

import (
	"context"
	"reflect"

	"github.com/go-logr/logr"
	monitoringv1 "github.com/prometheus-operator/prometheus-operator/pkg/apis/monitoring/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	hcoutil "github.com/kubevirt/hyperconverged-cluster-operator/pkg/util"
)

type serviceMonitorReconciler struct {
	theServiceMonitor *monitoringv1.ServiceMonitor
}

func newServiceMonitorReconciler(namespace string, owner metav1.OwnerReference) *serviceMonitorReconciler {
	return &serviceMonitorReconciler{theServiceMonitor: NewServiceMonitor(namespace, owner)}
}

func (r serviceMonitorReconciler) Kind() string {
	return monitoringv1.ServiceMonitorsKind
}

func (r serviceMonitorReconciler) ResourceName() string {
	return serviceName
}

func (r serviceMonitorReconciler) GetFullResource() client.Object {
	return r.theServiceMonitor.DeepCopy()
}

func (r serviceMonitorReconciler) EmptyObject() client.Object {
	return &monitoringv1.ServiceMonitor{}
}

func (r serviceMonitorReconciler) UpdateExistingResource(ctx context.Context, cl client.Client, resource client.Object, logger logr.Logger) (client.Object, bool, error) {
	found := resource.(*monitoringv1.ServiceMonitor)
	modified := false
	if !reflect.DeepEqual(found.Spec, r.theServiceMonitor.Spec) {
		r.theServiceMonitor.Spec.DeepCopyInto(&found.Spec)
		modified = true
	}

	modified = updateCommonDetails(&r.theServiceMonitor.ObjectMeta, &found.ObjectMeta) || modified

	if modified {
		err := cl.Update(ctx, found)
		if err != nil {
			logger.Error(err, "failed to update the ServiceMonitor")
			return nil, false, err
		}
		logger.Info("successfully updated the ServiceMonitor")
	}
	return found, modified, nil
}

func NewServiceMonitor(namespace string, owner metav1.OwnerReference) *monitoringv1.ServiceMonitor {
	labels := hcoutil.GetLabels(hcoutil.HyperConvergedName, hcoutil.AppComponentMonitoring)
	spec := monitoringv1.ServiceMonitorSpec{
		Selector: metav1.LabelSelector{
			MatchLabels: labels,
		},
		Endpoints: []monitoringv1.Endpoint{{Port: operatorPortName}},
	}

	return &monitoringv1.ServiceMonitor{
		TypeMeta: metav1.TypeMeta{
			APIVersion: monitoringv1.SchemeGroupVersion.String(),
			Kind:       monitoringv1.ServiceMonitorsKind,
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:            serviceName,
			Labels:          labels,
			Namespace:       namespace,
			OwnerReferences: []metav1.OwnerReference{owner},
		},
		Spec: spec,
	}
}
