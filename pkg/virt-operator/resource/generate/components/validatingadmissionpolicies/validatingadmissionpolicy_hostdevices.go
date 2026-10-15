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

package validatingadmissionpolicies

import (
	"fmt"
	"strings"

	admissionregistrationv1 "k8s.io/api/admissionregistration/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	v1 "kubevirt.io/api/core/v1"

	"kubevirt.io/kubevirt/pkg/pointer"
)

const (
	// HostDevicesRestrictionPolicyName is the ValidatingAdmissionPolicy that
	// stops arbitrary pods from claiming host devices virt-handler registered.
	HostDevicesRestrictionPolicyName = "kubevirt-host-devices-restriction-policy"
	// HostDevicesRestrictionBindingName binds that policy at the KubeVirt CR
	// which declares permittedHostDevices.
	HostDevicesRestrictionBindingName = "kubevirt-host-devices-restriction-binding"
	hostDevicesRestrictionAppLabel    = "kubevirt-host-devices-restriction"

	// HostDevicesRestrictionErr is returned when a pod requests a KubeVirt-managed
	// host device and the caller is not virt-controller.
	HostDevicesRestrictionErr = "only the virt-controller service account may request host devices declared in permittedHostDevices"

	managedHostDeviceResourcesVar = "managedHostDeviceResources"
)

// NewHostDevicesValidatingAdmissionPolicy denies pods that request a host device
// virt-handler itself exposes, unless virt-controller is creating the pod.
//
// Device plugins have no authorization scope, so a resource name listed in
// spec.configuration.permittedHostDevices can otherwise be allocated to any pod
// in the cluster. Extended-resource requests cannot be added after the pod is
// created, so the check runs on CREATE only and does not interfere with later
// annotation or label updates. Devices with externalResourceProvider are left
// alone: KubeVirt does not own that device plugin, and other workloads may
// legitimately request it. Ephemeral containers are not covered; kubelet
// allocates device plugins when the pod sandbox is created.
//
// virtControllerServiceAccount is the full username, for example
// system:serviceaccount:kubevirt:kubevirt-controller.
func NewHostDevicesValidatingAdmissionPolicy(virtControllerServiceAccount string) *admissionregistrationv1.ValidatingAdmissionPolicy {
	return &admissionregistrationv1.ValidatingAdmissionPolicy{
		TypeMeta: metav1.TypeMeta{
			APIVersion: admissionregistrationv1.SchemeGroupVersion.String(),
			Kind:       "ValidatingAdmissionPolicy",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: HostDevicesRestrictionPolicyName,
		},
		Spec: admissionregistrationv1.ValidatingAdmissionPolicySpec{
			FailurePolicy: pointer.P(admissionregistrationv1.Fail),
			ParamKind: &admissionregistrationv1.ParamKind{
				APIVersion: v1.SchemeGroupVersion.String(),
				Kind:       "KubeVirt",
			},
			MatchConstraints: &admissionregistrationv1.MatchResources{
				ResourceRules: hostDevicePodRules(),
			},
			// Most pods never request an extended resource. Skipping them keeps a
			// CEL failure in the host-device check from rejecting ordinary workloads.
			MatchConditions: []admissionregistrationv1.MatchCondition{
				{
					Name:       "pod-requests-extended-resource",
					Expression: podRequestsExtendedResourceExpr(),
				},
			},
			Variables: []admissionregistrationv1.Variable{
				{
					Name:       managedHostDeviceResourcesVar,
					Expression: managedHostDeviceNamesExpr(),
				},
				{
					Name:       "requestsManagedHostDevice",
					Expression: podRequestsResourceExpr("variables." + managedHostDeviceResourcesVar),
				},
				{
					Name:       "isVirtController",
					Expression: fmt.Sprintf("request.userInfo.username == %q", virtControllerServiceAccount),
				},
			},
			Validations: []admissionregistrationv1.Validation{
				{
					Expression: `!variables.requestsManagedHostDevice || variables.isVirtController`,
					Message:    HostDevicesRestrictionErr,
					Reason:     pointer.P(metav1.StatusReasonForbidden),
				},
			},
		},
	}
}

