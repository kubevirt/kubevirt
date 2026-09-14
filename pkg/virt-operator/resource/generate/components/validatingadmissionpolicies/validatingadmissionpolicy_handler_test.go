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

package validatingadmissionpolicies_test

import (
	"fmt"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apiserver/pkg/admission/plugin/policy/validating"
	"k8s.io/apiserver/pkg/authentication/user"

	vap "kubevirt.io/kubevirt/pkg/virt-operator/resource/generate/components/validatingadmissionpolicies"
)

const (
	handlerSA     = "system:serviceaccount:kubevirt:kubevirt-handler"
	nodeNameKey   = "authentication.kubernetes.io/node-name"
	ownNodeName   = "own-node"
	otherNodeName = "other-node"
)

var _ = Describe("Validation Admission Policy", func() {
	Context("ValidatingAdmissionPolicy", func() {
		It("should generate the expected policy", func() {
			const userName = "system:serviceaccount:kubevirt-ns:kubevirt-handler"
			validatingAdmissionPolicy := vap.NewHandlerV1ValidatingAdmissionPolicy(userName)

			expectedMatchConditionExpression := fmt.Sprintf("request.userInfo.username == %q", userName)
			Expect(validatingAdmissionPolicy.Spec.MatchConditions[0].Expression).To(Equal(expectedMatchConditionExpression))
			Expect(validatingAdmissionPolicy.Kind).ToNot(BeEmpty())
		})
	})

	Context("ValidatingAdmissionPolicyBinding", func() {
		It("should generate the expected policy binding", func() {
			const userName = "system:serviceaccount:kubevirt-ns:kubevirt-handler"
			validatingAdmissionPolicy := vap.NewHandlerV1ValidatingAdmissionPolicy(userName)
			validatingAdmissionPolicyBinding := vap.NewHandlerV1ValidatingAdmissionPolicyBinding()

			Expect(validatingAdmissionPolicyBinding.Spec.PolicyName).To(Equal(validatingAdmissionPolicy.Name))
			Expect(validatingAdmissionPolicyBinding.Kind).ToNot(BeEmpty())
		})
	})

	Context("handler node restriction CEL evaluations", func() {
		var (
			validator validating.Validator
			old       *corev1.Node
		)

		BeforeEach(func() {
			policy := vap.NewHandlerV1ValidatingAdmissionPolicy(handlerSA)
			validator = compilePolicy(policy)
			old = testNode(ownNodeName)
		})

		DescribeTable("evaluates node updates",
			func(mutate func(*corev1.Node), info user.Info, want outcome, wantMessage string) {
				object := old.DeepCopy()
				if mutate != nil {
					mutate(object)
				}
				got, messages := evaluate(validator, object, old, info)
				Expect(got).To(Equal(want))
				if wantMessage != "" {
					Expect(containsMessage(messages, wantMessage)).To(BeTrue(), "messages %v should contain %q", messages, wantMessage)
				}
			},
			Entry("deny spec change", func(n *corev1.Node) {
				n.Spec.Unschedulable = true
			}, handlerUser(ownNodeName), deny, vap.NodeRestrictionErrModifySpec),
			Entry("deny metadata finalizers add", func(n *corev1.Node) {
				n.Finalizers = []string{"kubernetes.io/evil-finalizer"}
			}, handlerUser(ownNodeName), deny, vap.NodeRestrictionErrChangeMetadataFields),
			Entry("deny add non-kubevirt label", func(n *corev1.Node) {
				n.Labels["other.io/newNotAllowedLabel"] = "value"
			}, handlerUser(ownNodeName), deny, vap.NodeRestrictionErrAddDeleteLabels),
			Entry("deny update non-kubevirt label", func(n *corev1.Node) {
				n.Labels["other.io/notAllowedLabel"] = "other-value"
			}, handlerUser(ownNodeName), deny, vap.NodeRestrictionErrUpdateLabels),
			Entry("deny remove non-kubevirt label", func(n *corev1.Node) {
				delete(n.Labels, "other.io/notAllowedLabel")
			}, handlerUser(ownNodeName), deny, vap.NodeRestrictionErrAddDeleteLabels),
			Entry("deny add non-kubevirt annotation", func(n *corev1.Node) {
				n.Annotations["other.io/newNotAllowedAnnotation"] = "value"
			}, handlerUser(ownNodeName), deny, vap.NodeRestrictionErrAddDeleteAnnotations),
			Entry("deny update non-kubevirt annotation", func(n *corev1.Node) {
				n.Annotations["other.io/notAllowedAnnotation"] = "other-value"
			}, handlerUser(ownNodeName), deny, vap.NodeRestrictionErrUpdateAnnotations),
			Entry("deny remove non-kubevirt annotation", func(n *corev1.Node) {
				delete(n.Annotations, "other.io/notAllowedAnnotation")
			}, handlerUser(ownNodeName), deny, vap.NodeRestrictionErrAddDeleteAnnotations),
			Entry("admit add kubevirt.io label", func(n *corev1.Node) {
				n.Labels["kubevirt.io/newAllowedLabel"] = "value"
			}, handlerUser(ownNodeName), admit, ""),
			Entry("admit update kubevirt.io label", func(n *corev1.Node) {
				n.Labels["kubevirt.io/allowedLabel"] = "other-value"
			}, handlerUser(ownNodeName), admit, ""),
			Entry("admit remove kubevirt.io label", func(n *corev1.Node) {
				delete(n.Labels, "kubevirt.io/allowedLabel")
			}, handlerUser(ownNodeName), admit, ""),
			Entry("admit add kubevirt.io annotation", func(n *corev1.Node) {
				n.Annotations["kubevirt.io/newAllowedAnnotation"] = "value"
			}, handlerUser(ownNodeName), admit, ""),
			Entry("admit update kubevirt.io annotation", func(n *corev1.Node) {
				n.Annotations["kubevirt.io/allowedAnnotation"] = "other-value"
			}, handlerUser(ownNodeName), admit, ""),
			Entry("admit remove kubevirt.io annotation", func(n *corev1.Node) {
				delete(n.Annotations, "kubevirt.io/allowedAnnotation")
			}, handlerUser(ownNodeName), admit, ""),
			Entry("admit update cpumanager label", func(n *corev1.Node) {
				n.Labels["cpumanager"] = "false"
			}, handlerUser(ownNodeName), admit, ""),
			Entry("admit resourceVersion bump", func(n *corev1.Node) {
				n.ResourceVersion = "2"
			}, handlerUser(ownNodeName), admit, ""),
			Entry("admit managedFields change", func(n *corev1.Node) {
				n.ManagedFields = append(n.ManagedFields, metav1.ManagedFieldsEntry{
					Manager:    "virt-handler",
					Operation:  metav1.ManagedFieldsOperationUpdate,
					APIVersion: "v1",
					FieldsType: "FieldsV1",
				})
			}, handlerUser(ownNodeName), admit, ""),
			Entry("deny other node extra patching own node", func(n *corev1.Node) {
				n.Labels["kubevirt.io/newAllowedLabel"] = "value"
			}, handlerUser(otherNodeName), deny, vap.NodeRestrictionErrModifyAnother),
			Entry("deny empty extra map", func(n *corev1.Node) {
				n.Labels["kubevirt.io/newAllowedLabel"] = "value"
			}, &user.DefaultInfo{Name: handlerSA, Extra: map[string][]string{}}, deny, ""),
			Entry("deny extra missing node-name key", func(n *corev1.Node) {
				n.Labels["kubevirt.io/newAllowedLabel"] = "value"
			}, &user.DefaultInfo{
				Name:  handlerSA,
				Extra: map[string][]string{"other-key": {"value"}},
			}, deny, vap.NodeRestrictionErrModifyAnother),
			Entry("skip different service account", func(n *corev1.Node) {
				n.Spec.Unschedulable = true
			}, &user.DefaultInfo{
				Name:  "system:serviceaccount:kubevirt:kubelet",
				Extra: map[string][]string{nodeNameKey: {ownNodeName}},
			}, skip, ""),
		)
	})
})

func handlerUser(nodeName string) user.Info {
	extra := map[string][]string{}
	if nodeName != "" {
		extra[nodeNameKey] = []string{nodeName}
	}
	return &user.DefaultInfo{Name: handlerSA, Extra: extra}
}

func testNode(name string) *corev1.Node {
	return &corev1.Node{
		TypeMeta: metav1.TypeMeta{APIVersion: "v1", Kind: "Node"},
		ObjectMeta: metav1.ObjectMeta{
			Name:            name,
			UID:             "node-uid",
			ResourceVersion: "1",
			Labels: map[string]string{
				"kubevirt.io/allowedLabel": "value",
				"other.io/notAllowedLabel": "value",
				"cpumanager":               "true",
			},
			Annotations: map[string]string{
				"kubevirt.io/allowedAnnotation": "value",
				"other.io/notAllowedAnnotation": "value",
			},
			ManagedFields: []metav1.ManagedFieldsEntry{
				{
					Manager:    "kubelet",
					Operation:  metav1.ManagedFieldsOperationUpdate,
					APIVersion: "v1",
					FieldsType: "FieldsV1",
				},
			},
		},
		Spec: corev1.NodeSpec{Unschedulable: false},
	}
}
