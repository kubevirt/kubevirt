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
	"fmt"

	celgo "github.com/google/cel-go/cel"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"k8s.io/apiserver/pkg/cel/environment"

	admissionregistrationv1 "k8s.io/api/admissionregistration/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apiserver/pkg/admission/plugin/cel"

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
				compositionEnvTemplateWithoutStrictCost, err := cel.NewCompositionEnv(cel.VariablesTypeName, environment.MustBaseEnvSet(environment.DefaultCompatibilityVersion()))
				Expect(err).ToNot(HaveOccurred())
				celCompiler = cel.NewCompositedCompilerFromTemplate(compositionEnvTemplateWithoutStrictCost)
			})

			It("succeed compiling all the policy validations", func() {
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
