package components

import (
	"fmt"

	"github.com/coreos/prometheus-operator/pkg/apis/monitoring"
	v1 "github.com/coreos/prometheus-operator/pkg/apis/monitoring/v1"
	v12 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
)

const KUBEVIRT_PROMETHEUS_RULE_NAME = "prometheus-kubevirt-rules"
const prometheusLabelKey = "prometheus.kubevirt.io"
const prometheusLabelValue = "true"
const runbookUrlBasePath = "https://kubevirt.io/monitoring/runbooks/"

func NewServiceMonitorCR(namespace string, monitorNamespace string, insecureSkipVerify bool) *v1.ServiceMonitor {
	return &v1.ServiceMonitor{
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
		Spec: v1.ServiceMonitorSpec{
			Selector: v12.LabelSelector{
				MatchLabels: map[string]string{
					prometheusLabelKey: prometheusLabelValue,
				},
			},
			NamespaceSelector: v1.NamespaceSelector{
				MatchNames: []string{namespace},
			},
			Endpoints: []v1.Endpoint{
				{
					Port:   "metrics",
					Scheme: "https",
					TLSConfig: &v1.TLSConfig{
						InsecureSkipVerify: insecureSkipVerify,
					},
					HonorLabels: true,
				},
			},
		},
	}
}

// NewPrometheusRuleCR returns a PrometheusRule with a group of alerts for the KubeVirt deployment.
func NewPrometheusRuleCR(namespace string, workloadUpdatesEnabled bool) *v1.PrometheusRule {
	return &v1.PrometheusRule{
		TypeMeta: v12.TypeMeta{
			APIVersion: v12.SchemeGroupVersion.String(),
			Kind:       "PrometheusRule",
		},
		ObjectMeta: v12.ObjectMeta{
			Name:      KUBEVIRT_PROMETHEUS_RULE_NAME,
			Namespace: namespace,
			Labels: map[string]string{
				prometheusLabelKey: prometheusLabelValue,
				"k8s-app":          "kubevirt",
			},
		},
		Spec: *NewPrometheusRuleSpec(namespace, workloadUpdatesEnabled),
	}
}