// NewHostDevicesValidatingAdmissionPolicyBinding applies the host-device policy
// to every KubeVirt CR in namespace. parameterNotFoundAction is Allow so a
// missing CR does not fail every pod admission in the cluster.
func NewHostDevicesValidatingAdmissionPolicyBinding(namespace string) *admissionregistrationv1.ValidatingAdmissionPolicyBinding {
	return &admissionregistrationv1.ValidatingAdmissionPolicyBinding{
		TypeMeta: metav1.TypeMeta{
			APIVersion: admissionregistrationv1.SchemeGroupVersion.String(),
			Kind:       "ValidatingAdmissionPolicyBinding",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: HostDevicesRestrictionBindingName,
			Labels: map[string]string{
				v1.AppLabel:       hostDevicesRestrictionAppLabel,
				v1.ManagedByLabel: v1.ManagedByLabelOperatorValue,
			},
		},
		Spec: admissionregistrationv1.ValidatingAdmissionPolicyBindingSpec{
			PolicyName: HostDevicesRestrictionPolicyName,
			ValidationActions: []admissionregistrationv1.ValidationAction{
				admissionregistrationv1.Deny,
			},
			ParamRef: &admissionregistrationv1.ParamRef{
				Namespace: namespace,
				// Match every KubeVirt object in the install namespace. The CR name
				// is chosen by the user and is not known when the strategy is generated.
				Selector:                &metav1.LabelSelector{},
				ParameterNotFoundAction: pointer.P(admissionregistrationv1.AllowAction),
			},
			MatchResources: &admissionregistrationv1.MatchResources{
				ResourceRules: hostDevicePodRules(),
			},
		},
	}
}

func hostDevicePodRules() []admissionregistrationv1.NamedRuleWithOperations {
	return []admissionregistrationv1.NamedRuleWithOperations{
		{
			RuleWithOperations: admissionregistrationv1.RuleWithOperations{
				Operations: []admissionregistrationv1.OperationType{
					admissionregistrationv1.Create,
				},
				Rule: admissionregistrationv1.Rule{
					APIGroups:   []string{""},
					APIVersions: []string{"v1"},
					Resources:   []string{"pods"},
				},
			},
		},
	}
}

// managedHostDeviceNamesExpr is the resource names virt-handler registers.
// externalResourceProvider entries are excluded because another device plugin owns them.
func managedHostDeviceNamesExpr() string {
	fields := []string{"pciHostDevices", "mediatedDevices", "usb"}
	parts := make([]string, 0, len(fields))
	for _, field := range fields {
		parts = append(parts, fmt.Sprintf(`(has(params.spec) && has(params.spec.configuration) && has(params.spec.configuration.permittedHostDevices) && has(params.spec.configuration.permittedHostDevices.%[1]s) ? params.spec.configuration.permittedHostDevices.%[1]s.filter(d, has(d.resourceName) && d.resourceName != "" && (!has(d.externalResourceProvider) || !d.externalResourceProvider)).map(d, d.resourceName) : [])`, field))
	}
	return "(" + strings.Join(parts, " + ") + ")"
}

func podRequestsExtendedResourceExpr() string {
	return podRequestsKeyExpr(`k.contains("/")`)
}

func podRequestsResourceExpr(resourceNamesExpr string) string {
	return podRequestsKeyExpr(fmt.Sprintf("k in (%s)", resourceNamesExpr))
}

func podRequestsKeyExpr(keyTest string) string {
	fields := []string{"containers", "initContainers", "ephemeralContainers"}
	parts := make([]string, 0, len(fields))
	for _, field := range fields {
		parts = append(parts, fmt.Sprintf(
			`(has(object.spec.%[1]s) && object.spec.%[1]s.exists(c, %[2]s))`,
			field,
			containerUsesKey("c", keyTest),
		))
	}
	return `(has(object.spec) && (` + strings.Join(parts, " || ") + `))`
}

func containerUsesKey(container, keyTest string) string {
	return fmt.Sprintf(
		`(has(%[1]s.resources) && ((has(%[1]s.resources.requests) && %[1]s.resources.requests.exists(k, %[2]s)) || (has(%[1]s.resources.limits) && %[1]s.resources.limits.exists(k, %[2]s))))`,
		container,
		keyTest,
	)
}
