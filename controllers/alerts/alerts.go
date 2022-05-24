package alerts

// This package makes sure that the PrometheusRule is present with the right configurations.
// This code was taken out of the operator package, because the operand reconciliation is done
// only if the HyperConverged CR is present. But we need the alert in place even if the CR was
// not created.

import (
	"context"
	"fmt"
	"reflect"

	"github.com/go-logr/logr"
	monitoringv1 "github.com/prometheus-operator/prometheus-operator/pkg/apis/monitoring/v1"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/utils/pointer"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/kubevirt/hyperconverged-cluster-operator/controllers/common"
	hcoutil "github.com/kubevirt/hyperconverged-cluster-operator/pkg/util"
)

const (
	alertRuleGroup                = "kubevirt.hyperconverged.rules"
	outOfBandUpdateAlert          = "KubevirtHyperconvergedClusterOperatorCRModification"
	unsafeModificationAlert       = "KubevirtHyperconvergedClusterOperatorUSModification"
	installationNotCompletedAlert = "KubevirtHyperconvergedClusterOperatorInstallationNotCompletedAlert"
	severityAlertLabelKey         = "severity"
	partOfAlertLabelKey           = "kubernetes_operator_part_of"
	partOfAlertLabelValue         = "kubevirt"
	componentAlertLabelKey        = "kubernetes_operator_component"
	componentAlertLabelValue      = "hyperconverged-cluster-operator"
	ruleName                      = hcoutil.HyperConvergedName + "-prometheus-rule"
)

var (
	runbookUrlTemplate = "https://kubevirt.io/monitoring/runbooks/%s"

	outOfBandUpdateRunbookUrl          = fmt.Sprintf(runbookUrlTemplate, outOfBandUpdateAlert)
	unsafeModificationRunbookUrl       = fmt.Sprintf(runbookUrlTemplate, unsafeModificationAlert)
	installationNotCompletedRunbookUrl = fmt.Sprintf(runbookUrlTemplate, installationNotCompletedAlert)
)

type AlertRuleReconciler struct {
	client       client.Client
	namespace    string
	theRule      *monitoringv1.PrometheusRule
	latestRule   *monitoringv1.PrometheusRule
	eventEmitter hcoutil.EventEmitter
	scheme       *runtime.Scheme
}

// NewAlertRuleReconciler creates new AlertRuleReconciler instance and returns a pointer to it.
func NewAlertRuleReconciler(cl client.Client, ci hcoutil.ClusterInfo, ee hcoutil.EventEmitter, scheme *runtime.Scheme) *AlertRuleReconciler {
	deployment := ci.GetDeployment()
	namespace := deployment.Namespace
	return &AlertRuleReconciler{
		client:       cl,
		theRule:      newPrometheusRule(namespace, deployment),
		namespace:    namespace,
		eventEmitter: ee,
		scheme:       scheme,
	}
}

// Reconcile makes sure that the required PrometheusRule resource is present with the required fields
func (r *AlertRuleReconciler) Reconcile(ctx context.Context, logger logr.Logger) error {
	if r == nil {
		return nil // not initialized (not running on openshift). do nothing
	}

	logger.V(5).Info("Reconciling the PrometheusRule")

	existingRule := &monitoringv1.PrometheusRule{}
	logger.V(5).Info("Reading the current PrometheusRule")
	err := r.client.Get(ctx, client.ObjectKey{Namespace: r.namespace, Name: ruleName}, existingRule)

	if err != nil {
		if errors.IsNotFound(err) {
			logger.Info("Can't find the PrometheusRule; creating a new one")
			rule := r.theRule.DeepCopy()
			err := r.client.Create(ctx, rule)
			if err != nil {
				logger.Error(err, "failed to create PrometheusRule")
				r.eventEmitter.EmitEvent(rule, corev1.EventTypeWarning, "UnexpectedError", fmt.Sprintf("failed to create the %s %s", ruleName, monitoringv1.PrometheusRuleKind))
				return err
			}
			logger.Info("successfully created the PrometheusRule")
			r.eventEmitter.EmitEvent(rule, corev1.EventTypeNormal, "Created", fmt.Sprintf("Created %s %s", monitoringv1.PrometheusRuleKind, ruleName))

			// read back the new PrometheusRule
			err = r.client.Get(ctx, client.ObjectKey{Namespace: r.namespace, Name: ruleName}, existingRule)
			if err != nil {
				logger.Error(err, "failed to read PrometheusRule")
			}
			r.latestRule = existingRule
			return nil
		}

		logger.Error(err, "unexpected error while reading the PrometheusRule")
		return err
	}

	r.latestRule = existingRule

	return r.updateAlert(ctx, existingRule, logger)
}

