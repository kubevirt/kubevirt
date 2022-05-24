package operands

import (
	"errors"
	"os"
	"reflect"

	monitoringv1 "github.com/prometheus-operator/prometheus-operator/pkg/apis/monitoring/v1"
	corev1 "k8s.io/api/core/v1"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/intstr"
	"sigs.k8s.io/controller-runtime/pkg/client"

	hcov1beta1 "github.com/kubevirt/hyperconverged-cluster-operator/api/v1beta1"
	"github.com/kubevirt/hyperconverged-cluster-operator/controllers/common"
	"github.com/kubevirt/hyperconverged-cluster-operator/pkg/util"
	hcoutil "github.com/kubevirt/hyperconverged-cluster-operator/pkg/util"
)

const (
	operatorPortName    = "http-metrics"
	defaultOperatorName = "hyperconverged-cluster-operator"
	operatorNameEnv     = "OPERATOR_NAME"
	metricsSuffix       = "-operator-metrics"
)

// NewMetricsService creates service for prometheus metrics
func NewMetricsService(hc *hcov1beta1.HyperConverged) *corev1.Service {
	// Add to the below struct any other metrics ports you want to expose.
	servicePorts := []corev1.ServicePort{
		{Port: hcoutil.MetricsPort, Name: operatorPortName, Protocol: v1.ProtocolTCP, TargetPort: intstr.IntOrString{Type: intstr.Int, IntVal: hcoutil.MetricsPort}},
	}
	operatorName := defaultOperatorName
	val, ok := os.LookupEnv(operatorNameEnv)
	if ok && val != "" {
		operatorName = val
	}
	labelSelect := map[string]string{"name": operatorName}

	spec := corev1.ServiceSpec{
		Ports:    servicePorts,
		Selector: labelSelect,
	}

	return &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      hc.Name + metricsSuffix,
			Labels:    getLabels(hc, hcoutil.AppComponentMonitoring),
			Namespace: hc.Namespace,
		},
		Spec: spec,
	}
}

type metricsServiceMonitorHandler genericOperand

func newMetricsServiceMonitorHandler(Client client.Client, Scheme *runtime.Scheme) *metricsServiceMonitorHandler {
	return &metricsServiceMonitorHandler{
		Client:                 Client,
		Scheme:                 Scheme,
		crType:                 "ServiceMonitor",
		removeExistingOwner:    false,
		setControllerReference: true,
		hooks:                  &metricsServiceMonitorHooks{},
	}
}

type metricsServiceMonitorHooks struct{}

func (h metricsServiceMonitorHooks) getFullCr(hc *hcov1beta1.HyperConverged) (client.Object, error) {
	return NewServiceMonitor(hc, hc.Namespace), nil
}
func (h metricsServiceMonitorHooks) getEmptyCr() client.Object {
	return &monitoringv1.ServiceMonitor{}
}
func (h metricsServiceMonitorHooks) getObjectMeta(cr runtime.Object) *metav1.ObjectMeta {
	return &cr.(*monitoringv1.ServiceMonitor).ObjectMeta
}
func (h metricsServiceMonitorHooks) reset( /* No implementation */ ) {}

func (h *metricsServiceMonitorHooks) updateCr(req *common.HcoRequest, Client client.Client, exists runtime.Object, required runtime.Object) (bool, bool, error) {
	monitor, ok1 := required.(*monitoringv1.ServiceMonitor)
	found, ok2 := exists.(*monitoringv1.ServiceMonitor)
	if !ok1 || !ok2 {
		return false, false, errors.New("can't convert to ServiceMonitor")
	}
	if !reflect.DeepEqual(found.Spec, monitor.Spec) ||
		!reflect.DeepEqual(found.Labels, monitor.Labels) {
		if req.HCOTriggered {
			req.Logger.Info("Updating existing metrics ServiceMonitor Spec to new opinionated values")
		} else {
			req.Logger.Info("Reconciling an externally updated metrics ServiceMonitor Spec to its opinionated values")
		}
		monitor.Spec.DeepCopyInto(&found.Spec)
		util.DeepCopyLabels(&monitor.ObjectMeta, &found.ObjectMeta)
		err := Client.Update(req.Ctx, found)
		if err != nil {
			return false, false, err
		}
		return true, !req.HCOTriggered, nil
	}
	return false, false, nil
}

func (h metricsServiceMonitorHooks) justBeforeComplete(_ *common.HcoRequest) { /* no implementation */ }

// NewServiceMonitor creates ServiceMonitor resource to expose metrics endpoint
func NewServiceMonitor(hc *hcov1beta1.HyperConverged, namespace string) *monitoringv1.ServiceMonitor {
	labels := getLabels(hc, hcoutil.AppComponentMonitoring)
	spec := monitoringv1.ServiceMonitorSpec{
		Selector: metav1.LabelSelector{
			MatchLabels: labels,
		},
		Endpoints: []monitoringv1.Endpoint{{Port: operatorPortName}},
	}

	return &monitoringv1.ServiceMonitor{
		ObjectMeta: metav1.ObjectMeta{
			Name:      hc.Name + metricsSuffix,
			Labels:    labels,
			Namespace: namespace,
		},
		Spec: spec,
	}
}