// NewPrometheusRuleSpec makes a prometheus rule spec for kubevirt
func NewPrometheusRuleSpec(ns string, workloadUpdatesEnabled bool) *v1.PrometheusRuleSpec {
	getRestCallsFailedWarning := func(failingCallsPercentage int, component, duration string) string {
		const restCallsFailWarningTemplate = "More than %d%% of the rest calls failed in %s for the last %s"
		return fmt.Sprintf(restCallsFailWarningTemplate, failingCallsPercentage, component, duration)
	}

	ruleSpec := &v1.PrometheusRuleSpec{
		Groups: []v1.RuleGroup{
			{
				Name: "kubevirt.rules",
				Rules: []v1.Rule{
					{
						Record: "kubevirt_virt_api_up_total",
						Expr: intstr.FromString(
							fmt.Sprintf("sum(up{namespace='%s', pod=~'virt-api-.*'}) or vector(0)", ns),
						),
					},
					{
						Alert: "VirtAPIDown",
						Expr:  intstr.FromString("kubevirt_virt_api_up_total == 0"),
						For:   "10m",
						Annotations: map[string]string{
							"summary":     "All virt-api servers are down.",
							"runbook_url": runbookUrlBasePath + "VirtAPIDown",
						},
						Labels: map[string]string{
							"severity": "critical",
						},
					},
					{
						Record: "num_of_allocatable_nodes",
						Expr:   intstr.FromString("count(count (kube_node_status_allocatable) by (node))"),
					},
					{
						Alert: "LowVirtAPICount",
						Expr:  intstr.FromString("(num_of_allocatable_nodes > 1) and (kubevirt_virt_api_up_total < 2)"),
						For:   "60m",
						Annotations: map[string]string{
							"summary":     "More than one virt-api should be running if more than one worker nodes exist.",
							"runbook_url": runbookUrlBasePath + "LowVirtAPICount",
						},
						Labels: map[string]string{
							"severity": "warning",
						},
					},
					{
						Record: "num_of_kvm_available_nodes",
						Expr:   intstr.FromString("num_of_allocatable_nodes - count(kube_node_status_allocatable{resource=\"devices_kubevirt_io_kvm\"} == 0)"),
					},
					{
						Alert: "LowKVMNodesCount",
						Expr:  intstr.FromString("(num_of_allocatable_nodes > 1) and (num_of_kvm_available_nodes < 2)"),
						For:   "5m",
						Annotations: map[string]string{
							"description": "Low number of nodes with KVM resource available.",
							"summary":     "At least two nodes with kvm resource required for VM life migration.",
							"runbook_url": runbookUrlBasePath + "LowKVMNodesCount",
						},
						Labels: map[string]string{
							"severity": "warning",
						},
					},
					{
						Record: "kubevirt_virt_controller_up_total",
						Expr: intstr.FromString(
							fmt.Sprintf("sum(up{pod=~'virt-controller-.*', namespace='%s'}) or vector(0)", ns),
						),
					},
					{
						Record: "kubevirt_virt_controller_ready_total",
						Expr: intstr.FromString(
							fmt.Sprintf("sum(kubevirt_virt_controller_ready{namespace='%s'})", ns),
						),
					},
					{
						Alert: "LowReadyVirtControllersCount",
						Expr:  intstr.FromString("kubevirt_virt_controller_ready_total <  kubevirt_virt_controller_up_total"),
						For:   "10m",
						Annotations: map[string]string{
							"summary":     "Some virt controllers are running but not ready.",
							"runbook_url": runbookUrlBasePath + "LowReadyVirtControllersCount",
						},
						Labels: map[string]string{
							"severity": "warning",
						},
					},
					{
						Alert: "NoReadyVirtController",
						Expr:  intstr.FromString("kubevirt_virt_controller_ready_total == 0"),
						For:   "10m",
						Annotations: map[string]string{
							"summary":     "No ready virt-controller was detected for the last 10 min.",
							"runbook_url": runbookUrlBasePath + "NoReadyVirtController",
						},
						Labels: map[string]string{
							"severity": "critical",
						},
					},
					{
						Alert: "VirtControllerDown",
						Expr:  intstr.FromString("kubevirt_virt_controller_up_total == 0"),
						For:   "10m",
						Annotations: map[string]string{
							"summary":     "No running virt-controller was detected for the last 10 min.",
							"runbook_url": runbookUrlBasePath + "VirtControllerDown",
						},
						Labels: map[string]string{
							"severity": "critical",
						},
					},
					{
						Alert: "LowVirtControllersCount",
						Expr:  intstr.FromString("(num_of_allocatable_nodes > 1) and (kubevirt_virt_controller_ready_total < 2)"),
						For:   "10m",
						Annotations: map[string]string{
							"summary":     "More than one virt-controller should be ready if more than one worker node.",
							"runbook_url": runbookUrlBasePath + "LowVirtControllersCount",
						},
						Labels: map[string]string{
							"severity": "warning",
						},
					},
					{
						Record: "vec_by_virt_controllers_all_client_rest_requests_in_last_hour",
						Expr: intstr.FromString(
							fmt.Sprintf("sum by (pod) (sum_over_time(rest_client_requests_total{pod=~'virt-controller-.*', namespace='%s'}[60m]))", ns),
						),
					},
					{
						Record: "vec_by_virt_controllers_failed_client_rest_requests_in_last_hour",
						Expr: intstr.FromString(
							fmt.Sprintf("sum by (pod) (sum_over_time(rest_client_requests_total{pod=~'virt-controller-.*', namespace='%s', code=~'(4|5)[0-9][0-9]'}[60m]))", ns),
						),
					},
					{
						Record: "vec_by_virt_controllers_all_client_rest_requests_in_last_5m",
						Expr: intstr.FromString(
							fmt.Sprintf("sum by (pod) (sum_over_time(rest_client_requests_total{pod=~'virt-controller-.*', namespace='%s'}[5m]))", ns),
						),
					},
					{
						Record: "vec_by_virt_controllers_failed_client_rest_requests_in_last_5m",
						Expr: intstr.FromString(
							fmt.Sprintf("sum by (pod) (sum_over_time(rest_client_requests_total{pod=~'virt-controller-.*', namespace='%s', code=~'(4|5)[0-9][0-9]'}[5m]))", ns),
						),
					},
					{
						Alert: "VirtControllerRESTErrorsHigh",
						Expr:  intstr.FromString("(vec_by_virt_controllers_failed_client_rest_requests_in_last_hour / vec_by_virt_controllers_all_client_rest_requests_in_last_hour) >= 0.05"),
						For:   "5m",
						Annotations: map[string]string{
							"summary":     getRestCallsFailedWarning(5, "virt-controller", "hour"),
							"runbook_url": runbookUrlBasePath + "VirtControllerRESTErrorsHigh",
						},
						Labels: map[string]string{
							"severity": "warning",
						},
					},
					{
						Alert: "VirtControllerRESTErrorsBurst",
						Expr:  intstr.FromString("(vec_by_virt_controllers_failed_client_rest_requests_in_last_5m / vec_by_virt_controllers_all_client_rest_requests_in_last_5m) >= 0.8"),
						For:   "5m",
						Annotations: map[string]string{
							"summary":     getRestCallsFailedWarning(80, "virt-controller", "5 minutes"),
							"runbook_url": runbookUrlBasePath + "VirtControllerRESTErrorsBurst",
						},
						Labels: map[string]string{
							"severity": "critical",
						},
					},
					{
						Record: "kubevirt_virt_operator_up_total",
						Expr: intstr.FromString(
							fmt.Sprintf("sum(up{namespace='%s', pod=~'virt-operator-.*'}) or vector(0)", ns),
						),
					},
					{
						Alert: "VirtOperatorDown",
						Expr:  intstr.FromString("kubevirt_virt_operator_up_total == 0"),
						For:   "10m",
						Annotations: map[string]string{
							"summary":     "All virt-operator servers are down.",
							"runbook_url": runbookUrlBasePath + "VirtOperatorDown",
						},
						Labels: map[string]string{
							"severity": "critical",
						},
					},
					{
						Alert: "LowVirtOperatorCount",
						Expr:  intstr.FromString("(num_of_allocatable_nodes > 1) and (kubevirt_virt_operator_up_total < 2)"),
						For:   "60m",
						Annotations: map[string]string{
							"summary":     "More than one virt-operator should be running if more than one worker nodes exist.",
							"runbook_url": runbookUrlBasePath + "LowVirtOperatorCount",
						},
						Labels: map[string]string{
							"severity": "warning",
						},
					},
					{
						Record: "vec_by_virt_operators_all_client_rest_requests_in_last_hour",
						Expr: intstr.FromString(
							fmt.Sprintf("sum by (pod) (sum_over_time(rest_client_requests_total{pod=~'virt-operator-.*', namespace='%s'}[60m]))", ns),
						),
					},
					{
						Record: "vec_by_virt_operators_failed_client_rest_requests_in_last_hour",
						Expr: intstr.FromString(
							fmt.Sprintf("sum by (pod) (sum_over_time(rest_client_requests_total{pod=~'virt-operator-.*', namespace='%s', code=~'(4|5)[0-9][0-9]'}[60m]))", ns),
						),
					},
					{
						Record: "vec_by_virt_operators_all_client_rest_requests_in_last_5m",
						Expr: intstr.FromString(
							fmt.Sprintf("sum by (pod) (sum_over_time(rest_client_requests_total{pod=~'virt-operator-.*', namespace='%s'}[5m]))", ns),
						),
					},
					{
						Record: "vec_by_virt_operators_failed_client_rest_requests_in_last_5m",
						Expr: intstr.FromString(
							fmt.Sprintf("sum by (pod) (sum_over_time(rest_client_requests_total{pod=~'virt-operator-.*', namespace='%s', code=~'(4|5)[0-9][0-9]'}[5m]))", ns),
						),
					},
					{
						Alert: "VirtOperatorRESTErrorsHigh",
						Expr:  intstr.FromString("(vec_by_virt_operators_failed_client_rest_requests_in_last_hour / vec_by_virt_operators_all_client_rest_requests_in_last_hour) >= 0.05"),
						For:   "5m",
						Annotations: map[string]string{
							"summary":     getRestCallsFailedWarning(5, "virt-operator", "hour"),
							"runbook_url": runbookUrlBasePath + "VirtOperatorRESTErrorsHigh",
						},
						Labels: map[string]string{
							"severity": "warning",
						},
					},
					{
						Alert: "VirtOperatorRESTErrorsBurst",
						Expr:  intstr.FromString("(vec_by_virt_operators_failed_client_rest_requests_in_last_5m / vec_by_virt_operators_all_client_rest_requests_in_last_5m) >= 0.8"),
						For:   "5m",
						Annotations: map[string]string{
							"summary":     getRestCallsFailedWarning(80, "virt-operator", "5 minutes"),
							"runbook_url": runbookUrlBasePath + "VirtOperatorRESTErrorsBurst",
						},
						Labels: map[string]string{
							"severity": "critical",
						},
					},
					{
						Record: "kubevirt_virt_operator_ready_total",
						Expr: intstr.FromString(
							fmt.Sprintf("sum(kubevirt_virt_operator_ready{namespace='%s'})", ns),
						),
					},
					{
						Record: "kubevirt_virt_operator_leading_total",
						Expr: intstr.FromString(
							fmt.Sprintf("sum(kubevirt_virt_operator_leading{namespace='%s'})", ns),
						),
					},
					{
						Alert: "LowReadyVirtOperatorsCount",
						Expr:  intstr.FromString("kubevirt_virt_operator_ready_total <  kubevirt_virt_operator_up_total"),
						For:   "10m",
						Annotations: map[string]string{
							"summary":     "Some virt-operators are running but not ready.",
							"runbook_url": runbookUrlBasePath + "LowReadyVirtOperatorsCount",
						},
						Labels: map[string]string{
							"severity": "warning",
						},
					},
					{
						Alert: "NoReadyVirtOperator",
						Expr:  intstr.FromString("kubevirt_virt_operator_ready_total == 0"),
						For:   "10m",
						Annotations: map[string]string{
							"summary":     "No ready virt-operator was detected for the last 10 min.",
							"runbook_url": runbookUrlBasePath + "NoReadyVirtOperator",
						},
						Labels: map[string]string{
							"severity": "critical",
						},
					},
					{
						Alert: "NoLeadingVirtOperator",
						Expr:  intstr.FromString("kubevirt_virt_operator_leading_total == 0"),
						For:   "10m",
						Annotations: map[string]string{
							"summary":     "No leading virt-operator was detected for the last 10 min.",
							"runbook_url": runbookUrlBasePath + "NoLeadingVirtOperator",
						},
						Labels: map[string]string{
							"severity": "critical",
						},
					},
					{
						Record: "kubevirt_virt_handler_up_total",
						Expr:   intstr.FromString(fmt.Sprintf("sum(up{pod=~'virt-handler-.*', namespace='%s'}) or vector(0)", ns)),
					},
					{
						Alert: "VirtHandlerDaemonSetRolloutFailing",
						Expr: intstr.FromString(
							fmt.Sprintf("(%s - %s) != 0",
								fmt.Sprintf("kube_daemonset_status_number_ready{namespace='%s', daemonset='virt-handler'}", ns),
								fmt.Sprintf("kube_daemonset_status_desired_number_scheduled{namespace='%s', daemonset='virt-handler'}", ns))),
						For: "15m",
						Annotations: map[string]string{
							"summary":     "Some virt-handlers failed to roll out",
							"runbook_url": runbookUrlBasePath + "VirtHandlerDaemonSetRolloutFailing",
						},
						Labels: map[string]string{
							"severity": "warning",
						},
					},
					{
						Record: "vec_by_virt_handlers_all_client_rest_requests_in_last_5m",
						Expr: intstr.FromString(
							fmt.Sprintf("sum by (pod) (sum_over_time(rest_client_requests_total{pod=~'virt-handler-.*', namespace='%s'}[5m]))", ns),
						),
					},
					{
						Record: "vec_by_virt_handlers_all_client_rest_requests_in_last_hour",
						Expr: intstr.FromString(
							fmt.Sprintf("sum by (pod) (sum_over_time(rest_client_requests_total{pod=~'virt-handler-.*', namespace='%s'}[60m]))", ns),
						),
					},
					{
						Record: "vec_by_virt_handlers_failed_client_rest_requests_in_last_5m",
						Expr: intstr.FromString(
							fmt.Sprintf("sum by (pod) (sum_over_time(rest_client_requests_total{pod=~'virt-handler-.*', namespace='%s', code=~'(4|5)[0-9][0-9]'}[5m]))", ns),
						),
					},
					{
						Record: "vec_by_virt_handlers_failed_client_rest_requests_in_last_hour",
						Expr: intstr.FromString(
							fmt.Sprintf("sum by (pod) (sum_over_time(rest_client_requests_total{pod=~'virt-handler-.*', namespace='%s', code=~'(4|5)[0-9][0-9]'}[60m]))", ns),
						),
					},
					{
						Alert: "VirtHandlerRESTErrorsHigh",
						Expr:  intstr.FromString("(vec_by_virt_handlers_failed_client_rest_requests_in_last_hour / vec_by_virt_handlers_all_client_rest_requests_in_last_hour) >= 0.05"),
						For:   "5m",
						Annotations: map[string]string{
							"summary":     getRestCallsFailedWarning(5, "virt-handler", "hour"),
							"runbook_url": runbookUrlBasePath + "VirtHandlerRESTErrorsHigh",
						},
						Labels: map[string]string{
							"severity": "warning",
						},
					},
					{
						Alert: "VirtHandlerRESTErrorsBurst",
						Expr:  intstr.FromString("(vec_by_virt_handlers_failed_client_rest_requests_in_last_5m / vec_by_virt_handlers_all_client_rest_requests_in_last_5m) >= 0.8"),
						For:   "5m",
						Annotations: map[string]string{
							"summary":     getRestCallsFailedWarning(80, "virt-handler", "5 minutes"),
							"runbook_url": runbookUrlBasePath + "VirtHandlerRESTErrorsBurst",
						},
						Labels: map[string]string{
							"severity": "critical",
						},
					},
					{
						Record: "vec_by_virt_apis_all_client_rest_requests_in_last_5m",
						Expr: intstr.FromString(
							fmt.Sprintf("sum by (pod) (sum_over_time(rest_client_requests_total{pod=~'virt-api-.*', namespace='%s'}[5m]))", ns),
						),
					},
					{
						Record: "vec_by_virt_apis_all_client_rest_requests_in_last_hour",
						Expr: intstr.FromString(
							fmt.Sprintf("sum by (pod) (sum_over_time(rest_client_requests_total{pod=~'virt-api-.*', namespace='%s'}[60m]))", ns),
						),
					},
					{
						Record: "vec_by_virt_apis_failed_client_rest_requests_in_last_5m",
						Expr: intstr.FromString(
							fmt.Sprintf("sum by (pod) (sum_over_time(rest_client_requests_total{pod=~'virt-api-.*', namespace='%s', code=~'(4|5)[0-9][0-9]'}[5m]))", ns),
						),
					},
					{
						Record: "vec_by_virt_apis_failed_client_rest_requests_in_last_hour",
						Expr: intstr.FromString(
							fmt.Sprintf("sum by (pod) (sum_over_time(rest_client_requests_total{pod=~'virt-api-.*', namespace='%s', code=~'(4|5)[0-9][0-9]'}[60m]))", ns),
						),
					},
					{
						Alert: "VirtApiRESTErrorsHigh",
						Expr:  intstr.FromString("(vec_by_virt_apis_failed_client_rest_requests_in_last_hour / vec_by_virt_apis_all_client_rest_requests_in_last_hour) >= 0.05"),
						For:   "5m",
						Annotations: map[string]string{
							"summary": getRestCallsFailedWarning(5, "virt-api", "hour"),
						},
						Labels: map[string]string{
							"severity": "warning",
						},
					},
					{
						Alert: "VirtApiRESTErrorsBurst",
						Expr:  intstr.FromString("(vec_by_virt_apis_failed_client_rest_requests_in_last_5m / vec_by_virt_apis_all_client_rest_requests_in_last_5m) >= 0.8"),
						For:   "5m",
						Annotations: map[string]string{
							"summary": getRestCallsFailedWarning(80, "virt-api", "5 minutes"),
						},
						Labels: map[string]string{
							"severity": "critical",
						},
					},
					{
						Record: "kubevirt_vm_container_free_memory_bytes",
						Expr:   intstr.FromString("sum by(pod, container) ( kube_pod_container_resource_limits_memory_bytes{pod=~'virt-launcher-.*', container='compute'} - on(pod,container) container_memory_working_set_bytes{pod=~'virt-launcher-.*', container='compute'})"),
					},
					{
						Alert: "KubevirtVmHighMemoryUsage",
						Expr:  intstr.FromString("kubevirt_vm_container_free_memory_bytes < 20971520"),
						For:   "1m",
						Annotations: map[string]string{
							"description": "Container {{ $labels.container }} in pod {{ $labels.pod }} free memory is less than 20 MB and it is close to memory limit",
							"summary":     "VM is at risk of being terminated by the runtime.",
							"runbook_url": runbookUrlBasePath + "KubevirtVmHighMemoryUsage",
						},
						Labels: map[string]string{
							"severity": "warning",
						},
					},
					{
						Record: "kubevirt_num_virt_handlers_by_node_running_virt_launcher",
						Expr:   intstr.FromString("count by(node)(node_namespace_pod:kube_pod_info:{pod=~'virt-launcher-.*'} ) * on (node) group_left(pod) (1*(kube_pod_container_status_ready{pod=~'virt-handler-.*'} + on (pod) group_left(node) (0 * node_namespace_pod:kube_pod_info:{pod=~'virt-handler-.*'} ))) or on (node) (0 * node_namespace_pod:kube_pod_info:{pod=~'virt-launcher-.*'} )"),
					},
					{
						Alert: "OrphanedVirtualMachineImages",
						Expr:  intstr.FromString("(kubevirt_num_virt_handlers_by_node_running_virt_launcher) == 0"),
						For:   "60m",
						Annotations: map[string]string{
							"summary":     "No virt-handler pod detected on node {{ $labels.node }} with running vmis for more than an hour",
							"runbook_url": runbookUrlBasePath + "OrphanedVirtualMachineImages",
						},
						Labels: map[string]string{
							"severity": "warning",
						},
					},
					{
						Alert: "VMCannotBeEvicted",
						Expr:  intstr.FromString("kubevirt_vmi_non_evictable > 0"),
						For:   "1m",
						Annotations: map[string]string{
							"description": "Eviction policy for {{ $labels.name }} (on node {{ $labels.node }}) is set to Live Migration but the VM is not migratable",
							"summary":     "The VM's eviction strategy is set to Live Migration but the VM is not migratable",
							"runbook_url": runbookUrlBasePath + "VMCannotBeEvicted",
						},
						Labels: map[string]string{
							"severity": "warning",
						},
					},
					{
						Alert: "KubeVirtComponentExceedsRequestedMemory",
						Expr:  intstr.FromString(fmt.Sprintf(`((kube_pod_container_resource_requests{namespace="%s",container=~"virt-controller|virt-api|virt-handler|virt-operator",resource="memory"}) - on(pod) group_left(node) container_memory_usage_bytes{namespace="%s"}) < 0`, ns, ns)),
						For:   "5m",
						Annotations: map[string]string{
							"description": "Container {{ $labels.container }} in pod {{ $labels.pod }} memory usage exceeds the memory requested",
							"summary":     "The container is using more memory than what is defined in the containers resource requests",
							"runbook_url": runbookUrlBasePath + "KubeVirtComponentExceedsRequestedMemory",
						},
						Labels: map[string]string{
							"severity": "warning",
						},
					},
					{
						Alert: "KubeVirtComponentExceedsRequestedCPU",
						Expr: intstr.FromString(
							fmt.Sprintf(`((kube_pod_container_resource_requests{namespace="%s",container=~"virt-controller|virt-api|virt-handler|virt-operator",resource="cpu"}) - on(pod) group_left(node) node_namespace_pod_container:container_cpu_usage_seconds_total:sum_rate{namespace="%s"}) < 0`, ns, ns),
						),
						For: "5m",
						Annotations: map[string]string{
							"description": "Container {{ $labels.container }} in pod {{ $labels.pod }} cpu usage exceeds the CPU requested",
							"summary":     "The container is using more CPU than what is defined in the containers resource requests",
							"runbook_url": runbookUrlBasePath + "KubeVirtComponentExceedsRequestedCPU",
						},
						Labels: map[string]string{
							"severity": "warning",
						},
					},
				},
			},
		},
	}

	if workloadUpdatesEnabled {
		ruleSpec.Groups[0].Rules = append(ruleSpec.Groups[0].Rules, v1.Rule{

			Alert: "OutdatedVirtualMachineInstanceWorkloads",
			Expr:  intstr.FromString("kubevirt_vmi_outdated_count != 0"),
			For:   "1440m",
			Annotations: map[string]string{
				"summary":     "Some running VMIs are still active in outdated pods after KubeVirt control plane update has completed.",
				"runbook_url": runbookUrlBasePath + "OutdatedVirtualMachineInstanceWorkloads",
			},
		})
	}

	return ruleSpec
}
