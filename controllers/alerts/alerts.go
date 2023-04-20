package alerts

// This package makes sure that the PrometheusRule is present with the right configurations.
// This code was taken out of the operator package, because the operand reconciliation is done
// only if the HyperConverged CR is present. But we need the alert in place even if the CR was
// not created.

import (
	"context"
	"errors"
	"fmt"
	"os"
	"reflect"
	"strings"

	"github.com/go-logr/logr"
	monitoringv1 "github.com/prometheus-operator/prometheus-operator/pkg/apis/monitoring/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	"sigs.k8s.io/controller-runtime/pkg/client"

	hcoutil "github.com/kubevirt/hyperconverged-cluster-operator/pkg/util"
)

const (
	alertRuleGroup                = "kubevirt.hyperconverged.rules"
	outOfBandUpdateAlert          = "KubevirtHyperconvergedClusterOperatorCRModification"
	unsafeModificationAlert       = "KubevirtHyperconvergedClusterOperatorUSModification"
	installationNotCompletedAlert = "KubevirtHyperconvergedClusterOperatorInstallationNotCompletedAlert"
	severityAlertLabelKey         = "severity"
	healthImpactAlertLabelKey     = "operator_health_impact"
	partOfAlertLabelKey           = "kubernetes_operator_part_of"
	partOfAlertLabelValue         = "kubevirt"
	componentAlertLabelKey        = "kubernetes_operator_component"
	componentAlertLabelValue      = "hyperconverged-cluster-operator"
	ruleName                      = hcoutil.HyperConvergedName + "-prometheus-rule"
	defaultRunbookURLTemplate     = "https://kubevirt.io/monitoring/runbooks/%s"
	runbookURLTemplateEnv         = "RUNBOOK_URL_TEMPLATE"
)

type runbookCreator struct {
	template string
}

func newRunbookCreator() *runbookCreator {
	runbookURLTemplate, exists := os.LookupEnv(runbookURLTemplateEnv)
	if !exists {
		runbookURLTemplate = defaultRunbookURLTemplate
	}

	if strings.Count(runbookURLTemplate, "%s") != 1 {
		panic(errors.New("runbooks URL template must have exactly 1 %s substring"))
	}

	return &runbookCreator{
		template: runbookURLTemplate,
	}
}

func (r runbookCreator) getURL(alertName string) string {
	return fmt.Sprintf(r.template, alertName)
}

type AlertRuleReconciler struct {
	theRule *monitoringv1.PrometheusRule
}

// newAlertRuleReconciler creates new AlertRuleReconciler instance and returns a pointer to it.
func newAlertRuleReconciler(namespace string, owner metav1.OwnerReference) *AlertRuleReconciler {
	return &AlertRuleReconciler{
		theRule: newPrometheusRule(namespace, owner),
	}
}

func (r *AlertRuleReconciler) Kind() string {
	return monitoringv1.PrometheusRuleKind
}

func (r *AlertRuleReconciler) ResourceName() string {
	return r.theRule.Name
}

func (r *AlertRuleReconciler) GetFullResource() client.Object {
	return r.theRule.DeepCopy()
}

func (r *AlertRuleReconciler) EmptyObject() client.Object {
	return &monitoringv1.PrometheusRule{}
}

func (r *AlertRuleReconciler) UpdateExistingResource(ctx context.Context, cl client.Client, existing client.Object, logger logr.Logger) (client.Object, bool, error) {
	needUpdate := false
	rule := existing.(*monitoringv1.PrometheusRule)
	if !reflect.DeepEqual(r.theRule.Spec, rule.Spec) {
		needUpdate = true
		r.theRule.Spec.DeepCopyInto(&rule.Spec)
	}

	needUpdate = updateCommonDetails(&r.theRule.ObjectMeta, &rule.ObjectMeta) || needUpdate

	if needUpdate {
		logger.Info("updating the PrometheusRule")
		err := cl.Update(ctx, rule)
		if err != nil {
			logger.Error(err, "failed to update the PrometheusRule")
			return nil, false, err
		}
		logger.Info("successfully updated the PrometheusRule")
	}

	return rule, needUpdate, nil
}

func newPrometheusRule(namespace string, owner metav1.OwnerReference) *monitoringv1.PrometheusRule {
	return &monitoringv1.PrometheusRule{
		TypeMeta: metav1.TypeMeta{
			APIVersion: monitoringv1.SchemeGroupVersion.String(),
			Kind:       "PrometheusRule",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:            ruleName,
			Labels:          hcoutil.GetLabels(hcoutil.HyperConvergedName, hcoutil.AppComponentMonitoring),
			Namespace:       namespace,
			OwnerReferences: []metav1.OwnerReference{owner},
		},
		Spec: *NewPrometheusRuleSpec(),
	}
}

