/*
 * This file is part of the KubeVirt project
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 *
 * Copyright 2018 Red Hat, Inc.
 *
 */
package components

import (
	"fmt"
	"strings"

	"github.com/coreos/prometheus-operator/pkg/apis/monitoring"
	promv1 "github.com/coreos/prometheus-operator/pkg/apis/monitoring/v1"
	corev1 "k8s.io/api/core/v1"
	schedulingv1 "k8s.io/api/scheduling/v1"
	extv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	k8syaml "k8s.io/apimachinery/pkg/util/yaml"

	virtv1 "kubevirt.io/client-go/api/v1"
	snapshotv1 "kubevirt.io/client-go/apis/snapshot/v1alpha1"
)

const (
	KUBEVIRT_PROMETHEUS_RULE_NAME = "prometheus-kubevirt-rules"

	creationTimestampJSONPath = ".metadata.creationTimestamp"
	errorMessageJSONPath      = ".status.error.message"
	prometheusLabelKey        = "prometheus.kubevirt.io"
)

var (
	VIRTUALMACHINE                   = "virtualmachines." + virtv1.VirtualMachineInstanceGroupVersionKind.Group
	VIRTUALMACHINEINSTANCE           = "virtualmachineinstances." + virtv1.VirtualMachineInstanceGroupVersionKind.Group
	VIRTUALMACHINEINSTANCEPRESET     = "virtualmachineinstancepresets." + virtv1.VirtualMachineInstancePresetGroupVersionKind.Group
	VIRTUALMACHINEINSTANCEREPLICASET = "virtualmachineinstancereplicasets." + virtv1.VirtualMachineInstanceReplicaSetGroupVersionKind.Group
	VIRTUALMACHINEINSTANCEMIGRATION  = "virtualmachineinstancemigrations." + virtv1.VirtualMachineInstanceMigrationGroupVersionKind.Group
	KUBEVIRT                         = "kubevirts." + virtv1.KubeVirtGroupVersionKind.Group
	VIRTUALMACHINESNAPSHOT           = "virtualmachinesnapshots." + snapshotv1.SchemeGroupVersion.Group
	VIRTUALMACHINESNAPSHOTCONTENT    = "virtualmachinesnapshotcontents." + snapshotv1.SchemeGroupVersion.Group
	PreserveUnknownFieldsFalse       = false
)

func getVersion(crd *extv1.CustomResourceDefinition, version string) (*extv1.CustomResourceDefinitionVersion, error) {
	for i := range crd.Spec.Versions {
		if version == crd.Spec.Versions[i].Name {
			return &crd.Spec.Versions[i], nil
		}
	}
	return nil, fmt.Errorf("version %s not found in CustomResourceDefinition: %v", version, crd.Name)
}

func addFieldsToVersion(version *extv1.CustomResourceDefinitionVersion, fields ...interface{}) error {
	for _, field := range fields {
		switch v := field.(type) {
		case []extv1.CustomResourceColumnDefinition:
			version.AdditionalPrinterColumns = v
		case *extv1.CustomResourceSubresources:
			version.Subresources = v
		case *extv1.CustomResourceValidation:
			version.Schema = v
		default:
			return fmt.Errorf("cannot add field of type %T to a CustomResourceDefinitionVersion", v)
		}
	}
	return nil
}

func addFieldsToAllVersions(crd *extv1.CustomResourceDefinition, fields ...interface{}) error {
	for i := range crd.Spec.Versions {
		if err := addFieldsToVersion(&crd.Spec.Versions[i], fields...); err != nil {
			return err
		}
	}
	return nil
}

func patchValidation(crd *extv1.CustomResourceDefinition, version *extv1.CustomResourceDefinitionVersion) error {
	name := crd.Spec.Names.Singular

	crd.Spec.PreserveUnknownFields = PreserveUnknownFieldsFalse
	validation, ok := CRDsValidation[name]
	if !ok {
		return nil
	}
	crvalidation := extv1.CustomResourceValidation{}
	err := k8syaml.NewYAMLToJSONDecoder(strings.NewReader(validation)).Decode(&crvalidation)
	if err != nil {
		return fmt.Errorf("Could not decode validation for %s, %v", name, err)
	}
	if err = addFieldsToVersion(version, &crvalidation); err != nil {
		return err
	}
	return nil
}

