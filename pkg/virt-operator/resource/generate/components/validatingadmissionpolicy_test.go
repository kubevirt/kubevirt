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

package components_test

import (
	"context"
	"fmt"
	"strings"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	celgo "github.com/google/cel-go/cel"
	celtypes "github.com/google/cel-go/common/types"

	admissionregistrationv1 "k8s.io/api/admissionregistration/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apiserver/pkg/admission"
	"k8s.io/apiserver/pkg/admission/plugin/cel"
	celconfig "k8s.io/apiserver/pkg/apis/cel"
	"k8s.io/apiserver/pkg/cel/environment"

	"kubevirt.io/kubevirt/pkg/virt-operator/resource/generate/components"
)

var _ = Describe("Validation Admission Policy", func() {
	Context("ValidatingAdmissionPolicyBinding", func() {
		It("should generate the expected policy binding", func() {
			const userName = "system:serviceaccount:kubevirt-ns:kubevirt-handler"
			validatingAdmissionPolicy := components.NewHandlerV1ValidatingAdmissionPolicy(userName)
			validatingAdmissionPolicyBinding := components.NewHandlerV1ValidatingAdmissionPolicyBinding()

			Expect(validatingAdmissionPolicyBinding.Spec.PolicyName).To(Equal(validatingAdmissionPolicy.Name))
			Expect(validatingAdmissionPolicyBinding.Kind).ToNot(BeEmpty())
		})
	})

	Context("ValidatingAdmissionPolicy", func() {
		It("should generate the expected policy", func() {
			const userName = "system:serviceaccount:kubevirt-ns:kubevirt-handler"
			validatingAdmissionPolicy := components.NewHandlerV1ValidatingAdmissionPolicy(userName)

			expectedMatchConditionExpression := fmt.Sprintf("request.userInfo.username == %q", userName)
			Expect(validatingAdmissionPolicy.Spec.MatchConditions[0].Expression).To(Equal(expectedMatchConditionExpression))
			Expect(validatingAdmissionPolicy.Kind).ToNot(BeEmpty())
		})
		Context("Validation Compile test", func() {
			var celCompiler *cel.CompositedCompiler
			BeforeEach(func() {
				compositionEnvTemplateWithoutStrictCost, err := cel.NewCompositionEnv(cel.VariablesTypeName, environment.MustBaseEnvSet(environment.DefaultCompatibilityVersion(), false))
				Expect(err).ToNot(HaveOccurred())
				celCompiler = cel.NewCompositedCompilerFromTemplate(compositionEnvTemplateWithoutStrictCost)
			})

			It("succeed compiling all the policy validations with variables", func() {
				const userName = "system:serviceaccount:kubevirt-ns:kubevirt-handler"
				validatingAdmissionPolicy := components.NewHandlerV1ValidatingAdmissionPolicy(userName)

				options := cel.OptionalVariableDeclarations{
					HasParams:     false,
					HasAuthorizer: false,
				}
				mode := environment.NewExpressions
				celCompiler.CompileAndStoreVariables(convertV1Variables(validatingAdmissionPolicy.Spec.Variables), options, mode)

				for _, validation := range validatingAdmissionPolicy.Spec.Validations {
					compilationResult := celCompiler.CompileCELExpression(convertV1Validation(validation), options, mode)
					Expect(compilationResult).ToNot(BeNil())
					Expect(compilationResult.Error).To(BeNil())
				}
			})
		})
		Context("Validation Filter test", func() {
			var celCompiler *cel.CompositedCompiler
			const nodeName = "node01"
			BeforeEach(func() {
				compositionEnvTemplateWithoutStrictCost, err := cel.NewCompositionEnv(cel.VariablesTypeName, environment.MustBaseEnvSet(environment.DefaultCompatibilityVersion(), false))
				Expect(err).ToNot(HaveOccurred())
				celCompiler = cel.NewCompositedCompilerFromTemplate(compositionEnvTemplateWithoutStrictCost)
			})
			DescribeTable("should succeed patching the node with allowed actions", func(oldNode, newNode *corev1.Node) {
				const userName = "system:serviceaccount:kubevirt-ns:kubevirt-handler"
				validatingAdmissionPolicy := components.NewHandlerV1ValidatingAdmissionPolicy(userName)

				// currently variables are not calculated when running the filter.
				// to work around it - replacing variables args in validations' expression.
				injectVariablesToValidations(validatingAdmissionPolicy.Spec.Validations, validatingAdmissionPolicy.Spec.Variables)

				filterResults := compileValidations(validatingAdmissionPolicy.Spec.Validations, celCompiler)
				Expect(filterResults.CompilationErrors()).To(HaveLen(0))

				versionedAttr, err := setNodeUpdateAttribute(oldNode, newNode)
				Expect(err).ToNot(HaveOccurred())

				evalResults, _, err := filterResults.ForInput(
					context.TODO(),
					versionedAttr,
					cel.CreateAdmissionRequest(versionedAttr.Attributes, metav1.GroupVersionResource(versionedAttr.GetResource()), metav1.GroupVersionKind(versionedAttr.VersionedKind)),
					cel.OptionalVariableBindings{},
					nil,
					celconfig.RuntimeCELCostBudget)
				Expect(err).ToNot(HaveOccurred())

				for resultIdx := range evalResults {
					result := evalResults[resultIdx]
					validation := validatingAdmissionPolicy.Spec.Validations[resultIdx]
					Expect(result.Error).To(BeNil(), fmt.Sprintf("validation policy expression %q failed", result.ExpressionAccessor.GetExpression()))
					Expect(result.EvalResult).To(Equal(celtypes.True), fmt.Sprintf("validation policy expression %q returned false. reason given: %q", result.ExpressionAccessor.GetExpression(), validation.Message))
				}
			},
				Entry("when adding a kubevirt-owned annotation",
					newNode(nodeName),
					newNode(nodeName, withAnnotations(map[string]string{"kubevirt.io/permittedAnnotation": ""}))),
				Entry("when adding a kubevirt-owned label",
					newNode(nodeName),
					newNode(nodeName, withLabels(map[string]string{"kubevirt.io/permittedLabel": "", "cpumanager": "true"}))),
			)

			DescribeTable("should fail patching the node with not allowed actions", func(oldNode, newNode *corev1.Node, expectedErrMessage string) {
				const userName = "system:serviceaccount:kubevirt-ns:kubevirt-handler"
				validatingAdmissionPolicy := components.NewHandlerV1ValidatingAdmissionPolicy(userName)

				// currently variables are not calculated when running the filter.
				// to work around it - replacing variables args in validations' expression.
				injectVariablesToValidations(validatingAdmissionPolicy.Spec.Validations, validatingAdmissionPolicy.Spec.Variables)

				filterResults := compileValidations(validatingAdmissionPolicy.Spec.Validations, celCompiler)
				Expect(filterResults.CompilationErrors()).To(HaveLen(0))

				versionedAttr, err := setNodeUpdateAttribute(oldNode, newNode)
				Expect(err).ToNot(HaveOccurred())

				evalResults, _, err := filterResults.ForInput(
					context.TODO(),
					versionedAttr,
					cel.CreateAdmissionRequest(versionedAttr.Attributes, metav1.GroupVersionResource(versionedAttr.GetResource()), metav1.GroupVersionKind(versionedAttr.VersionedKind)),
					cel.OptionalVariableBindings{},
					nil,
					celconfig.RuntimeCELCostBudget)
				Expect(err).ToNot(HaveOccurred())

				var resultIdxFailures []int
				for resultIdx := range evalResults {
					result := evalResults[resultIdx]
					Expect(result.Error).To(BeNil(), fmt.Sprintf("validation policy expression %q failed", result.ExpressionAccessor.GetExpression()))
					if result.EvalResult == celtypes.False {
						resultIdxFailures = append(resultIdxFailures, resultIdx)
					}
				}

				getErrMessage := func(resultIdx int) string { return validatingAdmissionPolicy.Spec.Validations[resultIdx].Message }
				Expect(resultIdxFailures).To(ContainElement(
					WithTransform(getErrMessage, Equal(expectedErrMessage))), fmt.Sprintf("validation did not fail with expected error message %q", expectedErrMessage))
			},
				Entry("when changing node spec",
					newNode(nodeName),
					newNode(nodeName, withUnschedulable(true)),
					components.NodeRestrictionErrModifySpec,
				),
				Entry("when changing not allowed metadata field",
					newNode(nodeName),
					newNode(nodeName, withOwnerReference("user", "1234")),
					components.NodeRestrictionErrChangeMetadataFields,
				),
				Entry("when adding a non kubevirt-owned label",
					newNode(nodeName),
					newNode(nodeName, withLabels(map[string]string{"other.io/notPermittedLabel": ""})),
					components.NodeRestrictionErrAddDeleteLabels,
				),
				Entry("when updating a non kubevirt-owned label",
					newNode(nodeName, withLabels(map[string]string{"other.io/notPermittedLabel": "old-value"})),
					newNode(nodeName, withLabels(map[string]string{"other.io/notPermittedLabel": "new-value"})),
					components.NodeRestrictionErrUpdateLabels,
				),
				Entry("when removing a non kubevirt-owned label",
					newNode(nodeName, withLabels(map[string]string{"other.io/notPermittedLabel": "old-value"})),
					newNode(nodeName),
					components.NodeRestrictionErrAddDeleteLabels,
				),
				Entry("when adding a non kubevirt-owned annotation",
					newNode(nodeName),
					newNode(nodeName, withAnnotations(map[string]string{"other.io/notPermittedAnnotation": ""})),
					components.NodeRestrictionErrAddDeleteAnnotations,
				),
				Entry("when updating a non kubevirt-owned annotation",
					newNode(nodeName, withAnnotations(map[string]string{"other.io/notPermittedAnnotation": "old-value"})),
					newNode(nodeName, withAnnotations(map[string]string{"other.io/notPermittedAnnotation": "new-value"})),
					components.NodeRestrictionErrUpdateAnnotations,
				),
				Entry("when removing a non kubevirt-owned annotation",
					newNode(nodeName, withAnnotations(map[string]string{"other.io/notPermittedAnnotation": "old-value"})),
					newNode(nodeName),
					components.NodeRestrictionErrAddDeleteAnnotations,
				),
			)
		})
	})

})

