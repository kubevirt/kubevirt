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
 * Copyright The KubeVirt Authors.
 *
 */

package components

import (
	k8sv1 "k8s.io/api/core/v1"
	networkv1 "k8s.io/api/networking/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"

	"kubevirt.io/kubevirt/pkg/pointer"
)

const (
	allowIngressToMetrics             = "kubevirt-allow-ingress-to-metrics"
	allowIngressToVirtApiWebhook      = "kubevirt-allow-ingress-to-virt-api-webhook-server"
	allowVirtApiToComponents          = "kubevirt-allow-virt-api-to-components"
	allowVirtApiToLaunchers           = "kubevirt-allow-virt-api-to-launchers"
	allowVirtApiToHandlers            = "kubevirt-allow-virt-api-to-virt-handler"
	allowIngressToHandler             = "kubevirt-allow-ingress-to-virt-handler"
	allowIngressToVirtOperatorWebhook = "kubevirt-allow-ingress-to-virt-operator-webhook-server"
	allowExportProxyCommunications    = "kubevirt-allow-virt-exportproxy-communications"
	allowHandlerToHandler             = "kubevirt-allow-handler-to-handler"
	allowHandlerToPrometheus          = "kubevirt-allow-handler-to-prometheus"
)

// NewKubeVirtNetworkPolicies returns the network policies required by kv to operate
func NewKubeVirtNetworkPolicies(namespace string) []*networkv1.NetworkPolicy {
	return []*networkv1.NetworkPolicy{
		newIngressToMetricsNP(namespace),
		newVirtApiWebhookNP(namespace),
		newVirtApiToComponentsNP(namespace),
		newVirtApiToLaunchersNP(namespace),
		newVirtApiToHandlersNP(namespace),
		newHandlersToVirtApiNP(namespace),
		newVirtOperatorWebhookNP(namespace),
		newExportProxyNP(namespace),
		newHandlerToHandlerNP(namespace),
		newHandlerToPrometheusNP(namespace),
	}
}

func newNetworkPolicy(namespace, name string, spec *networkv1.NetworkPolicySpec) *networkv1.NetworkPolicy {
	return &networkv1.NetworkPolicy{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "networking.k8s.io/v1",
			Kind:       "NetworkPolicy",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
		Spec: *spec,
	}
}

func newIngressToMetricsNP(namespace string) *networkv1.NetworkPolicy {
	return newNetworkPolicy(
		namespace,
		allowIngressToMetrics,
		&networkv1.NetworkPolicySpec{
			PodSelector: metav1.LabelSelector{
				MatchExpressions: []metav1.LabelSelectorRequirement{
					{
						Key:      kubevirtLabelKey,
						Operator: metav1.LabelSelectorOpIn,
						Values: []string{
							VirtOperatorName,
							VirtHandlerName,
							VirtControllerName,
							VirtAPIName,
							VirtExportProxyName,
							VirtSynchronizationControllerName,
						},
					},
				},
			},
			PolicyTypes: []networkv1.PolicyType{networkv1.PolicyTypeIngress},
			Ingress: []networkv1.NetworkPolicyIngressRule{
				{
					Ports: []networkv1.NetworkPolicyPort{
						{
							Port:     pointer.P(intstr.FromInt32(8443)),
							Protocol: pointer.P(k8sv1.ProtocolTCP),
						},
					},
				},
			},
		},
	)
}

func newVirtApiWebhookNP(namespace string) *networkv1.NetworkPolicy {
	return newNetworkPolicy(
		namespace,
		allowIngressToVirtApiWebhook,
		&networkv1.NetworkPolicySpec{
			PodSelector: metav1.LabelSelector{
				MatchLabels: map[string]string{kubevirtLabelKey: VirtAPIName},
			},
			PolicyTypes: []networkv1.PolicyType{networkv1.PolicyTypeIngress},
			Ingress: []networkv1.NetworkPolicyIngressRule{
				{
					Ports: []networkv1.NetworkPolicyPort{
						{
							Port:     pointer.P(intstr.FromInt32(8443)),
							Protocol: pointer.P(k8sv1.ProtocolTCP),
						},
					},
				},
			},
		},
	)
}

