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
 * Copyright 2017 Red Hat, Inc.
 *
 */

package testutils_test

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"kubevirt.io/kubevirt/pkg/testutils"
)

type myCustomType struct {
	s   string
	n   int
	f   float32
	arr []string
}

var _ = Describe("BeIn", func() {
	Context("when passed a supported type", func() {
		It("should do the right thing", func() {
			Expect(2).Should(testutils.BeIn([2]int{1, 2}))
			Expect(3).ShouldNot(testutils.BeIn([2]int{1, 2}))

			Expect(2).Should(testutils.BeIn([]int{1, 2}))
			Expect(3).ShouldNot(testutils.BeIn([]int{1, 2}))

			Expect(2).Should(testutils.BeIn(1, 2))
			Expect(3).ShouldNot(testutils.BeIn(1, 2))

			Expect("abc").Should(testutils.BeIn("abc"))
			Expect("abc").ShouldNot(testutils.BeIn("def"))

			Expect("abc").ShouldNot(testutils.BeIn())
			Expect(7).ShouldNot(testutils.BeIn(nil))

			Expect(2).Should(testutils.BeIn(map[string]int{"foo": 1, "bar": 2}))
			Expect(3).ShouldNot(testutils.BeIn(map[int]int{3: 1, 4: 2}))

			arr := make([]myCustomType, 2)
			arr[0] = myCustomType{s: "foo", n: 3, f: 2.0, arr: []string{"a", "b"}}
			arr[1] = myCustomType{s: "foo", n: 3, f: 2.0, arr: []string{"a", "c"}}
			Expect(myCustomType{s: "foo", n: 3, f: 2.0, arr: []string{"a", "b"}}).Should(testutils.BeIn(arr))
			Expect(myCustomType{s: "foo", n: 3, f: 2.0, arr: []string{"b", "c"}}).ShouldNot(testutils.BeIn(arr))
		})
	})

	Context("when passed a correctly typed nil", func() {
		It("should operate succesfully on the passed in value", func() {
			var nilSlice []int
			Expect(1).ShouldNot(testutils.BeIn(nilSlice))

			var nilMap map[int]string
			Expect("foo").ShouldNot(testutils.BeIn(nilMap))
		})
	})

	Context("when passed an unsupported type", func() {
		It("should error", func() {
			success, err := (&testutils.BeInMatcher{Elements: []interface{}{0}}).Match(nil)
			Expect(success).Should(BeFalse())
			Expect(err).Should(HaveOccurred())

			success, err = (&testutils.BeInMatcher{Elements: nil}).Match(nil)
			Expect(success).Should(BeFalse())
			Expect(err).Should(HaveOccurred())
		})
	})
})
