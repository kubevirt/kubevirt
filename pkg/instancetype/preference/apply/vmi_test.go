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

package apply

import (
	"fmt"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	virtv1 "kubevirt.io/api/core/v1"
	v1beta1 "kubevirt.io/api/instancetype/v1beta1"

	"kubevirt.io/kubevirt/pkg/pointer"
)

var _ = Describe("Preference spec applier", func() {
	var entries []TableEntry
	for _, applier := range append(NewSpec().appliers, NewSpec()) {
		entries = append(entries, Entry(fmt.Sprintf("%T", applier), applier))
	}

	DescribeTable("should ignore nil preference specs without modifying the target", func(applier preferenceApply) {
		applier.Apply(nil, nil, nil)

		vmi := &virtv1.VirtualMachineInstance{}
		expected := vmi.DeepCopy()
		applier.Apply(nil, &vmi.Spec, &vmi.ObjectMeta)
		Expect(vmi).To(Equal(expected))
	}, entries)
})

var _ = Describe("VMI preference applier", func() {
	It("should return early for nil preferences without dereferencing the VMI", func() {
		NewVMI().Apply(nil, nil)
	})

	It("should apply preferences to the VMI spec and metadata", func() {
		preferenceSpec := &v1beta1.VirtualMachinePreferenceSpec{
			PreferredSubdomain: pointer.P("preferred-subdomain"),
			Annotations:        map[string]string{"preferred-annotation": "value"},
		}
		vmi := &virtv1.VirtualMachineInstance{}
		NewVMI().Apply(preferenceSpec, vmi)

		Expect(vmi.Spec.Subdomain).To(Equal(*preferenceSpec.PreferredSubdomain))
		Expect(vmi.Annotations).To(Equal(preferenceSpec.Annotations))
	})

	It("should ignore nil preferences when applying auto-attach preferences", func() {
		ApplyAutoAttachPreferences(nil, nil)

		vmiSpec := &virtv1.VirtualMachineInstanceSpec{}
		expected := vmiSpec.DeepCopy()
		ApplyAutoAttachPreferences(nil, vmiSpec)
		Expect(vmiSpec).To(Equal(expected))
	})
})