func patchValidationForAllVersions(crd *extv1.CustomResourceDefinition) error {
	for i := range crd.Spec.Versions {
		if err := patchValidation(crd, &crd.Spec.Versions[i]); err != nil {
			return err
		}
	}
	return nil
}

func newBlankCrd() *extv1.CustomResourceDefinition {
	return &extv1.CustomResourceDefinition{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "apiextensions.k8s.io/v1",
			Kind:       "CustomResourceDefinition",
		},
		ObjectMeta: metav1.ObjectMeta{
			Labels: map[string]string{
				virtv1.AppLabel: "",
			},
		},
	}
}

func newCRDVersions() []extv1.CustomResourceDefinitionVersion {
	versions := make([]extv1.CustomResourceDefinitionVersion, len(virtv1.ApiSupportedVersions))
	copy(versions, virtv1.ApiSupportedVersions)
	return versions
}

func NewVirtualMachineInstanceCrd() (*extv1.CustomResourceDefinition, error) {
	crd := newBlankCrd()

	crd.ObjectMeta.Name = VIRTUALMACHINEINSTANCE
	crd.Spec = extv1.CustomResourceDefinitionSpec{
		Group:    virtv1.VirtualMachineInstanceGroupVersionKind.Group,
		Versions: newCRDVersions(),
		Scope:    "Namespaced",

		Names: extv1.CustomResourceDefinitionNames{
			Plural:     "virtualmachineinstances",
			Singular:   "virtualmachineinstance",
			Kind:       virtv1.VirtualMachineInstanceGroupVersionKind.Kind,
			ShortNames: []string{"vmi", "vmis"},
			Categories: []string{
				"all",
			},
		},
	}
	err := addFieldsToAllVersions(crd, []extv1.CustomResourceColumnDefinition{
		{Name: "Age", Type: "date", JSONPath: creationTimestampJSONPath},
		{Name: "Phase", Type: "string", JSONPath: ".status.phase"},
		{Name: "IP", Type: "string", JSONPath: ".status.interfaces[0].ipAddress"},
		{Name: "NodeName", Type: "string", JSONPath: ".status.nodeName"},
		{Name: "Live-Migratable", Type: "string", JSONPath: ".status.conditions[?(@.type=='LiveMigratable')].status", Priority: 1},
		{Name: "Paused", Type: "string", JSONPath: ".status.conditions[?(@.type=='Paused')].status", Priority: 1},
	})
	if err != nil {
		return nil, err
	}

	if err := patchValidationForAllVersions(crd); err != nil {
		return nil, err
	}
	return crd, nil
}

func NewVirtualMachineCrd() (*extv1.CustomResourceDefinition, error) {
	crd := newBlankCrd()

	crd.ObjectMeta.Name = VIRTUALMACHINE
	crd.Spec = extv1.CustomResourceDefinitionSpec{
		Group:    virtv1.VirtualMachineGroupVersionKind.Group,
		Versions: newCRDVersions(),
		Scope:    "Namespaced",

		Names: extv1.CustomResourceDefinitionNames{
			Plural:     "virtualmachines",
			Singular:   "virtualmachine",
			Kind:       virtv1.VirtualMachineGroupVersionKind.Kind,
			ShortNames: []string{"vm", "vms"},
			Categories: []string{
				"all",
			},
		},
	}
	err := addFieldsToAllVersions(crd, []extv1.CustomResourceColumnDefinition{
		{Name: "Age", Type: "date", JSONPath: creationTimestampJSONPath},
		{Name: "Status", Description: "Human Readable Status", Type: "string", JSONPath: ".status.printableStatus"},
		{Name: "Volume", Description: "Primary Volume", Type: "string", JSONPath: ".spec.volumes[0].name"},
		{Name: "Created", Type: "boolean", JSONPath: ".status.created", Priority: 1},
	}, &extv1.CustomResourceSubresources{
		Status: &extv1.CustomResourceSubresourceStatus{}})
	if err != nil {
		return nil, err
	}

	if err = patchValidationForAllVersions(crd); err != nil {
		return nil, err
	}
	return crd, nil
}

