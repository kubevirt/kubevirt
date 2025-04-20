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

var _ = Describe("Owner", func() {

	var toNilPointer *v1.Pod = nil

	var ownedPod = func(ownerReferences []metav1.OwnerReference) *v1.Pod {
		return &v1.Pod{
			ObjectMeta: metav1.ObjectMeta{
				OwnerReferences: ownerReferences,
			},
		}
	}

	DescribeTable("should", func(pod interface{}, match bool) {
		success, err := HaveOwners().Match(pod)
		Expect(err).ToNot(HaveOccurred())
		Expect(success).To(Equal(match))
		Expect(HaveOwners().FailureMessage(pod)).ToNot(BeEmpty())
		Expect(HaveOwners().NegatedFailureMessage(pod)).ToNot(BeEmpty())
	},
		Entry("with an owner present report it as present", ownedPod([]metav1.OwnerReference{{}}), true),
		Entry("with no owner present report it as missing", ownedPod([]metav1.OwnerReference{}), false),
		Entry("cope with a nil pod", nil, false),
		Entry("cope with an object pointing to nil", toNilPointer, false),
		Entry("cope with an object which has nil as owners array", ownedPod(nil), false),
	)
})