// Variable is a named expression for composition.
type Variable struct {
	Name       string
	Expression string
}

func (v *Variable) GetExpression() string {
	return v.Expression
}

func (v *Variable) ReturnTypes() []*celgo.Type {
	return []*celgo.Type{celgo.AnyType, celgo.DynType}
}

func (v *Variable) GetName() string {
	return v.Name
}

func convertV1Variables(variables []admissionregistrationv1.Variable) []cel.NamedExpressionAccessor {
	namedExpressions := make([]cel.NamedExpressionAccessor, len(variables))
	for i, variable := range variables {
		namedExpressions[i] = &Variable{Name: variable.Name, Expression: variable.Expression}
	}
	return namedExpressions
}

// ValidationCondition contains the inputs needed to compile, evaluate and validate a cel expression
type ValidationCondition struct {
	Expression string
	Message    string
	Reason     *metav1.StatusReason
}

func (v *ValidationCondition) GetExpression() string {
	return v.Expression
}

func (v *ValidationCondition) ReturnTypes() []*celgo.Type {
	return []*celgo.Type{celgo.BoolType}
}

func convertV1Validation(validation admissionregistrationv1.Validation) cel.ExpressionAccessor {
	return &ValidationCondition{
		Expression: validation.Expression,
		Message:    validation.Message,
		Reason:     validation.Reason,
	}
}

