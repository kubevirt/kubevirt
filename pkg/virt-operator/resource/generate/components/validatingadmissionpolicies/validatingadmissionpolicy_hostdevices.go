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

	admissionregistrationv1 "k8s.io/api/admissionregistration/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	v1 "kubevirt.io/api/core/v1"

	"kubevirt.io/kubevirt/pkg/pointer"
	"kubevirt.io/kubevirt/pkg/virt-config/featuregate"
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
	HostDevicesRestrictionErr = "only the virt-controller service account may request host devices registered by virt-handler"

	managedHostDeviceResourcesVar = "managedHostDeviceResources"

	// Resource names registered by virt-handler outside permittedHostDevices.
	// Keep them aligned with pkg/virt-handler/device-manager and the constants in
	// pkg/virt-controller/services/template.go.
	sevResource        = "devices.kubevirt.io/sev"
	vhostVsockResource = "devices.kubevirt.io/vhost-vsock"
	tdxResource        = "devices.kubevirt.io/tdx"
	iommufdResource    = "devices.kubevirt.io/iommufd"
	prHelperResource   = "devices.kubevirt.io/pr-helper"
)

// managedHostDeviceNamesExpr lists the device-plugin resources virt-handler
// registers from KubeVirt.spec.configuration.permittedHostDevices, plus the
// feature-gated plugins it registers beside that list.
//
// Keep the permittedHostDevices half in sync with every JSON field of
// v1.PermittedHostDevices. A new device category that is missing here is still
// registered by virt-handler and becomes allocatable by any pod. The element
// types differ, so each field is mapped on its own before the lists are
// concatenated. The unit test walks the struct and fails when a JSON name is
// absent from this expression. externalResourceProvider devices are excluded:
// another plugin owns them.
//
// The feature-gated half follows the same enablement virt-handler uses.
// WorkloadEncryptionSEV and VSOCK are beta, so they are on unless
// disabledFeatureGates lists them; an explicit featureGates entry wins.
// WorkloadEncryptionTDX and IOMMUFD are alpha, so they are on only when
// featureGates lists them. The pr-helper follows
// persistentReservationConfiguration.enabled when that pointer is set, and
// otherwise the legacy PersistentReservation feature gate.
// devices.kubevirt.io/kvm, tun, and vhost-net stay out of this list.
var managedHostDeviceNamesExpr = fmt.Sprintf(`
	params.spec.?configuration.?permittedHostDevices.?pciHostDevices.orValue([]).filter(d,
		d.resourceName != "" && !d.?externalResourceProvider.orValue(false)).map(d, d.resourceName)
	+ params.spec.?configuration.?permittedHostDevices.?mediatedDevices.orValue([]).filter(d,
		d.resourceName != "" && !d.?externalResourceProvider.orValue(false)).map(d, d.resourceName)
	+ params.spec.?configuration.?permittedHostDevices.?usb.orValue([]).filter(d,
		d.resourceName != "" && !d.?externalResourceProvider.orValue(false)).map(d, d.resourceName)
	+ ((params.spec.?configuration.?developerConfiguration.?featureGates.orValue([]).exists(g, g == %[1]q)
		|| !(params.spec.?configuration.?developerConfiguration.?disabledFeatureGates.orValue([]).exists(g, g == %[1]q)))
		? [%[2]q] : [])
	+ ((params.spec.?configuration.?developerConfiguration.?featureGates.orValue([]).exists(g, g == %[3]q)
		|| !(params.spec.?configuration.?developerConfiguration.?disabledFeatureGates.orValue([]).exists(g, g == %[3]q)))
		? [%[4]q] : [])
	+ (params.spec.?configuration.?developerConfiguration.?featureGates.orValue([]).exists(g, g == %[5]q)
		? [%[6]q] : [])
	+ (params.spec.?configuration.?developerConfiguration.?featureGates.orValue([]).exists(g, g == %[7]q)
		? [%[8]q] : [])
	+ (params.spec.?configuration.?persistentReservationConfiguration.?enabled.orValue(
		params.spec.?configuration.?developerConfiguration.?featureGates.orValue([]).exists(g, g == %[9]q))
		? [%[10]q] : [])`,
	featuregate.WorkloadEncryptionSEV, sevResource,
	featuregate.VSOCKGate, vhostVsockResource,
	featuregate.WorkloadEncryptionTDX, tdxResource,
	featuregate.IOMMUFDGate, iommufdResource,
	featuregate.PersistentReservation, prHelperResource,
)

// podRequestsExtendedResourceExpr matches a pod whose container or init-container
// limits name an extended resource. Pod.spec is required, so it is not guarded
// with has(). Init containers are optional. Ephemeral containers cannot be set
// on pod create and cannot carry resources, and this policy matches only CREATE.
//
// Only limits are checked. An extended resource must set a limit equal to its
// request, and the apiserver copies limits into requests before validating
// admission. A request without a limit is rejected by API validation.
const podRequestsExtendedResourceExpr = `
	object.spec.containers.exists(c, has(c.resources) && has(c.resources.limits) &&
		c.resources.limits.exists(k, k.contains("/")))
	|| object.spec.?initContainers.orValue([]).exists(c, has(c.resources) && has(c.resources.limits) &&
		c.resources.limits.exists(k, k.contains("/")))`

// podRequestsManagedHostDeviceExpr matches a pod whose container or init-container
// limits name a resource from managedHostDeviceNamesExpr.
const podRequestsManagedHostDeviceExpr = `
	object.spec.containers.exists(c, has(c.resources) && has(c.resources.limits) &&
		c.resources.limits.exists(k, k in variables.managedHostDeviceResources))
	|| object.spec.?initContainers.orValue([]).exists(c, has(c.resources) && has(c.resources.limits) &&
		c.resources.limits.exists(k, k in variables.managedHostDeviceResources))`

// NewHostDevicesValidatingAdmissionPolicy denies pods that request a host device
// virt-handler itself exposes, unless virt-controller is creating the pod.
//
// Device plugins have no authorization scope, so a resource name virt-handler
// registers can otherwise be allocated to any pod in the cluster. Extended-resource
// requests cannot be added after the pod is created, so the check runs on CREATE
// only and does not interfere with later
// annotation or label updates. Devices with externalResourceProvider are left
// alone: KubeVirt does not own that device plugin, and other workloads may
// legitimately request it. Container and init-container limits are checked.
// Requests are already defaulted from those limits, and ephemeral containers
// cannot be set on create or carry resources.
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
					Expression: podRequestsExtendedResourceExpr,
				},
			},
			Variables: []admissionregistrationv1.Variable{
				{
					Name:       managedHostDeviceResourcesVar,
					Expression: managedHostDeviceNamesExpr,
				},
				{
					Name:       "isVirtController",
					Expression: fmt.Sprintf("request.userInfo.username == %q", virtControllerServiceAccount),
				},
			},
			Validations: []admissionregistrationv1.Validation{
				{
					Expression: "!(" + podRequestsManagedHostDeviceExpr + ") || variables.isVirtController",
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
