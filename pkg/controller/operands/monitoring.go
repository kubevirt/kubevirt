package operands

import (
	"errors"
	"os"
	"reflect"

	monitoringv1 "github.com/coreos/prometheus-operator/pkg/apis/monitoring/v1"
	hcov1beta1 "github.com/kubevirt/hyperconverged-cluster-operator/pkg/apis/hco/v1beta1"
	"github.com/kubevirt/hyperconverged-cluster-operator/pkg/controller/common"
	hcoutil "github.com/kubevirt/hyperconverged-cluster-operator/pkg/util"
	conditionsv1 "github.com/openshift/custom-resource-status/conditions/v1"
	corev1 "k8s.io/api/core/v1"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/intstr"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	operatorPortName     = "http-metrics"
	defaultOperatorName  = "hyperconverged-cluster-operator"
	operatorNameEnv      = "OPERATOR_NAME"
	metricsSuffix        = "-operator-metrics"
	alertRuleGroup       = "kubevirt.hyperconverged.rules"
	outOfBandUpdateAlert = "Kubevirt_hyperconverged_out-of-band_CR_modification_detected"
)

type metricsServiceHandler genericOperand

func newMetricsServiceHandler(Client client.Client, Scheme *runtime.Scheme) *metricsServiceHandler {
	return &metricsServiceHandler{
		Client:                 Client,
		Scheme:                 Scheme,
		crType:                 "MetricsService",
		removeExistingOwner:    false,
		setControllerReference: true,
		isCr:                   false,
		hooks:                  &metricsServiceHooks{},
	}
}

type metricsServiceHooks struct{}

func (h metricsServiceHooks) getFullCr(hc *hcov1beta1.HyperConverged) runtime.Object {
	return NewMetricsService(hc, hc.Namespace)
}
func (h metricsServiceHooks) getEmptyCr() runtime.Object                              { return &corev1.Service{} }
func (h metricsServiceHooks) validate() error                                         { return nil }
func (h metricsServiceHooks) postFound(*common.HcoRequest, runtime.Object) error      { return nil }
func (h metricsServiceHooks) getConditions(_ runtime.Object) []conditionsv1.Condition { return nil }
func (h metricsServiceHooks) checkComponentVersion(_ runtime.Object) bool             { return true }
func (h metricsServiceHooks) getObjectMeta(cr runtime.Object) *metav1.ObjectMeta {
	return &cr.(*corev1.Service).ObjectMeta
}

func (h *metricsServiceHooks) updateCr(req *common.HcoRequest, Client client.Client, exists runtime.Object, required runtime.Object) (bool, bool, error) {
	service, ok1 := required.(*corev1.Service)
	found, ok2 := exists.(*corev1.Service)
	if !ok1 || !ok2 {
		return false, false, errors.New("can't convert to Service")
	}

	if !reflect.DeepEqual(found.Spec.Ports, service.Spec.Ports) || !reflect.DeepEqual(found.Spec.Selector, service.Spec.Selector) {
		if req.HCOTriggered {
			req.Logger.Info("Updating existing metrics Service Spec to new opinionated values")
		} else {
			req.Logger.Info("Reconciling an externally updated metrics Service Spec to its opinionated values")
		}
		found.Spec.Ports = service.Spec.Ports
		found.Spec.Selector = service.Spec.Selector
		err := Client.Update(req.Ctx, found)
		if err != nil {
			return false, false, err
		}
		return true, !req.HCOTriggered, nil
	}
	return false, false, nil
}

// NewMetricsService creates service for prometheus metrics
func NewMetricsService(hc *hcov1beta1.HyperConverged, namespace string) *corev1.Service {
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
			Labels:    getLabels(hc),
			Namespace: namespace,
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
		isCr:                   false,
		hooks:                  &metricsServiceMonitorHooks{},
	}
}

type metricsServiceMonitorHooks struct{}

func (h metricsServiceMonitorHooks) getFullCr(hc *hcov1beta1.HyperConverged) runtime.Object {
	return NewServiceMonitor(hc, hc.Namespace)
}
func (h metricsServiceMonitorHooks) getEmptyCr() runtime.Object {
	return &monitoringv1.ServiceMonitor{}
}
func (h metricsServiceMonitorHooks) validate() error                                    { return nil }
func (h metricsServiceMonitorHooks) postFound(*common.HcoRequest, runtime.Object) error { return nil }
func (h metricsServiceMonitorHooks) getConditions(_ runtime.Object) []conditionsv1.Condition {
	return nil
}
func (h metricsServiceMonitorHooks) checkComponentVersion(_ runtime.Object) bool { return true }
func (h metricsServiceMonitorHooks) getObjectMeta(cr runtime.Object) *metav1.ObjectMeta {
	return &cr.(*monitoringv1.ServiceMonitor).ObjectMeta
}

