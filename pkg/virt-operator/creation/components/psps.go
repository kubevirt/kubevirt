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
 * Copyright 2019 Red Hat, Inc.
 *
 */
package components

import (
	policyv1beta1 "k8s.io/api/policy/v1beta1"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	virtv1 "kubevirt.io/client-go/api/v1"
	"kubevirt.io/kubevirt/pkg/virt-operator/creation/rbac"
)

func NewControllerPodSecurityPolicy() *policyv1beta1.PodSecurityPolicy {
	return &policyv1beta1.PodSecurityPolicy{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "policy/v1beta1",
			Kind:       "PodSecurityPolicy",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: rbac.ControllerServiceAccountName,
			Labels: map[string]string{
				virtv1.AppLabel: "",
			},
			Annotations: map[string]string{
				"apparmor.security.beta.kubernetes.io/defaultProfileName":  "runtime/default",
				"seccomp.security.alpha.kubernetes.io/allowedProfileNames": "*",
				"seccomp.security.alpha.kubernetes.io/defaultProfileName":  "runtime/default",
			},
		},
		Spec: policyv1beta1.PodSecurityPolicySpec{
			AllowedCapabilities: []corev1.Capability{
				"NET_ADMIAN",
				"SYS_NICE",
			},
			FSGroup: policyv1beta1.FSGroupStrategyOptions{
				Rule: policyv1beta1.FSGroupStrategyRunAsAny,
			},
			RunAsUser: policyv1beta1.RunAsUserStrategyOptions{
				Rule: policyv1beta1.RunAsUserStrategyRunAsAny,
			},
			SELinux: policyv1beta1.SELinuxStrategyOptions{
				Rule: policyv1beta1.SELinuxStrategyRunAsAny,
			},
			SupplementalGroups: policyv1beta1.SupplementalGroupsStrategyOptions{
				Rule: policyv1beta1.SupplementalGroupsStrategyRunAsAny,
			},
			Volumes: []policyv1beta1.FSType{
				policyv1beta1.ConfigMap,
				policyv1beta1.Secret,
				policyv1beta1.EmptyDir,
				policyv1beta1.DownwardAPI,
				policyv1beta1.PersistentVolumeClaim,
				policyv1beta1.HostPath,
			},
		},
	}
}

func NewHandlerPodSecurityPolicy() *policyv1beta1.PodSecurityPolicy {
	return &policyv1beta1.PodSecurityPolicy{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "policy/v1beta1",
			Kind:       "PodSecurityPolicy",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: rbac.HandlerServiceAccountName,
			Labels: map[string]string{
				virtv1.AppLabel: "",
			},
			Annotations: map[string]string{
				"apparmor.security.beta.kubernetes.io/defaultProfileName":  "runtime/default",
				"seccomp.security.alpha.kubernetes.io/allowedProfileNames": "*",
				"seccomp.security.alpha.kubernetes.io/defaultProfileName":  "runtime/default",
			},
		},
		Spec: policyv1beta1.PodSecurityPolicySpec{
			HostPID:    true,
			Privileged: true,
			FSGroup: policyv1beta1.FSGroupStrategyOptions{
				Rule: policyv1beta1.FSGroupStrategyMustRunAs,
				Ranges: []policyv1beta1.IDRange{
					policyv1beta1.IDRange{
						Min: 1000,
						Max: 65535,
					},
				},
			},
			RunAsUser: policyv1beta1.RunAsUserStrategyOptions{
				Rule: policyv1beta1.RunAsUserStrategyMustRunAsNonRoot,
			},
			SELinux: policyv1beta1.SELinuxStrategyOptions{
				Rule: policyv1beta1.SELinuxStrategyRunAsAny,
			},
			SupplementalGroups: policyv1beta1.SupplementalGroupsStrategyOptions{
				Rule: policyv1beta1.SupplementalGroupsStrategyMustRunAs,
				Ranges: []policyv1beta1.IDRange{
					policyv1beta1.IDRange{
						Min: 1000,
						Max: 65535,
					},
				},
			},
			Volumes: []policyv1beta1.FSType{
				policyv1beta1.ConfigMap,
				policyv1beta1.Secret,
				policyv1beta1.EmptyDir,
				policyv1beta1.DownwardAPI,
				policyv1beta1.PersistentVolumeClaim,
				policyv1beta1.HostPath,
			},
		},
	}
}
