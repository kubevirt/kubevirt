package operands

import (
	"errors"
	"fmt"
	"os"
	"reflect"

	monitoringv1 "github.com/prometheus-operator/prometheus-operator/pkg/apis/monitoring/v1"
	corev1 "k8s.io/api/core/v1"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/intstr"
	"sigs.k8s.io/controller-runtime/pkg/client"

	hcov1beta1 "github.com/kubevirt/hyperconverged-cluster-operator/pkg/apis/hco/v1beta1"
	"github.com/kubevirt/hyperconverged-cluster-operator/pkg/controller/common"
	"github.com/kubevirt/hyperconverged-cluster-operator/pkg/util"
	hcoutil "github.com/kubevirt/hyperconverged-cluster-operator/pkg/util"
)

const (
	operatorPortName        = "http-metrics"
	defaultOperatorName     = "hyperconverged-cluster-operator"
	operatorNameEnv         = "OPERATOR_NAME"
	metricsSuffix           = "-operator-metrics"
	alertRuleGroup          = "kubevirt.hyperconverged.rules"
	outOfBandUpdateAlert    = "KubevirtHyperconvergedClusterOperatorCRModification"
	unsafeModificationAlert = "KubevirtHyperconvergedClusterOperatorUSModification"
	runbookUrlTemplate      = "https://kubevirt.io/monitoring/runbooks/%s"
)

var (
	outOfBandUpdateRunbookUrl    = fmt.Sprintf(runbookUrlTemplate, outOfBandUpdateAlert)
	unsafeModificationRunbookUrl = fmt.Sprintf(runbookUrlTemplate, unsafeModificationAlert)
)

type metricsServiceHandler genericOperand

func newMetricsServiceHandler(Client client.Client, Scheme *runtime.Scheme) *metricsServiceHandler {
	return &metricsServiceHandler{
		Client:                 Client,
		Scheme:                 Scheme,
		crType:                 "MetricsService",
		removeExistingOwner:    false,
		setControllerReference: true,
		hooks:                  &metricsServiceHooks{},
	}
}

type metricsServiceHooks struct{}

func (h metricsServiceHooks) getFullCr(hc *hcov1beta1.HyperConverged) (client.Object, error) {
	return NewMetricsService(hc, hc.Namespace), nil
}
func (h metricsServiceHooks) getEmptyCr() client.Object { return &corev1.Service{} }
func (h metricsServiceHooks) getObjectMeta(cr runtime.Object) *metav1.ObjectMeta {
	return &cr.(*corev1.Service).ObjectMeta
}
func (h metricsServiceHooks) reset( /* No implementation */ ) {}

