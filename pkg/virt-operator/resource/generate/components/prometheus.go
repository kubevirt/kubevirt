package components

import (
	"fmt"

	"github.com/coreos/prometheus-operator/pkg/apis/monitoring"
	v1 "github.com/coreos/prometheus-operator/pkg/apis/monitoring/v1"
	v12 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
)

const (
	KUBEVIRT_PROMETHEUS_RULE_NAME = "prometheus-kubevirt-rules"
	prometheusLabelKey            = "prometheus.kubevirt.io"
	prometheusLabelValue          = "true"
	runbookUrlBasePath            = "https://kubevirt.io/monitoring/runbooks/"
	severityAlertLabelKey         = "severity"
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
	getRestCallsFailedWarning := func(failingCallsPercentage int, component, duration string) string {
		const restCallsFailWarningTemplate = "More than %d%% of the rest calls failed in %s for the last %s"
		return fmt.Sprintf(restCallsFailWarningTemplate, failingCallsPercentage, component, duration)
	}
	getErrorRatio := func(ns string, podName string, errorCodeRegex string, durationInMinutes int) string {
		errorRatioQuery := "sum ( rate ( rest_client_requests_total{namespace=\"%s\",pod=~\"%s-.*\",code=~\"%s\"} [%dm] ) )  /  sum ( rate ( rest_client_requests_total{namespace=\"%s\",pod=~\"%s-.*\"} [%dm] ) )"
		return fmt.Sprintf(errorRatioQuery, ns, podName, errorCodeRegex, durationInMinutes, ns, podName, durationInMinutes)
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
							severityAlertLabelKey: "critical",
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
							severityAlertLabelKey: "warning",
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
							"summary":     "At least two nodes with kvm resource required for VM live migration.",
							"runbook_url": runbookUrlBasePath + "LowKVMNodesCount",
						},
						Labels: map[string]string{
							severityAlertLabelKey: "warning",
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
							fmt.Sprintf("sum(kubevirt_virt_controller_ready{namespace='%s'}) or vector(0)", ns),
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
							severityAlertLabelKey: "warning",
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
							severityAlertLabelKey: "critical",
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
							severityAlertLabelKey: "critical",
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
							severityAlertLabelKey: "warning",
						},
					},
					{
						Alert: "VirtControllerRESTErrorsHigh",
						Expr:  intstr.FromString(getErrorRatio(ns, "virt-controller", "(4|5)[0-9][0-9]", 60) + " >= 0.05"),
						Annotations: map[string]string{
							"summary":     getRestCallsFailedWarning(5, "virt-controller", "hour"),
							"runbook_url": runbookUrlBasePath + "VirtControllerRESTErrorsHigh",
						},
						Labels: map[string]string{
							severityAlertLabelKey: "warning",
						},
					},
					{
						Alert: "VirtControllerRESTErrorsBurst",
						Expr:  intstr.FromString(getErrorRatio(ns, "virt-controller", "(4|5)[0-9][0-9]", 5) + " >= 0.8"),
						For:   "5m",
						Annotations: map[string]string{
							"summary":     getRestCallsFailedWarning(80, "virt-controller", durationFiveMinutes),
							"runbook_url": runbookUrlBasePath + "VirtControllerRESTErrorsBurst",
						},
						Labels: map[string]string{
							severityAlertLabelKey: "critical",
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
							severityAlertLabelKey: "critical",
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
							severityAlertLabelKey: "warning",
						},
					},
					{
						Alert: "VirtOperatorRESTErrorsHigh",
						Expr:  intstr.FromString(getErrorRatio(ns, "virt-operator", "(4|5)[0-9][0-9]", 60) + " >= 0.05"),
						Annotations: map[string]string{
							"summary":     getRestCallsFailedWarning(5, "virt-operator", "hour"),
							"runbook_url": runbookUrlBasePath + "VirtOperatorRESTErrorsHigh",
						},
						Labels: map[string]string{
							severityAlertLabelKey: "warning",
						},
					},
					{
						Alert: "VirtOperatorRESTErrorsBurst",
						Expr:  intstr.FromString(getErrorRatio(ns, "virt-operator", "(4|5)[0-9][0-9]", 5) + " >= 0.8"),
						For:   "5m",
						Annotations: map[string]string{
							"summary":     getRestCallsFailedWarning(80, "virt-operator", durationFiveMinutes),
							"runbook_url": runbookUrlBasePath + "VirtOperatorRESTErrorsBurst",
						},
						Labels: map[string]string{
							severityAlertLabelKey: "critical",
						},
					},
					{
						Record: "kubevirt_virt_operator_ready_total",
						Expr: intstr.FromString(
							fmt.Sprintf("sum(kubevirt_virt_operator_ready{namespace='%s'}) or vector(0)", ns),
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
							severityAlertLabelKey: "warning",
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
							severityAlertLabelKey: "critical",
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
							severityAlertLabelKey: "critical",
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
							severityAlertLabelKey: "warning",
						},
					},
					{
						Alert: "VirtHandlerRESTErrorsHigh",
						Expr:  intstr.FromString(getErrorRatio(ns, "virt-handler", "(4|5)[0-9][0-9]", 60) + " >= 0.05"),
						Annotations: map[string]string{
							"summary":     getRestCallsFailedWarning(5, "virt-handler", "hour"),
							"runbook_url": runbookUrlBasePath + "VirtHandlerRESTErrorsHigh",
						},
						Labels: map[string]string{
							severityAlertLabelKey: "warning",
						},
					},
					{
						Alert: "VirtHandlerRESTErrorsBurst",
						Expr:  intstr.FromString(getErrorRatio(ns, "virt-handler", "(4|5)[0-9][0-9]", 5) + " >= 0.8"),
						For:   "5m",
						Annotations: map[string]string{
							"summary":     getRestCallsFailedWarning(80, "virt-handler", durationFiveMinutes),
							"runbook_url": runbookUrlBasePath + "VirtHandlerRESTErrorsBurst",
						},
						Labels: map[string]string{
							severityAlertLabelKey: "critical",
						},
					},
					{
						Alert: "VirtApiRESTErrorsHigh",
						Expr:  intstr.FromString(getErrorRatio(ns, "virt-api", "(4|5)[0-9][0-9]", 60) + " >= 0.05"),
						Annotations: map[string]string{
							"summary":     getRestCallsFailedWarning(5, "virt-api", "hour"),
							"runbook_url": runbookUrlBasePath + "VirtApiRESTErrorsHigh",
						},
						Labels: map[string]string{
							severityAlertLabelKey: "warning",
						},
					},
					{
						Alert: "VirtApiRESTErrorsBurst",
						Expr:  intstr.FromString(getErrorRatio(ns, "virt-api", "(4|5)[0-9][0-9]", 5) + " >= 0.8"),
						For:   "5m",
						Annotations: map[string]string{
							"summary":     getRestCallsFailedWarning(80, "virt-api", durationFiveMinutes),
							"runbook_url": runbookUrlBasePath + "VirtApiRESTErrorsBurst",
						},
						Labels: map[string]string{
							severityAlertLabelKey: "critical",
						},
					},
					{
						Record: "kubevirt_vmi_memory_used_bytes",
						Expr:   intstr.FromString("kubevirt_vmi_memory_available_bytes-kubevirt_vmi_memory_usable_bytes"),
					},
					{
						Record: "kubevirt_vm_container_free_memory_bytes_based_on_working_set_bytes",
						Expr:   intstr.FromString("sum by(pod, container, namespace) (kube_pod_container_resource_requests{pod=~'virt-launcher-.*', container='compute', resource='memory'}- on(pod,container, namespace) container_memory_working_set_bytes{pod=~'virt-launcher-.*', container='compute'})"),
					},
					{
						Record: "kubevirt_vm_container_free_memory_bytes_based_on_rss",
						Expr:   intstr.FromString("sum by(pod, container, namespace) (kube_pod_container_resource_requests{pod=~'virt-launcher-.*', container='compute', resource='memory'}- on(pod,container, namespace) container_memory_rss{pod=~'virt-launcher-.*', container='compute'})"),
					},
					{
						Alert: "KubevirtVmHighMemoryUsage",
						Expr:  intstr.FromString("kubevirt_vm_container_free_memory_bytes_based_on_working_set_bytes < 20971520 or kubevirt_vm_container_free_memory_bytes_based_on_rss < 20971520"),
						For:   "1m",
						Annotations: map[string]string{
							"description": "Container {{ $labels.container }} in pod {{ $labels.pod }} in namespace {{ $labels.namespace }} free memory is less than 20 MB and it is close to requested memory",
							"summary":     "VM is at risk of being evicted and in serious cases of memory exhaustion being terminated by the runtime.",
							"runbook_url": runbookUrlBasePath + "KubevirtVmHighMemoryUsage",
						},
						Labels: map[string]string{
							severityAlertLabelKey: "warning",
						},
					},
					{
						Alert: "OrphanedVirtualMachineInstances",
						Expr:  intstr.FromString("((count by (node) (kube_pod_status_ready{condition='true',pod=~'virt-handler.*'} * on(pod) group_left(node) kube_pod_info{pod=~'virt-handler.*'})) or (count by (node)(kube_pod_info{pod=~'virt-launcher.*'})*0)) == 0"),
						For:   "10m",
						Annotations: map[string]string{
							"summary":     "No ready virt-handler pod detected on node {{ $labels.node }} with running vmis for more than 10 minutes",
							"runbook_url": runbookUrlBasePath + "OrphanedVirtualMachineInstances",
						},
						Labels: map[string]string{
							severityAlertLabelKey: "warning",
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
							severityAlertLabelKey: "warning",
						},
					},
					{
						Alert: "KubeVirtComponentExceedsRequestedMemory",
						Expr:  intstr.FromString(fmt.Sprintf(`((kube_pod_container_resource_requests{namespace="%s",container=~"virt-controller|virt-api|virt-handler|virt-operator",resource="memory"}) - on(pod) group_left(node) container_memory_working_set_bytes{container="",namespace="%s"}) < 0`, ns, ns)),
						For:   "5m",
						Annotations: map[string]string{
							"description": "Container {{ $labels.container }} in pod {{ $labels.pod }} memory usage exceeds the memory requested",
							"summary":     "The container is using more memory than what is defined in the containers resource requests",
							"runbook_url": runbookUrlBasePath + "KubeVirtComponentExceedsRequestedMemory",
						},
						Labels: map[string]string{
							severityAlertLabelKey: "warning",
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
							severityAlertLabelKey: "warning",
						},
					},
					{
						Record: "kubevirt_vmsnapshot_persistentvolumeclaim_labels",
						Expr:   intstr.FromString("label_replace(label_replace(kube_persistentvolumeclaim_labels{label_restore_kubevirt_io_source_vm_name!='', label_restore_kubevirt_io_source_vm_namespace!=''} == 1, 'vm_namespace', '$1', 'label_restore_kubevirt_io_source_vm_namespace', '(.*)'), 'vm_name', '$1', 'label_restore_kubevirt_io_source_vm_name', '(.*)')"),
					},
					{
						Record: "kubevirt_vmsnapshot_disks_restored_from_source_total",
						Expr:   intstr.FromString("sum by(vm_name, vm_namespace) (kubevirt_vmsnapshot_persistentvolumeclaim_labels)"),
					},
					{
						Record: "kubevirt_vmsnapshot_disks_restored_from_source_bytes",
						Expr:   intstr.FromString("sum by(vm_name, vm_namespace) (kube_persistentvolumeclaim_resource_requests_storage_bytes * on(persistentvolumeclaim, namespace) group_left(vm_name, vm_namespace) kubevirt_vmsnapshot_persistentvolumeclaim_labels)"),
					},
					{
						Alert: "KubeVirtVMIExcessiveMigrations",
						Expr:  intstr.FromString("floor(increase(sum by (vmi) (kubevirt_migrate_vmi_succeeded_total)[1d:1m])) >= 12"),
						For:   "1m",
						Annotations: map[string]string{
							"description": "VirtualMachineInstance {{ $labels.vmi }} has been migrated more than 12 times during the last 24 hours",
							"summary":     "An excessive amount of migrations have been detected on a VirtualMachineInstance in the last 24 hours.",
							"runbook_url": runbookUrlBasePath + "KubeVirtVMIExcessiveMigrations",
						},
						Labels: map[string]string{
							severityAlertLabelKey: "warning",
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
			Labels: map[string]string{
				severityAlertLabelKey: "warning",
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
