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

package cpudedicated_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	v1 "kubevirt.io/api/core/v1"

	"kubevirt.io/kubevirt/pkg/libvmi"
	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/api"
	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/cpudedicated"
)

var _ = Describe("GenerateDomainForTargetCPUSetAndTopology", func() {
	It("should return an error when MigrationState is nil", func() {
		vmi := libvmi.New(libvmi.WithDedicatedCPUPlacement())
		// MigrationState is nil by default on a freshly created VMI
		Expect(vmi.Status.MigrationState).To(BeNil())

		domSpec := &api.DomainSpec{}
		_, err := cpudedicated.GenerateDomainForTargetCPUSetAndTopology(vmi, domSpec)
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring("migration state is not initialized"))
	})
})