func NewPresetCrd() (*extv1.CustomResourceDefinition, error) {
	crd := newBlankCrd()

	crd.ObjectMeta.Name = VIRTUALMACHINEINSTANCEPRESET
	crd.Spec = extv1.CustomResourceDefinitionSpec{
		Group:    virtv1.VirtualMachineInstancePresetGroupVersionKind.Group,
		Versions: newCRDVersions(),
		Scope:    "Namespaced",

		Names: extv1.CustomResourceDefinitionNames{
			Plural:     "virtualmachineinstancepresets",
			Singular:   "virtualmachineinstancepreset",
			Kind:       virtv1.VirtualMachineInstancePresetGroupVersionKind.Kind,
			ShortNames: []string{"vmipreset", "vmipresets"},
			Categories: []string{
				"all",
			},
		},
	}

	if err := patchValidationForAllVersions(crd); err != nil {
		return nil, err
	}
	return crd, nil
}

func NewReplicaSetCrd() (*extv1.CustomResourceDefinition, error) {
	crd := newBlankCrd()
	labelSelector := ".status.labelSelector"

	crd.ObjectMeta.Name = VIRTUALMACHINEINSTANCEREPLICASET
	crd.Spec = extv1.CustomResourceDefinitionSpec{
		Group:    virtv1.VirtualMachineInstanceReplicaSetGroupVersionKind.Group,
		Versions: newCRDVersions(),
		Scope:    "Namespaced",

		Names: extv1.CustomResourceDefinitionNames{
			Plural:     "virtualmachineinstancereplicasets",
			Singular:   "virtualmachineinstancereplicaset",
			Kind:       virtv1.VirtualMachineInstanceReplicaSetGroupVersionKind.Kind,
			ShortNames: []string{"vmirs", "vmirss"},
			Categories: []string{
				"all",
			},
		},
	}
	err := addFieldsToAllVersions(crd,
		[]extv1.CustomResourceColumnDefinition{
			{Name: "Desired", Type: "integer", JSONPath: ".spec.replicas",
				Description: "Number of desired VirtualMachineInstances"},
			{Name: "Current", Type: "integer", JSONPath: ".status.replicas",
				Description: "Number of managed and not final or deleted VirtualMachineInstances"},
			{Name: "Ready", Type: "integer", JSONPath: ".status.readyReplicas",
				Description: "Number of managed VirtualMachineInstances which are ready to receive traffic"},
			{Name: "Age", Type: "date", JSONPath: creationTimestampJSONPath},
		}, &extv1.CustomResourceSubresources{
			Scale: &extv1.CustomResourceSubresourceScale{
				SpecReplicasPath:   ".spec.replicas",
				StatusReplicasPath: ".status.replicas",
				LabelSelectorPath:  &labelSelector,
			},
			Status: &extv1.CustomResourceSubresourceStatus{},
		})
	if err != nil {
		return nil, err
	}
	if err := patchValidationForAllVersions(crd); err != nil {
		return nil, err
	}
	return crd, nil
}

func NewVirtualMachineInstanceMigrationCrd() (*extv1.CustomResourceDefinition, error) {
	crd := newBlankCrd()

	crd.ObjectMeta.Name = VIRTUALMACHINEINSTANCEMIGRATION
	crd.Spec = extv1.CustomResourceDefinitionSpec{
		Group:    virtv1.VirtualMachineInstanceMigrationGroupVersionKind.Group,
		Versions: newCRDVersions(),
		Scope:    "Namespaced",

		Names: extv1.CustomResourceDefinitionNames{
			Plural:     "virtualmachineinstancemigrations",
			Singular:   "virtualmachineinstancemigration",
			Kind:       virtv1.VirtualMachineInstanceMigrationGroupVersionKind.Kind,
			ShortNames: []string{"vmim", "vmims"},
			Categories: []string{
				"all",
			},
		},
	}
	err := addFieldsToAllVersions(crd, &extv1.CustomResourceSubresources{
		Status: &extv1.CustomResourceSubresourceStatus{},
	})
	if err != nil {
		return nil, err
	}

	if err = patchValidationForAllVersions(crd); err != nil {
		return nil, err
	}
	return crd, nil
}

