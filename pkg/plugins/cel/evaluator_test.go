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

package cel_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	celgo "github.com/google/cel-go/cel"
	k8sv1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	v1 "kubevirt.io/api/core/v1"

	"kubevirt.io/kubevirt/pkg/plugins/cel"
)

var _ = Describe("CEL Evaluator", func() {
	var (
		evaluator *cel.Evaluator
		vmi       *v1.VirtualMachineInstance
	)

	BeforeEach(func() {
		var err error
		evaluator, err = cel.NewEvaluator()
		Expect(err).ToNot(HaveOccurred())

		vmi = &v1.VirtualMachineInstance{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test-vmi",
				Namespace: "default",
				Labels: map[string]string{
					"app": "test",
				},
			},
			Spec: v1.VirtualMachineInstanceSpec{
				Domain: v1.DomainSpec{
					CPU: &v1.CPU{
						Cores: 4,
					},
					Resources: v1.ResourceRequirements{
						Requests: k8sv1.ResourceList{
							k8sv1.ResourceMemory: resource.MustParse("512Mi"),
						},
					},
				},
			},
		}
	})

	Context("VMI field access", func() {
		It("should evaluate VMI name", func() {
			result, err := evaluator.EvaluateCondition(
				`vmi.metadata.name == "test-vmi"`,
				map[string]any{"vmi": vmi},
			)
			Expect(err).ToNot(HaveOccurred())
			Expect(result).To(BeTrue())
		})

		It("should evaluate CPU cores", func() {
			result, err := evaluator.EvaluateCondition(
				`vmi.spec.domain.cpu.cores > 2`,
				map[string]any{"vmi": vmi},
			)
			Expect(err).ToNot(HaveOccurred())
			Expect(result).To(BeTrue())
		})

		It("should evaluate labels", func() {
			result, err := evaluator.EvaluateCondition(
				`"app" in vmi.metadata.labels`,
				map[string]any{"vmi": vmi},
			)
			Expect(err).ToNot(HaveOccurred())
			Expect(result).To(BeTrue())
		})

		It("should return false for non-matching condition", func() {
			result, err := evaluator.EvaluateCondition(
				`vmi.spec.domain.cpu.cores < 2`,
				map[string]any{"vmi": vmi},
			)
			Expect(err).ToNot(HaveOccurred())
			Expect(result).To(BeFalse())
		})
	})

	Context("Compile-time validation", func() {
		It("should detect field typos at compile time", func() {
			err := evaluator.CompileCondition(`vmi.spec.doman.cpu.cores > 2`)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("compilation failed"))
		})

		It("should detect invalid syntax", func() {
			err := evaluator.CompileCondition(`vmi.spec.domain.cpu.cores >`)
			Expect(err).To(HaveOccurred())
		})

		It("should reject non-bool expressions", func() {
			err := evaluator.CompileCondition(`vmi.metadata.name`)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("must return bool"))
		})

		It("should accept valid expressions", func() {
			err := evaluator.CompileCondition(`vmi.spec.domain.cpu.cores > 0`)
			Expect(err).ToNot(HaveOccurred())
		})
	})

	Context("Expression caching", func() {
		It("should cache compiled expressions", func() {
			expr := `vmi.spec.domain.cpu.cores > 2`
			vars := map[string]any{"vmi": vmi}

			result1, err := evaluator.EvaluateCondition(expr, vars)
			Expect(err).ToNot(HaveOccurred())

			result2, err := evaluator.EvaluateCondition(expr, vars)
			Expect(err).ToNot(HaveOccurred())

			Expect(result1).To(Equal(result2))
		})

		It("should produce correct results with different inputs on cached expression", func() {
			expr := `vmi.spec.domain.cpu.cores > 2`

			result, err := evaluator.EvaluateCondition(expr, map[string]any{"vmi": vmi})
			Expect(err).ToNot(HaveOccurred())
			Expect(result).To(BeTrue())

			smallVMI := vmi.DeepCopy()
			smallVMI.Spec.Domain.CPU.Cores = 1
			result, err = evaluator.EvaluateCondition(expr, map[string]any{"vmi": smallVMI})
			Expect(err).ToNot(HaveOccurred())
			Expect(result).To(BeFalse())
		})
	})

	Context("Empty expression", func() {
		It("should return true for empty expression", func() {
			result, err := evaluator.EvaluateCondition("", map[string]any{"vmi": vmi})
			Expect(err).ToNot(HaveOccurred())
			Expect(result).To(BeTrue())
		})
	})

	Context("Custom variables", func() {
		It("should support additional variables via WithVariable", func() {
			eval, err := cel.NewEvaluator(
				cel.WithVariable("threshold", celgo.IntType),
			)
			Expect(err).ToNot(HaveOccurred())

			result, err := eval.EvaluateCondition(
				`vmi.spec.domain.cpu.cores > threshold`,
				map[string]any{"vmi": vmi, "threshold": int64(2)},
			)
			Expect(err).ToNot(HaveOccurred())
			Expect(result).To(BeTrue())
		})
	})

	Context("Error cases", func() {
		It("should return error for non-bool result at evaluation time", func() {
			eval, err := cel.NewEvaluator(
				cel.WithVariable("x", celgo.DynType),
			)
			Expect(err).ToNot(HaveOccurred())

			_, err = eval.EvaluateCondition(`x`, map[string]any{"vmi": vmi, "x": "not-a-bool"})
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("must return bool"))
		})

		It("should return error for missing variable at evaluation time", func() {
			_, err := evaluator.EvaluateCondition(
				`vmi.spec.domain.cpu.cores > 0`,
				map[string]any{},
			)
			Expect(err).To(HaveOccurred())
		})
	})
})