func (h *metricsServiceMonitorHooks) updateCr(req *common.HcoRequest, Client client.Client, exists runtime.Object, required runtime.Object) (bool, bool, error) {
	monitor, ok1 := required.(*monitoringv1.ServiceMonitor)
	found, ok2 := exists.(*monitoringv1.ServiceMonitor)
	if !ok1 || !ok2 {
		return false, false, errors.New("can't convert to ServiceMonitor")
	}
	if !reflect.DeepEqual(found.Spec, monitor.Spec) {
		if req.HCOTriggered {
			req.Logger.Info("Updating existing metrics ServiceMonitor Spec to new opinionated values")
		} else {
			req.Logger.Info("Reconciling an externally updated metrics ServiceMonitor Spec to its opinionated values")
		}
		monitor.Spec.DeepCopyInto(&found.Spec)
		err := Client.Update(req.Ctx, found)
		if err != nil {
			return false, false, err
		}
		return true, !req.HCOTriggered, nil
	}
	return false, false, nil
}

// NewServiceMonitor creates ServiceMonitor resource to expose metrics endpoint
func NewServiceMonitor(hc *hcov1beta1.HyperConverged, namespace string) *monitoringv1.ServiceMonitor {
	labels := getLabels(hc)

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

type monitoringPrometheusRuleHandler genericOperand

func newMonitoringPrometheusRuleHandler(Client client.Client, Scheme *runtime.Scheme) *monitoringPrometheusRuleHandler {
	return &monitoringPrometheusRuleHandler{
		Client:                 Client,
		Scheme:                 Scheme,
		crType:                 "PrometheusRule",
		removeExistingOwner:    false,
		setControllerReference: true,
		isCr:                   false,
		hooks:                  &prometheusRuleHooks{},
	}
}

type prometheusRuleHooks struct{}

func (h prometheusRuleHooks) getFullCr(hc *hcov1beta1.HyperConverged) runtime.Object {
	return NewPrometheusRule(hc, hc.Namespace)
}
func (h prometheusRuleHooks) getEmptyCr() runtime.Object                              { return &monitoringv1.PrometheusRule{} }
func (h prometheusRuleHooks) validate() error                                         { return nil }
func (h prometheusRuleHooks) postFound(*common.HcoRequest, runtime.Object) error      { return nil }
func (h prometheusRuleHooks) getConditions(_ runtime.Object) []conditionsv1.Condition { return nil }
func (h prometheusRuleHooks) checkComponentVersion(_ runtime.Object) bool             { return true }
func (h prometheusRuleHooks) getObjectMeta(cr runtime.Object) *metav1.ObjectMeta {
	return &cr.(*monitoringv1.PrometheusRule).ObjectMeta
}

func (h *prometheusRuleHooks) updateCr(req *common.HcoRequest, Client client.Client, exists runtime.Object, required runtime.Object) (bool, bool, error) {
	rule, ok1 := required.(*monitoringv1.PrometheusRule)
	found, ok2 := exists.(*monitoringv1.PrometheusRule)
	if !ok1 || !ok2 {
		return false, false, errors.New("can't convert to PrometheusRule")
	}
	if !reflect.DeepEqual(found.Spec, rule.Spec) {
		if req.HCOTriggered {
			req.Logger.Info("Updating existing PrometheusRule Spec to new opinionated values")
		} else {
			req.Logger.Info("Reconciling an externally updated PrometheusRule Spec to its opinionated values")
		}
		rule.Spec.DeepCopyInto(&found.Spec)
		err := Client.Update(req.Ctx, found)
		if err != nil {
			return false, false, err
		}
		return true, !req.HCOTriggered, nil
	}
	return false, false, nil
}

// NewPrometheusRule creates PrometheusRule resource to define alert rules
func NewPrometheusRule(hc *hcov1beta1.HyperConverged, namespace string) *monitoringv1.PrometheusRule {
	spec := monitoringv1.PrometheusRuleSpec{
		Groups: []monitoringv1.RuleGroup{{
			Name: alertRuleGroup,
			Rules: []monitoringv1.Rule{{
				Alert: outOfBandUpdateAlert,
				Expr:  intstr.FromString("sum(increase(hyperconverged_cluster_operator_out_of_band_modifications[1m])) by (component_name) > 0"),
				For:   "60m",
				Annotations: map[string]string{
					"description": "Out-of-band modification for {{ $labels.component_name }} .",
					"summary":     "Out-of-band CR modification was detected",
				},
				Labels: map[string]string{
					"severity": "warning",
				},
			}},
		}},
	}

	return &monitoringv1.PrometheusRule{
		TypeMeta: metav1.TypeMeta{
			APIVersion: monitoringv1.SchemeGroupVersion.String(),
			Kind:       "PrometheusRule",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      hc.Name + "-prometheus-rule",
			Labels:    getLabels(hc),
			Namespace: namespace,
		},
		Spec: spec,
	}
}