// Used by manifest generation
// If you change something here, you probably need to change the CSV manifest too,
// see /manifests/release/kubevirt.VERSION.csv.yaml.in
func NewKubeVirtCrd() (*extv1.CustomResourceDefinition, error) {

	// we use a different label here, so no newBlankCrd()
	crd := &extv1.CustomResourceDefinition{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "apiextensions.k8s.io/v1",
			Kind:       "CustomResourceDefinition",
		},
		ObjectMeta: metav1.ObjectMeta{
			Labels: map[string]string{
				"operator.kubevirt.io": "",
			},
		},
	}

	crd.ObjectMeta.Name = KUBEVIRT
	crd.Spec = extv1.CustomResourceDefinitionSpec{
		Group:    virtv1.KubeVirtGroupVersionKind.Group,
		Versions: newCRDVersions(),
		Scope:    "Namespaced",

		Names: extv1.CustomResourceDefinitionNames{
			Plural:     "kubevirts",
			Singular:   "kubevirt",
			Kind:       virtv1.KubeVirtGroupVersionKind.Kind,
			ShortNames: []string{"kv", "kvs"},
			Categories: []string{
				"all",
			},
		},
	}
	err := addFieldsToAllVersions(crd, []extv1.CustomResourceColumnDefinition{
		{Name: "Age", Type: "date", JSONPath: creationTimestampJSONPath},
		{Name: "Phase", Type: "string", JSONPath: ".status.phase"},
	}, &extv1.CustomResourceSubresources{
		Status: &extv1.CustomResourceSubresourceStatus{},
	})
	if err != nil {
		return nil, err
	}

	if err = patchValidationForAllVersions(crd); err != nil {
		return nil, err
	}
	return crd, nil
}

func NewVirtualMachineSnapshotCrd() (*extv1.CustomResourceDefinition, error) {
	crd := newBlankCrd()

	crd.ObjectMeta.Name = VIRTUALMACHINESNAPSHOT
	crd.Spec = extv1.CustomResourceDefinitionSpec{
		Group: snapshotv1.SchemeGroupVersion.Group,
		Versions: []extv1.CustomResourceDefinitionVersion{
			{
				Name:    snapshotv1.SchemeGroupVersion.Version,
				Served:  true,
				Storage: true,
			},
		},
		Scope: "Namespaced",
		Names: extv1.CustomResourceDefinitionNames{
			Plural:     "virtualmachinesnapshots",
			Singular:   "virtualmachinesnapshot",
			Kind:       "VirtualMachineSnapshot",
			ShortNames: []string{"vmsnapshot", "vmsnapshots"},
			Categories: []string{
				"all",
			},
		},
	}
	err := addFieldsToAllVersions(crd, []extv1.CustomResourceColumnDefinition{
		{Name: "SourceKind", Type: "string", JSONPath: ".spec.source.kind"},
		{Name: "SourceName", Type: "string", JSONPath: ".spec.source.name"},
		{Name: "ReadyToUse", Type: "boolean", JSONPath: ".status.readyToUse"},
		{Name: "CreationTime", Type: "date", JSONPath: ".status.creationTime"},
		{Name: "Error", Type: "string", JSONPath: errorMessageJSONPath},
	})
	if err != nil {
		return nil, err
	}

	if err = patchValidationForAllVersions(crd); err != nil {
		return nil, err
	}
	return crd, nil
}

