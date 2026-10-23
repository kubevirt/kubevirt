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

package main

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("infraReplicasToUint8", func() {
	DescribeTable("should convert values that fit in a uint8", func(replicas uint, expected uint8) {
		got, err := infraReplicasToUint8(replicas)
		Expect(err).ToNot(HaveOccurred())
		Expect(got).To(Equal(expected))
	},
		Entry("with the minimum non-zero value", uint(1), uint8(1)),
		Entry("with the maximum uint8 value", uint(255), uint8(255)),
	)

	DescribeTable("should reject values above 255", func(replicas uint) {
		_, err := infraReplicasToUint8(replicas)
		Expect(err).To(HaveOccurred())
	},
		Entry("with 256, which would wrap around to 0", uint(256)),
		Entry("with 1000", uint(1000)),
	)
})
