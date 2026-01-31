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

var _ = Describe("Test NewLauncherResourceRenderer", func() {
	It("should return KVM renderer for unknown hypervisor", func() {
		renderer := NewLauncherResourceRenderer("unknownHypervisor")
		_, ok := renderer.(*kvm.KvmLauncherResourceRenderer)
		Expect(ok).To(BeTrue())
	})
	It("should return KVM renderer for KVM hypervisor", func() {
		renderer := NewLauncherResourceRenderer(v1.KvmHypervisorName)
		_, ok := renderer.(*kvm.KvmLauncherResourceRenderer)
		Expect(ok).To(BeTrue())
	})
})