func newVirtApiToComponentsNP(namespace string) *networkv1.NetworkPolicy {
	return newNetworkPolicy(
		namespace,
		allowVirtApiToComponents,
		&networkv1.NetworkPolicySpec{
			PodSelector: metav1.LabelSelector{
				MatchLabels: map[string]string{kubevirtLabelKey: VirtAPIName},
			},
			PolicyTypes: []networkv1.PolicyType{networkv1.PolicyTypeEgress},
			Egress: []networkv1.NetworkPolicyEgressRule{
				{
					To: []networkv1.NetworkPolicyPeer{
						{
							PodSelector: &metav1.LabelSelector{
								MatchExpressions: []metav1.LabelSelectorRequirement{
									{
										Key:      kubevirtLabelKey,
										Operator: metav1.LabelSelectorOpIn,
										Values: []string{
											VirtOperatorName,
											VirtHandlerName,
											VirtControllerName,
											VirtAPIName,
										},
									},
								},
							},
						},
					},
					Ports: []networkv1.NetworkPolicyPort{
						{
							Port:     pointer.P(intstr.FromInt32(8443)),
							Protocol: pointer.P(k8sv1.ProtocolTCP),
						},
					},
				},
			},
		},
	)
}

func newVirtApiToLaunchersNP(namespace string) *networkv1.NetworkPolicy {
	return newNetworkPolicy(
		namespace,
		allowVirtApiToLaunchers,
		&networkv1.NetworkPolicySpec{
			PodSelector: metav1.LabelSelector{
				MatchLabels: map[string]string{kubevirtLabelKey: VirtAPIName},
			},
			PolicyTypes: []networkv1.PolicyType{networkv1.PolicyTypeEgress},
			Egress: []networkv1.NetworkPolicyEgressRule{
				{
					To: []networkv1.NetworkPolicyPeer{
						{
							PodSelector: &metav1.LabelSelector{
								MatchLabels: map[string]string{kubevirtLabelKey: "virt-launcher"},
							},
							NamespaceSelector: &metav1.LabelSelector{},
						},
					},
				},
			},
		},
	)
}

func newVirtApiToHandlersNP(namespace string) *networkv1.NetworkPolicy {
	return newNetworkPolicy(
		namespace,
		allowVirtApiToHandlers,
		&networkv1.NetworkPolicySpec{
			PodSelector: metav1.LabelSelector{
				MatchLabels: map[string]string{kubevirtLabelKey: VirtAPIName},
			},
			PolicyTypes: []networkv1.PolicyType{networkv1.PolicyTypeEgress},
			Egress: []networkv1.NetworkPolicyEgressRule{
				{
					To: []networkv1.NetworkPolicyPeer{
						{
							PodSelector: &metav1.LabelSelector{
								MatchLabels: map[string]string{kubevirtLabelKey: VirtHandlerName},
							},
						},
					},
					Ports: []networkv1.NetworkPolicyPort{
						{
							Protocol: pointer.P(k8sv1.ProtocolTCP),
						},
					},
				},
			},
		},
	)
}

func newHandlersToVirtApiNP(namespace string) *networkv1.NetworkPolicy {
	return newNetworkPolicy(
		namespace,
		allowIngressToHandler,
		&networkv1.NetworkPolicySpec{
			PodSelector: metav1.LabelSelector{
				MatchLabels: map[string]string{kubevirtLabelKey: VirtHandlerName},
			},
			PolicyTypes: []networkv1.PolicyType{networkv1.PolicyTypeIngress},
			Ingress: []networkv1.NetworkPolicyIngressRule{
				{
					From: []networkv1.NetworkPolicyPeer{
						{
							PodSelector: &metav1.LabelSelector{
								MatchLabels: map[string]string{kubevirtLabelKey: VirtAPIName},
							},
						},
					},
					Ports: []networkv1.NetworkPolicyPort{
						{
							Protocol: pointer.P(k8sv1.ProtocolTCP),
						},
					},
				},
			},
		},
	)
}

