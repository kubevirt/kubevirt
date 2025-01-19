package observability

import (
	"context"
	"fmt"
	"reflect"

	promv1 "github.com/prometheus-operator/prometheus-operator/pkg/apis/monitoring/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"

	"github.com/kubevirt/hyperconverged-cluster-operator/pkg/monitoring/observability/rules"
)

func (r *Reconciler) ReconcileAlerts(ctx context.Context) error {
	desiredPromRule, err := rules.BuildPrometheusRule(r.namespace, r.owner)
	if err != nil {
		return fmt.Errorf("failed to build PrometheusRule: %v", err)
	}

	existingPromRule := &promv1.PrometheusRule{}
	err = r.Get(ctx, types.NamespacedName{
		Name:      desiredPromRule.Name,
		Namespace: desiredPromRule.Namespace,
	}, existingPromRule)

	if err != nil {
		if apierrors.IsNotFound(err) {
			// if it doesn't exist, create it
			if createErr := r.Create(ctx, desiredPromRule); createErr != nil {
				return fmt.Errorf("failed to create PrometheusRule: %v", createErr)
			}

			return nil
		}

		return fmt.Errorf("failed to get PrometheusRule: %v", err)
	}

	// if it does exist, compare specs and update if different
	if !reflect.DeepEqual(existingPromRule.Spec, desiredPromRule.Spec) {
		existingPromRule.Spec = desiredPromRule.Spec
		if updateErr := r.Update(ctx, existingPromRule); updateErr != nil {
			return fmt.Errorf("failed to update PrometheusRule: %v", updateErr)
		}
	}

	return nil
}