// newObjectInterfacesForTest returns an ObjectInterfaces appropriate for test cases in this file.
func newObjectInterfacesForTest() admission.ObjectInterfaces {
	scheme := runtime.NewScheme()
	corev1.AddToScheme(scheme)
	return admission.NewObjectInterfacesFromScheme(scheme)
}

func injectVariablesToValidations(validations []admissionregistrationv1.Validation, variables []admissionregistrationv1.Variable) {
	for idx, _ := range validations {
		for _, variable := range variables {
			validations[idx].Expression = strings.ReplaceAll(validations[idx].Expression, "variables."+variable.Name, variable.Expression)
		}
	}
}

func compileValidations(validations []admissionregistrationv1.Validation, celCompiler *cel.CompositedCompiler) cel.Filter {
	var expressions []cel.ExpressionAccessor
	for _, validation := range validations {
		expressions = append(expressions, convertV1Validation(validation))
	}
	options := cel.OptionalVariableDeclarations{
		HasParams:     false,
		HasAuthorizer: false,
	}
	mode := environment.NewExpressions
	return celCompiler.FilterCompiler.Compile(expressions, options, mode)
}

func setNodeUpdateAttribute(oldNode, newNode *corev1.Node) (*admission.VersionedAttributes, error) {
	nodeAttribiute := admission.NewAttributesRecord(
		oldNode,
		newNode,
		corev1.SchemeGroupVersion.WithKind("Node"),
		corev1.NamespaceAll,
		oldNode.Name,
		corev1.SchemeGroupVersion.WithResource("nodes"),
		"",
		admission.Update,
		&metav1.CreateOptions{},
		false,
		nil,
	)
	return admission.NewVersionedAttributes(nodeAttribiute, nodeAttribiute.GetKind(), newObjectInterfacesForTest())
}

type Option func(node *corev1.Node)

func newNode(nodeName string, options ...Option) *corev1.Node {
	node := &corev1.Node{
		ObjectMeta: metav1.ObjectMeta{
			Name:        nodeName,
			Labels:      map[string]string{"label1": "val1"},
			Annotations: map[string]string{"annotations1": "val1"},
		},
		Spec: corev1.NodeSpec{},
	}

	for _, f := range options {
		f(node)
	}

	return node
}

func withOwnerReference(ownerName, ownerUID string) Option {
	return func(node *corev1.Node) {
		if ownerUID != "" && ownerName != "" {
			node.ObjectMeta.OwnerReferences = append(node.ObjectMeta.OwnerReferences, metav1.OwnerReference{
				APIVersion: "v1",
				Kind:       "Pod",
				Name:       ownerName,
				UID:        types.UID(ownerUID),
			})
		}
	}
}

func withAnnotations(annotations map[string]string) Option {
	return func(node *corev1.Node) {
		if node.Annotations == nil {
			node.ObjectMeta.Annotations = annotations
		} else {
			for annotation, value := range annotations {
				node.ObjectMeta.Annotations[annotation] = value
			}
		}
	}
}

func withLabels(labels map[string]string) Option {
	return func(node *corev1.Node) {
		if node.Labels == nil {
			node.ObjectMeta.Labels = labels
		} else {
			for label, value := range labels {
				node.ObjectMeta.Labels[label] = value
			}
		}
	}
}

func withUnschedulable(unschedulable bool) Option {
	return func(node *corev1.Node) {
		node.Spec.Unschedulable = unschedulable
	}
}
