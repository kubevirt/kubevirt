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

package hypervisor

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	v1 "kubevirt.io/api/core/v1"

	"kubevirt.io/kubevirt/pkg/hypervisor/kvm"
)

var _ = Describe("NewLauncherHypervisorResources", func() {
	DescribeTable("should return correct hypervisor resources", func(hypervisorType string, expectedType interface{}) {
		renderer := NewLauncherHypervisorResources(hypervisorType)
		Expect(renderer).To(BeAssignableToTypeOf(expectedType))
	},
		Entry("should return KVM renderer for empty hypervisor", "", (*kvm.KvmLauncherHypervisorResources)(nil)),
		Entry("should return KVM renderer for KVM hypervisor", v1.KvmHypervisorName, (*kvm.KvmLauncherHypervisorResources)(nil)),
	)

	It("should panic for unsupported hypervisor", func() {
		Expect(func() {
			NewLauncherHypervisorResources("unsupported")
		}).To(Panic())
	})
})
