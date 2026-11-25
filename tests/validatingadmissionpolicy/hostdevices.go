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

package validatingadmissionpolicy

import (
	"context"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	admissionregistrationv1 "k8s.io/api/admissionregistration/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	vap "kubevirt.io/kubevirt/pkg/virt-operator/resource/generate/components/validatingadmissionpolicies"
	"kubevirt.io/kubevirt/pkg/virt-operator/util"
	"kubevirt.io/kubevirt/tests/decorators"
	"kubevirt.io/kubevirt/tests/framework/kubevirt"
)

// Pod admission for a permitted host device is covered by the control-plane
// integration suite. A cluster only needs to show that virt-operator installed
// the policy and binding; that check does not need a device plugin.
var _ = Describe("[sig-compute] permitted host device admission policy", decorators.SigCompute, func() {
	It("should be installed with its binding by virt-operator", func() {
		virtClient := kubevirt.Client()
		enabled, err := util.IsValidatingAdmissionPolicyEnabled(virtClient)
		Expect(err).NotTo(HaveOccurred())
		Expect(enabled).To(BeTrue(), "ValidatingAdmissionPolicy should be enabled")

		policy, err := virtClient.AdmissionregistrationV1().ValidatingAdmissionPolicies().Get(
			context.Background(), vap.HostDevicesRestrictionPolicyName, metav1.GetOptions{})
		Expect(err).NotTo(HaveOccurred(), "host device validating admission policy should be installed")
		Expect(policy.Spec.Validations).NotTo(BeEmpty())
		Expect(policy.Spec.Validations[0].Message).To(Equal(vap.HostDevicesRestrictionErr))

		binding, err := virtClient.AdmissionregistrationV1().ValidatingAdmissionPolicyBindings().Get(
			context.Background(), vap.HostDevicesRestrictionBindingName, metav1.GetOptions{})
		Expect(err).NotTo(HaveOccurred(), "host device validating admission policy binding should be installed")
		Expect(binding.Spec.PolicyName).To(Equal(vap.HostDevicesRestrictionPolicyName))
		Expect(binding.Spec.ValidationActions).To(ConsistOf(admissionregistrationv1.Deny))
	})
})