func (r *AlertRuleReconciler) UpdateRelatedObjects(req *common.HcoRequest) error {
	if r == nil {
		return nil // not initialized (not running on openshift). do nothing
	}

	changed, err := hcoutil.AddCrToTheRelatedObjectList(&req.Instance.Status.RelatedObjects, r.latestRule, r.scheme)
	if err != nil {
		return err
	}

	if changed {
		req.StatusDirty = true
	}

	return nil
}

func (r AlertRuleReconciler) updateAlert(ctx context.Context, rule *monitoringv1.PrometheusRule, logger logr.Logger) error {
	needUpdate := false
	if !reflect.DeepEqual(r.theRule.Spec, rule.Spec) {
		needUpdate = true
		r.theRule.Spec.DeepCopyInto(&rule.Spec)
	}

	if !reflect.DeepEqual(r.theRule.ObjectMeta.OwnerReferences, rule.ObjectMeta.OwnerReferences) {
		needUpdate = true
		if numRefs := len(r.theRule.ObjectMeta.OwnerReferences); numRefs > 0 {
			rule.ObjectMeta.OwnerReferences = make([]metav1.OwnerReference, numRefs)
			for i, ref := range r.theRule.ObjectMeta.OwnerReferences {
				ref.DeepCopyInto(&rule.ObjectMeta.OwnerReferences[i])
			}
		} else {
			rule.ObjectMeta.OwnerReferences = nil
		}
	}

	if needUpdate {
		logger.Info("updating the PrometheusRule")
		err := r.client.Update(ctx, rule)
		if err != nil {
			logger.Error(err, "failed to update the PrometheusRule")
			r.eventEmitter.EmitEvent(rule, corev1.EventTypeWarning, "UnexpectedError", fmt.Sprintf("failed to update the %s %s", ruleName, monitoringv1.PrometheusRuleKind))
			return err
		}
		logger.Info("successfully updated the PrometheusRule")
		r.eventEmitter.EmitEvent(rule, corev1.EventTypeNormal, "Updated", fmt.Sprintf("Updated %s %s", monitoringv1.PrometheusRuleKind, ruleName))
	}

	return nil
}

func newPrometheusRule(namespace string, deployment *appsv1.Deployment) *monitoringv1.PrometheusRule {
	return &monitoringv1.PrometheusRule{
		TypeMeta: metav1.TypeMeta{
			APIVersion: monitoringv1.SchemeGroupVersion.String(),
			Kind:       "PrometheusRule",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      ruleName,
			Labels:    hcoutil.GetLabels(hcoutil.HyperConvergedName, hcoutil.AppComponentMonitoring),
			Namespace: namespace,
			OwnerReferences: []metav1.OwnerReference{
				getDeploymentReference(deployment),
			},
		},
		Spec: *NewPrometheusRuleSpec(),
	}
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

// NewPrometheusRuleSpec creates PrometheusRuleSpec for alert rules
func NewPrometheusRuleSpec() *monitoringv1.PrometheusRuleSpec {
	return &monitoringv1.PrometheusRuleSpec{
		Groups: []monitoringv1.RuleGroup{{
			Name: alertRuleGroup,
			Rules: []monitoringv1.Rule{
				createOutOfBandUpdateAlertRule(),
				createUnsafeModificationAlertRule(),
				createInstallationNotCompletedAlertRule(),
				createRequestCPUCoresRule(),
			},
		}},
	}
}

func createOutOfBandUpdateAlertRule() monitoringv1.Rule {
	return monitoringv1.Rule{
		Alert: outOfBandUpdateAlert,
		Expr:  intstr.FromString("sum by(component_name) ((round(increase(kubevirt_hco_out_of_band_modifications_count[10m]))>0 and kubevirt_hco_out_of_band_modifications_count offset 10m) or (kubevirt_hco_out_of_band_modifications_count != 0 unless kubevirt_hco_out_of_band_modifications_count offset 10m))"),
		Annotations: map[string]string{
			"description": "Out-of-band modification for {{ $labels.component_name }}.",
			"summary":     "{{ $value }} out-of-band CR modifications were detected in the last 10 minutes.",
			"runbook_url": outOfBandUpdateRunbookUrl,
		},
		Labels: map[string]string{
			severityAlertLabelKey:  "warning",
			partOfAlertLabelKey:    partOfAlertLabelValue,
			componentAlertLabelKey: componentAlertLabelValue,
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
			"runbook_url": unsafeModificationRunbookUrl,
		},
		Labels: map[string]string{
			severityAlertLabelKey:  "info",
			partOfAlertLabelKey:    partOfAlertLabelValue,
			componentAlertLabelKey: componentAlertLabelValue,
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
			"runbook_url": installationNotCompletedRunbookUrl,
		},
		For: "1h",
		Labels: map[string]string{
			severityAlertLabelKey:  "info",
			partOfAlertLabelKey:    partOfAlertLabelValue,
			componentAlertLabelKey: componentAlertLabelValue,
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