func NewVirtualMachineSnapshotContentCrd() (*extv1.CustomResourceDefinition, error) {
	crd := newBlankCrd()

	crd.ObjectMeta.Name = VIRTUALMACHINESNAPSHOTCONTENT
	crd.Spec = extv1.CustomResourceDefinitionSpec{
		Group: snapshotv1.SchemeGroupVersion.Group,
		Versions: []extv1.CustomResourceDefinitionVersion{
			{
				Name:    snapshotv1.SchemeGroupVersion.Version,
				Served:  true,
				Storage: true,
			},
		},
		Scope: "Namespaced",
		Names: extv1.CustomResourceDefinitionNames{
			Plural:     "virtualmachinesnapshotcontents",
			Singular:   "virtualmachinesnapshotcontent",
			Kind:       "VirtualMachineSnapshotContent",
			ShortNames: []string{"vmsnapshotcontent", "vmsnapshotcontents"},
			Categories: []string{
				"all",
			},
		},
	}
	err := addFieldsToAllVersions(crd, []extv1.CustomResourceColumnDefinition{
		{Name: "ReadyToUse", Type: "boolean", JSONPath: ".status.readyToUse"},
		{Name: "CreationTime", Type: "date", JSONPath: ".status.creationTime"},
		{Name: "Error", Type: "string", JSONPath: errorMessageJSONPath},
	})
	if err != nil {
		return nil, err
	}

	if err = patchValidationForAllVersions(crd); err != nil {
		return nil, err
	}
	return crd, nil
}

func NewVirtualMachineRestoreCrd() (*extv1.CustomResourceDefinition, error) {
	crd := newBlankCrd()

	crd.ObjectMeta.Name = "virtualmachinerestores." + snapshotv1.SchemeGroupVersion.Group
	crd.Spec = extv1.CustomResourceDefinitionSpec{
		Group: snapshotv1.SchemeGroupVersion.Group,
		Versions: []extv1.CustomResourceDefinitionVersion{
			{
				Name:    snapshotv1.SchemeGroupVersion.Version,
				Served:  true,
				Storage: true,
			},
		},
		Scope: "Namespaced",
		Names: extv1.CustomResourceDefinitionNames{
			Plural:     "virtualmachinerestores",
			Singular:   "virtualmachinerestore",
			Kind:       "VirtualMachineRestore",
			ShortNames: []string{"vmrestore", "vmrestores"},
			Categories: []string{
				"all",
			},
		},
	}
	err := addFieldsToAllVersions(crd, []extv1.CustomResourceColumnDefinition{
		{Name: "TargetKind", Type: "string", JSONPath: ".spec.target.kind"},
		{Name: "TargetName", Type: "string", JSONPath: ".spec.target.name"},
		{Name: "Complete", Type: "boolean", JSONPath: ".status.complete"},
		{Name: "RestoreTime", Type: "date", JSONPath: ".status.restoreTime"},
		{Name: "Error", Type: "string", JSONPath: errorMessageJSONPath},
	})
	if err != nil {
		return nil, err
	}

	if err = patchValidationForAllVersions(crd); err != nil {
		return nil, err
	}
	return crd, nil
}

func NewServiceMonitorCR(namespace string, monitorNamespace string, insecureSkipVerify bool) *promv1.ServiceMonitor {
	return &promv1.ServiceMonitor{
		TypeMeta: metav1.TypeMeta{
			APIVersion: monitoring.GroupName,
			Kind:       "ServiceMonitor",
		},
		ObjectMeta: metav1.ObjectMeta{
			Namespace: monitorNamespace,
			Name:      KUBEVIRT_PROMETHEUS_RULE_NAME,
			Labels: map[string]string{
				"openshift.io/cluster-monitoring": "",
				prometheusLabelKey:                "",
				"k8s-app":                         "kubevirt",
			},
		},
		Spec: promv1.ServiceMonitorSpec{
			Selector: metav1.LabelSelector{
				MatchLabels: map[string]string{
					prometheusLabelKey: "",
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
						InsecureSkipVerify: insecureSkipVerify,
					},
					RelabelConfigs: []*promv1.RelabelConfig{
						{
							Regex:  "namespace",
							Action: "labeldrop",
						},
					},
				},
			},
		},
	}
}

// NewPrometheusRuleCR returns a PrometheusRule with a group of alerts for the KubeVirt deployment.
func NewPrometheusRuleCR(namespace string, workloadUpdatesEnabled bool) *promv1.PrometheusRule {
	return &promv1.PrometheusRule{
		TypeMeta: metav1.TypeMeta{
			APIVersion: promv1.SchemeGroupVersion.String(),
			Kind:       "PrometheusRule",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      KUBEVIRT_PROMETHEUS_RULE_NAME,
			Namespace: namespace,
			Labels: map[string]string{
				prometheusLabelKey: "",
				"k8s-app":          "kubevirt",
			},
		},
		Spec: *NewPrometheusRuleSpec(namespace, workloadUpdatesEnabled),
	}
}