// NewPrometheusRuleSpec creates PrometheusRuleSpec for alert rules
func NewPrometheusRuleSpec() *monitoringv1.PrometheusRuleSpec {
	runbookCreator := newRunbookCreator()

	spec := &monitoringv1.PrometheusRuleSpec{
		Groups: []monitoringv1.RuleGroup{{
			Name: alertRuleGroup,
			Rules: []monitoringv1.Rule{
				createOutOfBandUpdateAlertRule(),
				createUnsafeModificationAlertRule(),
				createInstallationNotCompletedAlertRule(),
				createRequestCPUCoresRule(),
				createOperatorHealthStatusRule(),
			},
		}},
	}

	for _, rule := range spec.Groups[0].Rules {
		if rule.Alert != "" {
			rule.Annotations["runbook_url"] = runbookCreator.getURL(rule.Alert)
			rule.Labels[partOfAlertLabelKey] = partOfAlertLabelValue
			rule.Labels[componentAlertLabelKey] = componentAlertLabelValue
		}
	}

	return spec
}

func createOutOfBandUpdateAlertRule() monitoringv1.Rule {
	return monitoringv1.Rule{
		Alert: outOfBandUpdateAlert,
		Expr:  intstr.FromString("sum by(component_name) ((round(increase(kubevirt_hco_out_of_band_modifications_count[10m]))>0 and kubevirt_hco_out_of_band_modifications_count offset 10m) or (kubevirt_hco_out_of_band_modifications_count != 0 unless kubevirt_hco_out_of_band_modifications_count offset 10m))"),
		Annotations: map[string]string{
			"description": "Out-of-band modification for {{ $labels.component_name }}.",
			"summary":     "{{ $value }} out-of-band CR modifications were detected in the last 10 minutes.",
		},
		Labels: map[string]string{
			severityAlertLabelKey:     "warning",
			healthImpactAlertLabelKey: "warning",
		},
	}
}

func createUnsafeModificationAlertRule() monitoringv1.Rule {
	return monitoringv1.Rule{
		Alert: unsafeModificationAlert,
		Expr:  intstr.FromString("sum by(annotation_name) ((kubevirt_hco_unsafe_modification_count)>0)"),
		Annotations: map[string]string{
			"description": "unsafe modification for the {{ $labels.annotation_name }} annotation in the HyperConverged resource.",
			"summary":     "{{ $value }} unsafe modifications were detected in the HyperConverged resource.",
		},
		Labels: map[string]string{
			severityAlertLabelKey:     "info",
			healthImpactAlertLabelKey: "warning",
		},
	}
}

func createInstallationNotCompletedAlertRule() monitoringv1.Rule {
	return monitoringv1.Rule{
		Alert: installationNotCompletedAlert,
		Expr:  intstr.FromString("kubevirt_hco_hyperconverged_cr_exists == 0"),
		Annotations: map[string]string{
			"description": "the installation was not completed; the HyperConverged custom resource is missing. In order to complete the installation of the Hyperconverged Cluster Operator you should create the HyperConverged custom resource.",
			"summary":     "the installation was not completed; to complete the installation, create a HyperConverged custom resource.",
		},
		For: "1h",
		Labels: map[string]string{
			severityAlertLabelKey:     "info",
			healthImpactAlertLabelKey: "critical",
		},
	}
}

// Recording rules for openshift/cluster-monitoring-operator
func createRequestCPUCoresRule() monitoringv1.Rule {
	return monitoringv1.Rule{
		Record: "cluster:vmi_request_cpu_cores:sum",
		Expr:   intstr.FromString(`sum(kube_pod_container_resource_requests{resource="cpu"} and on (pod) kube_pod_status_phase{phase="Running"} * on (pod) group_left kube_pod_labels{ label_kubevirt_io="virt-launcher"} > 0)`),
	}
}

func createOperatorHealthStatusRule() monitoringv1.Rule {
	return monitoringv1.Rule{
		Record: "kubevirt_hyperconverged_operator_health_status",
		Expr:   intstr.FromString(`label_replace(vector(2) and on() ((kubevirt_hco_system_health_status>1) or (count(ALERTS{kubernetes_operator_part_of="kubevirt", alertstate="firing", operator_health_impact="critical"})>0)) or (vector(1) and on() ((kubevirt_hco_system_health_status==1) or (count(ALERTS{kubernetes_operator_part_of="kubevirt", alertstate="firing", operator_health_impact="warning"})>0))) or vector(0),"name","kubevirt-hyperconverged","","")`),
	}
}