func newVirtOperatorWebhookNP(namespace string) *networkv1.NetworkPolicy {
	return newNetworkPolicy(
		namespace,
		allowIngressToVirtOperatorWebhook,
		&networkv1.NetworkPolicySpec{
			PodSelector: metav1.LabelSelector{
				MatchLabels: map[string]string{kubevirtLabelKey: VirtOperatorName},
			},
			PolicyTypes: []networkv1.PolicyType{networkv1.PolicyTypeIngress},
			Ingress: []networkv1.NetworkPolicyIngressRule{
				{
					Ports: []networkv1.NetworkPolicyPort{
						{
							Port:     pointer.P(intstr.FromInt32(8444)),
							Protocol: pointer.P(k8sv1.ProtocolTCP),
						},
					},
				},
			},
		},
	)
}

func newExportProxyNP(namespace string) *networkv1.NetworkPolicy {
	return newNetworkPolicy(
		namespace,
		allowExportProxyCommunications,
		&networkv1.NetworkPolicySpec{
			PodSelector: metav1.LabelSelector{
				MatchLabels: map[string]string{kubevirtLabelKey: VirtExportProxyName},
			},
			PolicyTypes: []networkv1.PolicyType{networkv1.PolicyTypeIngress, networkv1.PolicyTypeEgress},
			Ingress: []networkv1.NetworkPolicyIngressRule{
				{
					Ports: []networkv1.NetworkPolicyPort{
						{
							Port:     pointer.P(intstr.FromInt32(8443)),
							Protocol: pointer.P(k8sv1.ProtocolTCP),
						},
					},
				},
			},
			Egress: []networkv1.NetworkPolicyEgressRule{
				{
					To: []networkv1.NetworkPolicyPeer{
						{
							PodSelector: &metav1.LabelSelector{
								MatchExpressions: []metav1.LabelSelectorRequirement{
									{
										Key:      "kubevirt.io.virt-export-service",
										Operator: metav1.LabelSelectorOpExists,
									},
								},
							},
							NamespaceSelector: &metav1.LabelSelector{},
						},
					},
					Ports: []networkv1.NetworkPolicyPort{
						{
							Port:     pointer.P(intstr.FromInt32(8443)),
							Protocol: pointer.P(k8sv1.ProtocolTCP),
						},
					},
				},
			},
		},
	)
}

func newHandlerToHandlerNP(namespace string) *networkv1.NetworkPolicy {
	return newNetworkPolicy(
		namespace,
		allowHandlerToHandler,
		&networkv1.NetworkPolicySpec{
			PodSelector: metav1.LabelSelector{
				MatchLabels: map[string]string{kubevirtLabelKey: VirtHandlerName},
			},
			PolicyTypes: []networkv1.PolicyType{networkv1.PolicyTypeIngress, networkv1.PolicyTypeEgress},
			Ingress: []networkv1.NetworkPolicyIngressRule{
				{
					From: []networkv1.NetworkPolicyPeer{
						{
							PodSelector: &metav1.LabelSelector{
								MatchLabels: map[string]string{kubevirtLabelKey: VirtHandlerName},
							},
						},
					},
				},
			},
			Egress: []networkv1.NetworkPolicyEgressRule{
				{
					To: []networkv1.NetworkPolicyPeer{
						{
							PodSelector: &metav1.LabelSelector{
								MatchLabels: map[string]string{kubevirtLabelKey: VirtHandlerName},
							},
						},
					},
				},
			},
		},
	)
}

func newHandlerToPrometheusNP(namespace string) *networkv1.NetworkPolicy {
	return newNetworkPolicy(
		namespace,
		allowHandlerToPrometheus,
		&networkv1.NetworkPolicySpec{
			PodSelector: metav1.LabelSelector{
				MatchLabels: map[string]string{kubevirtLabelKey: VirtHandlerName},
			},
			PolicyTypes: []networkv1.PolicyType{networkv1.PolicyTypeEgress},
			Egress: []networkv1.NetworkPolicyEgressRule{
				{
					Ports: []networkv1.NetworkPolicyPort{
						{
							Port:     pointer.P(intstr.FromInt32(8443)),
							Protocol: pointer.P(k8sv1.ProtocolTCP),
						},
					},
				},
			},
		},
	)
}
