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
 */

package matcher

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var _ = Describe("Existence matchers", func() {

	var toNilPointer *v1.Pod = nil

	DescribeTable("should detect with the positive matcher", func(obj interface{}, existence bool) {
		exists, err := Exist().Match(obj)
		Expect(err).ToNot(HaveOccurred())
		Expect(exists).To(Equal(existence))
		Expect(Exist().FailureMessage(obj)).ToNot(BeEmpty())
		Expect(Exist().NegatedFailureMessage(obj)).ToNot(BeEmpty())
	},
		Entry("a nil object", nil, false),
		Entry("a pod", &v1.Pod{}, true),
	)
	DescribeTable("should detect with the negative matcher", func(obj interface{}, existence bool) {
		exists, err := BeGone().Match(obj)
		Expect(err).ToNot(HaveOccurred())
		Expect(exists).To(Equal(existence))
		Expect(BeGone().FailureMessage(obj)).ToNot(BeEmpty())
		Expect(BeGone().NegatedFailureMessage(obj)).ToNot(BeEmpty())
	},
		Entry("the existence of a set of pods", []*v1.Pod{{}, {}}, false),
		Entry("the absence of a set of pods", []*v1.Pod{}, true),
		Entry("a nil object", nil, true),
		Entry("an object pointing to nil", toNilPointer, true),
		Entry("a pod", &v1.Pod{}, false),
	)

	It("formating", func() {
		obj := &v1.Pod{
			ObjectMeta: metav1.ObjectMeta{
				Name: "HI",
				ManagedFields: []metav1.ManagedFieldsEntry{
					{
						Manager: "something",
					},
				},
			},
		}
		Expect(BeGone().FailureMessage(obj.DeepCopy())).To(
			SatisfyAll(
				ContainSubstring("metadata"),
				ContainSubstring("status"),
				Not(ContainSubstring("something")),
				Not(ContainSubstring("Spec")),
			),
		)
		Expect(BeGone().NegatedFailureMessage(obj.DeepCopy())).To(
			SatisfyAll(
				ContainSubstring("metadata"),
				ContainSubstring("status"),
				Not(ContainSubstring("something")),
				Not(ContainSubstring("Spec")),
			),
		)
	})
})
