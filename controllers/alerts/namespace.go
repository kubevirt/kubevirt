package alerts

import (
	"context"
	"fmt"

	"github.com/go-logr/logr"
	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	hcoutil "github.com/kubevirt/hyperconverged-cluster-operator/pkg/util"
)

func reconcileNamespace(ctx context.Context, cl client.Client, namespace string, logger logr.Logger) error {
	ns := &corev1.Namespace{}

	err := cl.Get(ctx, client.ObjectKey{Name: namespace}, ns)
	if err != nil {
		return fmt.Errorf("can't read namespace %s; %w", namespace, err)
	}

	needUpdate := false
	if ns.Annotations == nil {
		ns.Annotations = make(map[string]string)
	}
	if val, hasKey := ns.Annotations[hcoutil.OpenshiftNodeSelectorAnn]; !hasKey || val != "" {
		ns.Annotations[hcoutil.OpenshiftNodeSelectorAnn] = ""
		needUpdate = true
	}

	if ns.Labels == nil {
		ns.Labels = make(map[string]string)
	}
	if val, hasKey := ns.Labels[hcoutil.PrometheusNSLabel]; !hasKey || val != "true" {
		ns.Labels[hcoutil.PrometheusNSLabel] = "true"
		needUpdate = true
	}

	if needUpdate {
		logger.Info(fmt.Sprintf("updating the %s namespace", namespace))
		if err = cl.Update(ctx, ns); err != nil {
			return fmt.Errorf("failed to update namespace %s; %w", namespace, err)
		}
	}
	return nil
}
