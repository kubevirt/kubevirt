package recordingrules

import (
	"github.com/machadovilaca/operator-observability/pkg/operatormetrics"
	"github.com/machadovilaca/operator-observability/pkg/operatorrules"
	"k8s.io/apimachinery/pkg/util/intstr"
)

var operatorRecordingRules = []operatorrules.RecordingRule{
	{
		MetricsOpts: operatormetrics.MetricOpts{
			Name: "kubevirt_hyperconverged_operator_health_status",
			Help: "Indicates whether HCO and its secondary resources health status is healthy (0), warning (1) or critical (2), based both on the firing alerts that impact the operator health, and on kubevirt_hco_system_health_status metric",
		},
		MetricType: operatormetrics.GaugeType,
		Expr:       intstr.FromString(`label_replace(vector(2) and on() ((kubevirt_hco_system_health_status>1) or (count(ALERTS{kubernetes_operator_part_of="kubevirt", alertstate="firing", operator_health_impact="critical"})>0)) or (vector(1) and on() ((kubevirt_hco_system_health_status==1) or (count(ALERTS{kubernetes_operator_part_of="kubevirt", alertstate="firing", operator_health_impact="warning"})>0))) or vector(0),"name","kubevirt-hyperconverged","","")`),
	},
	{
		MetricsOpts: operatormetrics.MetricOpts{
			Name: "cluster:vmi_request_cpu_cores:sum",
			Help: "Sum of CPU core requests for all running virt-launcher VMIs across the entire Kubevirt cluster",
		},
		MetricType: operatormetrics.GaugeType,
		Expr:       intstr.FromString(`sum(kube_pod_container_resource_requests{resource="cpu"} and on (pod) kube_pod_status_phase{phase="Running"} * on (pod) group_left kube_pod_labels{ label_kubevirt_io="virt-launcher"} > 0)`),
	},
	{
		MetricsOpts: operatormetrics.MetricOpts{
			Name: "cnv_abnormal",
			Help: "Monitors resources for potential problems",
		},
		MetricType: operatormetrics.GaugeType,
		Expr:       intstr.FromString(`sum by (container, reason)(kubevirt_memory_delta_from_requested_bytes)`),
	},
}
