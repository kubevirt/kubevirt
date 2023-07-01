package components

import (
	"errors"
	"fmt"
	"os"
	"strings"

	prometheusv1 "github.com/prometheus/client_golang/api/prometheus/v1"

	"github.com/coreos/prometheus-operator/pkg/apis/monitoring"
	v1 "github.com/coreos/prometheus-operator/pkg/apis/monitoring/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	v12 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
)

const (
	KUBEVIRT_PROMETHEUS_RULE_NAME = "prometheus-kubevirt-rules"
	prometheusLabelKey            = "prometheus.kubevirt.io"
	prometheusLabelValue          = "true"
	defaultRunbookURLTemplate     = "https://kubevirt.io/monitoring/runbooks/%s"
	runbookURLTemplateEnv         = "RUNBOOK_URL_TEMPLATE"
	severityAlertLabelKey         = "severity"
	operatorHealthImpactLabelKey  = "operator_health_impact"
	partOfAlertLabelKey           = "kubernetes_operator_part_of"
	partOfAlertLabelValue         = "kubevirt"
	componentAlertLabelKey        = "kubernetes_operator_component"
	componentAlertLabelValue      = "kubevirt"
	durationFiveMinutes           = "5 minutes"
)

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
	fiftyMB := resource.MustParse("50Mi")

	getRestCallsFailedWarning := func(failingCallsPercentage int, component, duration string) string {
		const restCallsFailWarningTemplate = "More than %d%% of the rest calls failed in %s for the last %s"
		return fmt.Sprintf(restCallsFailWarningTemplate, failingCallsPercentage, component, duration)
	}
	getErrorRatio := func(ns string, podName string, errorCodeRegex string, durationInMinutes int) string {
		errorRatioQuery := "sum ( rate ( rest_client_requests_total{namespace=\"%s\",pod=~\"%s-.*\",code=~\"%s\"} [%dm] ) )  /  sum ( rate ( rest_client_requests_total{namespace=\"%s\",pod=~\"%s-.*\"} [%dm] ) )"
		return fmt.Sprintf(errorRatioQuery, ns, podName, errorCodeRegex, durationInMinutes, ns, podName, durationInMinutes)
	}

	runbookURLTemplate := getRunbookURLTemplate()

	var kubevirtRules []v1.Rule
	for _, rule := range GetRecordingRules(ns) {
		kubevirtRules = append(kubevirtRules, rule.Rule)
	}

	kubevirtRules = append(kubevirtRules, []v1.Rule{
		{
			Alert: "VirtAPIDown",
			Expr:  intstr.FromString("kubevirt_virt_api_up_total == 0"),
			For:   "10m",
			Annotations: map[string]string{
				"summary":     "All virt-api servers are down.",
				"runbook_url": fmt.Sprintf(runbookURLTemplate, "VirtAPIDown"),
			},
			Labels: map[string]string{
				severityAlertLabelKey:        "critical",
				operatorHealthImpactLabelKey: "critical",
			},
		},
		{
			Alert: "LowVirtAPICount",
			Expr:  intstr.FromString("(kubevirt_allocatable_nodes_count > 1) and (kubevirt_virt_api_up_total < 2)"),
			For:   "60m",
			Annotations: map[string]string{
				"summary":     "More than one virt-api should be running if more than one worker nodes exist.",
				"runbook_url": fmt.Sprintf(runbookURLTemplate, "LowVirtAPICount"),
			},
			Labels: map[string]string{
				severityAlertLabelKey:        "warning",
				operatorHealthImpactLabelKey: "warning",
			},
		},
		{
			Alert: "LowKVMNodesCount",
			Expr:  intstr.FromString("(kubevirt_allocatable_nodes_count > 1) and (kubevirt_kvm_available_nodes_count < 2)"),
			For:   "5m",
			Annotations: map[string]string{
				"description": "Low number of nodes with KVM resource available.",
				"summary":     "At least two nodes with kvm resource required for VM live migration.",
				"runbook_url": fmt.Sprintf(runbookURLTemplate, "LowKVMNodesCount"),
			},
			Labels: map[string]string{
				severityAlertLabelKey:        "warning",
				operatorHealthImpactLabelKey: "warning",
			},
		},
		{
			Alert: "LowReadyVirtControllersCount",
			Expr:  intstr.FromString("kubevirt_virt_controller_ready_total <  kubevirt_virt_controller_up_total"),
			For:   "10m",
			Annotations: map[string]string{
				"summary":     "Some virt controllers are running but not ready.",
				"runbook_url": fmt.Sprintf(runbookURLTemplate, "LowReadyVirtControllersCount"),
			},
			Labels: map[string]string{
				severityAlertLabelKey:        "warning",
				operatorHealthImpactLabelKey: "warning",
			},
		},
		{
			Alert: "NoReadyVirtController",
			Expr:  intstr.FromString("kubevirt_virt_controller_ready_total == 0"),
			For:   "10m",
			Annotations: map[string]string{
				"summary":     "No ready virt-controller was detected for the last 10 min.",
				"runbook_url": fmt.Sprintf(runbookURLTemplate, "NoReadyVirtController"),
			},
			Labels: map[string]string{
				severityAlertLabelKey:        "critical",
				operatorHealthImpactLabelKey: "critical",
			},
		},
		{
			Alert: "VirtControllerDown",
			Expr:  intstr.FromString("kubevirt_virt_controller_up_total == 0"),
			For:   "10m",
			Annotations: map[string]string{
				"summary":     "No running virt-controller was detected for the last 10 min.",
				"runbook_url": fmt.Sprintf(runbookURLTemplate, "VirtControllerDown"),
			},
			Labels: map[string]string{
				severityAlertLabelKey:        "critical",
				operatorHealthImpactLabelKey: "critical",
			},
		},
		{
			Alert: "LowVirtControllersCount",
			Expr:  intstr.FromString("(kubevirt_allocatable_nodes_count > 1) and (kubevirt_virt_controller_ready_total < 2)"),
			For:   "10m",
			Annotations: map[string]string{
				"summary":     "More than one virt-controller should be ready if more than one worker node.",
				"runbook_url": fmt.Sprintf(runbookURLTemplate, "LowVirtControllersCount"),
			},
			Labels: map[string]string{
				severityAlertLabelKey:        "warning",
				operatorHealthImpactLabelKey: "warning",
			},
		},
		{
			Alert: "VirtControllerRESTErrorsHigh",
			Expr:  intstr.FromString(getErrorRatio(ns, "virt-controller", "(4|5)[0-9][0-9]", 60) + " >= 0.05"),
			Annotations: map[string]string{
				"summary":     getRestCallsFailedWarning(5, "virt-controller", "hour"),
				"runbook_url": fmt.Sprintf(runbookURLTemplate, "VirtControllerRESTErrorsHigh"),
			},
			Labels: map[string]string{
				severityAlertLabelKey:        "warning",
				operatorHealthImpactLabelKey: "warning",
			},
		},
		{
			Alert: "VirtControllerRESTErrorsBurst",
			Expr:  intstr.FromString(getErrorRatio(ns, "virt-controller", "(4|5)[0-9][0-9]", 5) + " >= 0.8"),
			For:   "5m",
			Annotations: map[string]string{
				"summary":     getRestCallsFailedWarning(80, "virt-controller", durationFiveMinutes),
				"runbook_url": fmt.Sprintf(runbookURLTemplate, "VirtControllerRESTErrorsBurst"),
			},
			Labels: map[string]string{
				severityAlertLabelKey:        "critical",
				operatorHealthImpactLabelKey: "critical",
			},
		},
		{
			Alert: "VirtOperatorDown",
			Expr:  intstr.FromString("kubevirt_virt_operator_up_total == 0"),
			For:   "10m",
			Annotations: map[string]string{
				"summary":     "All virt-operator servers are down.",
				"runbook_url": fmt.Sprintf(runbookURLTemplate, "VirtOperatorDown"),
			},
			Labels: map[string]string{
				severityAlertLabelKey:        "critical",
				operatorHealthImpactLabelKey: "critical",
			},
		},
		{
			Alert: "LowVirtOperatorCount",
			Expr:  intstr.FromString("(kubevirt_allocatable_nodes_count > 1) and (kubevirt_virt_operator_up_total < 2)"),
			For:   "60m",
			Annotations: map[string]string{
				"summary":     "More than one virt-operator should be running if more than one worker nodes exist.",
				"runbook_url": fmt.Sprintf(runbookURLTemplate, "LowVirtOperatorCount"),
			},
			Labels: map[string]string{
				severityAlertLabelKey:        "warning",
				operatorHealthImpactLabelKey: "warning",
			},
		},
		{
			Alert: "VirtOperatorRESTErrorsHigh",
			Expr:  intstr.FromString(getErrorRatio(ns, "virt-operator", "(4|5)[0-9][0-9]", 60) + " >= 0.05"),
			Annotations: map[string]string{
				"summary":     getRestCallsFailedWarning(5, "virt-operator", "hour"),
				"runbook_url": fmt.Sprintf(runbookURLTemplate, "VirtOperatorRESTErrorsHigh"),
			},
			Labels: map[string]string{
				severityAlertLabelKey:        "warning",
				operatorHealthImpactLabelKey: "warning",
			},
		},
		{
			Alert: "VirtOperatorRESTErrorsBurst",
			Expr:  intstr.FromString(getErrorRatio(ns, "virt-operator", "(4|5)[0-9][0-9]", 5) + " >= 0.8"),
			For:   "5m",
			Annotations: map[string]string{
				"summary":     getRestCallsFailedWarning(80, "virt-operator", durationFiveMinutes),
				"runbook_url": fmt.Sprintf(runbookURLTemplate, "VirtOperatorRESTErrorsBurst"),
			},
			Labels: map[string]string{
				severityAlertLabelKey:        "critical",
				operatorHealthImpactLabelKey: "critical",
			},
		},
		{
			Alert: "LowReadyVirtOperatorsCount",
			Expr:  intstr.FromString("kubevirt_virt_operator_ready_total <  kubevirt_virt_operator_up_total"),
			For:   "10m",
			Annotations: map[string]string{
				"summary":     "Some virt-operators are running but not ready.",
				"runbook_url": fmt.Sprintf(runbookURLTemplate, "LowReadyVirtOperatorsCount"),
			},
			Labels: map[string]string{
				severityAlertLabelKey:        "warning",
				operatorHealthImpactLabelKey: "warning",
			},
		},
		{
			Alert: "NoReadyVirtOperator",
			Expr:  intstr.FromString("kubevirt_virt_operator_ready_total == 0"),
			For:   "10m",
			Annotations: map[string]string{
				"summary":     "No ready virt-operator was detected for the last 10 min.",
				"runbook_url": fmt.Sprintf(runbookURLTemplate, "NoReadyVirtOperator"),
			},
			Labels: map[string]string{
				severityAlertLabelKey:        "critical",
				operatorHealthImpactLabelKey: "critical",
			},
		},
		{
			Alert: "NoLeadingVirtOperator",
			Expr:  intstr.FromString("kubevirt_virt_operator_leading_total == 0"),
			For:   "10m",
			Annotations: map[string]string{
				"summary":     "No leading virt-operator was detected for the last 10 min.",
				"runbook_url": fmt.Sprintf(runbookURLTemplate, "NoLeadingVirtOperator"),
			},
			Labels: map[string]string{
				severityAlertLabelKey:        "critical",
				operatorHealthImpactLabelKey: "critical",
			},
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
				"runbook_url": fmt.Sprintf(runbookURLTemplate, "VirtHandlerDaemonSetRolloutFailing"),
			},
			Labels: map[string]string{
				severityAlertLabelKey:        "warning",
				operatorHealthImpactLabelKey: "warning",
			},
		},
		{
			Alert: "VirtHandlerRESTErrorsHigh",
			Expr:  intstr.FromString(getErrorRatio(ns, "virt-handler", "(4|5)[0-9][0-9]", 60) + " >= 0.05"),
			Annotations: map[string]string{
				"summary":     getRestCallsFailedWarning(5, "virt-handler", "hour"),
				"runbook_url": fmt.Sprintf(runbookURLTemplate, "VirtHandlerRESTErrorsHigh"),
			},
			Labels: map[string]string{
				severityAlertLabelKey:        "warning",
				operatorHealthImpactLabelKey: "warning",
			},
		},
		{
			Alert: "VirtHandlerRESTErrorsBurst",
			Expr:  intstr.FromString(getErrorRatio(ns, "virt-handler", "(4|5)[0-9][0-9]", 5) + " >= 0.8"),
			For:   "5m",
			Annotations: map[string]string{
				"summary":     getRestCallsFailedWarning(80, "virt-handler", durationFiveMinutes),
				"runbook_url": fmt.Sprintf(runbookURLTemplate, "VirtHandlerRESTErrorsBurst"),
			},
			Labels: map[string]string{
				severityAlertLabelKey:        "critical",
				operatorHealthImpactLabelKey: "critical",
			},
		},
		{
			Alert: "VirtApiRESTErrorsHigh",
			Expr:  intstr.FromString(getErrorRatio(ns, "virt-api", "(4|5)[0-9][0-9]", 60) + " >= 0.05"),
			Annotations: map[string]string{
				"summary":     getRestCallsFailedWarning(5, "virt-api", "hour"),
				"runbook_url": fmt.Sprintf(runbookURLTemplate, "VirtApiRESTErrorsHigh"),
			},
			Labels: map[string]string{
				severityAlertLabelKey:        "warning",
				operatorHealthImpactLabelKey: "warning",
			},
		},
		{
			Alert: "VirtApiRESTErrorsBurst",
			Expr:  intstr.FromString(getErrorRatio(ns, "virt-api", "(4|5)[0-9][0-9]", 5) + " >= 0.8"),
			For:   "5m",
			Annotations: map[string]string{
				"summary":     getRestCallsFailedWarning(80, "virt-api", durationFiveMinutes),
				"runbook_url": fmt.Sprintf(runbookURLTemplate, "VirtApiRESTErrorsBurst"),
			},
			Labels: map[string]string{
				severityAlertLabelKey:        "critical",
				operatorHealthImpactLabelKey: "critical",
			},
		},
		{
			Alert: "KubevirtVmHighMemoryUsage",
			Expr:  intstr.FromString("kubevirt_vm_container_free_memory_bytes_based_on_working_set_bytes < 52428800 or kubevirt_vm_container_free_memory_bytes_based_on_rss < 52428800"),
			For:   "1m",
			Annotations: map[string]string{
				"description": fmt.Sprintf("Container {{ $labels.container }} in pod {{ $labels.pod }} in namespace {{ $labels.namespace }} free memory is less than %s and it is close to requested memory", fiftyMB.String()),
				"summary":     "VM is at risk of being evicted and in serious cases of memory exhaustion being terminated by the runtime.",
				"runbook_url": fmt.Sprintf(runbookURLTemplate, "KubevirtVmHighMemoryUsage"),
			},
			Labels: map[string]string{
				severityAlertLabelKey:        "warning",
				operatorHealthImpactLabelKey: "none",
			},
		},
		{
			Alert: "OrphanedVirtualMachineInstances",
			Expr:  intstr.FromString("(((max by (node) (kube_pod_status_ready{condition='true',pod=~'virt-handler.*'} * on(pod) group_left(node) max by(pod,node)(kube_pod_info{pod=~'virt-handler.*',node!=''})) ) == 1) or (count by (node)( kube_pod_info{pod=~'virt-launcher.*',node!=''})*0)) == 0"),
			For:   "10m",
			Annotations: map[string]string{
				"summary":     "No ready virt-handler pod detected on node {{ $labels.node }} with running vmis for more than 10 minutes",
				"runbook_url": fmt.Sprintf(runbookURLTemplate, "OrphanedVirtualMachineInstances"),
			},
			Labels: map[string]string{
				severityAlertLabelKey:        "warning",
				operatorHealthImpactLabelKey: "none",
			},
		},
		{
			Alert: "VMCannotBeEvicted",
			Expr:  intstr.FromString("kubevirt_vmi_non_evictable > 0"),
			For:   "1m",
			Annotations: map[string]string{
				"description": "Eviction policy for {{ $labels.name }} (on node {{ $labels.node }}) is set to Live Migration but the VM is not migratable",
				"summary":     "The VM's eviction strategy is set to Live Migration but the VM is not migratable",
				"runbook_url": fmt.Sprintf(runbookURLTemplate, "VMCannotBeEvicted"),
			},
			Labels: map[string]string{
				severityAlertLabelKey:        "warning",
				operatorHealthImpactLabelKey: "none",
			},
		},
		{
			Alert: "KubeVirtComponentExceedsRequestedMemory",
			Expr: intstr.FromString(
				// In 'container_memory_working_set_bytes', 'container=""' filters the accumulated metric for the pod slice to measure total Memory usage for all containers within the pod
				fmt.Sprintf(`((kube_pod_container_resource_requests{namespace="%s",container=~"virt-controller|virt-api|virt-handler|virt-operator",resource="memory"}) - on(pod) group_left(node) container_memory_working_set_bytes{container="",namespace="%s"}) < 0`, ns, ns)),
			For: "5m",
			Annotations: map[string]string{
				"description": "Container {{ $labels.container }} in pod {{ $labels.pod }} memory usage exceeds the memory requested",
				"summary":     "The container is using more memory than what is defined in the containers resource requests",
				"runbook_url": fmt.Sprintf(runbookURLTemplate, "KubeVirtComponentExceedsRequestedMemory"),
			},
			Labels: map[string]string{
				severityAlertLabelKey:        "warning",
				operatorHealthImpactLabelKey: "none",
			},
		},
		{
			Alert: "KubeVirtComponentExceedsRequestedCPU",
			Expr: intstr.FromString(
				// In 'container_cpu_usage_seconds_total', 'container=""' filters the accumulated metric for the pod slice to measure total CPU usage for all containers within the pod
				fmt.Sprintf(`((kube_pod_container_resource_requests{namespace="%s",container=~"virt-controller|virt-api|virt-handler|virt-operator",resource="cpu"}) - on(pod) sum(rate(container_cpu_usage_seconds_total{container="",namespace="%s"}[5m])) by (pod)) < 0`, ns, ns),
			),
			For: "5m",
			Annotations: map[string]string{
				"description": "Pod {{ $labels.pod }} cpu usage exceeds the CPU requested",
				"summary":     "The containers in the pod are using more CPU than what is defined in the containers resource requests",
				"runbook_url": fmt.Sprintf(runbookURLTemplate, "KubeVirtComponentExceedsRequestedCPU"),
			},
			Labels: map[string]string{
				severityAlertLabelKey:        "warning",
				operatorHealthImpactLabelKey: "none",
			},
		},
		{
			Alert: "KubeVirtVMIExcessiveMigrations",
			Expr:  intstr.FromString("sum by (vmi) (max_over_time(kubevirt_migrate_vmi_succeeded[1d])) >= 12"),
			Annotations: map[string]string{
				"description": "VirtualMachineInstance {{ $labels.vmi }} has been migrated more than 12 times during the last 24 hours",
				"summary":     "An excessive amount of migrations have been detected on a VirtualMachineInstance in the last 24 hours.",
				"runbook_url": fmt.Sprintf(runbookURLTemplate, "KubeVirtVMIExcessiveMigrations"),
			},
			Labels: map[string]string{
				severityAlertLabelKey:        "warning",
				operatorHealthImpactLabelKey: "none",
			},
		},
		{
			Alert: "KubeVirtNoAvailableNodesToRunVMs",
			Expr:  intstr.FromString("((sum(kube_node_status_allocatable{resource='devices_kubevirt_io_kvm'}) or on() vector(0)) == 0 and (sum(kubevirt_configuration_emulation_enabled) or on() vector(0)) == 0) or (sum(kube_node_labels{label_kubevirt_io_schedulable='true'}) or on() vector(0)) == 0"),
			For:   "5m",
			Annotations: map[string]string{
				"summary":     "There are no available nodes in the cluster to run VMs.",
				"runbook_url": fmt.Sprintf(runbookURLTemplate, "KubeVirtNoAvailableNodesToRunVMs"),
			},
			Labels: map[string]string{
				severityAlertLabelKey:        "warning",
				operatorHealthImpactLabelKey: "critical",
			},
		},
		{
			Alert: "KubeVirtDeprecatedAPIsRequested",
			Expr:  intstr.FromString("sum by (resource,group,version) ((round(increase(kubevirt_api_request_deprecated_total{verb!~\"LIST|WATCH\"}[10m])) > 0 and kubevirt_api_request_deprecated_total{verb!~\"LIST|WATCH\"} offset 10m) or (kubevirt_api_request_deprecated_total{verb!~\"LIST|WATCH\"} != 0 unless kubevirt_api_request_deprecated_total{verb!~\"LIST|WATCH\"} offset 10m))"),
			Annotations: map[string]string{
				"description": "Detected requests to the deprecated {{ $labels.resource }}.{{ $labels.group }}/{{ $labels.version }} API.",
				"summary":     "Detected {{ $value }} requests in the last 10 minutes.",
				"runbook_url": fmt.Sprintf(runbookURLTemplate, "KubeVirtDeprecatedAPIsRequested"),
			},
			Labels: map[string]string{
				severityAlertLabelKey:        "info",
				operatorHealthImpactLabelKey: "none",
			},
		},
	}...)

	ruleSpec := &v1.PrometheusRuleSpec{
		Groups: []v1.RuleGroup{
			{
				Name:  "kubevirt.rules",
				Rules: kubevirtRules,
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
				"runbook_url": fmt.Sprintf(runbookURLTemplate, "OutdatedVirtualMachineInstanceWorkloads"),
			},
			Labels: map[string]string{
				severityAlertLabelKey:        "warning",
				operatorHealthImpactLabelKey: "none",
			},
		})
	}

	for _, group := range ruleSpec.Groups {
		for _, rule := range group.Rules {
			if rule.Alert == "" {
				continue
			}
			rule.Labels[partOfAlertLabelKey] = partOfAlertLabelValue
			rule.Labels[componentAlertLabelKey] = componentAlertLabelValue
		}
	}

	return ruleSpec
}

