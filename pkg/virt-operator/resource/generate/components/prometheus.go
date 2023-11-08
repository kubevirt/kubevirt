package components

import (
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/prometheus-operator/prometheus-operator/pkg/apis/monitoring"
	promv1 "github.com/prometheus-operator/prometheus-operator/pkg/apis/monitoring/v1"
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
	durationFiveMinutes           = "5 minutes"
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

func GetPrometheusAlerts(ns string) []promv1.Rule {
	getRestCallsFailedWarning := func(failingCallsPercentage int, component, duration string) string {
		const restCallsFailWarningTemplate = "More than %d%% of the rest calls failed in %s for the last %s"
		return fmt.Sprintf(restCallsFailWarningTemplate, failingCallsPercentage, component, duration)
	}
	getErrorRatio := func(ns string, podName string, errorCodeRegex string, durationInMinutes int) string {
		errorRatioQuery := "sum ( rate ( rest_client_requests_total{namespace=\"%s\",pod=~\"%s-.*\",code=~\"%s\"} [%dm] ) )  /  sum ( rate ( rest_client_requests_total{namespace=\"%s\",pod=~\"%s-.*\"} [%dm] ) )"
		return fmt.Sprintf(errorRatioQuery, ns, podName, errorCodeRegex, durationInMinutes, ns, podName, durationInMinutes)
	}

	fiftyMB := resource.MustParse("50Mi")

	runbookURLTemplate := getRunbookURLTemplate()

	oneMinute := promv1.Duration("1m")
	fiveMinutes := promv1.Duration("5m")
	tenMinutes := promv1.Duration("10m")
	fifteenMinutes := promv1.Duration("15m")
	sixtyMinutes := promv1.Duration("60m")
	oneDay := promv1.Duration("24h")

	return []promv1.Rule{
		{
			Alert: "VirtAPIDown",
			Expr:  intstr.FromString("kubevirt_virt_api_up == 0"),
			For:   &tenMinutes,
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
			Expr:  intstr.FromString("(kubevirt_allocatable_nodes > 1) and (kubevirt_virt_api_up < 2)"),
			For:   &sixtyMinutes,
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
			Expr:  intstr.FromString("(kubevirt_allocatable_nodes > 1) and (kubevirt_nodes_with_kvm < 2)"),
			For:   &fiveMinutes,
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
			Expr:  intstr.FromString("kubevirt_virt_controller_ready <  kubevirt_virt_controller_up"),
			For:   &tenMinutes,
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
			Expr:  intstr.FromString("kubevirt_virt_controller_ready == 0"),
			For:   &tenMinutes,
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
			Expr:  intstr.FromString("kubevirt_virt_controller_up == 0"),
			For:   &tenMinutes,
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
			Expr:  intstr.FromString("(kubevirt_allocatable_nodes > 1) and (kubevirt_virt_controller_ready < 2)"),
			For:   &tenMinutes,
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
			For:   &fiveMinutes,
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
			Expr:  intstr.FromString("kubevirt_virt_operator_up == 0"),
			For:   &tenMinutes,
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
			Expr:  intstr.FromString("(kubevirt_allocatable_nodes > 1) and (kubevirt_virt_operator_up < 2)"),
			For:   &sixtyMinutes,
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
			For:   &fiveMinutes,
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
			Expr:  intstr.FromString("kubevirt_virt_operator_ready <  kubevirt_virt_operator_up"),
			For:   &tenMinutes,
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
			Expr:  intstr.FromString("kubevirt_virt_operator_ready == 0"),
			For:   &tenMinutes,
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
			Expr:  intstr.FromString("kubevirt_virt_operator_leading == 0"),
			For:   &tenMinutes,
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
			For: &fifteenMinutes,
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
			For:   &fiveMinutes,
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
			For:   &fiveMinutes,
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
			For:   &oneMinute,
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
			For:   &tenMinutes,
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
			For:   &oneMinute,
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
			For: &fiveMinutes,
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
			For: &fiveMinutes,
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
			Expr:  intstr.FromString("sum by (vmi) (max_over_time(kubevirt_vmi_migration_succeeded[1d])) >= 12"),
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
			For:   &fiveMinutes,
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
			Alert: "KubeVirtDeprecatedAPIRequested",
			Expr:  intstr.FromString("sum by (resource,group,version) ((round(increase(kubevirt_api_request_deprecated_total{verb!~\"LIST|WATCH\"}[10m])) > 0 and kubevirt_api_request_deprecated_total{verb!~\"LIST|WATCH\"} offset 10m) or (kubevirt_api_request_deprecated_total{verb!~\"LIST|WATCH\"} != 0 unless kubevirt_api_request_deprecated_total{verb!~\"LIST|WATCH\"} offset 10m))"),
			Annotations: map[string]string{
				"description": "Detected requests to the deprecated {{ $labels.resource }}.{{ $labels.group }}/{{ $labels.version }} API.",
				"summary":     "Detected {{ $value }} requests in the last 10 minutes.",
				"runbook_url": fmt.Sprintf(runbookURLTemplate, "KubeVirtDeprecatedAPIRequested"),
			},
			Labels: map[string]string{
				severityAlertLabelKey:        "info",
				operatorHealthImpactLabelKey: "none",
			},
		},
		{
			Alert: "OutdatedVirtualMachineInstanceWorkloads",
			Expr:  intstr.FromString("kubevirt_vmi_number_of_outdated != 0"),
			For:   &oneDay,
			Annotations: map[string]string{
				"summary":     "Some running VMIs are still active in outdated pods after KubeVirt control plane update has completed.",
				"runbook_url": fmt.Sprintf(runbookURLTemplate, "OutdatedVirtualMachineInstanceWorkloads"),
			},
			Labels: map[string]string{
				severityAlertLabelKey:        "warning",
				operatorHealthImpactLabelKey: "none",
			},
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
