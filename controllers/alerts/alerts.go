package alerts

import (
	"context"
	"reflect"

	"github.com/go-logr/logr"
	promv1 "github.com/prometheus-operator/prometheus-operator/pkg/apis/monitoring/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/kubevirt/hyperconverged-cluster-operator/pkg/monitoring/rules"
	hcoutil "github.com/kubevirt/hyperconverged-cluster-operator/pkg/util"
)

const (
	ruleName                  = hcoutil.HyperConvergedName + "-prometheus-rule"
	defaultRunbookURLTemplate = "https://kubevirt.io/monitoring/runbooks/%s"
	runbookURLTemplateEnv     = "RUNBOOK_URL_TEMPLATE"
)

type AlertRuleReconciler struct {
	theRule *promv1.PrometheusRule
}

// newAlertRuleReconciler creates new AlertRuleReconciler instance and returns a pointer to it.
func newAlertRuleReconciler(namespace string, owner metav1.OwnerReference) (*AlertRuleReconciler, error) {
	err := rules.SetupRules()
	if err != nil {
		return nil, err
	}

	rule, err := rules.BuildPrometheusRule(namespace, owner)
	if err != nil {
		return nil, err
	}

	return &AlertRuleReconciler{
		theRule: rule,
	}, nil
}

func (r *AlertRuleReconciler) Kind() string {
	return promv1.PrometheusRuleKind
}

func (r *AlertRuleReconciler) ResourceName() string {
	return r.theRule.Name
}

func (r *AlertRuleReconciler) GetFullResource() client.Object {
	return r.theRule.DeepCopy()
}

func (r *AlertRuleReconciler) EmptyObject() client.Object {
	return &promv1.PrometheusRule{}
}

func (r *AlertRuleReconciler) UpdateExistingResource(ctx context.Context, cl client.Client, existing client.Object, logger logr.Logger) (client.Object, bool, error) {
	needUpdate := false
	rule := existing.(*promv1.PrometheusRule)
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
