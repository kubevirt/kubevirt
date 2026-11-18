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

package virtio_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/converter/virtio"
)

var _ = Describe("InterpretTransitionalModelType", func() {
	DescribeTable("should return the correct model type",
		func(useVirtioTransitional *bool, arch, expectedModel string) {
			Expect(virtio.InterpretTransitionalModelType(useVirtioTransitional, arch)).To(Equal(expectedModel))
		},
		Entry("amd64 with transitional enabled", new(true), "amd64", "virtio-transitional"),
		Entry("amd64 with transitional disabled", new(false), "amd64", "virtio-non-transitional"),
		Entry("amd64 with nil", nil, "amd64", "virtio-non-transitional"),
		Entry("arm64 with transitional enabled", new(true), "arm64", "virtio-transitional"),
		Entry("arm64 with transitional disabled", new(false), "arm64", "virtio-non-transitional"),
		Entry("arm64 with nil", nil, "arm64", "virtio-non-transitional"),
		Entry("s390x with transitional enabled", new(true), "s390x", "virtio"),
		Entry("s390x with transitional disabled", new(false), "s390x", "virtio"),
		Entry("s390x with nil", nil, "s390x", "virtio"),
	)
})
