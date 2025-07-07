package components

import (
	"github.com/prometheus-operator/prometheus-operator/pkg/apis/monitoring"
	promv1 "github.com/prometheus-operator/prometheus-operator/pkg/apis/monitoring/v1"
	v12 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	KUBEVIRT_PROMETHEUS_RULE_NAME = "prometheus-kubevirt-rules"
	prometheusLabelKey            = "prometheus.kubevirt.io"
	prometheusLabelValue          = "true"
)

func NewServiceMonitorCR(namespace string, monitorNamespace string, insecureSkipVerify bool) *promv1.ServiceMonitor {
	return &promv1.ServiceMonitor{
		TypeMeta: v12.TypeMeta{
			APIVersion: monitoring.GroupName,
			Kind:       "ServiceMonitor",
		},
		ObjectMeta: v12.ObjectMeta{
			Namespace: monitorNamespace,
			Name:      KUBEVIRT_PROMETHEUS_RULE_NAME,
			Labels: map[string]string{
				"openshift.io/cluster-monitoring": "",
				prometheusLabelKey:                prometheusLabelValue,
				"k8s-app":                         "kubevirt",
			},
		},
		Spec: promv1.ServiceMonitorSpec{
			Selector: v12.LabelSelector{
				MatchLabels: map[string]string{
					prometheusLabelKey: prometheusLabelValue,
				},
			},
			NamespaceSelector: promv1.NamespaceSelector{
				MatchNames: []string{namespace},
			},
			Endpoints: []promv1.Endpoint{
				{
					Port:   "metrics",
					Scheme: "https",
					TLSConfig: &promv1.TLSConfig{
						SafeTLSConfig: promv1.SafeTLSConfig{
							InsecureSkipVerify: insecureSkipVerify,
						},
					},
					HonorLabels: true,
				},
			},
		},
	}
}
