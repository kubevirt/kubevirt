package alerts

import (
	promv1 "github.com/prometheus-operator/prometheus-operator/pkg/apis/monitoring/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/utils/ptr"
)

func clusterAlerts() []promv1.Rule {
	return []promv1.Rule{
		{
			Alert: "HighCPUWorkload",
			Expr:  intstr.FromString("instance:node_cpu_utilisation:rate1m >= 0.9"),
			For:   ptr.To(promv1.Duration("5m")),
			Annotations: map[string]string{
				"summary":     "High CPU usage on host {{ $labels.instance }}",
				"description": "CPU utilization for {{ $labels.instance }} has been above 90% for more than 5 minutes.",
			},
			Labels: map[string]string{
				"severity":               "warning",
				"operator_health_impact": "none",
			},
		},
		{
			Alert: "HAControlPlaneDown",
			Expr:  intstr.FromString("kube_node_role{role=\"control-plane\"} * on(node) kube_node_status_condition{condition=\"Ready\",status=\"true\"} == 0"),
			For:   ptr.To(promv1.Duration("5m")),
			Annotations: map[string]string{
				"summary":     "Control plane node {{ $labels.node }} is not ready",
				"description": "Control plane node {{ $labels.node }} has been not ready for more than 5 minutes.",
			},
			Labels: map[string]string{
				"severity":               "critical",
				"operator_health_impact": "none",
			},
		},
		{
			Alert: "NodeNetworkInterfaceDown",
			Expr:  intstr.FromString("count by (instance) (node_network_up{device!~\"veth.+|tunbr\"} == 0) > 0"),
			For:   ptr.To(promv1.Duration("5m")),
			Annotations: map[string]string{
				"summary":     "Network interfaces are down",
				"description": "{{ $value }} network devices have been down on instance {{ $labels.instance }} for more than 5 minutes.",
			},
			Labels: map[string]string{
				"severity":               "warning",
				"operator_health_impact": "none",
			},
		},
	}
}