type KubevirtRecordingRule struct {
	v1.Rule
	MType       prometheusv1.MetricType
	Description string
}

func GetRecordingRules(namespace string) []KubevirtRecordingRule {
	return []KubevirtRecordingRule{
		{
			Rule: v1.Rule{
				Record: "kubevirt_virt_api_up_total",
				Expr: intstr.FromString(
					fmt.Sprintf("sum(up{namespace='%s', pod=~'virt-api-.*'}) or vector(0)", namespace),
				),
			},
			MType:       prometheusv1.MetricTypeGauge,
			Description: "The number of virt-api pods that are up.",
		},
		{
			Rule: v1.Rule{
				Record: "kubevirt_allocatable_nodes_count",
				Expr:   intstr.FromString("count(count (kube_node_status_allocatable) by (node))"),
			},
			MType:       prometheusv1.MetricTypeGauge,
			Description: "The number of nodes in the cluster that have the devices.kubevirt.io/kvm resource available.",
		},
		{
			Rule: v1.Rule{
				Record: "kubevirt_kvm_available_nodes_count",
				Expr:   intstr.FromString("kubevirt_allocatable_nodes_count - count(kube_node_status_allocatable{resource=\"devices_kubevirt_io_kvm\"} == 0)"),
			},
			MType:       prometheusv1.MetricTypeGauge,
			Description: "The number of nodes in the cluster that have the devices.kubevirt.io/kvm resource available.",
		},
		{
			Rule: v1.Rule{
				Record: "kubevirt_virt_controller_up_total",
				Expr: intstr.FromString(
					fmt.Sprintf("sum(up{pod=~'virt-controller-.*', namespace='%s'}) or vector(0)", namespace),
				),
			},
			MType:       prometheusv1.MetricTypeGauge,
			Description: "The number of virt-controller pods that are up.",
		},
		{
			Rule: v1.Rule{
				Record: "kubevirt_virt_controller_ready_total",
				Expr: intstr.FromString(
					fmt.Sprintf("sum(kubevirt_virt_controller_ready{namespace='%s'}) or vector(0)", namespace),
				),
			},
			MType:       prometheusv1.MetricTypeGauge,
			Description: "The number of virt-controller pods that are ready.",
		},
		{
			Rule: v1.Rule{
				Record: "kubevirt_virt_operator_up_total",
				Expr: intstr.FromString(
					fmt.Sprintf("sum(up{namespace='%s', pod=~'virt-operator-.*'}) or vector(0)", namespace),
				),
			},
			MType:       prometheusv1.MetricTypeGauge,
			Description: "The number of virt-operator pods that are up.",
		},
		{
			Rule: v1.Rule{
				Record: "kubevirt_virt_operator_ready_total",
				Expr: intstr.FromString(
					fmt.Sprintf("sum(kubevirt_virt_operator_ready{namespace='%s'}) or vector(0)", namespace),
				),
			},
			MType:       prometheusv1.MetricTypeGauge,
			Description: "The number of virt-operator pods that are ready.",
		},
		{
			Rule: v1.Rule{
				Record: "kubevirt_virt_operator_leading_total",
				Expr: intstr.FromString(
					fmt.Sprintf("sum(kubevirt_virt_operator_leading{namespace='%s'})", namespace),
				),
			},
			MType:       prometheusv1.MetricTypeGauge,
			Description: "The number of virt-operator pods that are leading.",
		},
		{
			Rule: v1.Rule{
				Record: "kubevirt_virt_handler_up_total",
				Expr:   intstr.FromString(fmt.Sprintf("sum(up{pod=~'virt-handler-.*', namespace='%s'}) or vector(0)", namespace)),
			},
			MType:       prometheusv1.MetricTypeGauge,
			Description: "The number of virt-handler pods that are up.",
		},
		{
			Rule: v1.Rule{
				Record: "kubevirt_vmi_memory_used_bytes",
				Expr:   intstr.FromString("kubevirt_vmi_memory_available_bytes-kubevirt_vmi_memory_usable_bytes"),
			},
			MType:       prometheusv1.MetricTypeGauge,
			Description: "Amount of `used` memory as seen by the domain.",
		},
		{
			Rule: v1.Rule{
				Record: "kubevirt_vm_container_free_memory_bytes_based_on_working_set_bytes",
				Expr:   intstr.FromString("sum by(pod, container, namespace) (kube_pod_container_resource_requests{pod=~'virt-launcher-.*', container='compute', resource='memory'}- on(pod,container, namespace) container_memory_working_set_bytes{pod=~'virt-launcher-.*', container='compute'})"),
			},
			MType:       prometheusv1.MetricTypeGauge,
			Description: "The current available memory of the VM containers based on the working set.",
		},
		{
			Rule: v1.Rule{
				Record: "kubevirt_vm_container_free_memory_bytes_based_on_rss",
				Expr:   intstr.FromString("sum by(pod, container, namespace) (kube_pod_container_resource_requests{pod=~'virt-launcher-.*', container='compute', resource='memory'}- on(pod,container, namespace) container_memory_rss{pod=~'virt-launcher-.*', container='compute'})"),
			},
			MType:       prometheusv1.MetricTypeGauge,
			Description: "The current available memory of the VM containers based on the rss.",
		},
		{
			Rule: v1.Rule{
				Record: "kubevirt_vmsnapshot_persistentvolumeclaim_labels",
				Expr:   intstr.FromString("label_replace(label_replace(kube_persistentvolumeclaim_labels{label_restore_kubevirt_io_source_vm_name!='', label_restore_kubevirt_io_source_vm_namespace!=''} == 1, 'vm_namespace', '$1', 'label_restore_kubevirt_io_source_vm_namespace', '(.*)'), 'vm_name', '$1', 'label_restore_kubevirt_io_source_vm_name', '(.*)')"),
			},
			MType:       prometheusv1.MetricTypeInfo,
			Description: "Returns the labels of the persistent volume claims that are used for restoring virtual machines.",
		},
		{
			Rule: v1.Rule{
				Record: "kubevirt_vmsnapshot_disks_restored_from_source_total",
				Expr:   intstr.FromString("sum by(vm_name, vm_namespace) (kubevirt_vmsnapshot_persistentvolumeclaim_labels)"),
			},
			MType:       prometheusv1.MetricTypeGauge,
			Description: "Returns the total number of virtual machine disks restored from the source virtual machine.",
		},
		{
			Rule: v1.Rule{
				Record: "kubevirt_vmsnapshot_disks_restored_from_source_bytes",
				Expr:   intstr.FromString("sum by(vm_name, vm_namespace) (kube_persistentvolumeclaim_resource_requests_storage_bytes * on(persistentvolumeclaim, namespace) group_left(vm_name, vm_namespace) kubevirt_vmsnapshot_persistentvolumeclaim_labels)"),
			},
			MType:       prometheusv1.MetricTypeGauge,
			Description: "Returns the amount of space in bytes restored from the source virtual machine.",
		},
		{
			Rule: v1.Rule{
				Record: "kubevirt_number_of_vms",
				Expr:   intstr.FromString("sum by (namespace) (count by (name,namespace) (kubevirt_vm_error_status_last_transition_timestamp_seconds + kubevirt_vm_migrating_status_last_transition_timestamp_seconds + kubevirt_vm_non_running_status_last_transition_timestamp_seconds + kubevirt_vm_running_status_last_transition_timestamp_seconds + kubevirt_vm_starting_status_last_transition_timestamp_seconds))"),
			},
			MType:       prometheusv1.MetricTypeGauge,
			Description: "The number of VMs in the cluster by namespace.",
		},
		{
			Rule: v1.Rule{
				Record: "kubevirt_api_request_deprecated_total",
				Expr:   intstr.FromString("group by (group,version,resource,subresource) (apiserver_requested_deprecated_apis{group=\"kubevirt.io\"}) * on (group,version,resource,subresource) group_right() sum by (group,version,resource,subresource,verb) (apiserver_request_total)"),
			},
			MType:       prometheusv1.MetricTypeCounter,
			Description: "The total number of requests to deprecated KubeVirt APIs.",
		},
	}
}

func getRunbookURLTemplate() string {
	runbookURLTemplate, exists := os.LookupEnv(runbookURLTemplateEnv)
	if !exists {
		runbookURLTemplate = defaultRunbookURLTemplate
	}

	if strings.Count(runbookURLTemplate, "%s") != 1 {
		panic(errors.New("runbook URL template must have exactly 1 %s substring"))
	}

	return runbookURLTemplate
}