func (h *metricsServiceHooks) updateCr(req *common.HcoRequest, Client client.Client, exists runtime.Object, required runtime.Object) (bool, bool, error) {
	service, ok1 := required.(*corev1.Service)
	found, ok2 := exists.(*corev1.Service)
	if !ok1 || !ok2 {
		return false, false, errors.New("can't convert to Service")
	}

	if !reflect.DeepEqual(found.Spec.Ports, service.Spec.Ports) ||
		!reflect.DeepEqual(found.Spec.Selector, service.Spec.Selector) ||
		!reflect.DeepEqual(found.Labels, service.Labels) {
		if req.HCOTriggered {
			req.Logger.Info("Updating existing metrics Service Spec to new opinionated values")
		} else {
			req.Logger.Info("Reconciling an externally updated metrics Service Spec to its opinionated values")
		}
		found.Spec.Ports = service.Spec.Ports
		found.Spec.Selector = service.Spec.Selector
		util.DeepCopyLabels(&service.ObjectMeta, &found.ObjectMeta)
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
			Labels:    getLabels(hc, hcoutil.AppComponentMonitoring),
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

type monitoringPrometheusRuleHandler genericOperand

func newMonitoringPrometheusRuleHandler(Client client.Client, Scheme *runtime.Scheme) *monitoringPrometheusRuleHandler {
	return &monitoringPrometheusRuleHandler{
		Client:                 Client,
		Scheme:                 Scheme,
		crType:                 "PrometheusRule",
		removeExistingOwner:    false,
		setControllerReference: true,
		hooks:                  &prometheusRuleHooks{},
	}
}

type prometheusRuleHooks struct{}

func (h prometheusRuleHooks) getFullCr(hc *hcov1beta1.HyperConverged) (client.Object, error) {
	return NewPrometheusRule(hc, hc.Namespace), nil
}
func (h prometheusRuleHooks) getEmptyCr() client.Object { return &monitoringv1.PrometheusRule{} }
func (h prometheusRuleHooks) getObjectMeta(cr runtime.Object) *metav1.ObjectMeta {
	return &cr.(*monitoringv1.PrometheusRule).ObjectMeta
}
func (h prometheusRuleHooks) reset( /* No implementation */ ) {}

func (h *prometheusRuleHooks) updateCr(req *common.HcoRequest, Client client.Client, exists runtime.Object, required runtime.Object) (bool, bool, error) {
	rule, ok1 := required.(*monitoringv1.PrometheusRule)
	found, ok2 := exists.(*monitoringv1.PrometheusRule)
	if !ok1 || !ok2 {
		return false, false, errors.New("can't convert to PrometheusRule")
	}
	if !reflect.DeepEqual(found.Spec, rule.Spec) ||
		!reflect.DeepEqual(found.Labels, rule.Labels) {
		if req.HCOTriggered {
			req.Logger.Info("Updating existing PrometheusRule Spec to new opinionated values")
		} else {
			req.Logger.Info("Reconciling an externally updated PrometheusRule Spec to its opinionated values")
		}
		rule.Spec.DeepCopyInto(&found.Spec)
		util.DeepCopyLabels(&rule.ObjectMeta, &found.ObjectMeta)
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
	return &monitoringv1.PrometheusRule{
		TypeMeta: metav1.TypeMeta{
			APIVersion: monitoringv1.SchemeGroupVersion.String(),
			Kind:       "PrometheusRule",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      hc.Name + "-prometheus-rule",
			Labels:    getLabels(hc, hcoutil.AppComponentMonitoring),
			Namespace: namespace,
		},
		Spec: *NewPrometheusRuleSpec(),
	}
}

// NewPrometheusRuleSpec creates PrometheusRuleSpec for alert rules
func NewPrometheusRuleSpec() *monitoringv1.PrometheusRuleSpec {
	return &monitoringv1.PrometheusRuleSpec{
		Groups: []monitoringv1.RuleGroup{{
			Name: alertRuleGroup,
			Rules: []monitoringv1.Rule{
				{
					Alert: outOfBandUpdateAlert,
					Expr:  intstr.FromString("sum by(component_name) ((round(increase(kubevirt_hco_out_of_band_modifications_count[10m]))>0 and kubevirt_hco_out_of_band_modifications_count offset 10m) or (kubevirt_hco_out_of_band_modifications_count != 0 unless kubevirt_hco_out_of_band_modifications_count offset 10m))"),
					Annotations: map[string]string{
						"description": "Out-of-band modification for {{ $labels.component_name }}.",
						"summary":     "{{ $value }} out-of-band CR modifications were detected in the last 10 minutes.",
						"runbook_url": outOfBandUpdateRunbookUrl,
					},
					Labels: map[string]string{
						"severity":                      "warning",
						"kubernetes_operator_part_of":   "kubevirt",
						"kubernetes_operator_component": "hyperconverged-cluster-operator",
					},
				},
				{
					Alert: unsafeModificationAlert,
					Expr:  intstr.FromString("sum by(annotation_name) ((kubevirt_hco_unsafe_modification_count)>0)"),
					Annotations: map[string]string{
						"description": "unsafe modification for the {{ $labels.annotation_name }} annotation in the HyperConverged resource.",
						"summary":     "{{ $value }} unsafe modifications were detected in the HyperConverged resource.",
						"runbook_url": unsafeModificationRunbookUrl,
					},
					Labels: map[string]string{
						"severity":                      "info",
						"kubernetes_operator_part_of":   "kubevirt",
						"kubernetes_operator_component": "hyperconverged-cluster-operator",
					},
				},
				// Recording rules for openshift/cluster-monitoring-operator
				{
					Record: "cluster:vmi_request_cpu_cores:sum",
					Expr:   intstr.FromString(`sum(kube_pod_container_resource_requests{resource="cpu"} and on (pod) kube_pod_status_phase{phase="Running"} * on (pod) group_left kube_pod_labels{ label_kubevirt_io="virt-launcher"} > 0)`),
				},
			},
		}},
	}
}