// NewPrometheusRuleSpec makes a prometheus rule spec for kubevirt
func NewPrometheusRuleSpec(ns string, workloadUpdatesEnabled bool) *promv1.PrometheusRuleSpec {
	ruleSpec := &promv1.PrometheusRuleSpec{
		Groups: []promv1.RuleGroup{
			{
				Name: "kubevirt.rules",
				Rules: []promv1.Rule{
					{
						Record: "kubevirt_virt_api_up_total",
						Expr: intstr.FromString(
							fmt.Sprintf("sum(up{namespace='%s', pod=~'virt-api-.*'})", ns),
						),
					},
					{
						Alert: "VirtAPIDown",
						Expr:  intstr.FromString("kubevirt_virt_api_up_total == 0"),
						For:   "5m",
						Annotations: map[string]string{
							"summary": "All virt-api servers are down.",
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
							"summary": "More than one virt-api should be running if more than one worker nodes exist.",
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
						},
						Labels: map[string]string{
							"severity": "warning",
						},
					},
					{
						Record: "kubevirt_virt_controller_up_total",
						Expr: intstr.FromString(
							fmt.Sprintf("sum(up{pod=~'virt-controller-.*', namespace='%s'})", ns),
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
						For:   "5m",
						Annotations: map[string]string{
							"summary": "Some virt controllers are running but not ready.",
						},
					},
					{
						Alert: "NoReadyVirtController",
						Expr:  intstr.FromString("kubevirt_virt_controller_ready_total == 0"),
						For:   "5m",
						Annotations: map[string]string{
							"summary": "No ready virt-controller was detected for the last 5 min.",
						},
					},
					{
						Alert: "VirtControllerDown",
						Expr:  intstr.FromString("kubevirt_virt_controller_up_total == 0"),
						For:   "5m",
						Annotations: map[string]string{
							"summary": "No running virt-controller was detected for the last 5 min.",
						},
					},
					{
						Alert: "LowVirtControllersCount",
						Expr:  intstr.FromString("(num_of_allocatable_nodes > 1) and (kubevirt_virt_controller_ready_total < 2)"),
						For:   "5m",
						Annotations: map[string]string{
							"summary": "More than one virt-controller should be ready if more than one worker node.",
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
							"summary": "More than 5% of the rest calls failed in virt-controller for the last hour",
						},
					},
					{
						Alert: "VirtControllerRESTErrorsBurst",
						Expr:  intstr.FromString("(vec_by_virt_controllers_failed_client_rest_requests_in_last_5m / vec_by_virt_controllers_all_client_rest_requests_in_last_5m) >= 0.8"),
						For:   "5m",
						Annotations: map[string]string{
							"summary": "More than 80% of the rest calls failed in virt-controller for the last 5 minutes",
						},
					},
					{
						Record: "kubevirt_virt_operator_up_total",
						Expr: intstr.FromString(
							fmt.Sprintf("sum(up{namespace='%s', pod=~'virt-operator-.*'})", ns),
						),
					},
					{
						Alert: "VirtOperatorDown",
						Expr:  intstr.FromString("kubevirt_virt_operator_up_total == 0"),
						For:   "5m",
						Annotations: map[string]string{
							"summary": "All virt-operator servers are down.",
						},
					},
					{
						Alert: "LowVirtOperatorCount",
						Expr:  intstr.FromString("(num_of_allocatable_nodes > 1) and (kubevirt_virt_operator_up_total < 2)"),
						For:   "60m",
						Annotations: map[string]string{
							"summary": "More than one virt-operator should be running if more than one worker nodes exist.",
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
							"summary": "More than 5% of the rest calls failed in virt-operator for the last hour",
						},
					},
					{
						Alert: "VirtOperatorRESTErrorsBurst",
						Expr:  intstr.FromString("(vec_by_virt_operators_failed_client_rest_requests_in_last_5m / vec_by_virt_operators_all_client_rest_requests_in_last_5m) >= 0.8"),
						For:   "5m",
						Annotations: map[string]string{
							"summary": "More than 80% of the rest calls failed in virt-operator for the last 5 minutes",
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
						For:   "5m",
						Annotations: map[string]string{
							"summary": "Some virt-operators are running but not ready.",
						},
					},
					{
						Alert: "NoReadyVirtOperator",
						Expr:  intstr.FromString("kubevirt_virt_operator_up_total == 0"),
						For:   "5m",
						Annotations: map[string]string{
							"summary": "No ready virt-operator was detected for the last 5 min.",
						},
					},
					{
						Alert: "NoLeadingVirtOperator",
						Expr:  intstr.FromString("kubevirt_virt_operator_leading_total == 0"),
						For:   "5m",
						Annotations: map[string]string{
							"summary": "No leading virt-operator was detected for the last 5 min.",
						},
					},
					{
						Record: "kubevirt_virt_handler_up_total",
						Expr:   intstr.FromString(fmt.Sprintf("sum(up{pod=~'virt-handler-.*', namespace='%s'})", ns)),
					},
					{
						Alert: "VirtHandlerDaemonSetRolloutFailing",
						Expr: intstr.FromString(
							fmt.Sprintf("(%s - %s) != 0",
								fmt.Sprintf("kube_daemonset_status_number_ready{namespace='%s', daemonset='virt-handler'}", ns),
								fmt.Sprintf("kube_daemonset_status_desired_number_scheduled{namespace='%s', daemonset='virt-handler'}", ns))),
						For: "15m",
						Annotations: map[string]string{
							"summary": "Some virt-handlers failed to roll out",
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
							"summary": "More than 5% of the rest calls failed in virt-handler for the last hour",
						},
					},
					{
						Alert: "VirtHandlerRESTErrorsBurst",
						Expr:  intstr.FromString("(vec_by_virt_handlers_failed_client_rest_requests_in_last_5m / vec_by_virt_handlers_all_client_rest_requests_in_last_5m) >= 0.8"),
						For:   "5m",
						Annotations: map[string]string{
							"summary": "More than 80% of the rest calls failed in virt-handler for the last 5 minutes",
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
							"summary": "No virt-handler pod detected on node {{ $labels.node }} with running vmis for more than an hour",
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
		ruleSpec.Groups[0].Rules = append(ruleSpec.Groups[0].Rules, promv1.Rule{

			Alert: "OutdatedVirtualMachineInstanceWorkloads",
			Expr:  intstr.FromString("kubevirt_vmi_outdated_count != 0"),
			For:   "1440m",
			Annotations: map[string]string{
				"summary": "Some running VMIs are still active in outdated pods after KubeVirt control plane update has completed.",
			},
		})
	}

	return ruleSpec
}

// Used by manifest generation
func NewKubeVirtCR(namespace string, pullPolicy corev1.PullPolicy, featureGates string) *virtv1.KubeVirt {
	cr := &virtv1.KubeVirt{
		TypeMeta: metav1.TypeMeta{
			APIVersion: virtv1.GroupVersion.String(),
			Kind:       "KubeVirt",
		},
		ObjectMeta: metav1.ObjectMeta{
			Namespace: namespace,
			Name:      "kubevirt",
		},
		Spec: virtv1.KubeVirtSpec{
			ImagePullPolicy: pullPolicy,
		},
	}

	if featureGates != "" {
		cr.Spec.Configuration = virtv1.KubeVirtConfiguration{
			DeveloperConfiguration: &virtv1.DeveloperConfiguration{
				FeatureGates: strings.Split(featureGates, ","),
			},
		}
	}

	return cr
}

// NewKubeVirtPriorityClassCR is used for manifest generation
func NewKubeVirtPriorityClassCR() *schedulingv1.PriorityClass {
	return &schedulingv1.PriorityClass{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "scheduling.k8s.io/v1",
			Kind:       "PriorityClass",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: "kubevirt-cluster-critical",
		},
		// 1 billion is the highest value we can set
		// https://kubernetes.io/docs/concepts/configuration/pod-priority-preemption/#priorityclass
		Value:         1000000000,
		GlobalDefault: false,
		Description:   "This priority class should be used for KubeVirt core components only.",
	}
}
